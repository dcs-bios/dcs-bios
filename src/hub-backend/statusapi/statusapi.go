// Package statusapi provides an API to query small pieces of information
// such as version info and whether connections to the Lua scripts or the
// Lua Console exist.
package statusapi

import (
	"sync"

	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

type StatusInfo struct {
	Version                        string `json:"version"`
	GitSha1                        string `json:"gitSHA1"`
	IsDcsConnected                 bool   `json:"isDcsConnected"`
	IsLuaConsoleConnected          bool   `json:"isLuaConsoleConnected"`
	IsLuaConsoleEnabled            bool   `json:"isLuaConsoleEnabled"`
	IsExternalNetworkAccessEnabled bool   `json:"isExternalNetworkAccessEnabled"`
	UnitType                       string `json:"unittype"`
}

var currentStatus StatusInfo
var statusLock sync.Mutex

var statusChannels map[chan StatusInfo]struct{} = make(map[chan StatusInfo]struct{})

func RegisterApiCalls(jsonAPI *jsonapi.JsonApi) {
	jsonAPI.RegisterType("get_status_updates", GetStatusUpdatesRequest{})
	jsonAPI.RegisterApiCall("get_status_updates", HandleGetStatusUpdatesRequest)
	jsonAPI.RegisterType("status_update", StatusInfo{})
}

func WithStatusInfoDo(mutator func(status *StatusInfo)) {
	statusLock.Lock()
	defer statusLock.Unlock()

	// notify observers
	mutator(&currentStatus)
	for ch := range statusChannels {
		ch <- currentStatus
	}
}

type GetStatusUpdatesRequest struct{}

func HandleGetStatusUpdatesRequest(req *GetStatusUpdatesRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	defer close(responseCh)

	statusChannel := make(chan StatusInfo)

	statusLock.Lock()
	statusChannels[statusChannel] = struct{}{}
	responseCh <- currentStatus
	statusLock.Unlock()

	for {
		select {
		case newStatus := <-statusChannel:
			responseCh <- newStatus
		case <-followupCh:
			// when the connection is closed, unsubscribe
			statusLock.Lock()
			delete(statusChannels, statusChannel)
			statusLock.Unlock()
			return
		}
	}
}
