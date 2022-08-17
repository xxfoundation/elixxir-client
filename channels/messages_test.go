////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"testing"
)

func TestUserMessageInternal_GetChannelMessage(t *testing.T) {
	channelMsg := &ChannelMessage{
		Payload: []byte("ban_badUSer"),
	}

	serialized, err := proto.Marshal(channelMsg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	usrMsg := &UserMessage{
		Message:             serialized,
		ValidationSignature: []byte("sig"),
		Signature:           []byte("sig"),
		Username:            "hunter",
	}

	internal := NewUserMessageInternal(usrMsg)
	received, err := internal.GetChannelMessage()
	if err != nil {
		t.Fatalf("GetChannelMessage error: %v", err)
	}

	if !bytes.Equal(received.Payload, channelMsg.Payload) {
		t.Fatalf("GetChannelMessage did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", channelMsg, received)
	}
}
