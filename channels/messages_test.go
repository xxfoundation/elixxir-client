////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
)

func Test_unmarshalUserMessageInternal(t *testing.T) {
	internal, usrMsg, _ := builtTestUMI(t, 7)
	channelID := &id.ID{}

	usrMsgMarshaled, err := proto.Marshal(usrMsg)
	if err != nil {
		t.Fatalf("Failed to marshal user message: %+v", err)
	}

	umi, err := unmarshalUserMessageInternal(usrMsgMarshaled, channelID)
	if err != nil {
		t.Fatalf("Failed to unmarshal user message: %+v", err)
	}

	if !proto.Equal(umi.userMessage, internal.userMessage) {
		t.Errorf("Unmarshalled UserMessage does not match original."+
			"\nexpected: %+v\nreceived: %+v",
			internal.userMessage, umi.userMessage)
	}

	umi.userMessage = internal.userMessage
	if !reflect.DeepEqual(umi, internal) {
		t.Errorf("Unmarshalled userMessageInternal does not match original."+
			"\nexpected: %+v\nreceived: %+v", internal, umi)
	}
}

func TestUnmarshalUserMessageInternal_BadUserMessage(t *testing.T) {
	chID := &id.ID{}
	_, err := unmarshalUserMessageInternal([]byte("Malformed"), chID)
	if err == nil {
		t.Fatalf("Error not returned on unmarshaling a bad user " +
			"message")
	}
}

func TestUnmarshalUserMessageInternal_BadChannelMessage(t *testing.T) {
	_, usrMsg, _ := builtTestUMI(t, 7)

	usrMsg.Message = []byte("Malformed")

	chID := &id.ID{}

	usrMsgMarshaled, err := proto.Marshal(usrMsg)
	if err != nil {
		t.Fatalf("Failed to marshal user message: %+v", err)
	}

	_, err = unmarshalUserMessageInternal(usrMsgMarshaled, chID)
	if err == nil {
		t.Fatalf("Error not returned on unmarshaling a user message " +
			"with a bad channel message")
	}
}

func Test_newUserMessageInternal_BadChannelMessage(t *testing.T) {
	_, usrMsg, _ := builtTestUMI(t, 7)

	usrMsg.Message = []byte("Malformed")

	chID := &id.ID{}

	_, err := newUserMessageInternal(usrMsg, chID)

	if err == nil {
		t.Fatalf("failed to produce error with malformed user message")
	}
}

func TestUserMessageInternal_GetChannelMessage(t *testing.T) {
	internal, _, channelMsg := builtTestUMI(t, 7)
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
	internal, usrMsg, _ := builtTestUMI(t, 7)
	received := internal.GetUserMessage()

	if !reflect.DeepEqual(received.Message, usrMsg.Message) ||
		!reflect.DeepEqual(received.Signature, usrMsg.Signature) ||
		!reflect.DeepEqual(received.ECCPublicKey, usrMsg.ECCPublicKey) {
		t.Fatalf("GetUserMessage did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", usrMsg, received)
	}
}

func TestUserMessageInternal_GetMessageID(t *testing.T) {
	internal, usrMsg, _ := builtTestUMI(t, 7)
	received := internal.GetMessageID()

	chID := &id.ID{}

	expected := message.DeriveChannelMessageID(chID, 42, usrMsg.Message)

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("GetMessageID did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}
}

// Ensures the serialization has not changed, changing the message IDs. The
// protocol is tolerant of this because only the sender serializes, but it would
// be good to know when this changes. If this test breaks, report it, but it
// should be safe to update the expected.
func TestUserMessageInternal_GetMessageID_Consistency(t *testing.T) {
	expected := "MsgID-/9l5HhCSBPgz+CPw+PUBxO4EqmkCrG8z8/39ZUWj+ks="

	internal, _, _ := builtTestUMI(t, 7)

	received := internal.GetMessageID()

	if expected != received.String() {
		t.Fatalf("GetMessageID did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}
}

func builtTestUMI(t *testing.T, mt MessageType) (
	*userMessageInternal, *UserMessage, *ChannelMessage) {
	channelMsg := &ChannelMessage{
		Lease:       69,
		RoundID:     42,
		PayloadType: uint32(mt),
		Payload:     []byte("ban_badUSer"),
		Nickname:    "paul",
	}

	serialized, err := proto.Marshal(channelMsg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	usrMsg := &UserMessage{
		Message:      serialized,
		Signature:    []byte("sig2"),
		ECCPublicKey: []byte("key"),
	}

	chID := &id.ID{}

	internal, _ := newUserMessageInternal(usrMsg, chID)

	return internal, usrMsg, channelMsg
}
