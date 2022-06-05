////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/restlike"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/csprng"
	"google.golang.org/protobuf/proto"
)

// Request allows for making REST-like requests to a RestServer using connect.Connection
// Can be used as stateful or declared inline without state
type Request struct {
	Net    connect.Connection
	Rng    csprng.Source
	E2eGrp *cyclic.Group
}

// Request provides several Method of sending Data to the given URI
// and blocks until the Message is returned
func (s *Request) Request(method restlike.Method, path restlike.URI,
	content restlike.Data, headers *restlike.Headers, e2eParams e2e.Params) (*restlike.Message, error) {
	// Build the Message
	newMessage := &restlike.Message{
		Content: content,
		Headers: headers,
		Method:  uint32(method),
		Uri:     string(path),
	}
	msg, err := proto.Marshal(newMessage)
	if err != nil {
		return nil, err
	}

	// Build callback for the response
	signalChannel := make(chan *restlike.Message, 1)
	cb := func(msg *restlike.Message) {
		signalChannel <- msg
	}
	s.Net.RegisterListener(catalog.XxMessage, &response{responseCallback: cb})

	// Transmit the Message
	_, _, _, err = s.Net.SendE2E(catalog.XxMessage, msg, e2eParams)
	if err != nil {
		return nil, err
	}

	// Block waiting for single-use response
	jww.DEBUG.Printf("Restlike waiting for connect response from %s...",
		s.Net.GetPartner().PartnerId().String())
	newResponse := <-signalChannel
	jww.DEBUG.Printf("Restlike connect response received from %s",
		s.Net.GetPartner().PartnerId().String())

	return newResponse, nil
}

// AsyncRequest provides several Method of sending Data to the given URI
// and will return the Message to the given Callback when received
func (s *Request) AsyncRequest(method restlike.Method, path restlike.URI,
	content restlike.Data, headers *restlike.Headers, cb restlike.RequestCallback, e2eParams e2e.Params) error {
	// Build the Message
	newMessage := &restlike.Message{
		Content: content,
		Headers: headers,
		Method:  uint32(method),
		Uri:     string(path),
	}
	msg, err := proto.Marshal(newMessage)
	if err != nil {
		return err
	}

	// Build callback for the response
	s.Net.RegisterListener(catalog.XxMessage, &response{responseCallback: cb})

	// Transmit the Message
	_, _, _, err = s.Net.SendE2E(catalog.XxMessage, msg, e2eParams)
	return err
}
