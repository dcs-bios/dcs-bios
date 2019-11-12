// Package luastate holds the central LuaState for user scripting
// and provides a goroutine-safe API to access it.
package luastate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"dcs-bios.a10c.de/dcs-bios-hub/configstore"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
	"dcs-bios.a10c.de/dcs-bios-hub/luastate/shmmodule"
	"github.com/nubix-io/gluabit32"
	lua "github.com/yuin/gopher-lua"
)

var luaState = lua.NewState()
var luaLock sync.Mutex

type ScriptListEntry struct {
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

var scriptList []ScriptListEntry

var scriptListSubscriptions map[chan []ScriptListEntry]bool = make(map[chan []ScriptListEntry]bool)

// Reset creates a new Lua state and executes all user lua scripts
func Reset(logBuffer io.Writer) {
	luaLock.Lock()
	defer luaLock.Unlock()

	shmmodule.Reset()

	luaState.Close()
	inputCallbacks = nil
	outputCallbacks = nil

	luaState = lua.NewState()
	luaState.PreloadModule("hub", Loader)
	shmmodule.Preload(luaState)
	gluabit32.Preload(luaState)
	err := luaState.DoString(`hub = require("hub")
local _FILEENV = {}
function setFileEnv(name, env)
	_FILEENV[name] = env
end
function enterEnv(name)
	for filename, env in pairs(_FILEENV) do
		if filename:sub(filename:len() - name:len() + 1):lower() == name:lower() then
			setfenv(2, env)
			return true
		end
	end
	return false
end`)
	if err != nil {
		panic(err)
	}

	configstore.Load("scriptlist.json", &scriptList)

	for _, script := range scriptList {
		if !script.Enabled {
			continue
		}

		stat, err := os.Stat(script.Path)
		if err != nil || stat.IsDir() {
			fmt.Fprintf(logBuffer, "file not found: %s\n", script.Path)
			continue
		}

		fmt.Fprintf(logBuffer, "loading: %s\n", script.Path)
		scriptDir := filepath.Dir(script.Path) + string(os.PathSeparator)

		err = luaState.DoString(`
		local path=[[` + script.Path + `]]
		local newgt = {}
		newgt["_G"] = newgt
		newgt["_SCRIPTDIR"] = [[` + scriptDir + `]]
		
		for _, varname in pairs({
			"package",
			"_VERSION",
			"_GOPHER_LUA_VERSION",
			"next",
			"getfenv",
			"setfenv",
			"setmetatable",
			"unpack",
			"rawget",
			"select",
			"assert",
			"loadfile",
			"require",
			"getmetatable",
			"print",
			"type",
			"dofile",
			"error",
			"load",
			"xpcall",
			"tonumber",
			"rawequal",
			"rawset",
			"setfenv",
			"tostring",
			"module",
			"loadstring",
			"pcall",
			"ipairs",
			"pairs",
			"table",
			"io",
			"os",
			"string",
			"math",
			"debug",
			"channel",
			"coroutine",
			"hub"
		}) do newgt[varname] = _G[varname] end

		newgt["loadstring"] = function(s)
			local f, err = loadstring(s)
			if f == nil then
				return f, err
			end
			setfenv(f, newgt)
			return f, err
		end
		newgt["loadfile"] = function(path)
			local f, err = loadfile(path)
			if f == nil then
				return f, err
			end
			setfenv(f, newgt)
			return f, err
		end
		newgt["dofile"] = function(path)
			local f, err = newgt["loadfile"](path)
			if err ~= nil then error(err) end
			return f()
		end

		local f = loadfile(path)
		setFileEnv(path:lower(), newgt)
		setfenv(f, newgt)
		f()
		`)
		if err != nil {
			fmt.Fprintf(logBuffer, "lua error: %v\n", err)
		}
	}
	luaState.Options.IncludeGoStackTrace = true

}

type ReloadHooksLuaRequest struct{}

func HandleReloadHooksLuaRequest(req *ReloadHooksLuaRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)

	logBuffer := bytes.NewBuffer([]byte{})
	Reset(logBuffer)
	responseCh <- jsonapi.SuccessResult{
		Message: logBuffer.String(),
	}
}

type MonitorScriptListRequest struct{}
type ScriptList []ScriptListEntry

func HandleMonitorScriptListRequest(req *MonitorScriptListRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	subscription := make(chan []ScriptListEntry)
	go func() {
		for s := range subscription {
			select {
			case responseCh <- ScriptList(s):
			case _, ok := <-followupCh:
				if !ok {
					luaLock.Lock()
					defer luaLock.Unlock()
					delete(scriptListSubscriptions, subscription)
					return
				}
			}
		}
		close(responseCh)
	}()
	luaLock.Lock()
	scriptListSubscriptions[subscription] = true
	listCopy := make([]ScriptListEntry, len(scriptList))
	copy(listCopy, scriptList)
	subscription <- listCopy
	luaLock.Unlock()
}

// notifyScriptListSubscribers sends a copy of the UserLuaScript list to all subscribers.
// The caller must hold luaLock.
func notifyScriptListSubscribers() {
	listCopy := make([]ScriptListEntry, len(scriptList))
	copy(listCopy, scriptList)
	for subscription := range scriptListSubscriptions {
		select {
		case subscription <- listCopy:
		case <-time.After(200 * time.Millisecond):
		}
	}
}

type SetScriptListRequest ScriptList

func HandleSetScriptListRequest(req *SetScriptListRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	luaLock.Lock()
	defer luaLock.Unlock()

	scriptList = nil
	for _, item := range *req {
		scriptList = append(scriptList, item)
	}
	notifyScriptListSubscribers()
	configstore.Store("scriptlist.json", scriptList)

	responseCh <- jsonapi.SuccessResult{}
}

func RegisterJsonApiCalls(jsonAPI *jsonapi.JsonApi) {
	jsonAPI.RegisterType("reload_scripts", ReloadHooksLuaRequest{})
	jsonAPI.RegisterApiCall("reload_scripts", HandleReloadHooksLuaRequest)

	jsonAPI.RegisterType("monitor_script_list", MonitorScriptListRequest{})
	jsonAPI.RegisterApiCall("monitor_script_list", HandleMonitorScriptListRequest)
	jsonAPI.RegisterType("script_list", ScriptList(nil))

	jsonAPI.RegisterType("set_script_list", SetScriptListRequest{})
	jsonAPI.RegisterApiCall("set_script_list", HandleSetScriptListRequest)
}

// DoString executes a snippet of Lua code in the environment.
// If the code throws an error or cannot be parsed, the function
// returns the error. On success, nil is returned.
func DoString(code string) error {
	luaLock.Lock()
	defer luaLock.Unlock()

	err := luaState.CallByParam(lua.P{
		Fn:      luaState.GetGlobal("loadstring"),
		NRet:    1,
		Protect: true,
	}, lua.LString(code))

	if err != nil {
		return err
	}

	loadstringReturn := luaState.Get(-1)
	luaState.Pop(1)

	if loadstringReturn.Type() != lua.LTFunction {
		return errors.New(loadstringReturn.String())
	}

	err = luaState.CallByParam(lua.P{
		Fn:      loadstringReturn,
		NRet:    0,
		Protect: true,
	})
	if err != nil {
		return err
	}

	return nil
}

// DoStringAndSerializeResult executes a block of Lua code,
// and returns the Lua return value as a human readable string.
func DoStringAndSerializeResult(code string) (string, error) {
	luaLock.Lock()
	defer luaLock.Unlock()

	err := luaState.CallByParam(lua.P{
		Fn:      luaState.GetGlobal("loadstring"),
		NRet:    1,
		Protect: true,
	}, lua.LString(code))

	// err := luaState.DoString(code)
	if err != nil {
		return "", err
	}

	loadstringReturn := luaState.Get(-1)
	luaState.Pop(1)

	if loadstringReturn.Type() != lua.LTFunction {
		return "", errors.New(loadstringReturn.String())
	}

	err = luaState.CallByParam(lua.P{
		Fn:      loadstringReturn,
		NRet:    1,
		Protect: true,
	})
	if err != nil {
		return "", err
	}

	ret := luaState.Get(-1)
	luaState.Pop(1)

	// convert to string

	table := luaState.NewTable()
	table.RawSet(lua.LString("svalue"), ret)
	table.RawSetString("table", luaState.GetGlobal("table"))
	table.RawSetString("print", luaState.GetGlobal("print"))
	table.RawSetString("tostring", luaState.GetGlobal("tostring"))
	table.RawSetString("string", luaState.GetGlobal("string"))
	table.RawSetString("type", luaState.GetGlobal("type"))
	table.RawSetString("pairs", luaState.GetGlobal("pairs"))

	err = luaState.CallByParam(lua.P{
		Fn:      luaState.GetGlobal("loadstring"),
		NRet:    1,
		Protect: true,
	}, lua.LString(`
	local seenTables = {}
	local retlist = {}
	local indentLevel = 0
	local function serializeRecursive(value)
		if type(value) == "string" then return table.insert(retlist, string.format("%q", value)) end
		if type(value) ~= "table" then return table.insert(retlist, tostring(value)) end
			
		if seenTables[value] == true then
			   table.insert(retlist, tostring(value))
			return
		end
		seenTables[value] = true
		
		-- we have a table, iterate over the keys

		table.insert(retlist, "{\n")
		indentLevel = indentLevel + 4
		for k, v in pairs(value) do
			table.insert(retlist, string.rep(" ", indentLevel).."[")
			if type(k) == "table" then
				   table.insert(retlist, tostring(k))
			else
				serializeRecursive(k)
			end
			table.insert(retlist, "] = ")
			serializeRecursive(v)
			table.insert(retlist, ",\n")
		end
		indentLevel = indentLevel - 4
		table.insert(retlist, string.rep(" ", indentLevel).."}")
	end
	serializeRecursive(svalue, "    ")
	local stringResult = table.concat(retlist)
	return stringResult
	`))
	serializeFunction := luaState.Get(-1)
	luaState.Pop(1)

	if serializeFunction.Type() != lua.LTFunction {
		panic("not a function: " + serializeFunction.String())
	}

	luaState.SetFEnv(serializeFunction, table)

	err = luaState.CallByParam(lua.P{
		Fn:      serializeFunction,
		Protect: true,
		NRet:    1,
	})

	if err != nil {
		panic(err.Error())
	}
	ret = luaState.Get(-1)
	luaState.Pop(1)

	return ret.String(), nil
}

// DoFile executes a .lua file given a path to the file.
func DoFile(path string) error {
	luaLock.Lock()
	defer luaLock.Unlock()
	return luaState.DoFile(path)
}

// SetGlobal sets a global variable in the Lua state.
func SetGlobal(name string, value lua.LValue) {
	luaLock.Lock()
	defer luaLock.Unlock()
	luaState.SetGlobal(name, value)
}

// SetGlobalFunction makes a Go function available from Lua.
// The Go function must take a Lua state as an argument and
// return the number of return values it pushed onto the Lua stack.
// While the Go function is called, the luaLock mutex will be held.
func SetGlobalFunction(name string, goFunction lua.LGFunction) {
	luaLock.Lock()
	defer luaLock.Unlock()
	luaState.SetGlobal(name, luaState.NewFunction(goFunction))
}

// WithLuaStateDo locks the luaLock mutex,
// calls the given Go function with the lua.LState as an argument,
// then releases the luaLock mutex.
func WithLuaStateDo(action func(*lua.LState)) {
	luaLock.Lock()
	defer luaLock.Unlock()
	action(luaState)
}
