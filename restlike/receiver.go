////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
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
func (s *singleReceiver) Callback(req *single.Request, receptionId receptionID.EphemeralIdentity, rounds []rounds.Round) {
	// Unmarshal the payload
	newMessage := &Message{}
	err := proto.Unmarshal(req.GetPayload(), newMessage)
	if err != nil {
		jww.ERROR.Printf("Unable to unmarshal restlike message: %+v", err)
		return
	}

	// Send the payload to the proper Callback
	if cb, err := s.endpoints.Get(URI(newMessage.GetUri()), Method(newMessage.GetMethod())); err == nil {
		cb(newMessage)
	} else {
		// If no callback, send an error response
		responseMessage := &Message{Error: err.Error()}
		payload, err := proto.Marshal(responseMessage)
		if err != nil {
			jww.ERROR.Printf("Unable to marshal restlike response message: %+v", err)
			return
		}
		// Send the response
		// TODO: Parameterize params and timeout
		_, err = req.Respond(payload, cmix.GetDefaultCMIXParams(), 30*time.Second)
		if err != nil {
			jww.ERROR.Printf("Unable to send restlike response message: %+v", err)
		}
	}
}
