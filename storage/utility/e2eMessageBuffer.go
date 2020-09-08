package utility

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"time"
)

const currentE2EMessageVersion = 0

type e2eMessageHandler struct{}

type e2eMessage struct {
	Recipient   []byte
	Payload     []byte
	MessageType uint32
}

// saveMessage saves the message as a versioned object.
func (emh *e2eMessageHandler) SaveMessage(kv *versioned.KV, m interface{}, key string) error {
	msg := m.(e2eMessage)

	b, err := json.Marshal(&msg)
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal e2e message for "+
			"storage: %s", err)
	}

	// Create versioned object
	obj := versioned.Object{
		Version:   currentE2EMessageVersion,
		Timestamp: time.Now(),
		Data:      b,
	}

	// Save versioned object
	return kv.Set(key, &obj)
}

// loadMessage loads the message with the specified key.
func (emh *e2eMessageHandler) LoadMessage(kv *versioned.KV, key string) (interface{}, error) {
	// Load the versioned object
	vo, err := kv.Get(key)
	if err != nil {
		return nil, err
	}

	msg := e2eMessage{}

	if err := json.Unmarshal(vo.Data, &msg); err != nil {
		jww.FATAL.Panicf("Failed to unmarshal e2e message for "+
			"storage: %s", err)
	}
	// Create message from data
	return msg, err
}

// DeleteMessage deletes the message with the specified key.
func (emh *e2eMessageHandler) DeleteMessage(kv *versioned.KV, key string) error {
	return kv.Delete(key)
}

// hashMessage generates a hash of the message.
func (emh *e2eMessageHandler) HashMessage(m interface{}) MessageHash {
	msg := m.(e2eMessage)

	var digest []byte
	digest = append(digest, msg.Recipient...)
	digest = append(digest, msg.Payload...)

	mtBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(mtBytes, msg.MessageType)
	digest = append(digest, mtBytes...)

	// Create message from data
	return md5.Sum(digest)
}
