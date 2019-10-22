package serialconnection

import (
	"fmt"
	"sync"
	"time"

	"dcs-bios.a10c.de/dcs-bios-hub/configstore"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
	"dcs-bios.a10c.de/dcs-bios-hub/serialportlist"
)

type comportsJsonConfigFile struct {
	AutoConnect []string `json:"autoConnect"`
}

type PortPreference struct {
	// If AutoConnect is true, as soon as the port appears it will be connected to (unless Suspend=true)
	// This setting is persisted in the configuration file.
	AutoConnect bool `json:"autoConnect"`

	// Connect is a runtime setting. It defaults to the value of AutoConnect when the configuration is being loaded.
	// If Connect=true, a connection attempt will be made at regular intervals as long as the port is available.
	// If Connect=false, it will be closed if it is currently open and no (re)connection attempts will be made.
	ShouldBeConnected bool `json:"shouldBeConnected"`
}

type PortState struct {
	PortPreference
	IsConnected      bool `json:"isConnected"`
	IsPresent        bool `json:"isPresent"`
	serialConnection *SerialConnection
}

type InputCommand struct {
	SourcePortName string
	Command        []byte
}

type PortStateSnapshot map[string]PortState

// PortManager manages a list of desired port configurations.
// It also monitors available COM ports in the system and
// connects or disconnects them.
type PortManager struct {
	InputCommands        chan InputCommand
	portState            map[string]*PortState
	portStateDirtyFlag   bool // portStateDirtyFlag is set whenever a port state changes and the UI has to be notified
	portStateLock        sync.Mutex
	stateSubscribers     map[chan PortStateSnapshot]struct{}
	stateSubscribersLock sync.Mutex
	jsonAPI              *jsonapi.JsonApi
}

func NewPortManager() *PortManager {
	return &PortManager{
		InputCommands:    make(chan InputCommand),
		portState:        make(map[string]*PortState),
		stateSubscribers: make(map[chan PortStateSnapshot]struct{}),
	}
}

type SetPortPrefRequest struct {
	PortPreference
	PortName string `json:portName`
}

func (p *PortManager) HandlePortPrefRequest(req *SetPortPrefRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	p.SetPortPreference(req.PortName, req.PortPreference)
	responseCh <- jsonapi.SuccessResult{
		Message: "Configured port " + req.PortName,
	}
}

type MonitorSerialPortRequest struct{}

func (p *PortManager) HandleMonitorPortRequest(req *MonitorSerialPortRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	updateChan := make(chan PortStateSnapshot)
	p.SubscribePortStateUpdateChannel(updateChan)
	defer p.UnsubscribePortStateUpdateChannel(updateChan)

	for {
		select {
		case responseCh <- <-updateChan:
		case _, ok := <-followupCh:
			if !ok {
				break // channel closed
			}
		}
	}
}

func (p *PortManager) SetupJSONApi(api *jsonapi.JsonApi) {
	api.RegisterType("set_port_pref", SetPortPrefRequest{})
	api.RegisterApiCall("set_port_pref", p.HandlePortPrefRequest)

	api.RegisterType("monitor_serial_ports", MonitorSerialPortRequest{})
	api.RegisterApiCall("monitor_serial_ports", p.HandleMonitorPortRequest)
	api.RegisterType("port_state_snapshot", PortStateSnapshot{})
}

// SubscribePortStateJsonChannel subscribes a channel to receive the PortState as JSON whenever it changes
// After subscribing, the current PortState is immediately sent to the channel.
func (p *PortManager) SubscribePortStateUpdateChannel(ch chan PortStateSnapshot) {
	p.portStateLock.Lock()
	defer p.portStateLock.Unlock()
	p.stateSubscribersLock.Lock()
	defer p.stateSubscribersLock.Unlock()
	if _, ok := p.stateSubscribers[ch]; ok {
		return // do not subscribe a channel twice
	}

	p.sendPortStateCopyToChannel(ch)
	p.stateSubscribers[ch] = struct{}{}
}

func (p *PortManager) sendPortStateCopyToChannel(ch chan PortStateSnapshot) {
	portStateCopy := make(map[string]PortState)
	for k, v := range p.portState {
		portStateCopy[k] = *v
	}
	go func() { ch <- portStateCopy }()
}

// UnsubscribePortUpdateChannel unsubscribes a channel from receiving new PortStates and closes the channel.
func (p *PortManager) UnsubscribePortStateUpdateChannel(ch chan PortStateSnapshot) {
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
	p.portStateDirtyFlag = true

	p.persistConfig()
}

func (p *PortManager) persistConfig() {
	config := comportsJsonConfigFile{}
	for portName, state := range p.portState {
		if state.AutoConnect {
			config.AutoConnect = append(config.AutoConnect, portName)
		}
	}

	configstore.Store("comports.json", config)
}

// getPortState returns p.portState[portName]. If the portName does not have
// an entry yet, a new PortState struct is allocated and stored in the map.
// The caller must hold the p.portStateLock when calling this function.
func (p *PortManager) getPortState(portName string) (portState *PortState) {
	portState = p.portState[portName]
	if portState == nil {
		portState = &PortState{}
		p.portState[portName] = portState
		p.portStateDirtyFlag = true
	}
	return
}

func (p *PortManager) updatePortState() {
	p.portStateLock.Lock()
	defer p.portStateLock.Unlock()
	// clean up closed SerialConnections
	for _, portState := range p.portState {
		if (portState.IsConnected) && (portState.serialConnection.GetState() == StateClosed) {
			portState.serialConnection = nil
			portState.IsConnected = false
			p.portStateDirtyFlag = true
		}
	}
	// detect ports that have connected successfully
	for _, portState := range p.portState {
		if portState.serialConnection != nil {
			if !portState.IsConnected && portState.serialConnection.GetState() == StateOpen {
				portState.IsConnected = true
				p.portStateDirtyFlag = true
			}
		}
	}
	// get the list of all serial ports on the system
	availablePorts, err := serialportlist.GetSerialPortList()
	if err != nil {
		fmt.Printf("could not get serial port list: %s", err.Error())
		availablePorts = make([]string, 0)
	}
	// detect removed ports
	for portName, portState := range p.portState {
		found := false
		for _, p := range availablePorts {
			if p == portName {
				found = true
				break
			}
		}
		if !found {
			if portState.AutoConnect {
				portState.IsPresent = false
			} else {
				delete(p.portState, portName)
			}
			p.portStateDirtyFlag = true
		}
	}
	// detect new ports and connect to new ports
	for _, availablePortName := range availablePorts {
		portState := p.getPortState(availablePortName)
		if !portState.IsPresent {
			// detected a new port
			portState.IsPresent = true
			portState.ShouldBeConnected = portState.AutoConnect
			p.portStateDirtyFlag = true
		}
		if !portState.ShouldBeConnected && portState.IsConnected {
			// port is connected but shouldn't be, trigger disconnect
			portState.serialConnection.Close()
		} else if portState.ShouldBeConnected && portState.serialConnection == nil {
			// port is not connected but should be, try to connect
			portState.serialConnection = NewSerialConnection(availablePortName)
			portState.IsConnected = false
			p.portStateDirtyFlag = true
			go func(sc *SerialConnection) {
				for s := range sc.InputCommands {
					fmt.Printf("[%s] %s\n", sc.GetPortName(), s)
					p.InputCommands <- InputCommand{SourcePortName: sc.GetPortName(), Command: s}
				}
			}(portState.serialConnection)
		}
	}
	if p.portStateDirtyFlag {
		p.notifyPortState()
	}
}

func (p *PortManager) Run() {
	// load configuration
	initialPrefs := comportsJsonConfigFile{}
	configstore.Load("comports.json", &initialPrefs)
	for _, portName := range initialPrefs.AutoConnect {
		portState := p.getPortState(portName)
		portState.AutoConnect = true
		portState.ShouldBeConnected = true
	}

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

	for ch := range p.stateSubscribers {
		p.sendPortStateCopyToChannel(ch)
	}
	p.portStateDirtyFlag = false
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
