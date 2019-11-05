// Package pluginmanager manages module definitions.
// Module definitions are developed in separate git repositories from the main project.
// The module definition manager clones each repository into a subdirectory of
// %APPDATA%/DCS-BIOS/module-definitions,
// allows the user to update the definitions or revert a specific checkout to a previous
// tag, and updates a Lua file to include a dofile(...) call for each updated module
// definition.
package pluginmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	git "gopkg.in/src-d/go-git.v4"
	plumbing "gopkg.in/src-d/go-git.v4/plumbing"

	"dcs-bios.a10c.de/dcs-bios-hub/controlreference"
	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

const PluginManagerRemoteName = "dcs_bios_plugin_manager_remote"

type pluginManager struct {
	CheckoutPath                string
	jsonAPI                     *jsonapi.JsonApi
	state                       map[string]*PluginState
	stateLock                   sync.Mutex // needs to be held when changing a progressState to "working" or inserting a new item into the state map, to ensure only one operation starts concurrently per repository
	stateSubscriptions          map[chan []PluginState]bool
	initialPluginLoadInProgress bool // if true, updatePluginIndex() becomes a no-op, and has to be called after all plugins are loaded. This is to prevent writing the same file over and over again.
	controlReferenceStore       *controlreference.ControlReferenceStore
}

func NewPluginManager(checkoutPath string, jsonAPI *jsonapi.JsonApi, crs *controlreference.ControlReferenceStore) (*pluginManager, error) {
	stat, err := os.Stat(checkoutPath)
	if err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("could not create PluginManager: not a directory: %s", checkoutPath)
	}

	pm := &pluginManager{
		CheckoutPath:          checkoutPath,
		jsonAPI:               jsonAPI,
		state:                 make(map[string]*PluginState),
		stateSubscriptions:    make(map[chan []PluginState]bool),
		controlReferenceStore: crs,
	}

	pm.stateLock.Lock()
	pm.initialPluginLoadInProgress = true
	mStates, _ := pm.ReadAllInstalledPluginsFromDisk()
	for _, mState := range mStates {
		pm.state[mState.LocalName] = mState
	}
	pm.initialPluginLoadInProgress = false
	pm.updateDcsLuaIndex()

	pm.stateLock.Unlock()

	jsonAPI.RegisterType("install_plugin", InstallPluginRequest{})
	jsonAPI.RegisterApiCall("install_plugin", pm.HandleInstallPluginRequest)

	jsonAPI.RegisterType("remove_plugin", RemovePluginListRequest{})
	jsonAPI.RegisterApiCall("remove_plugin", pm.HandleRemovePluginRequest)

	jsonAPI.RegisterType("monitor_plugin_list", MonitorPluginListRequest{})
	jsonAPI.RegisterType("plugin_list", PluginList(nil))
	jsonAPI.RegisterApiCall("monitor_plugin_list", pm.HandleMonitorPluginListRequest)

	jsonAPI.RegisterType("check_for_plugin_updates", CheckForPluginUpdatesRequest{})
	jsonAPI.RegisterApiCall("check_for_plugin_updates", pm.HandleCheckForPluginUpdatesRequest)

	jsonAPI.RegisterType("apply_plugin_updates", ApplyPluginUpdatesRequest{})
	jsonAPI.RegisterApiCall("apply_plugin_updates", pm.HandleApplyPluginUpdateRequest)

	pm.registerPluginCatalogApi()

	return pm, nil
}

// PluginState describes the current state of a module definition that has been installed
// or is in the process of being installed, updated or removed.
type PluginState struct {
	LocalName            string   `json:"localName"`            // Name of the on-disk folder
	RemoteURL            string   `json:"remoteURL"`            // remote URL (origin)
	CheckedOutCommitHash string   `json:"checkedOutCommitHash"` // the commit hash that is currently checked out (HEAD)
	CheckedOutTags       []string `json:"checkedOutTags"`       // The tags that point to the current HEAD
	CheckedOutBranch     string   `json:"checkedOutBranch"`     // the checked out branch. Empty if in detached head state.
	CanApplyUpdates      bool     `json:"canApplyUpdates"`      // true if HEAD is a branch and the remote branch differs
	Tags                 []string `json:"tags"`                 // available tags that can be checked out
	Branches             []string `json:"branches"`             // available remote branches that can be checked out
	IsManagedManually    bool     `json:"isManagedManually"`    // if true, this repository will not be touched by the plugin manager
	ProgressMessage      string   `json:"progressMessage"`
	ProgressState        string   `json:"progressState"` // "", "working", "error", "success"
	IsLoaded             bool     `json:"isLoaded"`      // true if the plugin was loaded correctly
	LoadError            string   `json:"loadError"`     // if IsLoaded is false, this contains the error description
	ModuleDefinitionName string   `json:"moduleDefinitionName"`
}

type MonitorPluginListRequest struct {
}
type PluginList []PluginState

func (pm *pluginManager) HandleMonitorPluginListRequest(req *MonitorPluginListRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	subscription := make(chan []PluginState)
	go func() {
		defer close(responseCh)
		for newState := range subscription {
			select {
			case responseCh <- PluginList(newState):
			case _, ok := <-followupCh:
				if !ok {
					pm.stateLock.Lock()
					defer pm.stateLock.Unlock()
					delete(pm.stateSubscriptions, subscription)
					return
				}
			}

		}
	}()

	pm.stateLock.Lock()
	pm.stateSubscriptions[subscription] = true
	stateCopy := make([]PluginState, 0, len(pm.state))
	names := make([]string, 0, len(pm.state))
	for name := range pm.state {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		stateCopy = append(stateCopy, *pm.state[name])
	}
	pm.stateLock.Unlock()
	subscription <- stateCopy

	<-followupCh // wait for connection to be closed

}

type RemovePluginListRequest struct {
	LocalName string
}

func (pm *pluginManager) HandleRemovePluginRequest(req *RemovePluginListRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)
	pm.stateLock.Lock()
	mState, ok := pm.state[req.LocalName]
	if !ok || mState.IsManagedManually {
		pm.stateLock.Unlock()
		responseCh <- jsonapi.ErrorResult{
			Message: "plugin is not installed or is manually managed: " + req.LocalName,
		}
		return
	}

	if mState.ProgressState == "working" {
		pm.stateLock.Unlock()
		responseCh <- jsonapi.ErrorResult{
			Message: "module definition " + req.LocalName + " is locked by another operation.",
		}
		return
	}

	mState.ProgressState = "working"
	mState.ProgressMessage = "removing..."
	pm.unloadPlugin(mState)
	pm.notifyStateObservers()
	pm.stateLock.Unlock()

	err := os.RemoveAll(filepath.Join(pm.CheckoutPath, req.LocalName))

	pm.stateLock.Lock()
	delete(pm.state, req.LocalName)
	pm.updateDcsLuaIndex()
	pm.notifyStateObservers()
	pm.stateLock.Unlock()

	if err == nil {
		responseCh <- jsonapi.SuccessResult{}
	} else {
		responseCh <- jsonapi.ErrorResult{
			Message: err.Error(),
		}
	}
}

type InstallPluginRequest struct {
	RemoteURL string `json:"remoteURL"`
}

func localNameFromRemoteURL(url *url.URL) string {
	localName := path.Base(url.Path)
	if strings.Trim(localName, ". /\\") == "" {
		return url.String()
	}
	if localName == ".git" {
		localName = path.Base(url.Path)
	}
	localName = strings.TrimSuffix(localName, ".git")
	return localName
}

func (pm *pluginManager) HandleInstallPluginRequest(req *InstallPluginRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	pm.stateLock.Lock()

	// check if plugin is already installed
	for _, state := range pm.state {
		if strings.ToLower(state.RemoteURL) == strings.ToLower(req.RemoteURL) {
			pm.stateLock.Unlock()
			responseCh <- jsonapi.ErrorResult{
				Message: "plugin is already installed.",
			}
			close(responseCh)
			return
		}
	}

	// determine local URL from module name
	url, err := url.Parse(req.RemoteURL)
	if err == nil && !(url.Scheme == "http" || url.Scheme == "https" || url.Scheme == "file") {
		err = errors.New("URL must start with https:// or http:// or file://")
	}
	if err != nil {
		pm.stateLock.Unlock()
		responseCh <- jsonapi.ErrorResult{
			Message: "invalid remote URL: " + req.RemoteURL + ": " + err.Error(),
		}
		close(responseCh)
		return
	}

	localName := localNameFromRemoteURL(url)

	// make sure that localName is unique by appending a number if necessary
	suffix := ""
	suffixCnt := 0
	for _, ok := pm.state[localName+suffix]; ok; {
		suffixCnt++
		suffix = strconv.Itoa(suffixCnt)
	}
	localName += suffix

	pm.state[localName] = &PluginState{
		LocalName:       localName,
		ProgressState:   "working",
		ProgressMessage: "installing...",
		CheckedOutTags:  make([]string, 0),
		Tags:            make([]string, 0),
		Branches:        make([]string, 0),
	}
	pm.notifyStateObservers()
	pm.stateLock.Unlock()

	responseCh <- jsonapi.SuccessResult{
		Message: "installing.",
	}
	close(responseCh)

	err = pm.InstallModuleDefinition(localName, req.RemoteURL)

	pm.stateLock.Lock()
	defer pm.stateLock.Unlock()
	if err != nil {
		pm.state[localName].ProgressState = "error"
		pm.state[localName].ProgressMessage = err.Error()
		pm.notifyStateObservers()
	}

	pm.state[localName].ProgressState = "success"
	pm.state[localName].ProgressMessage = "installed."
	pm.notifyStateObservers()
}

func (pm *pluginManager) ReadSinglePluginStateFromDisk(localName string) (*PluginState, error) {
	repo, err := git.PlainOpen(filepath.Join(pm.CheckoutPath, localName))
	if err != nil {
		return nil, err
	}
	state := &PluginState{
		LocalName:      localName,
		CheckedOutTags: make([]string, 0),
		Tags:           make([]string, 0),
		Branches:       make([]string, 0),
	}

	// determine remoteURL
	origin, err := repo.Remote(PluginManagerRemoteName)
	if err != nil {
		state.IsManagedManually = true
	} else {
		if len(origin.Config().URLs) == 1 {
			state.RemoteURL = origin.Config().URLs[0]
		}
	}

	// determine current HEAD state
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	if head.Name().IsBranch() {
		state.CheckedOutBranch = head.Name().Short()
		// check remote tracking branch
		if !state.IsManagedManually {
			ref, err := repo.Reference(plumbing.NewRemoteReferenceName(PluginManagerRemoteName, head.Name().Short()), true)
			if err != nil {
				fmt.Printf("error resolving ref: %v\n", err)
			}
			if err == nil {
				if ref.Hash() != head.Hash() {
					state.CanApplyUpdates = true
				}
			}
		}
	} else {
		state.CheckedOutBranch = ""
	}
	state.CheckedOutCommitHash = head.Hash().String()

	// list tags
	tags, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	tags.ForEach(func(tag *plumbing.Reference) error {
		//fmt.Printf("tag: %v %v %v\n", tag.Target(), tag.Name(), tag.Hash())
		state.Tags = append(state.Tags, tag.Name().Short())
		if tag.Hash() == head.Hash() {
			state.CheckedOutTags = append(state.CheckedOutTags, tag.Name().Short())
		}
		return nil
	})

	// list branches
	refIter, err := repo.Branches()
	if err != nil {
		return nil, err
	}
	refIter.ForEach(func(branch *plumbing.Reference) error {
		state.Branches = append(state.Branches, branch.Name().Short())
		return nil
	})

	// determine remote branches
	refs, err := repo.References()
	refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			state.Branches = append(state.Branches, ref.Name().Short())
		}
		return nil
	})

	type pluginManifest struct {
		ManifestVersion      int    `json:"manifestVersion"`
		ModuleDefinitionName string `json:"moduleDefinitionName"`
	}

	// check for manifest
	manifestFilename := filepath.Join(pm.CheckoutPath, localName, "dcs-bios-plugin-manifest.json")
	stat, err := os.Stat(manifestFilename)
	if err == nil && !stat.IsDir() {
		file, err := os.Open(manifestFilename)
		if err == nil {
			defer file.Close()
			dec := json.NewDecoder(file)
			var manifest pluginManifest
			err = dec.Decode(&manifest)
			if err == nil {
				if manifest.ManifestVersion == 1 {
					if manifest.ModuleDefinitionName != "" {
						state.ModuleDefinitionName = manifest.ModuleDefinitionName
					}
				} else {
					state.LoadError = "invalid manifest version"
				}
			}
		}

	}

	return state, nil
}

func (pm *pluginManager) loadPlugin(state *PluginState) {
	if state.ModuleDefinitionName != "" {
		jsonPath := filepath.Join(pm.CheckoutPath, state.LocalName, state.ModuleDefinitionName+".json")
		pm.controlReferenceStore.LoadFile(jsonPath)
		pm.updateDcsLuaIndex()
	}
}

func (pm *pluginManager) unloadPlugin(state *PluginState) {
	if state.ModuleDefinitionName != "" {
		pm.controlReferenceStore.UnloadModuleDefinition(state.ModuleDefinitionName)
	}
	pm.updateDcsLuaIndex()
	return
}

// ReadAllInstalledPluginsFromDisk returns a list of currently installed
// module definitions.
func (pm *pluginManager) ReadAllInstalledPluginsFromDisk() ([]*PluginState, error) {
	files, err := ioutil.ReadDir(pm.CheckoutPath)
	if err != nil {
		return nil, err
	}

	pluginStateList := make([]*PluginState, 0)
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		state, err := pm.ReadSinglePluginStateFromDisk(f.Name())
		if err != nil {
			return nil, err
		}
		pluginStateList = append(pluginStateList, state)
		pm.loadPlugin(state)
	}
	return pluginStateList, nil
}

type CheckForPluginUpdatesRequest struct{}

func (pm *pluginManager) HandleCheckForPluginUpdatesRequest(req *CheckForPluginUpdatesRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)

	pluginNamesToCheck := make([]string, 0)
	pm.stateLock.Lock()
	for localName, state := range pm.state {
		if state.IsManagedManually { // handle manually managed repositories
			pm.unloadPlugin(state)
			// manual repos: just re-read current state
			newState, err := pm.ReadSinglePluginStateFromDisk(state.LocalName)
			if err != nil {
				state.ProgressState = "error"
				state.ProgressMessage = "failed to read from disk: " + err.Error()
			} else {
				pm.state[state.LocalName] = newState
				pm.loadPlugin(newState)
			}
			continue
		}

		if state.ProgressState == "working" {
			continue // skip plugins that are already being worked on
		}
		pluginNamesToCheck = append(pluginNamesToCheck, localName)
		state.ProgressState = "working"
		state.ProgressMessage = "wait for update check"
	}
	pm.notifyStateObservers()
	pm.stateLock.Unlock()

	fetchJobChannel := make(chan string)
	const numConcurrentFetches = 2
	wg := sync.WaitGroup{}
	wg.Add(numConcurrentFetches)
	for i := 0; i < numConcurrentFetches; i++ {
		go func() {
			for localName := range fetchJobChannel {
				pm.stateLock.Lock()
				state := pm.state[localName]
				state.ProgressMessage = "checking for updates"
				pm.notifyStateObservers()
				pm.stateLock.Unlock()

				err := pm.CheckForUpdates(localName)
				if err == git.NoErrAlreadyUpToDate {
					err = nil
				}

				pm.stateLock.Lock()
				var newState *PluginState
				if err == nil {
					newState, err = pm.ReadSinglePluginStateFromDisk(localName)
				}

				if err != nil {
					pm.state[localName].ProgressState = "error"
					pm.state[localName].ProgressMessage = err.Error()
				} else {
					pm.state[localName] = newState
					pm.state[localName].ProgressState = "success"
				}
				pm.notifyStateObservers()
				pm.stateLock.Unlock()

			}
		}()

		responseCh <- jsonapi.SuccessResult{
			Message: "started update checks",
		}
	}

	go func() {
		for _, name := range pluginNamesToCheck {
			fetchJobChannel <- name
		}
		close(fetchJobChannel)
	}()
	wg.Wait()

}

func (pm *pluginManager) CheckForUpdates(localName string) error {
	repo, err := git.PlainOpen(filepath.Join(pm.CheckoutPath, localName))
	if err != nil {
		return err
	}
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: PluginManagerRemoteName,
		Tags:       git.AllTags,
	})
	if err != nil {
		return err
	}

	return nil
}

func (pm *pluginManager) startWorkOnPlugin(localName string, progressMessage string) (state *PluginState, ok bool) {
	pm.stateLock.Lock()
	defer pm.stateLock.Unlock()
	state, ok = pm.state[localName]
	if !ok {
		return nil, false
	}
	if state.ProgressState == "working" {
		return nil, false
	}
	state.ProgressState = "working"
	state.ProgressMessage = progressMessage
	return state, true
}

func (pm *pluginManager) finishWorkOnPlugin(localName string, progressState, progressMessage string) {
	pm.stateLock.Lock()
	defer pm.stateLock.Unlock()
	state, ok := pm.state[localName]
	if !ok {
		return
	}
	state.ProgressState = progressState
	state.ProgressMessage = progressMessage
	pm.notifyStateObservers()
}

type ApplyPluginUpdatesRequest struct {
	LocalName string `json:"localName"`
}

func (pm *pluginManager) HandleApplyPluginUpdateRequest(req *ApplyPluginUpdatesRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)

	currentState, ok := pm.startWorkOnPlugin(req.LocalName, "applying updates...")
	if !ok {
		responseCh <- jsonapi.ErrorResult{
			Message: "Plugin does not exist or is already in use.",
		}
		return
	}

	pm.unloadPlugin(currentState)
	err := pm.ApplyUpdates(currentState.LocalName)

	if err != nil {
		pm.finishWorkOnPlugin(req.LocalName, "error", err.Error())
		responseCh <- jsonapi.ErrorResult{
			Message: err.Error(),
		}
	}
	newState, err := pm.ReadSinglePluginStateFromDisk(req.LocalName)

	if err != nil {
		pm.finishWorkOnPlugin(req.LocalName, "error", "error re-reading plugin contents:"+err.Error())
	} else {
		pm.stateLock.Lock()
		pm.state[req.LocalName] = newState
		pm.loadPlugin(newState)
		pm.stateLock.Unlock()

		pm.finishWorkOnPlugin(req.LocalName, "success", "updates applied")
	}
	responseCh <- jsonapi.SuccessResult{}
}

func (pm *pluginManager) ApplyUpdates(localName string) error {
	repo, err := git.PlainOpen(filepath.Join(pm.CheckoutPath, localName))
	if err != nil {
		return err
	}

	// determine current HEAD state
	head, err := repo.Head()
	if err != nil {
		return err
	}
	if !head.Name().IsBranch() {
		return errors.New(fmt.Sprintf("Module definition %s is not on a branch.", localName))
	}
	// check remote tracking branch
	ref, err := repo.Reference(plumbing.NewRemoteReferenceName(PluginManagerRemoteName, head.Name().Short()), true)
	if err != nil {
		return errors.New(fmt.Sprintf("error resolving ref: %v\n", err))
	}
	if ref.Hash() == head.Hash() {
		return errors.New(fmt.Sprintf("Module definition %s is already up-to-date.", localName))
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	err = wt.Reset(&git.ResetOptions{
		Commit: ref.Hash(),
		Mode:   git.HardReset,
	})

	if err != nil {
		return err
	}
	return nil
}

// notifyStateObservers sends a copy of the current state
// to each observer. The caller of this function has to hold the
// stateLock.
func (pm *pluginManager) notifyStateObservers() {
	pluginStateList := make([]PluginState, 0, len(pm.state))
	names := make([]string, 0, len(pm.state))
	for name := range pm.state {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pluginStateList = append(pluginStateList, *pm.state[name])
	}
	for sub, _ := range pm.stateSubscriptions {
		select {
		case sub <- pluginStateList:
		case <-time.After(200 * time.Millisecond):
			fmt.Printf("PluginManager: notifyStateObservers(): failed to send state to subscriber\n")
		}

	}
}

// InstallModuleDefinition installs a module definition.
func (pm *pluginManager) InstallModuleDefinition(localName string, repoURL string) error {
	clonePath := filepath.Join(pm.CheckoutPath, localName)
	_, err := os.Stat(clonePath)
	if err == nil {
		return errors.New("module definition already installed, folder exists: " + clonePath)
	}
	_, err = git.PlainClone(clonePath, false, &git.CloneOptions{
		URL:          repoURL,
		SingleBranch: false,
		RemoteName:   PluginManagerRemoteName,
	})
	if err != nil {
		return err
	}

	pluginState, err := pm.ReadSinglePluginStateFromDisk(localName)
	if err != nil {
		return errors.New("InstallModuleDefinition(): failed to read state after clone: " + err.Error())
	}

	pm.stateLock.Lock()
	pm.state[localName] = pluginState
	pm.loadPlugin(pluginState)
	pm.notifyStateObservers()
	pm.stateLock.Unlock()

	return nil
}

type DcsLuaIndexEntry struct {
	PluginDir string `json:"pluginDir"`
	LuaFile   string `json:"luaFile"`
}

// updateDcsLuaIndex writes a list of all installed plugins
// to CheckoutPath\dcs-lua-index.json
// the caller has to hold the stateLock.
func (pm *pluginManager) updateDcsLuaIndex() {
	if pm.initialPluginLoadInProgress {
		return
	}

	indexFilename := filepath.Join(pm.CheckoutPath, "dcs-lua-index.json")
	index := make([]DcsLuaIndexEntry, 0)

	for _, pluginState := range pm.state {
		if pluginState.ModuleDefinitionName != "" {
			index = append(index, DcsLuaIndexEntry{
				PluginDir: filepath.Join(pm.CheckoutPath, pluginState.LocalName) + string(os.PathSeparator),
				LuaFile:   filepath.Join(pm.CheckoutPath, pluginState.LocalName, pluginState.ModuleDefinitionName+".lua"),
			})
		}
	}

	file, _ := os.Create(indexFilename)
	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	enc.Encode(index)
	file.Close()

}
