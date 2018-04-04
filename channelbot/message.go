////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channelbot

import (
	"bytes"
	"encoding/gob"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/crypto/format"
)

// This is the message type that the subscribers to the channelbot send to the
// channel bot, by serialization.
// TODO: serialize this for the network/cross-language communication using something better than gobs
type ChannelbotMessage struct {
	// This is the same as the user ID of the person who sent the message to
	// the channelbot
	SpeakerID uint64
	// This holds the actual message.
	Message string
}

// Returns all channelbot message fields packed into a payload string
func (m ChannelbotMessage) SerializeChannelbotMessage() string {
	var result bytes.Buffer
	enc := gob.NewEncoder(&result)
	err := enc.Encode(m)
	if err != nil {
		jww.ERROR.Printf("Failed to encode gob for channelbot message: %v", err.Error())
	}
	return result.String()
}

func ParseChannelbotMessage(
	serializedChannelMessage string) (*ChannelbotMessage, error) {
	dec := gob.NewDecoder(bytes.NewBufferString(serializedChannelMessage))
	var result ChannelbotMessage
	err := dec.Decode(&result)
	if err != nil {
		err = errors.New("Failed to decode gob for channelbot message: " + err.Error())
	}
	return &result, err
}

func NewSerializedChannelbotMessages(GroupID, SpeakerID uint64,
	Message string) []string {
	// Try to serialize a gob with the message fields left untouched,
	// and see if it fits in the length of a message payload
	length := ChannelbotMessage{SpeakerID: SpeakerID,
		Message: Message}.SerializeChannelbotMessage()

	if uint64(len(length)) > format.DATA_LEN {
		// If in this loop, the gob was too long and we need to break the
		// message up into multiple messages
		// TODO: we should have some sort of ordering guarantee for
		// sending these, like a unique identifier for the group of messages
		result := make([]string, 0)
		for uint64(len(length)) > format.DATA_LEN {
			partition := uint64(len(length)) - format.DATA_LEN
			nextMessagePayload := Message[partition:]
			Message = Message[:partition]

			nextChannelMessage := ChannelbotMessage{
				SpeakerID: SpeakerID, Message: nextMessagePayload}.
				SerializeChannelbotMessage()
			// prepend the resulting message
			result = append([]string{nextChannelMessage}, result...)
			length = ChannelbotMessage{SpeakerID: SpeakerID,
				Message: Message}.SerializeChannelbotMessage()
		}
		result = append([]string{length}, result...)
		return result
	} else {
		return []string{length}
	}
}
