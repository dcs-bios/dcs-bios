package shmmodule

import (
	"github.com/hidez8891/shm"
	lua "github.com/yuin/gopher-lua"
)

var sharedMemoryAreas map[string]*shm.Memory

// Loader is called by luastate.go to provide the "hub" module
// to the Lua environment.
func Loader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), exports)
	L.Push(mod)
	return 1
}

func Preload(L *lua.LState) {
	L.PreloadModule("shm", Loader)
}

var exports = map[string]lua.LGFunction{
	"create": shmCreate,
	"write":  shmWrite,
	"close":  shmClose,
}

func Reset() {
	for _, v := range sharedMemoryAreas {
		v.Close()
	}
	sharedMemoryAreas = make(map[string]*shm.Memory)
}

func shmCreate(L *lua.LState) int {
	name := L.ToString(1)
	size := int32(L.ToInt(2))

	if _, ok := sharedMemoryAreas[name]; ok {
		L.Push(lua.LFalse)
		L.Push(lua.LString("already exists"))
		return 2
	}

	s, err := shm.Create(name, size)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	sharedMemoryAreas[name] = s
	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

func shmClose(L *lua.LState) int {
	name := L.ToString(1)
	mem, ok := sharedMemoryAreas[name]
	if !ok {
		L.Push(lua.LFalse)
		return 1
	}
	mem.Close()
	delete(sharedMemoryAreas, name)
	L.Push(lua.LTrue)
	return 1
}

func shmWrite(L *lua.LState) int {
	name := L.ToString(1)
	offset := L.ToInt64(2)
	value := L.ToString(3)

	mem, ok := sharedMemoryAreas[name]
	if !ok {
		L.Push(lua.LFalse)
		return 1
	}

	n, err := mem.WriteAt([]byte(value), offset)
	L.Push(lua.LNumber(n))
	if err != nil {
		L.Push(lua.LString(err.Error()))
	} else {
		L.Push(lua.LNil)
	}
	return 2
}
