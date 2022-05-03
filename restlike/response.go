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
)

// processor is the response handler for a Request
type singleResponse struct {
	responseCallback Callback
}

// Callback is the handler for single-use message responses for a Request
func (s *singleResponse) Callback(payload []byte, receptionID receptionID.EphemeralIdentity, rounds []rounds.Round, err error) {
	newMessage := &message{}

	// Handle response errors
	if err != nil {
		newMessage.err = err.Error()
		s.responseCallback(newMessage)
	}

	// Unmarshal the payload
	err = json.Unmarshal(payload, newMessage)
	if err != nil {
		jww.ERROR.Printf("Unable to unmarshal restlike message: %+v", err)
		return
	}

	// Send the response payload to the responseCallback
	s.responseCallback(newMessage)
}
