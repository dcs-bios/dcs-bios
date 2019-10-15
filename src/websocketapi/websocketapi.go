/*
The websocketapi package provides a WebSocket API to watch
the hub configuration and the data exported from the simulator.

Work in progress.
*/
package websocketapi

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"dcs-bios.a10c.de/dcs-bios-hub/gui"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// if external network access is enabled, allow any origin
		if gui.IsExternalNetworkAccessEnabled() {
			return true
		}

		// otherwise, only allow localhost
		origin, err := url.Parse(r.Header.Get("Origin"))
		if err != nil {
			return false
		}

		return origin.Hostname() == "localhost" || origin.Hostname() == "127.0.0.1"
	},
}

type WebSocketCommand interface {
	Handle()
}

var JsonApi *jsonapi.JsonApi

// AddHandler adds a handler for the request path /hubapi/websocket to the default net.http ServeMux
func AddHandler() {
	http.HandleFunc("/api/websocket", wsHandler)
}

type SetSerialPortStateMessage struct {
	PortName               string `json:portName`
	DesiredConnectionState bool
}

func (m *SetSerialPortStateMessage) Handle() {
	fmt.Printf("setting serial port prefs for %s\n", m.PortName)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade websocket request: %s\n", err)
		return
	}
	if JsonApi == nil {
		log.Printf("websocket api: JsonApi instance not set\n")
		return
	}

	_, wsData, wsError := conn.ReadMessage()
	if wsError != nil {
		log.Printf("websocket api: cannot read first message: %s\n", wsError.Error())
		return
	}

	followupJson := make(chan []byte)
	responses, callError := JsonApi.HandleApiCall(wsData, followupJson)
	if callError != nil {
		fmt.Printf("api call error: %s\n", callError)
		return
	}
	go func() {
		for resp := range responses {
			conn.WriteMessage(websocket.TextMessage, resp)
		}
	}()
	go func() {
		for {
			msgType, fmsg, followupErr := conn.ReadMessage()
			if followupErr != nil {
				// WebSocket was closed
				break
			}
			fmt.Println("websocket followup msg received:", msgType)
			followupJson <- fmsg
		}
		close(followupJson)
	}()
}
