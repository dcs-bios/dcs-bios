package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"dcs-bios.a10c.de/dcs-bios-hub/configstore"
	"dcs-bios.a10c.de/dcs-bios-hub/controlreference"
	"dcs-bios.a10c.de/dcs-bios-hub/dcsconnection"
	"dcs-bios.a10c.de/dcs-bios-hub/dcssetup"
	"dcs-bios.a10c.de/dcs-bios-hub/exportdataparser"
	"dcs-bios.a10c.de/dcs-bios-hub/gui"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
	"dcs-bios.a10c.de/dcs-bios-hub/livedataapi"
	"dcs-bios.a10c.de/dcs-bios-hub/luaconsole"
	"dcs-bios.a10c.de/dcs-bios-hub/luastate"
	"dcs-bios.a10c.de/dcs-bios-hub/pluginmanager"
	"dcs-bios.a10c.de/dcs-bios-hub/serialconnection"
	"dcs-bios.a10c.de/dcs-bios-hub/statusapi"
	"dcs-bios.a10c.de/dcs-bios-hub/webappserver"
	"dcs-bios.a10c.de/dcs-bios-hub/websocketapi"
)

var gitSha1 string = "development build"
var gitTag string = "development build"
var autorunMode *bool = flag.Bool("autorun-mode", false, "Silently exit when binding TCP port 5010 fails. This prevents a message box when the program is being started by DCS but is already running.")

func runHttpServer(listenURI string) error {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		// redirect "/" to "/app/hubconfig"
		if r.RequestURI == "/" {
			http.Redirect(w, r, "/app/hubconfig", 302)
			return
		}
		http.DefaultServeMux.ServeHTTP(w, r)
		return
	}

	server := &http.Server{Addr: listenURI, Handler: http.HandlerFunc(handlerFunc)}
	listenSocket, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	go server.Serve(listenSocket)
	return nil
}

func startServices() {
	// find out where our executable is
	executableFilePath, err := os.Executable()
	if err != nil {
		fmt.Printf("could not determine current directory: %s\n", err.Error())
		os.Exit(1)
	}
	executableDir := filepath.Dir(executableFilePath)

	// create configuration directory
	if err := configstore.MakeDirs(); err != nil {
		fmt.Println("failed to create configuration directory:", err.Error())
		os.Exit(1)
	}

	os.Chdir(configstore.GetFilePath(""))

	// create jsonAPI instance
	// this is passed to the other services to make their API calls available
	jsonAPI := jsonapi.NewJsonApi()

	// run a web server on port 5010
	// the jsonAPI will be available via websockets at /api/websocket
	// Web pages will be served from /apps/appname.
	webappserver.JsonApi = jsonAPI
	webappserver.AddHandler(filepath.Join(executableDir, "apps"))

	statusapi.RegisterApiCalls(jsonAPI)
	statusapi.WithStatusInfoDo(func(si *statusapi.StatusInfo) {
		si.Version = gitTag
		si.GitSha1 = gitSha1
	})

	websocketapi.JsonApi = jsonAPI
	websocketapi.AddHandler()
	err = runHttpServer(":5010")
	if err != nil {
		// already running
		if !*autorunMode {
			gui.ErrorMsgBox("Could not listen on TCP port 5010.\nMost likely, another instance of DCS-BIOS Hub is already running. You can access it via the system tray icon.\n\nIf that is not the case, make sure nothing else is using TCP port 5010 and that your firewall is not interfering.", "DCS-BIOS Hub")
		}
		fmt.Println("could not listen on TCP :5010, is another instance running?")
		gui.Quit()
		return
	}

	// Control Reference Documentation
	cref := controlreference.NewControlReferenceStore(jsonAPI)
	cref.LoadFile(filepath.Join(executableDir, "control-reference-json", "MetadataStart.json"))
	cref.LoadFile(filepath.Join(executableDir, "control-reference-json", "MetadataEnd.json"))

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

	dcssetup.RegisterApi(jsonAPI)
	dcssetup.GetInstalledModulesList()

	_, err = pluginmanager.NewPluginManager(configstore.GetPluginDir(), jsonAPI, cref)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
	}

	// the Lua state that user-defined remapping scripts are executed in
	if err := luastate.DoFile("hooks.lua"); err != nil {
		workdir, getwderr := os.Getwd()
		if getwderr != nil {
			workdir = "(could not determine current directory)"
		}
		fmt.Printf("error loading hooks.lua from %s: %s", workdir, err.Error())
	}
	luastate.RegisterJsonApiCalls(jsonAPI)

	exportDataParser := exportdataparser.NewParser(cref)

	go func() {
		exportBuffer := exportdataparser.NewDataBuffer(cref)
		enc := exportdataparser.NewEncoder(exportBuffer)
		simData := exportdataparser.NewDataBuffer(cref)

		luastate.ExportDataBuffer = exportBuffer
		luastate.SimDataBuffer = simData

		for {
			select {
			case simData = <-exportDataParser.FrameData:
				luastate.SimDataBuffer = simData
				// remap here
				for _, v := range simData.BinaryData {
					exportBuffer.SetUint16(v.Address, v.Data)
				}

				luastate.NotifyOutputCallbacks()

				updatePacket := enc.Update()
				portManager.Write(updatePacket)
				lda.WriteExportData(updatePacket)

			case <-time.After(60 * time.Millisecond):
				updatePacket := enc.Update()
				portManager.Write(updatePacket)
				lda.WriteExportData(updatePacket)
			}
		}
	}()

	// transmit data between DCS and the serial ports
	go func() {
		for {
			select {

			case icstr := <-lda.InputCommands:
				cmd := []byte(string(icstr) + "\n")
				dcsConn.TrySend(cmd)

			case cmdFromLua := <-luastate.SimCommandChannel:
				cmd := []byte(cmdFromLua + "\n")
				dcsConn.TrySend(cmd)

			case ic := <-portManager.InputCommands:
				if !luastate.NotifyInputCallbacks(string(ic.Command)) {
					cmd := []byte(string(ic.Command) + "\n")
					dcsConn.TrySend(cmd)
				}

			case data := <-dcsConn.ExportData:
				for _, b := range data {
					exportDataParser.ProcessByte(b)
				}
			}
		}
	}()

	fmt.Println("ready.")
}

func main() {
	flag.Parse()
	gui.Run(startServices)
}
