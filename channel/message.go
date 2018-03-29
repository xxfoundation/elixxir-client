package channel

import (
	"bytes"
	"encoding/gob"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/format"
)

// This is the message type that the subscribers to the channel send to the
// channel bot, by serialization.
type ChannelMessage struct {
	GroupID uint64
	// This is the same as the user ID of the person who sent the message to
	// the channel
	SpeakerID uint64
	// This holds the actual message.
	Message string
}

// Returns all channel message fields packed into a payload string
func (m ChannelMessage) SerializeChannelMessage() *bytes.Buffer {
	var result bytes.Buffer
	enc := gob.NewEncoder(&result)
	err := enc.Encode(m)
	if err != nil {
		jww.ERROR.Printf("Failed to encode gob for channel message: %v", err.Error())
	}
	return &result
}

func ParseChannelMessage(serializedChannelMessage *bytes.
Buffer) *ChannelMessage {
	dec := gob.NewDecoder(serializedChannelMessage)
	var result ChannelMessage
	err := dec.Decode(&result)
	if err != nil {
		jww.ERROR.Printf("Failed to decode gob for channel message: %v", err.Error())
	}
	return &result
}

func NewSerializedChannelMessages(GroupID, SpeakerID uint64,
	Message string) []*bytes.Buffer {
	// Try to serialize a gob with the message fields left untouched,
	// and see if it fits in the length of a message payload
	length := ChannelMessage{GroupID: GroupID, SpeakerID: SpeakerID,
		Message: Message}.SerializeChannelMessage()

	if uint64(length.Len()) > format.DATA_LEN {
		// If in this loop, the gob was too long and we need to break the
		// message up into multiple messages
		// TODO: we should have some sort of ordering guarantee for
		// sending these, like a unique identifier for the group of messages
		result := make([]*bytes.Buffer, 0)
		for uint64(length.Len()) > format.DATA_LEN {
			partition := uint64(length.Len()) - format.DATA_LEN
			nextMessagePayload := Message[partition:]
			Message = Message[:partition]

			nextChannelMessage := ChannelMessage{GroupID: GroupID,
				SpeakerID: SpeakerID, Message: nextMessagePayload}.SerializeChannelMessage()
			// prepend the resulting message
			result = append([]*bytes.Buffer{nextChannelMessage}, result...)
			length = ChannelMessage{GroupID: GroupID, SpeakerID: SpeakerID,
				Message: Message}.SerializeChannelMessage()
		}
		result = append([]*bytes.Buffer{length}, result...)
		return result
	} else {
		return []*bytes.Buffer{length}
	}
}
