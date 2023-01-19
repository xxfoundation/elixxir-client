////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"time"
)

// Constants for data length.
const (
	MessageIdLen          = 32
	TruncatedMessageIdLen = 8
)

// MessageID is the ID of a message stored in a Message.
type MessageID [MessageIdLen]byte

// truncatedMessageID represents the first64 bits of the MessageID.
type truncatedMessageID [TruncatedMessageIdLen]byte

// Message is the structure held in a ring buffer. It represents a received
// message by the user, which needs its reception verified to the original
// sender of the message.
type Message struct {
	id        uint32    // The sequential ID of the Message in the ring buffer
	MessageId MessageID // The ID of the message
	Timestamp time.Time
}

// newMessage is the constructor for a Message object.
func newMessage(id uint32, mid MessageID, timestamp time.Time) *Message {
	return &Message{
		id:        id,
		MessageId: mid,
		Timestamp: timestamp,
	}
}

// marshal creates a byte buffer containing the serialized information of a
// Message.
func (m *Message) marshal() []byte {
	buff := bytes.NewBuffer(nil)

	// Serialize and write the ID into a byte buffer
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, m.id)
	buff.Write(b)

	// Serialize and write the MessageID into a byte buffer
	buff.Write(m.MessageId.Bytes())

	// Serialize and write the timestamp into a byte buffer
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(m.Timestamp.UnixNano()))
	buff.Write(b)

	return buff.Bytes()
}

// unmarshalMessage deserializes byte data into a Message.
func unmarshalMessage(data []byte) *Message {
	buff := bytes.NewBuffer(data)

	// Deserialize the ID
	ID := binary.LittleEndian.Uint32(buff.Next(4))

	// Deserialize the message ID
	midData := buff.Next(MessageIdLen)
	mid := NewMessageIdFromBytes(midData)

	tsNano := binary.LittleEndian.Uint64(buff.Next(8))
	ts := time.Unix(0, int64(tsNano))

	return &Message{
		id:        ID,
		MessageId: mid,
		Timestamp: ts,
	}
}

// NewMessageIdFromBytes creates a MessageID from byte data.
func NewMessageIdFromBytes(data []byte) MessageID {
	mid := MessageID{}
	copy(mid[:], data)
	return mid
}

// String returns a base 64 encoding of the MessageID. This functions adheres to
// the fmt.Stringer interface.
func (mid MessageID) String() string {
	return base64.StdEncoding.EncodeToString(mid[:])
}

// truncate converts a MessageID into a truncatedMessageID.
func (mid MessageID) truncate() truncatedMessageID {
	return newTruncatedMessageID(mid.Bytes())
}

// Bytes returns the byte data of the MessageID.
func (mid MessageID) Bytes() []byte {
	return mid[:]
}

// newTruncatedMessageID creates a truncatedMessageID from byte data.
func newTruncatedMessageID(data []byte) truncatedMessageID {
	tmID := truncatedMessageID{}
	copy(tmID[:], data)
	return tmID

}

// String returns the base 64 encoding of the truncatedMessageID. This functions
// adheres to the fmt.Stringer interface.
func (tmID truncatedMessageID) String() string {
	return base64.StdEncoding.EncodeToString(tmID[:])

}

// Bytes returns the byte data of the truncatedMessageID.
func (tmID truncatedMessageID) Bytes() []byte {
	return tmID[:]
}
