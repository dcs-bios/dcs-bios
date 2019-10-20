package dcsconnection

import (
	"io"
	"net"
	"sync"
	"time"

	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

type ChanWriter struct {
	targetChannel chan<- []byte
}

func NewChanWriter(targetChannel chan<- []byte) *ChanWriter {
	cw := &ChanWriter{targetChannel: targetChannel}
	return cw
}
func (cw *ChanWriter) Write(part []byte) (n int, err error) {
	cw.targetChannel <- part
	return len(part), nil
}

type DcsConnectionState string

const (
	StateConnecting = DcsConnectionState("Connecting")
	StateConnected  = DcsConnectionState("Connected")
)

type DcsConnection struct {
	conn       net.Conn
	ExportData chan []byte
	state      DcsConnectionState
	mutex      sync.Mutex // synchronizes access to state and conn variables
	closeOnce  sync.Once
	done       chan struct{}
	jsonAPI    *jsonapi.JsonApi
}

func New(jsonAPI *jsonapi.JsonApi) *DcsConnection {
	return &DcsConnection{
		ExportData: make(chan []byte),
		state:      StateConnecting,
		done:       make(chan struct{}),
		jsonAPI:    jsonAPI,
	}
}

func (dc *DcsConnection) Close() {
	dc.closeOnce.Do(func() { close(dc.done) })
}

func (dc *DcsConnection) GetState() DcsConnectionState {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	return dc.state
}

// TrySend sends a message to DCS if the connection is established. If no connection exists, the message is silently discarded.
func (dc *DcsConnection) TrySend(message []byte) {
	dc.mutex.Lock()
	conn := dc.conn
	state := dc.state
	dc.mutex.Unlock()
	if state == StateConnected && conn != nil {
		conn.Write(message)
	}
}

func (dc *DcsConnection) Run() {
	exportDataWriter := NewChanWriter(dc.ExportData)

	for {
		// phase 1: establish connection
		for {
			conn, err := net.Dial("tcp", ":7778")
			if err == nil {
				// connection established
				dc.mutex.Lock()
				dc.conn = conn
				dc.state = StateConnected
				dc.mutex.Unlock()
				break
			}
			// wait one second for the next connection attempt
			select {
			case <-dc.done:
				return
			case <-time.After(1 * time.Second):
			}
		}

		dcsConnectionClosed := make(chan struct{})
		// phase 2: read data
		go func() {
			io.Copy(exportDataWriter, dc.conn)
			dcsConnectionClosed <- struct{}{}
		}()

		select {
		case <-dcsConnectionClosed:
		case <-dc.done:
			return
		}

		dc.mutex.Lock()
		dc.state = StateConnecting
		dc.conn.Close()
		dc.conn = nil
		dc.mutex.Unlock()
	}
}
