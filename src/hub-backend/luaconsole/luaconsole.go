package luaconsole

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"dcs-bios.a10c.de/dcs-bios-hub/gui"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

type LuaResult struct {
	Type   string `json:"type"`
	Result string `json:"result"`
	Status string `json:"status"`
}

type LuaConsoleServer struct {
	jsonAPI         *jsonapi.JsonApi
	requestToDcs    chan interface{}
	responseFromDcs chan LuaResult
	conn            net.Conn
	connLock        sync.Mutex
}

func NewServer(jsonAPI *jsonapi.JsonApi) *LuaConsoleServer {
	return &LuaConsoleServer{
		jsonAPI:         jsonAPI,
		requestToDcs:    make(chan interface{}),
		responseFromDcs: make(chan LuaResult),
	}
}

func (lcs *LuaConsoleServer) Run() {
	listener, err := net.Listen("tcp", "localhost:3001")
	if err != nil {
		fmt.Println("luaconsole: could not listen on port 3001")
		return
	}

	lcs.jsonAPI.RegisterType("execute_lua_snippet", ExecuteSnippetRequest{})
	lcs.jsonAPI.RegisterType("lua_result", LuaResult{})
	lcs.jsonAPI.RegisterApiCall("execute_lua_snippet", lcs.HandleExecuteSnippetRequest)

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("luaconsole: error accepting connection: " + err.Error())
			continue
		}
		if lcs.conn != nil {
			lcs.conn.Close()
		}
		lcs.conn = conn
		go lcs.handleConnection(conn)
	}
}

func (lcs *LuaConsoleServer) handleConnection(conn net.Conn) {
	fmt.Println("luaconsole: DCS has connected.")
	go func() {
		enc := json.NewEncoder(conn)
		enc.SetEscapeHTML(false)
		for {
			fmt.Println("reading from requestToDcs")
			req := <-lcs.requestToDcs
			fmt.Println("sending", req)
			if err := enc.Encode(req); err != nil {
				conn.Close()
				return
			}
		}
	}()

	go func() {
		dec := json.NewDecoder(conn)
		var dcsResponse LuaResult
		for {
			if err := dec.Decode(&dcsResponse); err != nil {
				conn.Close()
				return
			}
			if dcsResponse.Type != "ping" {
				lcs.responseFromDcs <- dcsResponse
			}
		}
	}()
}

type ExecuteSnippetRequest struct {
	LuaEnvironment string `json:"luaEnvironment"`
	LuaCode        string `json:"luaCode"`
}

func (lcs *LuaConsoleServer) HandleExecuteSnippetRequest(req *ExecuteSnippetRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	if !gui.IsLuaConsoleEnabled() {
		responseCh <- jsonapi.ErrorResult{Message: "The Lua Console is disabled."}
		return
	}

	request := map[string]string{
		"type":   "lua",
		"name":   "irrelevant",
		"luaenv": req.LuaEnvironment,
		"code":   req.LuaCode,
	}

	lcs.connLock.Lock()
	defer lcs.connLock.Unlock()
	select {
	case lcs.requestToDcs <- request:
		var result LuaResult = <-lcs.responseFromDcs
		responseCh <- result
	case <-time.After(1 * time.Second):
		responseCh <- jsonapi.ErrorResult{
			Message: "could not send snippet to DCS within 1 second.",
		}
	}
}
