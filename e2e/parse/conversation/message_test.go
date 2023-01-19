////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

// Tests whether a marshalled Message deserializes into the same Message using
// unmarshalMessage.
func TestMessage_Marshal_unmarshalMessage(t *testing.T) {
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.Local)
	testId := NewMessageIdFromBytes([]byte("messageId123"))

	message := &Message{
		id:        0,
		MessageId: testId,
		Timestamp: timestamp,
	}

	serialized := message.marshal()

	unmarshalled := unmarshalMessage(serialized)

	if !reflect.DeepEqual(unmarshalled, message) {
		t.Errorf("Unmarshal did not output expected data."+
			"\nexpected: %v\nreceived: %v", message, unmarshalled)
	}

}

// Tests the MessageID truncate function.
func TestMessageID_truncate(t *testing.T) {
	testID := NewMessageIdFromBytes([]byte("This is going to be 32 bytes..."))

	tmID := testID.truncate()
	expected := truncatedMessageID{}
	copy(expected[:], testID.Bytes())
	if len(tmID.Bytes()) != TruncatedMessageIdLen {
		t.Errorf("truncatedMessageID has incorrect length."+
			"\nexpected: %v\nreceived: %v", expected, tmID)
	}
}

// Tests that NewMessageIdFromBytes properly constructs a MessageID.
func TestNewMessageIdFromBytes(t *testing.T) {
	expected := make([]byte, MessageIdLen)
	for i := range expected {
		expected[i] = byte(i)
	}

	testId := NewMessageIdFromBytes(expected)
	if !bytes.Equal(expected, testId.Bytes()) {
		t.Errorf("Unexpected output from NewMessageIdFromBytes."+
			"\nexpected: %v\nreceived: %v", expected, testId.Bytes())
	}

}

// Tests that newTruncatedMessageID constructs a proper truncatedMessageID.
func TestNewTruncatedMessageId(t *testing.T) {
	expected := make([]byte, 0, TruncatedMessageIdLen)
	for i := 0; i < TruncatedMessageIdLen; i++ {
		expected = append(expected, byte(i))
	}
	testId := newTruncatedMessageID(expected)
	if !bytes.Equal(expected, testId.Bytes()) {
		t.Fatalf("Unexpected output from newTruncatedMessageID."+
			"\nexpected: %v\nreceived: %v", expected, testId.Bytes())
	}
}
