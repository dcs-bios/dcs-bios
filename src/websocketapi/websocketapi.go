/*
The websocketapi package provides a WebSocket API to watch
the hub configuration and the data exported from the simulator.

Work in progress.
*/
package websocketapi

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func ApiHandler(prefix string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, "/hubapi/websocket") {

		} else {
			next(w, r)
			return
		}
	}
}

// AddHandler adds a handler for the request path /hubapi/websocket to the default net.http ServeMux
func AddHandler() {
	http.HandleFunc("/hubapi/websocket", wsHandler)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalln("failed to upgrade websocket request")
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte("Hello from Go!"))
}
