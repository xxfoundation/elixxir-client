////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/client/v5/e2e/receive"
	"gitlab.com/elixxir/client/v5/restlike"
	"google.golang.org/protobuf/proto"
)

// response is the response handler for a Request
type response struct {
	responseCallback restlike.RequestCallback
}

// Hear handles for connect.Connection message responses for a Request
func (r response) Hear(item receive.Message) {
	newMessage := &restlike.Message{}

	// Unmarshal the payload
	err := proto.Unmarshal(item.Payload, newMessage)
	if err != nil {
		newMessage.Error = err.Error()
	}

	// Send the response payload to the responseCallback
	r.responseCallback(newMessage)
}

// Name is used for debugging
func (r response) Name() string {
	return "Restlike"
}
