package exportdataparser

import (
	"sync"

	"dcs-bios.a10c.de/dcs-bios-hub/controlreference"
)

type BinaryExportData interface {
	SetUint16(address uint16, value uint16)
	SetIntegerValue(address uint16, mask uint16, shift uint16, value uint16)
	SetStringValue(address uint16, length uint16, data []byte)
}
type KeyValueExportData interface {
	SetIntegerValue(name string)
}

type DataWord struct {
	Address uint16
	Data    uint16
	Dirty   bool
}

type DataBuffer struct {
	controlReferenceStore *controlreference.ControlReferenceStore
	BinaryData            []DataWord
	lock                  sync.Mutex
}

func NewDataBuffer(crefstore *controlreference.ControlReferenceStore) *DataBuffer {
	return &DataBuffer{
		controlReferenceStore: crefstore,
	}
}

func (db *DataBuffer) setBytes(address uint16, data []byte) {
	i := 0
	addr := address
	for i < len(data) {
		word := db.GetValueAtAddress(addr)
		word = (word & 0xFF00) | uint16(data[i])
		i++
		if i < len(data) {
			word = (word & 0x00FF) | (uint16(data[i]) << 8)
		}
		i++
		db.SetUint16(addr, word)
		addr += 2
	}
}

func (db *DataBuffer) GetIntegerValue(valueIdentifier string) int {
	element := db.controlReferenceStore.GetIOElementByIdentifier(valueIdentifier)
	if element == nil {
		return -1
	}
	for _, output := range element.Outputs {
		if output.Type == "integer" {
			word := db.GetValueAtAddress(output.Address)
			return int((word & output.Mask) >> output.ShiftBy)
		}
	}
	return -1
}

func (db *DataBuffer) GetCStringValue(valueIdentifier string) string {
	element := db.controlReferenceStore.GetIOElementByIdentifier(valueIdentifier)
	if element == nil {
		return ""
	}
	for _, output := range element.Outputs {
		if output.Type == "string" {
			data := make([]byte, 0)
			bytesLeft := output.MaxLength
			addr := output.Address

			for bytesLeft > 0 {
				word := db.GetValueAtAddress(addr)
				thisByte := byte(word & 0x00FF)
				if thisByte == 0 {
					break
				}
				data = append(data, thisByte)
				bytesLeft--
				if bytesLeft == 0 {
					break
				}
				thisByte = byte(word & 0xFF00 >> 8)
				if thisByte == 0 {
					break
				}
				data = append(data, thisByte)
				bytesLeft--
				addr += 2
			}

			return string(data)
		}
	}
	return ""
}

func (db *DataBuffer) SetCStringValue(valueIdentifier string, value string) bool {
	element := db.controlReferenceStore.GetIOElementByIdentifier(valueIdentifier)
	if element == nil {
		return false
	}
	for _, output := range element.Outputs {
		if output.Type == "string" {
			newBytes := []byte(value)
			if uint16(len(newBytes)) > output.MaxLength {
				newBytes = newBytes[:output.MaxLength]
			}
			for len(newBytes) < int(output.MaxLength) {
				// fill up with spaces up to max length
				newBytes = append(newBytes, 32)
			}
			db.setBytes(output.Address, newBytes)
			return true
		}
	}
	return false
}

func (db *DataBuffer) SetIntegerValue(valueIdentifier string, value int) bool {
	element := db.controlReferenceStore.GetIOElementByIdentifier(valueIdentifier)
	if element == nil {
		return false
	}
	for _, output := range element.Outputs {
		if output.Type == "integer" {
			if value < 0 || uint16(value) > output.MaxValue {
				return false
			}
			currentWord := db.GetValueAtAddress(output.Address)
			//			fmt.Printf("currentWord: %v ", currentWord)
			// clear all bits
			var mask uint16 = 0x0001
			for mask < output.MaxValue {
				mask <<= 1
				mask |= 1
			}
			mask <<= output.ShiftBy

			currentWord &= ^mask

			currentWord |= (uint16(value) << output.ShiftBy)
			//			fmt.Printf("newWord: %v \n", currentWord)

			db.SetUint16(output.Address, currentWord)

			return true
		}
	}
	return false
}

// GetValueAtAddress returns the value for an entry at the given address, or 0x0000 if no entry is found.
func (db *DataBuffer) GetValueAtAddress(address uint16) uint16 {
	for i := range db.BinaryData {
		if db.BinaryData[i].Address == address {
			return db.BinaryData[i].Data
		}
	}
	return 0
}

// SetUint16 sets a 16-bit value in the data buffer at the given address and marks it as dirty.
func (db *DataBuffer) SetUint16(address uint16, value uint16) {
	db.lock.Lock()
	defer db.lock.Unlock()
	var insertBefore int = len(db.BinaryData)
	for i := range db.BinaryData {
		if db.BinaryData[i].Address == address {
			if db.BinaryData[i].Data != value {
				db.BinaryData[i].Data = value
				db.BinaryData[i].Dirty = true
			}
			return
		}
		if db.BinaryData[i].Address > address {
			insertBefore = i
			break
		}
	}
	db.BinaryData = append(db.BinaryData, DataWord{})
	if len(db.BinaryData) == 1 {
		insertBefore = 0
	}
	for i := len(db.BinaryData) - 1; i > insertBefore; i-- {
		db.BinaryData[i] = db.BinaryData[i-1]
	}
	// data has been shifted up by one index, so insertBefore is now the index of the new item
	db.BinaryData[insertBefore].Address = address
	db.BinaryData[insertBefore].Data = value
	db.BinaryData[insertBefore].Dirty = true
}

// Copy returns a new data buffer containing the same data and dirty bits.
func (db *DataBuffer) Copy() *DataBuffer {
	db.lock.Lock()
	defer db.lock.Unlock()
	newDB := &DataBuffer{
		BinaryData:            make([]DataWord, len(db.BinaryData)),
		controlReferenceStore: db.controlReferenceStore,
	}
	copy(newDB.BinaryData, db.BinaryData)
	return newDB
}

// Reset clears all data from the buffer.
func (db *DataBuffer) Reset() {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.BinaryData = make([]DataWord, 0)
}

// ClearDirtyFlags clears all dirty flags in the data buffer
func (db *DataBuffer) ClearDirtyFlags() {
	db.lock.Lock()
	defer db.lock.Unlock()
	for i := range db.BinaryData {
		db.BinaryData[i].Dirty = false
	}
}
