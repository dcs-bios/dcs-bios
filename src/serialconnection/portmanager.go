package serialconnection

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"dcs-bios.a10c.de/dcs-bios-hub/serialportlist"
)

type PortPreference struct {
	// If AutoConnect is true, as soon as the port appears it will be connected to (unless Suspend=true)
	// This setting is persisted in the configuration file.
	InitialConnectionState bool

	// Connect is a runtime setting. It defaults to the value of AutoConnect when the configuration is being loaded.
	// If Connect=true, a connection attempt will be made at regular intervals as long as the port is available.
	// If Connect=false, it will be closed if it is currently open and no (re)connection attempts will be made.
	DesiredConnectionState bool
}

type PortState struct {
	PortPreference
	CurrentConnectionState bool
	serialConnection       *SerialConnection
}

type InputCommand struct {
	SourcePortName string
	Command        []byte
}

// PortManager manages a list of desired port configurations.
// It also monitors available COM ports in the system and
// connects or disconnects them.
type PortManager struct {
	InputCommands        chan InputCommand
	portState            map[string]*PortState
	portStateLock        sync.Mutex
	stateSubscribers     map[chan []byte]struct{}
	stateSubscribersLock sync.Mutex
}

func NewPortManager() *PortManager {
	return &PortManager{
		InputCommands:    make(chan InputCommand),
		portState:        make(map[string]*PortState),
		stateSubscribers: make(map[chan []byte]struct{}),
	}
}

// SubscribePortStateJsonChannel subscribes a channel to receive the PortState as JSON whenever it changes
// After subscribing, the current PortState is immediately sent to the channel.
func (p *PortManager) SubscribePortStateJsonChannel(ch chan []byte) {
	p.portStateLock.Lock()
	defer p.portStateLock.Unlock()
	p.stateSubscribersLock.Lock()
	defer p.stateSubscribersLock.Unlock()
	if _, ok := p.stateSubscribers[ch]; ok {
		return // do not subscribe a channel twice
	}

	jsonMessage, _ := json.Marshal(p.portState)
	ch <- jsonMessage
	p.stateSubscribers[ch] = struct{}{}
}

// UnsubscribePortUpdateChannel unsubscribes a channel from receiving new PortStates and closes the channel.
func (p *PortManager) UnsubscribePortStateJsonChannel(ch chan []byte) {
	p.stateSubscribersLock.Lock()
	defer p.stateSubscribersLock.Unlock()
	delete(p.stateSubscribers, ch)
	close(ch)
}

func (p *PortManager) SetPortPreference(portName string, pref PortPreference) {
	p.portStateLock.Lock()
	defer p.portStateLock.Unlock()

	portState := p.getPortState(portName)
	portState.PortPreference = pref

}

// getPortState returns p.portState[portName]. If the portName does not have
// an entry yet, a new PortState struct is allocated and stored in the map.
// The caller must hold the p.portStateLock when calling this function.
func (p *PortManager) getPortState(portName string) (portState *PortState) {
	portState = p.portState[portName]
	if portState == nil {
		portState = &PortState{}
		p.portState[portName] = portState
	}
	return
}

func (p *PortManager) updatePortState() {
	p.portStateLock.Lock()
	defer p.portStateLock.Unlock()
	portStateUpdated := false
	// every 200 ms:
	// clean up closed SerialConnections
	for portName, portState := range p.portState {
		if (portState.CurrentConnectionState) && (portState.serialConnection.GetState() == StateClosed) {
			fmt.Printf("serial connection closed: %s\n", portName)
			portState.serialConnection = nil
			portState.CurrentConnectionState = false
			portStateUpdated = true
		}
	}

	// and check for new ports to connect to
	availablePorts, err := serialportlist.GetSerialPortList()
	if err != nil {
		log.Fatal("could not get serial port list: %s", err)
	}
	for _, availablePortName := range availablePorts {
		portState := p.getPortState(availablePortName)
		if !portState.DesiredConnectionState && portState.CurrentConnectionState {
			// trigger disconnect
			portState.serialConnection.Close()
		} else if portState.DesiredConnectionState && !portState.CurrentConnectionState {
			// try to connect
			fmt.Printf("connecting to %s\n", availablePortName)
			portState.serialConnection, err = NewSerialConnection(availablePortName)
			if err != nil {
				fmt.Printf("failed to connect to port %s: %s\n", availablePortName, err)
				continue
			}
			portState.CurrentConnectionState = true
			portStateUpdated = true
			go func(sc *SerialConnection) {
				for s := range sc.InputCommands {
					fmt.Printf("[%s] %s\n", sc.GetPortName(), s)
					p.InputCommands <- InputCommand{SourcePortName: sc.GetPortName(), Command: s}
				}
			}(portState.serialConnection)
		}
	}
	if portStateUpdated {
		p.notifyPortState()
	}
}

func (p *PortManager) Run() {
	for {
		select {
		case <-time.After(200 * time.Millisecond):
		}

		p.updatePortState()
	}
}

func (p *PortManager) notifyPortState() {
	p.stateSubscribersLock.Lock()
	defer p.stateSubscribersLock.Unlock()

	jsonMessage, _ := json.Marshal(p.portState)

	for ch := range p.stateSubscribers {
		ch <- jsonMessage
	}
}

func (p *PortManager) Write(data []byte) (n int, err error) {
	p.portStateLock.Lock()
	defer p.portStateLock.Unlock()
	for _, portState := range p.portState {
		if sercon := portState.serialConnection; sercon != nil {
			if sercon.GetState() != StateClosed {
				sercon.Write(data)
			}
		}
	}
	return len(data), nil
}
