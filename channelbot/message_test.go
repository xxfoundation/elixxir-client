////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import (
	"testing"
	"gitlab.com/privategrity/crypto/format"
	"strings"
)

func TestChannelMessageSerializationAndParsing(t *testing.T) {
	expected := ChannelbotMessage{1, 5, "what do you guys think about straws?"}
	serialization := expected.SerializeChannelbotMessage()
	actual := ParseChannelbotMessage(serialization)

	if actual.SpeakerID != expected.SpeakerID {
		t.Errorf("Speaker ID differed from expected. Expected: %v, got %v",
			expected.SpeakerID, actual.SpeakerID)
	}
	if actual.GroupID != expected.GroupID {
		t.Errorf("Group ID differed from expected. Expected: %v, got %v",
			expected.GroupID, actual.GroupID)
	}
	if actual.Message != expected.Message {
		t.Errorf("Message differed from expected. Expected: %v, got %v",
			expected.Message, actual.Message)
	}
}

func TestNewSerializedChannelMessages(t *testing.T) {
	longMessageToChannel := "Beginning pretium venenatis dui vitae rhoncus. " +
		"Nunc ut lorem id arcu eleifend porta ac a orci. " +
		"Quisque mattis maximus porta. Sed congue, " +
		"libero in ornare tincidunt, felis nunc tincidunt odio, " +
		"eu pharetra nisi tortor non mauris. In nunc odio, " +
		"vehicula eget dolor a, pretium fringilla lectus. " +
		"Sed at placerat neque. Nulla pellentesque vestibulum nulla quis" +
		" vulputate. Quisque ut tellus a orci vehicula facilisis. " +
		"Aliquam pretium venenatis dui vitae rhoncus. " +
		"Nunc ut lorem id arcu eleifend porta ac a orci. " +
		"Quisque mattis maximus porta. Sed congue, " +
		"libero in ornare tincidunt, felis nunc tincidunt odio, " +
		"eu pharetra nisi tortor non mauris. In nunc odio, " +
		"vehicula eget dolor a, pretium fringilla lectus. " +
		"Sed at placerat neque. Nulla pellentesque vestibulum nulla quis" +
		" vulputate. Quisque ut tellus a orci vehicula facilisis." +
		"Nunc ut lorem id arcu eleifend porta ac a orci. " +
		"Quisque mattis maximus porta. Sed congue, " +
		"libero in ornare tincidunt, felis nunc tincidunt odio, " +
		"eu pharetra nisi tortor non mauris. In nunc odio, " +
		"vehicula eget dolor a, pretium fringilla lectus. " +
		"Sed at placerat neque. Nulla pellentesque vestibulum nulla quis" +
		" vulputate. Quisque ut tellus a orci vehicula facilisis. " +
		"Aliquam pretium venenatis dui vitae rhoncus. " +
		"Nunc ut lorem id arcu eleifend porta ac a orci. " +
		"Quisque mattis maximus porta. Sed congue, " +
		"libero in ornare tincidunt, felis nunc tincidunt odio, " +
		"eu pharetra nisi tortor non mauris. In nunc odio, " +
		"vehicula eget dolor a, pretium fringilla lectus. " +
		"Sed at placerat neque. Nulla pellentesque vestibulum nulla quis end."

	multipleSerializedMessages := NewSerializedChannelbotMessages(1, 5,
		longMessageToChannel)

	// if there isn't too much metadata embedded in the channelbot messages,
	// you can expect this number of submessages to be needed.
	expectedNumberOfMessages := uint64(len(longMessageToChannel)) / format.
		DATA_LEN + 1

	if expectedNumberOfMessages != uint64(len(multipleSerializedMessages)) {
		t.Errorf("Got a different number of messages than expected. Got: %v," +
			" expected %v.")
	}
	if !strings.Contains(ParseChannelbotMessage(
		multipleSerializedMessages[0]).Message, "Beginning") {
		t.Errorf("First message didn't contain the beginning of the" +
			" long message")
	}
	if !strings.Contains(ParseChannelbotMessage(
		multipleSerializedMessages[len(multipleSerializedMessages)-1]).
		Message, "end.") {
		t.Errorf("Last message didn't contain the end of the long message")
	}
}
