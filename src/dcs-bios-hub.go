package main

import (
	"fmt"
	"log"
	"net/http"

	"dcs-bios.a10c.de/dcs-bios-hub/dcsconnection"
	"dcs-bios.a10c.de/dcs-bios-hub/gui"
	"dcs-bios.a10c.de/dcs-bios-hub/serialconnection"
	"dcs-bios.a10c.de/dcs-bios-hub/webappserver"
	"dcs-bios.a10c.de/dcs-bios-hub/websocketapi"
)

func runHttpServer(listenURI string) {
	err := http.ListenAndServe(listenURI, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// redirect "/" to "/app/hubconfig"
		if r.RequestURI == "/" {
			http.Redirect(w, r, "/app/hubconfig", 302)
			return
		}
		http.DefaultServeMux.ServeHTTP(w, r)
		return
	}))
	if err != nil {
		log.Fatalln("error: " + err.Error())
	}
}

func startServices() {
	fmt.Println("starting services")

	// serve web apps and WebSocket API via HTTP on port 5010
	webappserver.AddHandler("apps")
	websocketapi.AddHandler()
	go runHttpServer("localhost:5010")

	// connection to DCS-BIOS Lua Script via TCP port 7778
	dcsConn := dcsconnection.New()
	go dcsConn.Run()

	// connection to zero to many serial ports
	portMan := serialconnection.NewPortManager()
	go portMan.Run()

	var portsToConnect []string = []string{"COM4", "COM6", "COM7", "COM9"}
	for _, portName := range portsToConnect {
		portMan.SetPortPreference(portName, serialconnection.PortPreference{DesiredConnectionState: true})
	}

	go func() {
		for {
			select {
			case ic := <-portMan.InputCommands:
				cmd := append(ic.Command, byte('\n'))
				dcsConn.TrySend(cmd)
			case data := <-dcsConn.ExportData:
				portMan.Write(data)
			}
		}
	}()

	fmt.Println("ready.")
}

func main() {
	gui.Run(startServices)
}
