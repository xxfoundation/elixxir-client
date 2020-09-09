package utility

import (
	"crypto/md5"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/format"
	"time"
)

const currentCmixMessageVersion = 0

type cmixMessageHandler struct{}

// SaveMessage saves the message as a versioned object at the specified key
// in the key value store.
func (cmh *cmixMessageHandler) SaveMessage(kv *versioned.KV, m interface{}, key string) error {
	msg := m.(format.Message)

	// Create versioned object
	obj := versioned.Object{
		Version:   currentCmixMessageVersion,
		Timestamp: time.Now(),
		Data:      msg.Marshal(),
	}

	// Save versioned object
	return kv.Set(key, &obj)
}

// LoadMessage returns the message with the specified key from the key value
// store. An empty message and error are returned if the message could not be
// retrieved.
func (cmh *cmixMessageHandler) LoadMessage(kv *versioned.KV, key string) (interface{}, error) {
	// Load the versioned object
	vo, err := kv.Get(key)
	if err != nil {
		return format.Message{}, err
	}

	// Create message from data
	return format.Unmarshal(vo.Data), nil
}

// DeleteMessage deletes the message with the specified key from the key value
// store.
func (cmh *cmixMessageHandler) DeleteMessage(kv *versioned.KV, key string) error {
	return kv.Delete(key)
}

// HashMessage generates a hash of the message.
func (cmh *cmixMessageHandler) HashMessage(m interface{}) MessageHash {
	msg := m.(format.Message)

	return md5.Sum(msg.Marshal())
}

// CmixMessageBuffer wraps the message buffer to store and load raw cmix
// messages.
type CmixMessageBuffer struct {
	mb *MessageBuffer
}

func NewCmixMessageBuffer(kv *versioned.KV, key string) (*CmixMessageBuffer, error) {
	mb, err := NewMessageBuffer(kv, &cmixMessageHandler{}, key)
	if err != nil {
		return nil, err
	}

	return &CmixMessageBuffer{mb: mb}, nil
}

func LoadCmixMessageBuffer(kv *versioned.KV, key string) (*CmixMessageBuffer, error) {
	mb, err := LoadMessageBuffer(kv, &cmixMessageHandler{}, key)
	if err != nil {
		return nil, err
	}

	return &CmixMessageBuffer{mb: mb}, nil
}

func (cmb *CmixMessageBuffer) Add(m format.Message) {
	cmb.mb.Add(m)
}

func (cmb *CmixMessageBuffer) Next() (format.Message, bool) {
	m, ok := cmb.mb.Next()
	if !ok {
		return format.Message{}, false
	}

	msg := m.(format.Message)
	return msg, true
}

func (cmb *CmixMessageBuffer) Succeeded(m format.Message) {
	cmb.mb.Succeeded(m)
}

func (cmb *CmixMessageBuffer) Failed(m format.Message) {
	cmb.mb.Failed(m)
}
