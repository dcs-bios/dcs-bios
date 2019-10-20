package livedataapi

import (
	"sync"

	"dcs-bios.a10c.de/dcs-bios-hub/jsonapi"
)

type LiveDataApi struct {
	jsonAPI             *jsonapi.JsonApi
	InputCommands       chan []byte
	exportDataListeners map[chan []byte]struct{}
	listenerLock        sync.Mutex
}

func NewLiveDataApi(jsonAPI *jsonapi.JsonApi) *LiveDataApi {
	lda := &LiveDataApi{
		jsonAPI:             jsonAPI,
		InputCommands:       make(chan []byte),
		exportDataListeners: make(map[chan []byte]struct{}),
	}
	jsonAPI.RegisterType("live_data", LiveDataRequest{})
	jsonAPI.RegisterApiCall("live_data", lda.HandleLiveDataRequest)
	jsonAPI.RegisterType("input_command", InputCommandMessage(""))
	return lda
}

func (lda *LiveDataApi) WriteExportData(data []byte) {
	lda.listenerLock.Lock()
	for ch, _ := range lda.exportDataListeners {
		ch <- data
	}
	lda.listenerLock.Unlock()
}

type LiveDataRequest struct{}
type InputCommandMessage string

func (lda *LiveDataApi) HandleLiveDataRequest(req *LiveDataRequest, responseCh chan<- interface{}, followupCh <-chan interface{}) {
	// accept input commands from the web socket connection for as long as it is alive
	onClose := make(chan struct{}) // this will be closed by the command listener goroutine once the followupChannel is closed
	go func() {
		for msg := range followupCh {

			if msgStr, ok := msg.(*InputCommandMessage); ok {

				msgBytes := []byte(*msgStr)
				lda.InputCommands <- msgBytes
			}
		}
		close(onClose)
	}()

	exportDataChannel := make(chan []byte)

	lda.listenerLock.Lock()
	lda.exportDataListeners[exportDataChannel] = struct{}{}
	lda.listenerLock.Unlock()

	// copy export data to the response channel until the followup channel is closed
	for {
		select {
		case data := <-exportDataChannel:
			responseCh <- jsonapi.BinaryData(data)
		case <-onClose:
			lda.listenerLock.Lock()
			delete(lda.exportDataListeners, exportDataChannel)
			lda.listenerLock.Unlock()
			close(responseCh)
			return
		}
	}
}
