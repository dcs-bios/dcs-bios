package luastate

import (
	"strings"

	"dcs-bios.a10c.de/dcs-bios-hub/exportdataparser"
	lua "github.com/yuin/gopher-lua"
)

// SimDataBuffer is a pointer to the DataBuffer that stores the most recent
// cockpit state received from DCS: World.
// This will be set from dcs-bios-hub.go.
var SimDataBuffer *exportdataparser.DataBuffer

// ExportDataBuffer is a pointer to the DataBuffer that holds the data
// sent to Arduino boards connected over serial ports.
// This will be set from dcs-bios-hub.go.
var ExportDataBuffer *exportdataparser.DataBuffer

// SimCommandChannel is used to send commands triggered by Lua.
// It is monitored in dcs-bios-hub.go.
var SimCommandChannel = make(chan string, 10)

var inputCallbacks []*lua.LFunction
var exportDataCallbacks []*lua.LFunction

// Loader is called by luastate.go to provide the "hub" module
// to the Lua environment.
func Loader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), exports)
	L.Push(mod)
	return 1
}

var exports = map[string]lua.LGFunction{
	"getSimString":           getSimString,
	"getSimInteger":          getSimInteger,
	"setPanelString":         setPanelString,
	"setPanelInteger":        setPanelInteger,
	"registerInputCallback":  registerInputCallback,
	"registerOutputCallback": registerExportDataCallback,
	"sendSimCommand":         sendSimCommand,
	"clearCallbacks":         clearCallbacks,
}

// NotifyInputCallbacks passes the given command string to the
// callbacks registered from Lua. Returns true if the command was
// handled by a callback function and should not be passed on to DCS.
func NotifyInputCallbacks(cmdString string) (handledByLua bool) {
	parts := strings.Split(cmdString, " ")
	if len(parts) != 2 {
		return false
	}
	cmd := parts[0]
	arg := parts[1]

	handledByLua = false
	WithLuaStateDo(func(L *lua.LState) {
		for _, cb := range inputCallbacks {
			L.CallByParam(lua.P{
				Fn:      cb,
				NRet:    1,
				Protect: true,
			}, lua.LString(cmd), lua.LString(arg))
			returnValue := L.Get(-1)
			L.Pop(1)
			if lua.LVAsBool(returnValue) {
				handledByLua = true
				return
			}
		}
	})
	return handledByLua
}

func NotifyOutputCallbacks() {
	WithLuaStateDo(func(L *lua.LState) {
		for _, cb := range exportDataCallbacks {
			L.CallByParam(lua.P{
				Fn:      cb,
				NRet:    0,
				Protect: true,
			})
		}
	})
}

// getSimString takes a "module/element" identifier
// and returns the last string value for this identifer
// that was sent by DCS. Returns the empty string if not
// successful (e.g. the identifier was not found).
func getSimString(L *lua.LState) int {
	id := L.ToString(1)
	value := SimDataBuffer.GetCStringValue(id)
	L.Push(lua.LString(value))
	return 1
}

// getSimString takes a "module/element" identifier
// and returns the last integer value for this identifer
// that was sent by DCS. Returns -1 if not
// successful (e.g. the identifier was not found).
func getSimInteger(L *lua.LState) int {
	id := L.ToString(1)
	value := SimDataBuffer.GetIntegerValue(id)
	L.Push(lua.LNumber(value))
	return 1
}

// setPanelString sets a string value that is sent to
// the physical cockpit.
// It takes a "module/element" identifier as its first
// argument and the new string value as the second argument.
// Returns true if the value was set, false if the identifier
// was invalid.
func setPanelString(L *lua.LState) int {
	id := L.ToString(1)
	value := L.ToString(2)

	ok := ExportDataBuffer.SetCStringValue(id, value)
	L.Push(lua.LBool(ok))
	return 1
}

// setPanelInteger sets an integer value that is sent to
// the physical cockpit.
// It takes a "module/element" identifier as its first
// argument and the new integer value as the second argument.
// Returns true if the value was set, false if the identifier
// was invalid.
func setPanelInteger(L *lua.LState) int {
	id := L.ToString(1)
	value := L.ToInt(2)

	ok := ExportDataBuffer.SetIntegerValue(id, value)
	L.Push(lua.LBool(ok))
	return 1
}

// sendSimCommand(cmd, arg)
// queues a command to be sent to DCS. The command buffer
// holds 10 commands. If the command does not fit, returns false
// and ignores the command.
func sendSimCommand(L *lua.LState) int {
	cmd := L.ToString(1)
	arg := L.ToString(2)

	select {
	case SimCommandChannel <- cmd + " " + arg:
		L.Push(lua.LTrue)
	default:
		L.Push(lua.LFalse)
	}

	return 1
}

// registerInputCallback registers a function to be called
// whenever a command arrives from the physical panels.
// The function will be called with two arguments: command, argument
// If the function returns true, the command will not be passed on to DCS
// or to other callback functions.
func registerInputCallback(L *lua.LState) int {
	fn := L.ToFunction(1)
	inputCallbacks = append(inputCallbacks, fn)
	return 0
}

// registerExportDataCallback registers a function to be called
// every time new export data is available.
// This function can use hub.getSimString() and hub.setPanelString()
// to remap export data.
func registerExportDataCallback(L *lua.LState) int {
	fn := L.ToFunction(1)
	exportDataCallbacks = append(exportDataCallbacks, fn)
	return 0
}

// clearCallbacks unregisters all input and export data callbacks.
func clearCallbacks(L *lua.LState) int {
	inputCallbacks = nil
	exportDataCallbacks = nil
	return 0
}
