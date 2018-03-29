package channel

import (
	"testing"
	"gitlab.com/privategrity/crypto/format"
)

func TestChannelMessageSerializationAndParsing(t *testing.T) {
	expected := ChannelMessage{1, 5, "what do you guys think about straws?"}
	serialization := expected.SerializeChannelMessage()
	println(serialization.String())
	actual := ParseChannelMessage(serialization)

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

func TestChannelMessageInNormalPayload(t *testing.T) {
	expected := ChannelMessage{1, 5, "what do you guys think about straws?"}
	serialization := expected.SerializeChannelMessage()
	// in this example, the bot would be user 8 and would be sending to user 5,
	// the only user in the channel
	// TODO automate the generation of these messages to channel subscribers
	messages, err := format.NewMessage(8, 5, serialization.String())
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(messages) > 1 {
		// the message was too long for one buffer. this is the condition that
		// automating the message generation will ameliorate
		t.Errorf("Too many messages: %v messages generated", len(messages))
	}
	t.Logf("First message string: %v", messages[0].GetPayload())
}
