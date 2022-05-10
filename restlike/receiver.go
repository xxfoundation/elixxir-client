////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
	"google.golang.org/protobuf/proto"
	"time"
)

// processor is the reception handler for a RestServer
type singleReceiver struct {
	endpoints *Endpoints
}

// Callback is the handler for single-use message reception for a RestServer
// Automatically responds to invalid endpoint requests
func (s *singleReceiver) Callback(req *single.Request, receptionId receptionID.EphemeralIdentity, rounds []rounds.Round) {
	// Unmarshal the request payload
	newMessage := &Message{}
	err := proto.Unmarshal(req.GetPayload(), newMessage)
	if err != nil {
		jww.ERROR.Printf("Unable to unmarshal restlike message: %+v", err)
		return
	}

	var respondErr error
	if cb, err := s.endpoints.Get(URI(newMessage.GetUri()), Method(newMessage.GetMethod())); err == nil {
		// Send the payload to the proper Callback if it exists and respond with the result
		respondErr = respond(cb(newMessage), req)
	} else {
		// If no callback, automatically send an error response
		respondErr = respond(&Message{Error: err.Error()}, req)
	}
	if respondErr != nil {
		jww.ERROR.Printf("Unable to respond to request: %+v", err)
	}
}

// respond to a single.Request with the given Message
func respond(response *Message, req *single.Request) error {
	payload, err := proto.Marshal(response)
	if err != nil {
		return errors.Errorf("unable to marshal restlike response message: %+v", err)
	}

	// TODO: Parameterize params and timeout
	_, err = req.Respond(payload, cmix.GetDefaultCMIXParams(), 30*time.Second)
	if err != nil {
		return errors.Errorf("unable to send restlike response message: %+v", err)
	}
	return nil
}
