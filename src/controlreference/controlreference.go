// Package controlreference provides an API for the control reference documentation app
// to query the list of known controls.
package controlreference

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

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
	Length      uint16 `json:"length"`
	Description string `json:"description"`
	MaxValue    uint16 `json:"max_value"`
	ShiftBy     uint16 `json:"shift_by"`
	Suffix      string `json:"suffix"`
	Type        string `json:"type"`
}

type ControlReferenceStore struct {
	modules map[string]IOElementCategoriesMap
	jsonAPI *jsonapi.JsonApi
}

type IOElementCategoriesMap map[string]map[string]IOElement

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
	Module   string `json:"module"`
	Category string `json:"category"`
}
type QueryIOElementsResult []IOElement

func (crs *ControlReferenceStore) HandleQueryIOElementsRequest(req *QueryIOElementsRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	var ret QueryIOElementsResult = make([]IOElement, 0)

	module := crs.modules[req.Module]
	category := module[req.Category]

	for name, elem := range category {
		elem.Name = name
		elem.Module = req.Module
		ret = append(ret, elem)
	}
	responseCh <- ret
}

func (crs *ControlReferenceStore) LoadData() {
	exec, err := os.Executable()
	if err != nil {
		log.Print(err)
		return
	}
	dir := filepath.Dir(exec)
	datapath := filepath.Join(dir, "control-reference-json")
	files, err := filepath.Glob(filepath.Join(datapath, "*.json"))
	if err != nil {
		log.Print(err)
		return
	}
	for _, filename := range files {
		crs.loadFile(filename)
	}
	fmt.Printf("control reference: loaded data for %d modules.\n", len(files))

	// verify that IOElements have at most one string output and at most one integer output
	// the web UI control reference assumes this to make live data handling a bit easier
	for moduleName, module := range crs.modules {
		for categoryName, cat := range module {
			for elementName, elem := range cat {
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
	fmt.Println("control reference data check complete.")
}

func (crs *ControlReferenceStore) loadFile(filename string) {
	basename := filepath.Base(filename)
	moduleName := basename[:len(basename)-len(filepath.Ext(basename))]
	module := make(IOElementCategoriesMap)
	f, err := os.Open(filename)
	if err != nil {
		log.Printf("error loading module %s: %v", moduleName, err)
		return
	}
	dec := json.NewDecoder(f)
	dec.Decode(&module)

	crs.modules[moduleName] = module
}
