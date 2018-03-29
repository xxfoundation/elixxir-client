package channel

import (
	"bytes"
	"encoding/gob"
	jww "github.com/spf13/jwalterweatherman"
)

// This is the message type that the subscribers to the channel send to the
// channel bot, by serialization.
type ChannelMessage struct {
	GroupID uint64
	// TODO is this the same as the user ID or should each channel map user IDs to speaker IDs somehow
	SpeakerID uint64
	// TODO handle longer channel messages
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
