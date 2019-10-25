// Package configstore reads and writes configuration files.
package configstore

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

func getFilePath(filename string) string {
	//dir, err := os.UserConfigDir()  // needs go 1.13
	// if err != nil {
	// 	panic(err)
	// }
	dir := os.ExpandEnv("${APPDATA}")
	return filepath.Join(dir, "DCS-BIOS", "Config", filename)
}

func MakeDirs() error {
	//dir, err := os.UserConfigDir()  // needs go 1.13
	// if err != nil {
	// 	return err
	// }
	dir := os.ExpandEnv("${APPDATA}")
	os.MkdirAll(filepath.Join(dir, "DCS-BIOS", "Config"), 0600)
	return nil
}

func Load(filename string, v interface{}) error {
	file, err := os.Open(getFilePath(filename))
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(v)
	if err != nil {
		return err
	}
	return nil
}

func Store(filename string, data interface{}) error {
	buf := bytes.NewBuffer([]byte{})

	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	err := enc.Encode(data)
	if err != nil {
		return err
	}

	file, err := os.Create(getFilePath(filename))
	if err != nil {
		return err
	}
	buf.WriteTo(file)
	file.Close()
	return nil
}

func OpenFile(filename string) (*os.File, error) {
	return os.Open(getFilePath(filename))
}