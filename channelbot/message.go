////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import (
	"bytes"
	"encoding/gob"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/format"
)

// This is the message type that the subscribers to the channelbot send to the
// channel bot, by serialization.
type ChannelbotMessage struct {
	GroupID uint64
	// This is the same as the user ID of the person who sent the message to
	// the channelbot
	SpeakerID uint64
	// This holds the actual message.
	Message string
}

// Returns all channelbot message fields packed into a payload string
func (m ChannelbotMessage) SerializeChannelbotMessage() *bytes.Buffer {
	var result bytes.Buffer
	enc := gob.NewEncoder(&result)
	err := enc.Encode(m)
	if err != nil {
		jww.ERROR.Printf("Failed to encode gob for channelbot message: %v", err.Error())
	}
	return &result
}

func ParseChannelbotMessage(serializedChannelMessage *bytes.
Buffer) *ChannelbotMessage {
	dec := gob.NewDecoder(serializedChannelMessage)
	var result ChannelbotMessage
	err := dec.Decode(&result)
	if err != nil {
		jww.ERROR.Printf("Failed to decode gob for channelbot message: %v", err.Error())
	}
	return &result
}

func NewSerializedChannelbotMessages(GroupID, SpeakerID uint64,
	Message string) []*bytes.Buffer {
	// Try to serialize a gob with the message fields left untouched,
	// and see if it fits in the length of a message payload
	length := ChannelbotMessage{GroupID: GroupID, SpeakerID: SpeakerID,
		Message: Message}.SerializeChannelbotMessage()

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

			nextChannelMessage := ChannelbotMessage{GroupID: GroupID,
				SpeakerID: SpeakerID, Message: nextMessagePayload}.
					SerializeChannelbotMessage()
			// prepend the resulting message
			result = append([]*bytes.Buffer{nextChannelMessage}, result...)
			length = ChannelbotMessage{GroupID: GroupID, SpeakerID: SpeakerID,
				Message: Message}.SerializeChannelbotMessage()
		}
		result = append([]*bytes.Buffer{length}, result...)
		return result
	} else {
		return []*bytes.Buffer{length}
	}
}
