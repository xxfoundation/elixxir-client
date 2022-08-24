////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/restlike"
	"google.golang.org/protobuf/proto"
)

// receiver is the reception handler for a RestServer
type receiver struct {
	conn      connect.Connection
	endpoints *restlike.Endpoints
}

// Hear handles connect.Connection message reception for a RestServer
// Automatically responds to invalid endpoint requests
func (c receiver) Hear(item receive.Message) {
	// Unmarshal the request payload
	newMessage := &restlike.Message{}
	err := proto.Unmarshal(item.Payload, newMessage)
	if err != nil {
		jww.ERROR.Printf("Unable to unmarshal restlike message: %+v", err)
		return
	}

	var respondErr error
	if cb, err := c.endpoints.Get(restlike.URI(newMessage.GetUri()), restlike.Method(newMessage.GetMethod())); err == nil {
		// Send the payload to the proper Callback if it exists and singleRespond with the result
		respondErr = respond(cb(newMessage), c.conn)
	} else {
		// If no callback, automatically send an error response
		respondErr = respond(&restlike.Message{Error: err.Error()}, c.conn)
	}
	if respondErr != nil {
		jww.ERROR.Printf("Unable to singleRespond to request: %+v", err)
	}
}

// respond to connect.Connection with the given Message
func respond(response *restlike.Message, conn connect.Connection) error {
	payload, err := proto.Marshal(response)
	if err != nil {
		return errors.Errorf("unable to marshal restlike response message: %+v", err)
	}

	// TODO: Parameterize params
	_, _, _, _, err = conn.SendE2E(catalog.XxMessage, payload, e2e.GetDefaultParams())
	if err != nil {
		return errors.Errorf("unable to send restlike response message: %+v", err)
	}
	return nil
}

// Name is used for debugging
func (c receiver) Name() string {
	return "Restlike"
}
