// +build windows

package serialportlist

import (
	"golang.org/x/sys/windows/registry"
)

// GetSerialPortList returns a list of available serial ports
func GetSerialPortList() ([]string, error) {
	portlist := make([]string, 0)
	d, err := registry.OpenKey(registry.LOCAL_MACHINE, "HARDWARE\\DEVICEMAP\\SERIALCOMM", registry.QUERY_VALUE)
	if err != nil {
		return portlist, err
	}
	defer d.Close()

	valueNames, err := d.ReadValueNames(128)
	if err.Error() != "EOF" {
		return portlist, err
	}

	for _, vn := range valueNames {
		portname, _, err := d.GetStringValue(vn)
		if err != nil {
			return portlist, err
		}
		portlist = append(portlist, portname)
	}
	return portlist, nil
}
