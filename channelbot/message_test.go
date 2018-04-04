////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import (
	"gitlab.com/privategrity/crypto/format"
	"strings"
	"testing"
)

func TestChannelMessageSerializationAndParsing(t *testing.T) {
	expected := ChannelbotMessage{5, "what do you guys think about straws?"}
	serialization := expected.SerializeChannelbotMessage()
	actual, err := ParseChannelbotMessage(serialization)
	if err != nil {
		t.Errorf("Error parsing channelbot message: %v", err.Error())
	}

	if actual.SpeakerID != expected.SpeakerID {
		t.Errorf("Speaker ID differed from expected. Expected: %v, got %v",
			expected.SpeakerID, actual.SpeakerID)
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
	expectedNumberOfMessages := uint64(len(longMessageToChannel))/format.
		DATA_LEN + 1

	if expectedNumberOfMessages != uint64(len(multipleSerializedMessages)) {
		t.Errorf("Got a different number of messages than expected. Got: %v," +
			" expected %v.")
	}
	message, err := ParseChannelbotMessage(multipleSerializedMessages[0])
	if err != nil {
		t.Errorf("Failed to parse first channelbot message: %v", err.Error())
	}
	if !strings.Contains(message.Message, "Beginning") {
		t.Errorf("First message didn't contain the beginning of the" +
			" long message")
	}

	message, err = ParseChannelbotMessage(multipleSerializedMessages[len(multipleSerializedMessages)-1])
	if !strings.Contains(message.Message, "end.") {
		t.Errorf("Last message didn't contain the end of the long message")
	}
}
