////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"google.golang.org/protobuf/proto"
)

// processor is the response handler for a Request
type singleResponse struct {
	responseCallback RequestCallback
}

// Callback is the handler for single-use message responses for a Request
func (s *singleResponse) Callback(payload []byte, receptionID receptionID.EphemeralIdentity, rounds []rounds.Round, err error) {
	newMessage := &Message{}

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
