////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/crypto/channel"
	"reflect"
	"testing"
)

func TestUserMessageInternal_GetChannelMessage(t *testing.T) {
	channelMsg := &ChannelMessage{
		Lease:       69,
		RoundID:     42,
		PayloadType: 7,
		Payload:     []byte("ban_badUSer"),
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

	internal, _ := NewUserMessageInternal(usrMsg)
	received := internal.GetChannelMessage()

	if !reflect.DeepEqual(received.Payload, channelMsg.Payload) ||
		received.Lease != channelMsg.Lease ||
		received.RoundID != channelMsg.RoundID ||
		received.PayloadType != channelMsg.PayloadType {
		t.Fatalf("GetChannelMessage did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", channelMsg, received)
	}
}

func TestUserMessageInternal_GetUserMessage(t *testing.T) {
	channelMsg := &ChannelMessage{
		Lease:       69,
		RoundID:     42,
		PayloadType: 7,
		Payload:     []byte("ban_badUSer"),
	}

	serialized, err := proto.Marshal(channelMsg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	usrMsg := &UserMessage{
		Message:             serialized,
		ValidationSignature: []byte("sig"),
		Signature:           []byte("sig2"),
		Username:            "hunter2",
		ECCPublicKey:        []byte("key"),
		UsernameLease:       666,
	}

	internal, _ := NewUserMessageInternal(usrMsg)
	received := internal.GetUserMessage()

	if !reflect.DeepEqual(received.Message, usrMsg.Message) ||
		received.Username != usrMsg.Username ||
		received.UsernameLease != usrMsg.UsernameLease ||
		!reflect.DeepEqual(received.Signature, usrMsg.Signature) ||
		!reflect.DeepEqual(received.ValidationSignature, usrMsg.ValidationSignature) ||
		!reflect.DeepEqual(received.ECCPublicKey, usrMsg.ECCPublicKey) {
		t.Fatalf("GetUserMessage did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", usrMsg, received)
	}
}

func TestUserMessageInternal_GetMessageID(t *testing.T) {
	channelMsg := &ChannelMessage{
		Lease:       69,
		RoundID:     42,
		PayloadType: 7,
		Payload:     []byte("ban_badUSer"),
	}

	serialized, err := proto.Marshal(channelMsg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	usrMsg := &UserMessage{
		Message:             serialized,
		ValidationSignature: []byte("sig"),
		Signature:           []byte("sig2"),
		Username:            "hunter2",
		ECCPublicKey:        []byte("key"),
		UsernameLease:       666,
	}

	internal, _ := NewUserMessageInternal(usrMsg)
	received := internal.GetMessageID()

	expected := channel.MakeMessageID(usrMsg.Message)

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("GetMessageID did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}
}

// Ensures the serialization hasn't changed, changing the message IDs. The
// protocol is tolerant of this because only the sender seralizes, but
// it would be good to know when this changes. If this test breaks, report it,
// but it should be safe to update the expected
func TestUserMessageInternal_GetMessageID_Consistency(t *testing.T) {
	expected := "ChMsgID-cfw4O6M47N9pqdtTcQjm/SSVqehTPGQd7cAMrNP9bcc="

	channelMsg := &ChannelMessage{
		Lease:       69,
		RoundID:     42,
		PayloadType: 7,
		Payload:     []byte("ban_badUSer"),
	}

	serialized, err := proto.Marshal(channelMsg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	usrMsg := &UserMessage{
		Message:             serialized,
		ValidationSignature: []byte("sig"),
		Signature:           []byte("sig2"),
		Username:            "hunter2",
		ECCPublicKey:        []byte("key"),
		UsernameLease:       666,
	}

	internal, _ := NewUserMessageInternal(usrMsg)
	received := internal.GetMessageID()

	if expected != received.String() {
		t.Fatalf("GetMessageID did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}
}
