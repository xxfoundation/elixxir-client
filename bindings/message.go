package bindings

import "gitlab.com/elixxir/client/interfaces/message"

// Message is a message received from the cMix network in the clear
// or that has been decrypted using established E2E keys.
type Message interface {
	//Returns the id of the message
	GetID() []byte

	// Returns the message's sender ID, if available
	GetSender() []byte

	// Returns the message payload/contents
	// Parse this with protobuf/whatever according to the message type
	GetPayload() []byte

	// Returns the message's recipient ID
	// This is usually your userID but could be an ephemeral/group ID
	GetRecipient() []byte

	// Returns the message's type
	GetMessageType() int

	// Returns the message's timestamp in milliseconds since unix epoc
	GetTimestampMS() int
	// Returns the message's timestamp in ns since unix epoc
	GetTimestampNano() int
}

type messageInternal struct {
	m message.Receive
}

//Returns the id of the message
func (mi messageInternal) GetID() []byte {
	return mi.m.ID[:]
}

// Returns the message's sender ID, if available
func (mi messageInternal) GetSender() []byte {
	return mi.m.Sender.Bytes()
}
