package exportdataparser

type encoder struct {
	DataBuffer    *DataBuffer
	autosyncIndex int // index in dataBuffer.BinaryData to consider dirty
}

func NewEncoder(dataBuffer *DataBuffer) *encoder {
	return &encoder{
		DataBuffer: dataBuffer,
	}
}

func byteSliceFromUint16(value uint16) []byte {
	x := make([]byte, 2)
	x[0] = byte(value & 0xFF)
	x[1] = byte((value & 0xFF00) >> 8)
	return x
}

func (enc *encoder) Update() []byte {
	enc.DataBuffer.SetFFFEDirty()

	updatePacket := make([]byte, 0)

	dataBuffer := enc.DataBuffer
	binData := dataBuffer.BinaryData

	if len(binData) > 0 {
		if enc.autosyncIndex >= len(binData) {
			enc.autosyncIndex = 0
		}
		for j := 0; j < 5; j++ {
			binData[enc.autosyncIndex].Dirty = true
			enc.autosyncIndex = (enc.autosyncIndex + 1) % len(binData)
		}
	}

	firstDirtyIndex := -1
	for i := range binData {
		if binData[i].Dirty {
			firstDirtyIndex = i
			break
		}
	}
	if firstDirtyIndex == -1 {
		return updatePacket // nothing to update
	}

	ret := []byte{0x55, 0x55, 0x55, 0x55}

	writeStartAddress := binData[firstDirtyIndex].Address
	var writeLength uint16 = 2
	writeData := byteSliceFromUint16(binData[firstDirtyIndex].Data)
	lastWriteDataAddress := writeStartAddress
	binData[firstDirtyIndex].Dirty = false
	for i := firstDirtyIndex + 1; i < len(binData); i++ {
		entry := binData[i]
		if entry.Dirty {
			// figure out whether to start a new write packet
			if (entry.Address-lastWriteDataAddress <= 6) && entry.Data != 0x5555 {
				// append to existing write packet
				a := lastWriteDataAddress + 2
				for a <= entry.Address {
					writeLength += 2
					writeData = append(writeData, byteSliceFromUint16(enc.DataBuffer.GetValueAtAddress(a))...)
					lastWriteDataAddress = a
					a += 2
				}

			} else {
				// start new write packet
				// first, flush the existing packet
				if writeLength != uint16(len(writeData)) {
					panic("wrong writeLength")
				}
				ret = append(ret, byteSliceFromUint16(writeStartAddress)...)
				ret = append(ret, byteSliceFromUint16(writeLength)...)
				ret = append(ret, writeData...)

				// start the next one
				writeStartAddress = entry.Address
				writeLength = 2
				writeData = byteSliceFromUint16(entry.Data)
				lastWriteDataAddress = entry.Address
			}
		}
		binData[i].Dirty = false
	}
	// append last write packet
	if writeLength != uint16(len(writeData)) {
		panic("wrong writeLength")
	}
	ret = append(ret, byteSliceFromUint16(writeStartAddress)...)
	ret = append(ret, byteSliceFromUint16(writeLength)...)
	ret = append(ret, writeData...)

	return ret
}
