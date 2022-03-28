///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

/*
import (
	"encoding/binary"
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
)

const currentE2EMessageVersion = 0

type e2eMessageHandler struct{}

type e2eMessage struct {
	Recipient   []byte
	Payload     []byte
	MessageType uint32
	Params      params.E2E
}

// SaveMessage saves the e2eMessage as a versioned object at the specified key
// in the key value store.
func (emh *e2eMessageHandler) SaveMessage(kv *versioned.KV, m interface{}, key string) error {
	msg := m.(e2eMessage)

	b, err := json.Marshal(&msg)
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal e2e message for storage: %s", err)
	}

	// Create versioned object
	obj := versioned.Object{
		Version:   currentE2EMessageVersion,
		Timestamp: netTime.Now(),
		Data:      b,
	}

	// Save versioned object
	return kv.Set(key, currentE2EMessageVersion, &obj)
}

// LoadMessage returns the e2eMessage with the specified key from the key value
// store. An empty message and error are returned if the message could not be
// retrieved.
func (emh *e2eMessageHandler) LoadMessage(kv *versioned.KV, key string) (interface{}, error) {
	// Load the versioned object
	vo, err := kv.Get(key, currentE2EMessageVersion)
	if err != nil {
		return nil, err
	}

	// Unmarshal data into e2eMessage
	msg := e2eMessage{}
	if err := json.Unmarshal(vo.Data, &msg); err != nil {
		jww.FATAL.Panicf("Failed to unmarshal e2e message for storage: %s", err)
	}

	return msg, err
}

// DeleteMessage deletes the message with the specified key from the key value
// store.
func (emh *e2eMessageHandler) DeleteMessage(kv *versioned.KV, key string) error {
	return kv.Delete(key, currentE2EMessageVersion)
}

// HashMessage generates a hash of the e2eMessage.
// Do not include the params in the hash so it is not needed to resubmit the
// message into succeeded or failed
func (emh *e2eMessageHandler) HashMessage(m interface{}) MessageHash {
	h, _ := blake2b.New256(nil)

	msg := m.(e2eMessage)
	h.Write(msg.Recipient)
	h.Write(msg.Payload)
	mtBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(mtBytes, msg.MessageType)
	h.Write(mtBytes)

	var messageHash MessageHash
	copy(messageHash[:], h.Sum(nil))

	return messageHash
}

// E2eMessageBuffer wraps the message buffer to store and load raw e2eMessages.
type E2eMessageBuffer struct {
	mb *MessageBuffer
}

func NewE2eMessageBuffer(kv *versioned.KV, key string) (*E2eMessageBuffer, error) {
	mb, err := NewMessageBuffer(kv, &e2eMessageHandler{}, key)
	if err != nil {
		return nil, err
	}

	return &E2eMessageBuffer{mb: mb}, nil
}

func LoadE2eMessageBuffer(kv *versioned.KV, key string) (*E2eMessageBuffer, error) {
	mb, err := LoadMessageBuffer(kv, &e2eMessageHandler{}, key)
	if err != nil {
		return nil, err
	}

	return &E2eMessageBuffer{mb: mb}, nil
}

func (emb *E2eMessageBuffer) Add(m message.Send, p params.E2E) {
	e2eMsg := e2eMessage{
		Recipient:   m.Recipient.Marshal(),
		Payload:     m.Payload,
		MessageType: uint32(m.MessageType),
		Params:      p,
	}

	emb.mb.Add(e2eMsg)
}

func (emb *E2eMessageBuffer) AddProcessing(m message.Send, p params.E2E) {
	e2eMsg := e2eMessage{
		Recipient:   m.Recipient.Marshal(),
		Payload:     m.Payload,
		MessageType: uint32(m.MessageType),
		Params:      p,
	}

	emb.mb.AddProcessing(e2eMsg)
}

func (emb *E2eMessageBuffer) Next() (message.Send, params.E2E, bool) {
	m, ok := emb.mb.Next()
	if !ok {
		return message.Send{}, params.E2E{}, false
	}

	msg := m.(e2eMessage)
	recipient, err := id.Unmarshal(msg.Recipient)
	if err != nil {
		jww.FATAL.Panicf("Error unmarshaling Recipient: %v", err)
	}
	return message.Send{recipient, msg.Payload,
		message.Type(msg.MessageType)}, msg.Params, true
}

func (emb *E2eMessageBuffer) Succeeded(m message.Send, p params.E2E) {
	emb.mb.Succeeded(e2eMessage{m.Recipient.Marshal(),
		m.Payload, uint32(m.MessageType), p})
}

func (emb *E2eMessageBuffer) Failed(m message.Send, p params.E2E) {
	emb.mb.Failed(e2eMessage{m.Recipient.Marshal(),
		m.Payload, uint32(m.MessageType), p})
}
*/
