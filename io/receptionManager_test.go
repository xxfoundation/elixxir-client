////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/primitives/format"
	"reflect"
	"testing"
)

// Tests that AppendGarbledMessage properly appends an array of messages by
// testing that the final buffer matches the values appended.
func TestSessionObj_AppendGarbledMessage(t *testing.T) {
	session := &ReceptionManager{
		garbledMessages: make([]*format.Message, 0),
	}
	msgs := GenerateTestMessages(10)

	session.AppendGarbledMessage(msgs...)

	if !reflect.DeepEqual(msgs, session.garbledMessages) {
		t.Errorf("AppendGarbledMessage() did not append the correct values"+
			"\n\texpected: %v\n\trecieved: %v",
			msgs, session.garbledMessages)
	}
}

// Tests that PopGarbledMessages returns the correct data and that the buffer
// is cleared.
func TestSessionObj_PopGarbledMessages(t *testing.T) {
	session := &ReceptionManager{
		garbledMessages: make([]*format.Message, 0),
	}
	msgs := GenerateTestMessages(10)

	session.garbledMessages = msgs

	poppedMsgs := session.PopGarbledMessages()

	if !reflect.DeepEqual(msgs, poppedMsgs) {
		t.Errorf("PopGarbledMessages() did not pop the correct values"+
			"\n\texpected: %v\n\trecieved: %v",
			msgs, poppedMsgs)
	}

	if !reflect.DeepEqual([]*format.Message{}, session.garbledMessages) {
		t.Errorf("PopGarbledMessages() did not remove the values from the buffer"+
			"\n\texpected: %#v\n\trecieved: %#v",
			[]*format.Message{}, session.garbledMessages)
	}

}

func GenerateTestMessages(size int) []*format.Message {
	msgs := make([]*format.Message, size)

	for i := 0; i < size; i++ {
		msgs[i] = format.NewMessage()
		payloadBytes := make([]byte, format.PayloadLen)
		payloadBytes[0] = byte(i)
		msgs[i].SetPayloadA(payloadBytes)
		msgs[i].SetPayloadB(payloadBytes)
	}

	return msgs
}
