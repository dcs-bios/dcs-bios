// Package luastate holds the central LuaState for user scripting
// and provides a goroutine-safe API to access it.
package luastate

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
	lua "github.com/yuin/gopher-lua"
)

var luaState = lua.NewState()
var luaLock sync.Mutex

func init() {
	luaState.Options.IncludeGoStackTrace = true

	luaState.PreloadModule("hub", Loader)
	err := luaState.DoString(`hub = require("hub")`)
	if err != nil {
		panic(err)
	}
}

type ReloadHooksLuaRequest struct{}

func HandleReloadHooksLuaRequest(req *ReloadHooksLuaRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	// the Lua state that user-defined remapping scripts are executed in
	if err := DoFile("hooks.lua"); err != nil {
		workdir, getwderr := os.Getwd()
		if getwderr != nil {
			workdir = "(could not determine current directory)"
		}
		responseCh <- jsonapi.ErrorResult{
			Message: fmt.Sprintf("error loading hooks.lua from %s:\n%s\n", workdir, err.Error()),
		}
		return
	}
	responseCh <- jsonapi.SuccessResult{
		Message: "Reloaded hooks.lua",
	}
}

func RegisterJsonApiCalls(jsonAPI *jsonapi.JsonApi) {
	jsonAPI.RegisterType("reload_hooks_lua", ReloadHooksLuaRequest{})
	jsonAPI.RegisterApiCall("reload_hooks_lua", HandleReloadHooksLuaRequest)
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
