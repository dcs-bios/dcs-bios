package exportdataparser

import (
	"bytes"
	"sync"

	"dcs-bios.a10c.de/dcs-bios-hub/controlreference"
)

type parserState int

const (
	StateWaitForSync = parserState(0)
	StateAddressLow  = parserState(1)
	StateAddressHigh = parserState(2)
	StateCountLow    = parserState(3)
	StateCountHigh   = parserState(4)
	StateDataLow     = parserState(5)
	StateDataHigh    = parserState(6)
)

type stringBuffer struct {
	address  int
	length   int
	data     []byte
	callback func([]byte)
}

type twoByteBuffer [2]byte

func (buf *twoByteBuffer) AsUint16() uint16 {
	return uint16(buf[1])<<8 | uint16(buf[0])
}
func (buf *twoByteBuffer) SetUint16(n uint16) {
	buf[0] = uint8(n & 0x00FF)
	buf[1] = uint8((n & 0xFF00) >> 8)
}

type ExportDataParser struct {
	state                 parserState
	protocolSyncByteCount int
	protocolAddressBuffer twoByteBuffer
	protocolCountBuffer   twoByteBuffer
	protocolDataBuffer    twoByteBuffer
	totalBuffer           [65536]byte
	dataBuffer            DataBuffer
	stringBuffers         []stringBuffer
	stringBufferLock      sync.Mutex
	FrameData             chan *DataBuffer
}

func NewParser(crs *controlreference.ControlReferenceStore) *ExportDataParser {
	ep := &ExportDataParser{
		FrameData: make(chan *DataBuffer, 1),
		dataBuffer: DataBuffer{
			controlReferenceStore: crs,
		},
	}
	return ep
}

func (ep *ExportDataParser) SubscribeStringBuffer(address int, length int, callback func([]byte)) {
	sb := stringBuffer{
		address:  address,
		length:   length,
		data:     make([]byte, length),
		callback: callback,
	}
	ep.stringBufferLock.Lock()
	ep.stringBuffers = append(ep.stringBuffers, sb)
	ep.stringBufferLock.Unlock()
}

func (ep *ExportDataParser) ProcessByte(b uint8) {
	switch ep.state {
	case StateWaitForSync:

	case StateAddressLow:
		ep.protocolAddressBuffer[0] = b
		ep.state = StateAddressHigh

	case StateAddressHigh:
		ep.protocolAddressBuffer[1] = b
		if ep.protocolAddressBuffer.AsUint16() != 0x555 {
			ep.state = StateCountLow
		} else {
			ep.state = StateWaitForSync
		}

	case StateCountLow:
		ep.protocolCountBuffer[0] = b
		ep.state = StateCountHigh

	case StateCountHigh:
		ep.protocolCountBuffer[1] = b
		ep.state = StateDataLow

	case StateDataLow:
		ep.protocolDataBuffer[0] = b
		ep.state = StateDataHigh
		ep.protocolCountBuffer.SetUint16(ep.protocolCountBuffer.AsUint16() - 1)
		ep.state = StateDataHigh

	case StateDataHigh:
		ep.protocolDataBuffer[1] = b
		ep.protocolCountBuffer.SetUint16(ep.protocolCountBuffer.AsUint16() - 1)
		ep.totalBuffer[ep.protocolAddressBuffer.AsUint16()] = ep.protocolDataBuffer[0]
		ep.totalBuffer[ep.protocolAddressBuffer.AsUint16()+1] = ep.protocolDataBuffer[1]
		ep.dataBuffer.SetUint16(ep.protocolAddressBuffer.AsUint16(), ep.protocolDataBuffer.AsUint16())

		if ep.protocolAddressBuffer.AsUint16() == 0xfffe {
			// end of update
			ep.notify()
		}

		ep.protocolAddressBuffer.SetUint16(ep.protocolAddressBuffer.AsUint16() + 2)

		if ep.protocolCountBuffer.AsUint16() == 0 {
			ep.state = StateAddressLow
		} else {
			ep.state = StateDataLow
		}

	}

	if b == 0x55 {
		ep.protocolSyncByteCount++
	} else {
		ep.protocolSyncByteCount = 0
	}

	if ep.protocolSyncByteCount == 4 {
		ep.state = StateAddressLow
		ep.protocolSyncByteCount = 0
	}
}

func (ep *ExportDataParser) notify() {
	ep.stringBufferLock.Lock()
	for _, sb := range ep.stringBuffers {
		for i := range sb.data {
			sb.data[i] = ep.totalBuffer[sb.address+i]
		}

		nullTerminatorPos := bytes.IndexByte(sb.data, 0)
		if nullTerminatorPos == -1 {
			nullTerminatorPos = 0
		}

		sb.callback(sb.data[:nullTerminatorPos])
	}
	ep.stringBufferLock.Unlock()
	ep.FrameData <- ep.dataBuffer.Copy()
	ep.dataBuffer.ClearDirtyFlags()
}
