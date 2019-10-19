// +build windows

package dcsinstalledmodules

import (
	"fmt"

	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/windows/registry"

	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

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
}

// GetInstalledModulesLIst returns a list of all installed DCS: World modules.
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

	scanRegistryPath := func(regKey registry.Key, path string) {
		d, err := registry.OpenKey(regKey, path, registry.QUERY_VALUE)
		if err != nil {
			fmt.Println("regerr", err)
			return
		}
		defer d.Close()

		installPath, _, err := d.GetStringValue("Path")
		if err != nil {
			return
		}
		scanDcsInstallDir(installPath)

	}

	scanRegistryPath(registry.CURRENT_USER, "Software\\Eagle Dynamics\\DCS World")
	scanRegistryPath(registry.CURRENT_USER, "Software\\Eagle Dynamics\\DCS World OpenBeta")

	moduleList := make([]string, len(moduleSet))
	for s := range moduleSet {
		moduleList = append(moduleList, s)
	}
	sort.Strings(moduleList)
	return moduleList
}
