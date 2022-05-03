////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
)

// processor is the reception handler for a RestServer
type singleReceiver struct {
	endpoints Endpoints
}

// Callback is the handler for single-use message reception for a RestServer
func (s *singleReceiver) Callback(req *single.Request, receptionId receptionID.EphemeralIdentity, rounds []rounds.Round) {
	// Unmarshal the payload
	newMessage := &message{}
	err := json.Unmarshal(req.GetPayload(), newMessage)
	if err != nil {
		jww.ERROR.Printf("Unable to unmarshal restlike message: %+v", err)
		return
	}

	// Send the payload to the proper Callback
	if cb, err := s.endpoints.Get(newMessage.URI(), newMessage.Method()); err == nil {
		cb(newMessage)
	} else {
		jww.ERROR.Printf("Unable to call restlike endpoint: %+v", err)
	}
}
