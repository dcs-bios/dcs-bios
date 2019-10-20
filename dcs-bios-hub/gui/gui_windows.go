// Package gui displays the system tray icon menu
// that allows the user to open the web-based interface
// and to quit the DCS-BIOS Hub.
package gui

import (
	"os"
	"os/signal"
	"sync/atomic"

	"dcs-bios.a10c.de/dcs-bios-hub/icon"
	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
)

var externalNetworkAccessEnabled uint32

func IsExternalNetworkAccessEnabled() bool {
	return atomic.LoadUint32(&externalNetworkAccessEnabled) == 1
}

var luaConsoleEnabled uint32

func IsLuaConsoleEnabled() bool {
	return atomic.LoadUint32(&luaConsoleEnabled) == 1
}

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
		mURL := systray.AddMenuItem("Open web interface", "")
		systray.AddSeparator()
		mToggleExternalAccess := systray.AddMenuItem("Enable access over the network", "Allow the web interface and API to be accessed over the network.")
		mLuaConsoleEnabled := systray.AddMenuItem("Enable Lua Console", "Enable the Lua console. Warning: this allows anyone with access to the web interface to execute arbitrary code on your machine!")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit")

		go func() {
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
				case <-mToggleExternalAccess.ClickedCh:
					if mToggleExternalAccess.Checked() {
						mToggleExternalAccess.Uncheck()
					} else {
						mToggleExternalAccess.Check()
					}
					atomic.StoreUint32(&externalNetworkAccessEnabled,
						map[bool]uint32{false: 0, true: 1}[mToggleExternalAccess.Checked()])
				case <-mLuaConsoleEnabled.ClickedCh:
					if mLuaConsoleEnabled.Checked() {
						mLuaConsoleEnabled.Uncheck()
					} else {
						mLuaConsoleEnabled.Check()
					}
					atomic.StoreUint32(&luaConsoleEnabled,
						map[bool]uint32{false: 0, true: 1}[mLuaConsoleEnabled.Checked()])
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
