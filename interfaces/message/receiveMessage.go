package message

import (
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type Receive struct {
	ID          e2e.MessageID
	Payload     []byte
	MessageType Type
	Sender      *id.ID
	Timestamp   time.Time
	Encryption  EncryptionType
}

//Returns the id of the message
func (r Receive) GetID() []byte {
	return r.ID[:]
}

// Returns the message's sender ID, if available
func (r Receive) GetSender() []byte {
	return r.Sender.Bytes()
}

// Returns the message's payload/contents
func (r Receive) GetPayload() []byte {
	return r.Payload
}

// Returns the message's type
func (r Receive) GetMessageType() int {
	return int(r.MessageType)
}

// Returns the message's timestamp in ms
func (r Receive) GetTimestampMS() int {
	return int(r.Timestamp.Unix())
}

func (r Receive) GetTimestampNano() int {
	return int(r.Timestamp.UnixNano())
}