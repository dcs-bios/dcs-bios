// Package serialconnection provides the SerialConnection object, which
// connects to a COM port and provides means to read and write from it
// that are convenient for the other parts of the DCS-BIOS hub use case.
//
// Data is read from the serial port one line at a time and sent to the InputCommands
// channel, which is available as an attribute on the SerialConnection object.
//
// To write data to the serial port, SerialConnection objects implement the
// Writer interface. The implementation ob SerialConnection.Write() never
// returns an error. If the data cannot be sent for any reason, it is silently
// discarded.
package serialconnection

import (
	"bufio"
	"fmt"
	"sync"
	"sync/atomic"

	serial "github.com/tarm/serial"
)

const (
	// StateConnecting means the serial port is being opened
	StateConnecting = iota
	// StateOpen means the serial port is open and data is being exchanged
	StateOpen
	// StateClosed means the serial port has been closed and the inputCommands channel is closed
	StateClosed
)

// SerialConnection represents a connection to a COM port. Create with New().
// Newline-delimited lines are read from the COM port and sent to the .InputCommands channel
// (without the newline at the end).
// When creating a SerialConnection with New(), the caller should also spawn a goroutine that
// reads from the InputCommand channel.
//
// SerialConnection also implements the Writer interface to send data to the port.
// The Write() method never returns an error. If the data cannot be sent, for example
// because the port is not open, the data is silently discarded.
type SerialConnection struct {
	portName      string
	port          *serial.Port
	InputCommands chan []byte
	closeOnce     sync.Once
	done          chan struct{}
	state         uint32
}

// GetPortName returns the name of the COM port this SerialConnection connects to
func (sc *SerialConnection) GetPortName() string {
	return sc.portName
}

// New connects to a COM port and returns a new SerialConnection object.
// A SerialConnection object starts out in the Connecting state and will
// try to connect to the given COM port.
func NewSerialConnection(portName string) (conn *SerialConnection) {
	sc := &SerialConnection{
		portName:      portName,
		InputCommands: make(chan []byte),
		done:          make(chan struct{}),
		state:         StateConnecting,
	}
	go sc.run()
	return sc
}

// Close closes the underlying serial port.
// It is safe to call this multiple times and from different goroutines.
func (sc *SerialConnection) Close() {
	sc.closeOnce.Do(func() { close(sc.done) })
}

func (sc *SerialConnection) GetState() uint32 {
	return atomic.LoadUint32(&sc.state)
}

func (sc *SerialConnection) setState(newState uint32) {
	atomic.StoreUint32(&sc.state, newState)
}

// StartSerialPortConnector spawns a goroutine that connects to a COM port and transfers data
// between the port and the inputCommands and newExportData channels, which are passed as arguments.
func (sc *SerialConnection) run() {
	// open serial port at 250000 bits per second
	var err error
	config := &serial.Config{Name: sc.portName, Baud: 250000}
	sc.port, err = serial.OpenPort(config)
	if err != nil {
		sc.setState(StateClosed)
		return
	}
	sc.setState(StateOpen)

	// spawn a goroutine to read lines from the port
	// and write them to the InputCommands channel
	go func() {
		scanner := bufio.NewScanner(sc.port)
		for scanner.Scan() {
			sc.InputCommands <- scanner.Bytes()
		}
		sc.Close()
	}()

	<-sc.done

	err = sc.port.Close()
	if err != nil {
		fmt.Printf("failed to close port %s: %s\n", sc.portName, err)
	}
	sc.setState(StateClosed)
	close(sc.InputCommands)
}

func (sc *SerialConnection) Write(data []byte) (n int, err error) {
	if sc.GetState() != StateOpen {
		return
	}
	return sc.port.Write(data)
}
