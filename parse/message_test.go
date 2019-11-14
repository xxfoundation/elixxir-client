////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"bytes"
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
	"time"
)

//Shows that MessageHash ia an independent function of every field in Message
func TestMessage_Hash(t *testing.T) {
	m := Message{}
	m.MessageType = 0
	m.Body = []byte{0, 0}
	m.Sender = id.ZeroID
	m.Receiver = id.ZeroID
	m.Nonce = []byte{0, 0}

	baseHash := m.Hash()

	m.MessageType = 1

	typeHash := m.Hash()

	if reflect.DeepEqual(baseHash, typeHash) {
		t.Errorf("Message.Hash: Output did not change with modified type")
	}

	m.MessageType = 0

	m.Body = []byte{1, 1}

	bodyHash := m.Hash()

	if reflect.DeepEqual(baseHash, bodyHash) {
		t.Errorf("Message.Hash: Output did not change with modified body")
	}

	m.Body = []byte{0, 0}

	newID := id.NewUserFromUint(1, t)
	oldID := m.Sender
	m.Sender = newID

	senderHash := m.Hash()

	if reflect.DeepEqual(baseHash, senderHash) {
		t.Errorf("Message.Hash: Output did not change with modified sender")
	}

	m.Sender = oldID

	m.Receiver = newID

	receiverHash := m.Hash()

	if reflect.DeepEqual(baseHash, receiverHash) {
		t.Errorf("Message.Hash: Output did not change with modified receiver")
	}

	m.Receiver = oldID

	// FIXME Add a "bake" step to the message to partition and nonceify it
	// before hashing. We need this to be able to identify messages by their
	// hash on both the message's sending and receiving clients.
	//m.Nonce = []byte{1, 1}
	//
	//nonceHash := m.Hash()
	//
	//if reflect.DeepEqual(baseHash, nonceHash) {
	//	t.Errorf("Message.Hash: Output did not change with modified nonce")
	//}
	//
	//m.Nonce = []byte{0, 0}
}

func TestCryptoType_String(t *testing.T) {
	cs := CryptoType(0)

	if cs.String() != "None" {
		t.Errorf("String() did not return the correct value"+
			"\n\texpected: %s\n\treceived: %s",
			cs.String(), "None")
	}

	cs = CryptoType(1)

	if cs.String() != "Unencrypted" {
		t.Errorf("String() did not return the correct value"+
			"\n\texpected: %s\n\treceived: %s",
			cs.String(), "Unencrypted")
	}

	cs = CryptoType(2)

	if cs.String() != "Rekey" {
		t.Errorf("String() did not return the correct value"+
			"\n\texpected: %s\n\treceived: %s",
			cs.String(), "Rekey")
	}

	cs = CryptoType(3)

	if cs.String() != "E2E" {
		t.Errorf("String() did not return the correct value"+
			"\n\texpected: %s\n\treceived: %s",
			cs.String(), "E2E")
	}
}

func TestMessage_GetTimestamp(t *testing.T) {
	testTime := time.Now()
	message := Message{Timestamp: testTime}

	if message.GetTimestamp() != testTime {
		t.Errorf("GetTimestamp() did not return the correct timestamp"+
			"\n\texpected: %v\n\treceived: %v", message.GetTimestamp(), testTime)
	}
}

func TestBindingsMessageProxy_GetTimestamp(t *testing.T) {
	testTime := time.Now()
	message := BindingsMessageProxy{Proxy: &Message{Timestamp: testTime}}

	if message.GetTimestamp() != testTime.Unix() {
		t.Errorf("GetTimestamp() did not return the correct timestamp"+
			"\n\texpected: %v\n\treceived: %v", message.GetTimestamp(), testTime.Unix())
	}
}

func TestBindingsMessageProxy_GetMessage(t *testing.T) {
	expectedBody := []byte("testPayload")
	expectedMessageType := int32(42)
	message := BindingsMessageProxy{
		Proxy: &Message{TypedBody: TypedBody{MessageType: expectedMessageType, Body: expectedBody}}}
	observedBody := message.GetPayload()
	observedMessageType := message.GetMessageType()

	if bytes.Compare(expectedBody, observedBody) != 0 {
		t.Errorf("Failed to retrieve the body from the message. Expected %v Recieved: %v",
			expectedBody, observedBody)
	}

	if expectedMessageType != observedMessageType {
		t.Errorf("Failed to retrieve the messageType from the message. Expected: %v Recieved: %v",
			expectedMessageType, observedMessageType)
	}
}

func TestBindingsMessageProxy_GetTimestampNano(t *testing.T) {
	testTime := time.Now()
	message := BindingsMessageProxy{Proxy: &Message{Timestamp: testTime}}

	if message.GetTimestampNano() != testTime.UnixNano() {
		t.Errorf("GetTimestampNano() did not return the correct timestamp"+
			"\n\texpected: %v\n\treceived: %v", message.GetTimestampNano(), testTime.UnixNano())
	}
}
