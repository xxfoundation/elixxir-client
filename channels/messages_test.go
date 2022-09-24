////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
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

func TestUnmarshalUserMessageInternal(t *testing.T) {
	internal, usrMsg, _ := builtTestUMI(t, 7)

	usrMsgMarshaled, err := proto.Marshal(usrMsg)
	if err != nil {
		t.Fatalf("Failed to marshal user message: %+v", err)
	}

	umi, err := unmarshalUserMessageInternal(usrMsgMarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal user message: %+v", err)
	}

	if !umi.GetMessageID().Equals(internal.messageID) {
		t.Errorf("Message IDs were changed in the unmarshal "+
			"process, %s vs %s", internal.messageID, umi.GetMessageID())
	}
}

func TestUnmarshalUserMessageInternal_BadUserMessage(t *testing.T) {
	_, err := unmarshalUserMessageInternal([]byte("Malformed"))
	if err == nil {
		t.Fatalf("Error not returned on unmarshaling a bad user " +
			"message")
	}
}

func TestUnmarshalUserMessageInternal_BadChannelMessage(t *testing.T) {
	_, usrMsg, _ := builtTestUMI(t, 7)

	usrMsg.Message = []byte("Malformed")

	usrMsgMarshaled, err := proto.Marshal(usrMsg)
	if err != nil {
		t.Fatalf("Failed to marshal user message: %+v", err)
	}

	_, err = unmarshalUserMessageInternal(usrMsgMarshaled)
	if err == nil {
		t.Fatalf("Error not returned on unmarshaling a user message " +
			"with a bad channel message")
	}
}

func TestNewUserMessageInternal_BadChannelMessage(t *testing.T) {
	_, usrMsg, _ := builtTestUMI(t, 7)

	usrMsg.Message = []byte("Malformed")

	_, err := newUserMessageInternal(usrMsg)

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
	expected := "ChMsgID-s425CTIAcKxvhUEZNr6Dk1g6rrOOpzKOS9L97OzLJ2w="

	internal, _, _ := builtTestUMI(t, 7)

	received := internal.GetMessageID()

	if expected != received.String() {
		t.Fatalf("GetMessageID did not return expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", expected, received)
	}
}

func builtTestUMI(t *testing.T, mt MessageType) (*userMessageInternal, *UserMessage, *ChannelMessage) {
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

	internal, _ := newUserMessageInternal(usrMsg)

	return internal, usrMsg, channelMsg
}
