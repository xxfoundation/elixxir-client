package conversation

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"time"
)

// Constants for data length.
const (
	MessageIdLen          = 32
	TruncatedMessageIdLen = 8
)

// MessageId is the ID of a message stored in a Message.
type MessageId [MessageIdLen]byte

// truncatedMessageId represents the first64 bits of the MessageId.
type truncatedMessageId [TruncatedMessageIdLen]byte

// A Message is the structure held in a ring buffer.
// It represents a received message by the user, which needs
// its reception verified to the original sender of the message.
type Message struct {
	// Id is the sequential ID of the Message in the ring buffer
	Id uint32
	// The ID of the message
	MessageId MessageId
	Timestamp time.Time
}

// loadMessage loads a message given truncatedMessageId from storage.
func loadMessage(tmid truncatedMessageId, kv *versioned.KV) (*Message, error) {
	// Load message from storage
	vo, err := kv.Get(makeMessageKey(tmid), messageVersion)
	if err != nil {
		return nil, errors.Errorf(loadMessageErr, tmid, err)
	}

	// Unmarshal message
	return unmarshalMessage(vo.Data), nil
}

// marshal creates a byte buffer containing the serialized information
// of a Message.
func (m *Message) marshal() []byte {
	buff := bytes.NewBuffer(nil)

	// Serialize and write the ID into a byte buffer
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, m.Id)
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
	mid := MessageId{}
	copy(mid[:], midData)

	tsNano := binary.LittleEndian.Uint64(buff.Next(8))
	ts := time.Unix(0, int64(tsNano))

	return &Message{
		Id:        ID,
		MessageId: mid,
		Timestamp: ts,
	}

}

// String returns a base64 encode of the MessageId. This functions
// satisfies the fmt.Stringer interface.
func (mid MessageId) String() string {
	return base64.StdEncoding.EncodeToString(mid[:])
}

// Truncate converts a MessageId into a truncatedMessageId.
func (mid MessageId) Truncate() truncatedMessageId {
	tmid := truncatedMessageId{}
	copy(tmid[:], mid[:])
	return tmid
}

// Bytes returns the byte data of the MessageId.
func (mid MessageId) Bytes() []byte {
	return mid[:]
}

// String returns a base64 encode of the truncatedMessageId. This functions
// satisfies the fmt.Stringer interface.
func (tmid truncatedMessageId) String() string {
	return base64.StdEncoding.EncodeToString(tmid[:])

}

// Bytes returns the byte data of the truncatedMessageId.
func (tmid truncatedMessageId) Bytes() []byte {
	return tmid[:]
}
