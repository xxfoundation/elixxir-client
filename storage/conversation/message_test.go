///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"reflect"
	"testing"
	"time"
)

// TestMessage_MarshalUnmarshal tests whether a marshalled Message deserializes into
// the same Message using unmarshalMessage.
func TestMessage_MarshalUnmarshal(t *testing.T) {
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.Local)
	testId := NewMessageIdFromBytes([]byte("messageId123"))

	message := &Message{
		Id:        0,
		MessageId: testId,
		Timestamp: timestamp,
	}

	serialized := message.marshal()

	unmarshalled := unmarshalMessage(serialized)

	if !reflect.DeepEqual(unmarshalled, message) {
		t.Fatalf("Unmarshal did not output expected data."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", message, unmarshalled)
	}

}

// TestMessageId_truncate tests the MessageId truncate function.
func TestMessageId_truncate(t *testing.T) {
	testId := NewMessageIdFromBytes([]byte("This is going to be 32 bytes...."))

	tmid := testId.truncate()
	expected := truncatedMessageId{}
	copy(expected[:], testId.Bytes())
	if len(tmid.Bytes()) != TruncatedMessageIdLen {
		t.Fatalf("MessageId.Truncate() did not produce a truncatedMessageId of "+
			"TruncatedMessageIdLen (%d)."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", TruncatedMessageIdLen, expected, tmid)
	}

}
