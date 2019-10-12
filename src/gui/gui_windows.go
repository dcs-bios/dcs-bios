// Package gui displays the system tray icon menu
// that allows the user to open the web-based interface
// and to quit the DCS-BIOS Hub.
package gui

import (
	"os"
	"os/signal"

	"dcs-bios.a10c.de/dcs-bios-hub/icon"
	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

// Run displays the GUI. Needs to be called directly
// from main() before any goroutines are started.
func Run(onReady func()) {
	initGui := func() {
		systray.SetIcon(icon.IconData)
		systray.SetTitle("DCS-BIOS Hub")
		systray.SetTooltip("DCS-BIOS")
		mHeader := systray.AddMenuItem("DCS-BIOS", "DCS-BIOS")
		mHeader.Disable()
		systray.AddSeparator()
		mURL := systray.AddMenuItem("Open in Browser", "")
		mQuit := systray.AddMenuItem("Quit", "Quit")

		go func() {

			// a, _ := serialportlist.GetSerialPortList()
			// fmt.Println("detected serial ports:")
			// fmt.Println(a)

			// handle SIGINT so we gracefully exit
			// on Ctrl+C in case this is compiled
			// and run as a console application during development
			sigintChannel := make(chan os.Signal, 1)
			signal.Notify(sigintChannel, os.Interrupt)
			go func() {
				<-sigintChannel
				systray.Quit()

			}()

			go func() {
				onReady()
			}()

			for {
				select {
				case <-mURL.ClickedCh:
					open.Start("http://localhost:5010")
				case <-mQuit.ClickedCh:
					systray.Quit()
					return
				}
			}
		}()
	}

	onExit := func() {
	}
	systray.Run(initGui, onExit)
}
