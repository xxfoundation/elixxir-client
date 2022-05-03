////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/csprng"
)

// Request allows for making REST-like requests to a RestServer
type Request interface {
	// Request provides several Method of sending Data to the given URI
	// and blocks until the Message is returned
	Request(method Method, recipient contact.Contact, path URI, content Data, param Param) (Message, error)

	// AsyncRequest provides several Method of sending Data to the given URI
	// and will return the Message to the given Callback when received
	AsyncRequest(method Method, recipient contact.Contact, path URI, content Data, param Param, cb Callback) error
}

// SingleRequest implements the Request interface using single-use messages
// Can be used as stateful or declared inline without state
type SingleRequest struct {
	RequestParam single.RequestParams
	Net          single.Cmix
	Rng          csprng.Source
	E2eGrp       *cyclic.Group
	//func TransmitRequest(recipient contact.Contact, tag string, payload []byte,
	//	callback Response, param RequestParams, net Cmix, rng csprng.Source,
	//	e2eGrp *cyclic.Group) ([]id.Round, receptionID.EphemeralIdentity, error)
}

// Request provides several Method of sending Data to the given URI
// and blocks until the Message is returned
func (s *SingleRequest) Request(method Method, recipient contact.Contact, path URI, content Data, param Param) (Message, error) {
	// Build the Message
	newMessage := &message{
		content: content,
		headers: param,
		method:  method,
		uri:     path,
	}
	msg, err := json.Marshal(newMessage)
	if err != nil {
		return nil, err
	}

	// Build callback for the single-use response
	signalChannel := make(chan Message, 1)
	cb := func(msg Message) {
		signalChannel <- msg
	}

	// Transmit the Message
	_, _, err = single.TransmitRequest(recipient, catalog.RestLike, msg,
		&singleResponse{responseCallback: cb}, s.RequestParam, s.Net, s.Rng, s.E2eGrp)
	if err != nil {
		return nil, err
	}

	// Block waiting for single-use response
	jww.DEBUG.Printf("Restlike waiting for single-use response from %s...", recipient.ID.String())
	newResponse := <-signalChannel
	jww.DEBUG.Printf("Restlike single-use response received from %s", recipient.ID.String())

	return newResponse, nil
}

// AsyncRequest provides several Method of sending Data to the given URI
// and will return the Message to the given Callback when received
func (s *SingleRequest) AsyncRequest(method Method, recipient contact.Contact, path URI, content Data, param Param, cb Callback) error {
	// Build the Message
	newMessage := &message{
		content: content,
		headers: param,
		method:  method,
		uri:     path,
	}
	msg, err := json.Marshal(newMessage)
	if err != nil {
		return err
	}

	// Transmit the Message
	_, _, err = single.TransmitRequest(recipient, catalog.RestLike, msg,
		&singleResponse{responseCallback: cb}, s.RequestParam, s.Net, s.Rng, s.E2eGrp)
	return err
}
