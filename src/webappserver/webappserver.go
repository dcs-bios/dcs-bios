/*
Package webappserver serves web apps from the file system.

Apps are served under /app/<appname>.
Apps are located on the file system in <appname> folders under the
path that has been passed to the AddHandler function.
*/
package webappserver

import (
	"net"
	"net/http"

	"dcs-bios.a10c.de/dcs-bios-hub/gui"
)

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
}

func isLocalRequest(remoteAddr string) bool {

	ipStr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false // err on the side of caution
	}
	ip := net.ParseIP(ipStr)
	return ip.IsLoopback()
}

// requestHandler dispatches to the static file server or the reverse proxy depending on the existence of a proxy.txt configuration file
func requestHandler(w http.ResponseWriter, r *http.Request) {
	// uriParts is now ["", "app", <app name>, ...]
	//log.Println("serving URL with static file handler: " + r.RequestURI)

	if !gui.IsExternalNetworkAccessEnabled() && !isLocalRequest(r.RemoteAddr) {
		// request from the network, but external access is disabled
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden. Enable external network access through the system tray icon to allow DCS-BIOS to be accessed over the network."))
		return
	}

	staticFileHandler.ServeHTTP(w, r)
	return

}
