package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"dcs-bios.a10c.de/dcs-bios-hub/configstore"
	"dcs-bios.a10c.de/dcs-bios-hub/controlreference"
	"dcs-bios.a10c.de/dcs-bios-hub/dcsconnection"
	"dcs-bios.a10c.de/dcs-bios-hub/dcsinstalledmodules"
	"dcs-bios.a10c.de/dcs-bios-hub/gui"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
	"dcs-bios.a10c.de/dcs-bios-hub/livedataapi"
	"dcs-bios.a10c.de/dcs-bios-hub/luaconsole"
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
	// create configuration directory
	if err := configstore.MakeDirs(); err != nil {
		fmt.Println("failed to create configuration directory:", err.Error())
		os.Exit(1)
	}

	// create jsonAPI instance
	// this is passed to the other services to make their API calls available
	jsonAPI := jsonapi.NewJsonApi()

	// run a web server on port 5010
	// the jsonAPI will be available via websockets at /api/websocket
	// Web pages will be served from /apps/appname.
	webappserver.JsonApi = jsonAPI
	webappserver.AddHandler("apps")

	websocketapi.JsonApi = jsonAPI
	websocketapi.AddHandler()
	go runHttpServer(":5010")

	// Control Reference Documentation
	cref := controlreference.NewControlReferenceStore(jsonAPI)
	go cref.LoadData()

	// Lua console TCP server
	luaConsole := luaconsole.NewServer(jsonAPI)
	go luaConsole.Run()

	// connection to DCS-BIOS Lua Script via TCP port 7778
	dcsConn := dcsconnection.New(jsonAPI)
	go dcsConn.Run()

	// serial port connections
	portManager := serialconnection.NewPortManager()
	portManager.SetupJSONApi(jsonAPI)
	go portManager.Run()

	// live data API endpoint
	lda := livedataapi.NewLiveDataApi(jsonAPI)

	dcsinstalledmodules.RegisterApi(jsonAPI)
	dcsinstalledmodules.GetInstalledModulesList()

	// transmit data between DCS and the serial ports
	go func() {
		fmt.Println("main loop starting")
		for {
			select {
			case icstr := <-lda.InputCommands:
				cmd := append(icstr, byte('\n'))
				dcsConn.TrySend(cmd)

			case ic := <-portManager.InputCommands:
				cmd := append(ic.Command, byte('\n'))
				dcsConn.TrySend(cmd)

			case data := <-dcsConn.ExportData:
				portManager.Write(data)
				lda.WriteExportData(data)

			}
		}
	}()

	fmt.Println("ready.")
}

func main() {
	gui.Run(startServices)
}
