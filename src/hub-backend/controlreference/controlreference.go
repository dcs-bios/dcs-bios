// Package controlreference provides an API for the control reference documentation app
// to query the list of known controls.
package controlreference

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

// IOElement represents a named input or output element
type IOElement struct {
	Name        string   `json:"name"`
	Module      string   `json:"module"`
	Category    string   `json:"category"`
	Inputs      []Input  `json:"inputs"`
	Outputs     []Output `json:"outputs"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
}

type Input struct {
	Description string `json:"description"`
	Interface   string `json:"interface"`
	MaxValue    int    `json:"max_value"`
	Argument    string `json:"argument"`
}

type Output struct {
	Address     uint16 `json:"address"`
	Mask        uint16 `json:"mask"`
	MaxLength   uint16 `json:"max_length"`
	Description string `json:"description"`
	MaxValue    uint16 `json:"max_value"`
	ShiftBy     uint16 `json:"shift_by"`
	Suffix      string `json:"suffix"`
	Type        string `json:"type"`
}

type ControlReferenceStore struct {
	modules        map[string]IOElementCategoriesMap
	moduleDataLock sync.Mutex
	jsonAPI        *jsonapi.JsonApi
}

type IOElementCategoriesMap map[string]map[string]*IOElement

// GetIOElementByIdentifier takes an identifier of the form "module/element_name"
// and returns a pointer to the IOElement structure, or nil if not found.
func (crs *ControlReferenceStore) GetIOElementByIdentifier(identifier string) *IOElement {
	crs.moduleDataLock.Lock()
	defer crs.moduleDataLock.Unlock()

	parts := strings.SplitN(identifier, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	moduleId := parts[0]
	elementId := parts[1]

	for moduleName, category := range crs.modules {
		if strings.ToLower(moduleName) == strings.ToLower(moduleId) {
			for _, elements := range category {
				for elementName, element := range elements {
					if strings.ToLower(elementName) == strings.ToLower(elementId) {
						return element
					}
				}
			}
		}
	}
	return nil
}

func NewControlReferenceStore(jsonAPI *jsonapi.JsonApi) *ControlReferenceStore {
	crs := &ControlReferenceStore{
		modules: make(map[string]IOElementCategoriesMap),
		jsonAPI: jsonAPI,
	}
	jsonAPI.RegisterType("control_reference_get_modules", GetModulesRequest{})
	jsonAPI.RegisterApiCall("control_reference_get_modules", crs.HandleGetModulesListRequest)
	jsonAPI.RegisterType("module_list", GetModulesRequestResult{})

	jsonAPI.RegisterType("control_reference_query_ioelements", QueryIOElementsRequest{})
	jsonAPI.RegisterApiCall("control_reference_query_ioelements", crs.HandleQueryIOElementsRequest)
	jsonAPI.RegisterType("ioelements_query_result", QueryIOElementsResult{})

	return crs
}

type GetModulesRequest struct{}
type GetModulesRequestResult map[string][]string

func (crs *ControlReferenceStore) HandleGetModulesListRequest(req *GetModulesRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	crs.moduleDataLock.Lock()
	defer crs.moduleDataLock.Unlock()

	defer close(responseCh)
	var ret GetModulesRequestResult = make(map[string][]string)
	for moduleName, moduleData := range crs.modules {
		categories := make([]string, 0)
		for categoryName, _ := range moduleData {
			categories = append(categories, categoryName)
		}
		sort.Strings(categories)
		ret[moduleName] = categories
	}
	responseCh <- ret
}

type QueryIOElementsRequest struct {
	Module     string `json:"module"`
	Category   string `json:"category"`
	SearchTerm string `json:"searchTerm"`
}
type QueryIOElementsResult []IOElement

func (crs *ControlReferenceStore) HandleQueryIOElementsRequest(req *QueryIOElementsRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	crs.moduleDataLock.Lock()
	defer crs.moduleDataLock.Unlock()

	defer close(responseCh)
	var ret QueryIOElementsResult = make([]IOElement, 0)

	module, ok := crs.modules[req.Module]
	if !ok {
		responseCh <- ret
		return
	}

	if req.Category == "" { // look up by search term
		s := strings.ToLower(req.SearchTerm)
		for _, category := range module {
			for _, elem := range category {
				if strings.Contains(strings.ToLower(elem.Name), s) ||
					strings.Contains(strings.ToLower(elem.Description), s) {
					ret = append(ret, *elem)
				}
			}
		}
	}

	if req.Category != "" { // look up by category
		category, ok := module[req.Category]
		if !ok {
			// empty result if category does not exist
			responseCh <- ret
			return
		}

		for _, elem := range category {
			ret = append(ret, *elem)
		}
	}
	responseCh <- ret
}

func (crs *ControlReferenceStore) UnloadModuleDefinition(moduleName string) {
	crs.moduleDataLock.Lock()
	defer crs.moduleDataLock.Unlock()

	_, ok := crs.modules[moduleName]
	if ok {
		delete(crs.modules, moduleName)

		jsonCopyFilePath := filepath.Join(os.ExpandEnv("${APPDATA}"), "DCS-BIOS", "control-reference-json", moduleName+".json")
		stat, err := os.Stat(jsonCopyFilePath)
		if err == nil && !stat.IsDir() {
			os.Remove(jsonCopyFilePath)
		}
	}
}

func (crs *ControlReferenceStore) LoadFile(filename string) error {
	crs.moduleDataLock.Lock()
	defer crs.moduleDataLock.Unlock()

	basename := filepath.Base(filename)
	moduleName := basename[:len(basename)-len(filepath.Ext(basename))]

	if _, ok := crs.modules[moduleName]; ok {
		return fmt.Errorf("control reference: module already loaded: %s", moduleName)
	}

	module := make(IOElementCategoriesMap)
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error loading module %s: %v", moduleName, err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.Decode(&module)

	crs.modules[moduleName] = module

	jsonCopyFilePath := filepath.Join(os.ExpandEnv("${APPDATA}"), "DCS-BIOS", "control-reference-json", moduleName+".json")
	f.Seek(0, 0)
	copy, err := os.Create(jsonCopyFilePath)
	if err == nil {
		io.Copy(copy, f)
	}
	copy.Close()

	for moduleName, module := range crs.modules {
		for categoryName, cat := range module {
			for elementName, elem := range cat {
				elem.Name = elementName
				elem.Module = moduleName
				countStrOutputs := 0
				countIntOutputs := 0
				for _, out := range elem.Outputs {
					if out.Type == "string" {
						countStrOutputs++
					} else if out.Type == "integer" {
						countIntOutputs++
					} else {
						fmt.Println("unknown output type", out.Type)
					}
				}
				if countStrOutputs > 1 || countIntOutputs > 1 {
					fmt.Printf("warning: found element with more than one integer or string output: %s / %s / %s\n", moduleName, categoryName, elementName)
				}
			}
		}
	}

	return nil
}
