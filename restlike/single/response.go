////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/restlike"
	"google.golang.org/protobuf/proto"
)

// response is the response handler for a Request
type response struct {
	responseCallback restlike.RequestCallback
}

// Callback is the handler for single-use message responses for a Request
func (s *response) Callback(payload []byte, receptionID receptionID.EphemeralIdentity, rounds []rounds.Round, err error) {
	newMessage := &restlike.Message{}

	// Handle response errors
	if err != nil {
		newMessage.Error = err.Error()
		s.responseCallback(newMessage)
		return
	}

	// Unmarshal the payload
	err = proto.Unmarshal(payload, newMessage)
	if err != nil {
		newMessage.Error = err.Error()
	}

	// Send the response payload to the responseCallback
	s.responseCallback(newMessage)
}
