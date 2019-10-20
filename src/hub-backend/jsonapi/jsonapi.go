// Package jsonapi provides a way for other packages
// to register API calls.
//
// All JSON messages have the form {datatype: string, data: any}.
// The corresponding Go types must have been registered with RegisterType().
//
// The type of the first message that is received determines the API function
// that is called and will receive all subsequent messages as well as be able
// to send one or more response messages back to the caller.
//
// This message exchange stops when either the client or the server closes
// the connection (i.e. the followupMessage channel is closed by the client
// or the responseMessage channel is closed by the server).
//
// This model maps to web sockets, but can also be used via a REST API
// for API calls that close their responseData channel after one message.
//
// To understand the dynamic JSON decoding going on here, refer to:
// https://eagain.net/articles/go-json-kind/

package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type JsonMessageEnvelope struct {
	DataType string      `json:"datatype"`
	Data     interface{} `json:"data"`
}

type BinaryData []byte

type SuccessResult struct {
	Message string `json:"message"`
}

type ErrorResult struct {
	Message string `json:"message"`
}

type JsonApi struct {
	dataStructFactories map[string]func() interface{}

	handlerFunctions map[string]interface{}
	msgtypeToGoType  map[string]reflect.Type
	goTypeToMsgtype  map[reflect.Type]string
}

func (api *JsonApi) RegisterType(msgtype string, typExamle interface{}) {
	v := reflect.ValueOf(typExamle)
	var typ = v.Type()
	// TODO: locking

	api.msgtypeToGoType[msgtype] = typ
	api.goTypeToMsgtype[typ] = msgtype
}

func (api *JsonApi) RegisterApiCall(msgtype string, handlerFunc interface{}) error {
	v := reflect.ValueOf(handlerFunc)
	t := v.Type()
	k := t.Kind()
	if k != reflect.Func {
		return errors.New("jsonapi.RegisterApiCall(): handlerFunc must be a function")
	}
	if t.NumIn() != 3 {
		return errors.New("jsonapi.RegisterApiCall(): handlerFunc must accept 3 parameters")
	}
	// check channel directions
	responseChType := t.In(1)
	if responseChType.Kind() != reflect.Chan || responseChType.ChanDir() != reflect.SendDir {
		return errors.New("jsonapi.RegisterApiCall(): first parameter of handlerFunc (response channel) must be a send-only channel")
	}
	followupChType := t.In(2)
	if followupChType.Kind() != reflect.Chan || followupChType.ChanDir() != reflect.RecvDir {
		return errors.New("jsonapi.RegisterApiCall(): second parameter of handlerFunc (followup message channel) must be a receive-only channel")
	}
	api.handlerFunctions[msgtype] = handlerFunc
	return nil
}

func (api *JsonApi) decodeJson(envelopeJson []byte) (*JsonMessageEnvelope, error) {
	var raw json.RawMessage
	var err error
	env := JsonMessageEnvelope{
		Data: &raw,
	}
	err = json.Unmarshal(envelopeJson, &env)
	if err != nil {
		return nil, errors.New("jsonapi: decodeJson: invalid envelope")
	}

	typ, ok := api.msgtypeToGoType[env.DataType]
	if !ok {
		return nil, errors.New("jsonapi: decodeJson: unknown msgtype: " + env.DataType)
	}
	data := reflect.New(typ).Interface()
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return nil, errors.New("jsonapi: could not decode message of type " + env.DataType + ": " + err.Error())
	}

	env.Data = data
	return &env, nil
}

func (api *JsonApi) encodeJson(data interface{}) ([]byte, error) {
	v := reflect.ValueOf(data)
	t := v.Type()
	// fmt.Printf("encodeJson: 1: %v\n", t)
	// if t.Kind() == reflect.Ptr {
	// 	t = t.Elem()
	// 	v = v.Elem()
	// }
	// fmt.Printf("encodeJson: looking up: %v\n", t)
	msgtype, ok := api.goTypeToMsgtype[t]
	if !ok {
		return nil, errors.New("jsonapi.encodeJson(): unknown type:" + t.Name())
	}
	// fmt.Printf("encodeJson: serializing: %v\n", v)
	envelope := JsonMessageEnvelope{
		DataType: msgtype,
		Data:     data,
	}
	ret, err := json.Marshal(envelope)
	// fmt.Printf("encodeJson: returning: %v\n", string(ret))
	return ret, err
}

func NewJsonApi() *JsonApi {
	api := &JsonApi{
		dataStructFactories: make(map[string]func() interface{}),
		handlerFunctions:    make(map[string]interface{}),
		msgtypeToGoType:     make(map[string]reflect.Type),
		goTypeToMsgtype:     make(map[reflect.Type]string),
	}
	api.RegisterType("success", SuccessResult{})
	api.RegisterType("error", ErrorResult{})
	return api
}

type ApiResponse struct {
	Data   []byte
	IsUTF8 bool
}

func (api *JsonApi) HandleApiCall(envelopeJsonData []byte, followupMessagesJson chan []byte) (responseJsonChannel chan ApiResponse, err error) {
	responseJsonChannel = make(chan ApiResponse)

	// decode message
	envelope, err := api.decodeJson(envelopeJsonData)
	if err != nil {
		close(responseJsonChannel)
		return responseJsonChannel, err
	}

	handlerFunc, ok := api.handlerFunctions[envelope.DataType]
	if !ok {
		close(responseJsonChannel)
		return responseJsonChannel, errors.New("jsonapi: HandleApiCall: no handlerFunc for message type" + envelope.DataType)
	}
	// we know that handlerFunc expects 3 parameters and has Kind reflect.Func
	// assert that the first argument type matches the type of the inital message
	v := reflect.ValueOf(handlerFunc)
	t := v.Type()
	firstArgType := t.In(0)
	if !reflect.TypeOf(envelope.Data).AssignableTo(firstArgType) {
		close(responseJsonChannel)
		return responseJsonChannel, errors.New("jsonapi: HandleApiCall: initial message type not assignable to first argument of handler function")
	}

	followupChannel := make(chan interface{})
	responseChannel := make(chan interface{})
	// next: call handlerFunc(envelope.Data, responseChannel, followupChannel) via reflect
	argv := []reflect.Value{reflect.ValueOf(envelope.Data), reflect.ValueOf(responseChannel), reflect.ValueOf(followupChannel)}
	go reflect.ValueOf(handlerFunc).Call(argv)

	go func() {
		for response := range responseChannel {

			if binaryData, ok := response.(BinaryData); ok {
				responseJsonChannel <- ApiResponse{binaryData, false}
			} else {

				data, err := api.encodeJson(response)
				if err == nil {
					// if we could serialize the message, send it out
					responseJsonChannel <- ApiResponse{data, true}
				} else {
					fmt.Printf("error serializing response: %s\n", err.Error())
				}
			}
		}
		close(responseJsonChannel)
	}()
	// transport followup messages
	go func() {
		for followupData := range followupMessagesJson {
			envelope, err := api.decodeJson(followupData)

			if err != nil {
				fmt.Printf("jsonapi: could not decode followup message: %v\n", err)
				continue
			}
			followupChannel <- envelope.Data
		}
		close(followupChannel)
	}()

	return responseJsonChannel, nil
}
