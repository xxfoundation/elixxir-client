////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/csprng"
	"google.golang.org/protobuf/proto"
)

// SingleRequest allows for making REST-like requests to a RestServer using single-use messages
// Can be used as stateful or declared inline without state
type SingleRequest struct {
	Net    single.Cmix
	Rng    csprng.Source
	E2eGrp *cyclic.Group
}

// Request provides several Method of sending Data to the given URI
// and blocks until the Message is returned
func (s *SingleRequest) Request(method Method, recipient contact.Contact, path URI,
	content Data, headers *Headers, singleParams single.RequestParams) (*Message, error) {
	// Build the Message
	newMessage := &Message{
		Content: content,
		Headers: headers,
		Method:  uint32(method),
		Uri:     string(path),
	}
	msg, err := proto.Marshal(newMessage)
	if err != nil {
		return nil, err
	}

	// Build callback for the single-use response
	signalChannel := make(chan *Message, 1)
	cb := func(msg *Message) {
		signalChannel <- msg
	}

	// Transmit the Message
	_, _, err = single.TransmitRequest(recipient, catalog.RestLike, msg,
		&singleResponse{responseCallback: cb}, singleParams, s.Net, s.Rng, s.E2eGrp)
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
func (s *SingleRequest) AsyncRequest(method Method, recipient contact.Contact, path URI,
	content Data, headers *Headers, cb RequestCallback, singleParams single.RequestParams) error {
	// Build the Message
	newMessage := &Message{
		Content: content,
		Headers: headers,
		Method:  uint32(method),
		Uri:     string(path),
	}
	msg, err := proto.Marshal(newMessage)
	if err != nil {
		return err
	}

	// Transmit the Message
	_, _, err = single.TransmitRequest(recipient, catalog.RestLike, msg,
		&singleResponse{responseCallback: cb}, singleParams, s.Net, s.Rng, s.E2eGrp)
	return err
}
