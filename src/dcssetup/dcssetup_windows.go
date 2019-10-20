// +build windows

package dcssetup

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/windows/registry"

	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

type DcsInstallation struct {
	InstallDir          string `json:"installDir"`
	Variant             string `json:"variant"`
	ProfileDir          string `json:"profileDir"`
	LuaScriptsInstalled bool   `json:"luaScriptsInstalled"`
}

type GetInstalledModuleNamesRequest struct{}
type GetInstalledModuleNamesResult []string

func HandleGetInstalledModuleNamesRequest(req *GetInstalledModuleNamesRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	responseCh <- GetInstalledModuleNamesResult(GetInstalledModulesList())
}

func RegisterApi(jsonAPI *jsonapi.JsonApi) {
	jsonAPI.RegisterType("get_installed_module_names", GetInstalledModuleNamesRequest{})
	jsonAPI.RegisterApiCall("get_installed_module_names", HandleGetInstalledModuleNamesRequest)
	jsonAPI.RegisterType("module_names_list", GetInstalledModuleNamesResult(nil))

	jsonAPI.RegisterType("get_setup_info", GetSetupInfoRequest{})
	jsonAPI.RegisterApiCall("get_setup_info", HandleGetSetupInfoRequest)
	jsonAPI.RegisterType("setup_info", GetSetupInfoResponse{})

	jsonAPI.RegisterType("modify_export_lua", ModifyExportLuaRequest{})
	jsonAPI.RegisterApiCall("modify_export_lua", HandleModifyExportLuaRequest)
}

// GetInstalledModulesList returns a list of all installed DCS: World modules.
func GetInstalledModulesList() []string {
	moduleSet := make(map[string]struct{}, 0)

	scanDcsInstallDir := func(path string) {
		fileinfoList, err := ioutil.ReadDir(filepath.Join(path, "mods", "aircraft"))
		if err != nil {
			return
		}
		for _, fi := range fileinfoList {
			if fi.IsDir() {
				moduleSet[strings.ToLower(fi.Name())] = struct{}{}
			}
		}
	}

	for _, install := range GetDcsInstallations() {
		scanDcsInstallDir(install.InstallDir)
	}

	moduleList := make([]string, len(moduleSet))
	for s := range moduleSet {
		moduleList = append(moduleList, s)
	}
	sort.Strings(moduleList)
	return moduleList
}

type ModifyExportLuaRequest struct {
	ProfileDir        string `json:"profileDir"`
	ShouldBeInstalled bool   `json:"shouldBeInstalled"`
}

func HandleModifyExportLuaRequest(req *ModifyExportLuaRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)

	installs := GetDcsInstallations()
	for _, i := range installs {
		if i.ProfileDir == req.ProfileDir {
			ok, log := SetupExportLua(req.ProfileDir, req.ShouldBeInstalled)
			if !ok {
				responseCh <- jsonapi.ErrorResult{Message: log}
			} else {
				responseCh <- jsonapi.SuccessResult{Message: log}
			}
			return
		}
	}

	responseCh <- jsonapi.ErrorResult{Message: "could not find a DCS installation with profile path " + req.ProfileDir}
	return
}

type GetSetupInfoRequest struct{}
type GetSetupInfoResponse struct {
	ExportLuaSetupLine string            `json:"exportLuaSetupLine"`
	Installs           []DcsInstallation `json:"installs"`
}

func HandleGetSetupInfoRequest(req *GetSetupInfoRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	exportLuaSetupLine, err := GetExportLuaSetupLine()
	if err != nil {
		exportLuaSetupLine = "error: " + err.Error()
	}
	installs := GetDcsInstallations()
	responseCh <- GetSetupInfoResponse{
		Installs:           installs,
		ExportLuaSetupLine: exportLuaSetupLine,
	}

}

func GetDcsInstallations() []DcsInstallation {
	installs := make([]DcsInstallation, 0)

	addInstallPath := func(installPath string) {
		dcsInstall := DcsInstallation{
			InstallDir: installPath,
		}

		// verify that the directory exists
		stat, err := os.Stat(installPath)
		if err != nil || !stat.IsDir() {
			return
		}

		// determine variant
		dcsVariantTxtPath := filepath.Join(installPath, "dcs_variant.txt")
		stat, err = os.Stat(dcsVariantTxtPath)
		if stat != nil && !(stat.IsDir()) {
			dcsVariantTxtBytes, err := ioutil.ReadFile(dcsVariantTxtPath)
			if err == nil {
				dcsInstall.Variant = strings.Split(string(dcsVariantTxtBytes), "\n")[0]
			}
		}

		dcsInstall.ProfileDir = filepath.Join(os.ExpandEnv("${USERPROFILE}"), "Saved Games", "DCS")
		if dcsInstall.Variant != "" {
			dcsInstall.ProfileDir += "." + dcsInstall.Variant
		}

		dcsInstall.LuaScriptsInstalled = IsExportLuaSetup(dcsInstall.ProfileDir)

		installs = append(installs, dcsInstall)
	}

	scanRegistryPath := func(regKey registry.Key, path string) {
		d, err := registry.OpenKey(regKey, path, registry.QUERY_VALUE)
		if err != nil {
			return
		}
		defer d.Close()

		installPath, _, err := d.GetStringValue("Path")
		if err != nil {
			return
		}
		addInstallPath(installPath)
	}

	scanRegistryPath(registry.CURRENT_USER, "Software\\Eagle Dynamics\\DCS World")
	scanRegistryPath(registry.CURRENT_USER, "Software\\Eagle Dynamics\\DCS World OpenBeta")

	return installs
}

func GetExportLuaSetupLine() (string, error) {
	executableFile, err := os.Executable()
	if err != nil {
		return "", err
	}
	luaScriptDir := filepath.Dir(executableFile) + string(os.PathSeparator)
	entrypointLuaFile := filepath.Join(luaScriptDir, "BIOS.lua")
	exportLuaSetupLine := "BIOS = {}; BIOS.LuaScriptDir = [[" + entrypointLuaFile + "]];pcall(function() dofile(BIOS.LuaScriptDir..[[BIOS.lua]]) end) --[[DCS-BIOS Automatic Setup]]"
	return exportLuaSetupLine, nil
}

func IsExportLuaSetup(profileDir string) bool {
	exportLuaFilePath := filepath.Join(profileDir, "Scripts", "Export.lua")

	file, err := os.Open(exportLuaFilePath)
	if err != nil {
		return false
	}
	defer file.Close()

	exportLuaSetupLine, err := GetExportLuaSetupLine()
	if err != nil {
		return false
	}

	lineScanner := bufio.NewScanner(file)
	for lineScanner.Scan() {
		if lineScanner.Text() == exportLuaSetupLine {
			return true
		}
	}

	return false
}

func GetModifiedExportLua(oldExportLua io.Reader, shouldBeInstalled bool, logBuffer *bytes.Buffer) []byte {
	newExportLuaBuffer := bytes.Buffer{}
	exportLuaSetupLine, err := GetExportLuaSetupLine()
	if err != nil {
		fmt.Fprintf(logBuffer, "error: could not determine executable file path: %v\n", err)
		return nil
	}

	lineScanner := bufio.NewScanner(oldExportLua)
	for lineScanner.Scan() {
		line := lineScanner.Text()
		if strings.HasSuffix(line, "--[[DCS-BIOS Automatic Setup]]") {
			fmt.Fprintf(logBuffer, "removing line: %s\n", line)
		} else if strings.Contains(line, "dofile(lfs.writedir()..[[Scripts\\DCS-BIOS\\BIOS.lua]])") {
			fmt.Fprintf(logBuffer, "removing line: %s\n", line)
		} else {
			newExportLuaBuffer.WriteString(line + "\r\n")
		}
	}
	if shouldBeInstalled {
		fmt.Fprintf(logBuffer, "appending line: %s\n", exportLuaSetupLine)
		newExportLuaBuffer.WriteString(exportLuaSetupLine + "\n")
	}
	return newExportLuaBuffer.Bytes()
}

func SetupExportLua(profileDir string, shouldBeInstalled bool) (ok bool, logMessages string) {
	logBuffer := &bytes.Buffer{}

	// assert that profileDir exists and is a directory
	stat, err := os.Stat(profileDir)
	if err != nil || !stat.IsDir() {
		fmt.Fprintf(logBuffer, "error: not a directory: %s\n", profileDir)
		return false, logBuffer.String()
	}

	// make sure a Scripts directory exists
	scriptDirPath := filepath.Join(profileDir, "Scripts")
	stat, err = os.Stat(scriptDirPath)
	if err != nil {
		// does not exist
		err = os.Mkdir(scriptDirPath, 0777)
		if err != nil {
			fmt.Fprintf(logBuffer, "error: could not create directory %s: %v\n", scriptDirPath, err)
			return false, logBuffer.String()
		}
	} else {
		// exists, assert that it is a directory
		if !stat.IsDir() {
			fmt.Fprintf(logBuffer, "error: path exists but is not a directory: %s\n", scriptDirPath)
			return false, logBuffer.String()
		}
	}

	// open existing Export.lua for reading or provide an empty buffer instead
	var existingExportLuaReader io.Reader
	exportLuaFilePath := filepath.Join(scriptDirPath, "Export.lua")
	stat, err = os.Stat(exportLuaFilePath)
	if err != nil {
		// Export.lua does not exist yet
		existingExportLuaReader = &bytes.Buffer{}
	} else {
		existingExportLuaReader, err = os.Open(exportLuaFilePath)
		if err != nil {
			fmt.Fprintf(logBuffer, "error: could not open %s: %v\n", exportLuaFilePath, err)
			return false, logBuffer.String()
		}
	}

	// try setup
	newExportLuaContent := GetModifiedExportLua(existingExportLuaReader, shouldBeInstalled, logBuffer)
	if err != nil {
		fmt.Fprintf(logBuffer, "error while generating new Export.lua content: %v\n", err)
		return false, logBuffer.String()
	}
	file, err := os.Create(exportLuaFilePath)
	if err != nil {
		fmt.Fprintf(logBuffer, "error: could not open Export.lua for writing: %v\n", err)
		return false, logBuffer.String()
	}
	defer file.Close()
	file.Write(newExportLuaContent)
	fmt.Fprintf(logBuffer, "file saved: %s\n", exportLuaFilePath)

	return true, logBuffer.String()
}
