/*
Package webappserver serves web apps from the file system.

Apps are served under /app/<appname>.
Apps are located on the file system in <appname> folders under the
path that has been passed to the AddHandler function.
*/
package webappserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"dcs-bios.a10c.de/dcs-bios-hub/gui"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

var JsonApi *jsonapi.JsonApi

var settings struct {
	appPath string
}
var staticFileHandler http.Handler

// AddHandler adds a handler for the request path /app/ to the default net.http ServeMux
func AddHandler(appPath string) {
	// settings.appPath is used by the http.FileServer to locate static files
	// and by the requestHandler function to locate the proxy.txt configuration files
	settings.appPath = appPath

	// initialize static file handler to serve static file requests
	staticFileHandler = http.StripPrefix("/app/", http.FileServer(http.Dir(settings.appPath)))

	// add handler
	http.Handle("/app/", http.HandlerFunc(requestHandler))
	http.Handle("/api/postjson", http.HandlerFunc(requestHandler))
}

func isLocalRequest(remoteAddr string) bool {

	ipStr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false // err on the side of caution
	}
	ip := net.ParseIP(ipStr)
	return ip.IsLoopback()
}

// requestHandler dispatches to the static file server
func requestHandler(w http.ResponseWriter, r *http.Request) {
	// uriParts is now ["", "app", <app name>, ...]
	//log.Println("serving URL with static file handler: " + r.RequestURI)

	if !gui.IsExternalNetworkAccessEnabled() && !isLocalRequest(r.RemoteAddr) {
		// request from the network, but external access is disabled
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden. Enable external network access through the system tray icon to allow DCS-BIOS to be accessed over the network."))
		return
	}

	if (r.RequestURI == "/api/postjson") && r.Method == "POST" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		var request json.RawMessage
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "could not parse JSON request: %v", err)
		}
		followupChan := make(chan []byte)
		defer close(followupChan)
		responseChan, err := JsonApi.HandleApiCall(request, followupChan)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "internal error while handling request: %v", err)
			return
		}
		w.Write((<-responseChan).Data)
		return
	}

	staticFileHandler.ServeHTTP(w, r)
	return

}
