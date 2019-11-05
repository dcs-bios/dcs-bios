package pluginmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"dcs-bios.a10c.de/dcs-bios-hub/configstore"
	"dcs-bios.a10c.de/dcs-bios-hub/dcssetup"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

// PluginCatalogURL is the URL where the plugin catalog JSON file is downloaded from
const PluginCatalogURL = "https://dcs-bios.a10c.de/plugincatalog.json"

type pluginCatalogEntry struct {
	CloneURL            string `json:"cloneURL"`
	Description         string `json:"description"`
	Website             string `json:"website"`
	RecommendForModPath string `json:"recommendForModPath"`
}

type pluginCatalog struct {
	Version int                  `json:"version"`
	Plugins []pluginCatalogEntry `json:"plugins"`
}

func fetchPluginCatalog() (*pluginCatalog, error) {
	var jsonCatalog []byte = nil
	localFilePath := configstore.GetFilePath("plugincatalog.json")
	stat, err := os.Stat(localFilePath)
	if err == nil && !stat.IsDir() {
		jsonCatalog, err = ioutil.ReadFile(localFilePath)
		if err != nil {
			return nil, err
		}
	}

	if len(jsonCatalog) == 0 {
		return nil, errors.New("could not fetch JSON catalog")
	}

	var catalog pluginCatalog
	err = json.Unmarshal(jsonCatalog, &catalog)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON plugin catalog: %w", err)
	}

	if catalog.Version != 1 {
		return nil, fmt.Errorf("invalid plugin catalog version: %v", catalog.Version)
	}

	return &catalog, nil
	// TODO: do this over HTTP as well
}

func (pm *pluginManager) registerPluginCatalogApi() {
	pm.jsonAPI.RegisterType("get_plugin_catalog", GetPluginCatalogRequest{})
	pm.jsonAPI.RegisterApiCall("get_plugin_catalog", pm.HandleGetPluginCatalogRequest)
	pm.jsonAPI.RegisterType("plugin_catalog", GetPluginCatalogResponse{})
}

type GetPluginCatalogRequest struct{}
type PluginCatalogResponseEntry struct {
	pluginCatalogEntry
	LocalName          string `json:"localName"`
	IsRecommended      bool   `json:"isRecommended"`
	IsAlreadyInstalled bool   `json:"isAlreadyInstalled"`
}
type GetPluginCatalogResponse []PluginCatalogResponseEntry

func (pm *pluginManager) HandleGetPluginCatalogRequest(req *GetPluginCatalogRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	catalog, err := fetchPluginCatalog()
	if err != nil {
		responseCh <- jsonapi.ErrorResult{
			Message: "could not fetch plugin catalog: " + err.Error(),
		}
		return
	}

	dirExists := func(path string) bool {
		stat, err := os.Stat(path)
		return err == nil && stat.IsDir()
	}

	result := make([]PluginCatalogResponseEntry, len(catalog.Plugins))

	installs := dcssetup.GetDcsInstallations()
	for i, plugin := range catalog.Plugins {
		result[i].pluginCatalogEntry = plugin
		result[i].IsRecommended = false
		if plugin.RecommendForModPath != "" {
			for _, install := range installs {
				if dirExists(filepath.Join(install.InstallDir, plugin.RecommendForModPath)) {
					result[i].IsRecommended = true
					break
				}
			}
		}
		if plugin.CloneURL == "https://github.com/dcs-bios/module-commondata.git" {
			result[i].IsRecommended = true // always recommend CommonData module
		}
	}

	pm.stateLock.Lock()
	for i, plugin := range result {
		cloneURL, err := url.Parse(plugin.CloneURL)
		if err != nil {
			result[i].LocalName = plugin.CloneURL
		}
		result[i].LocalName = localNameFromRemoteURL(cloneURL)
		for _, state := range pm.state {
			if state.RemoteURL == plugin.CloneURL {
				result[i].IsAlreadyInstalled = true
				break
			}
		}
	}
	pm.stateLock.Unlock()

	responseCh <- GetPluginCatalogResponse(result)
}
