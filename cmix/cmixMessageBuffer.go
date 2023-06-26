////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
)

const currentCmixMessageVersion = 0

type cmixMessageHandler struct{}

type storedMessage struct {
	Msg       []byte     `json:"msg"`
	Recipient *id.ID     `json:"recipient"`
	Params    CMIXParams `json:"params"`
}

func (sm storedMessage) Marshal() []byte {
	data, err := json.Marshal(&sm)
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal stored message: %s", err)
	}

	return data
}

// SaveMessage saves the message as a versioned object at the specified key
// in the key value store.
func (cmh *cmixMessageHandler) SaveMessage(
	kv versioned.KV, m interface{}, key string) error {
	sm := m.(storedMessage)

	// Create versioned object
	obj := versioned.Object{
		Version:   currentCmixMessageVersion,
		Timestamp: netTime.Now(),
		Data:      sm.Marshal(),
	}

	// Save versioned object
	return kv.Set(key, &obj)
}

// LoadMessage returns the message with the specified key from the key value
// store. An empty message and error are returned if the message could not be
// retrieved.
func (cmh *cmixMessageHandler) LoadMessage(kv versioned.KV, key string) (
	interface{}, error) {

	// Load the versioned object
	vo, err := kv.Get(key, currentCmixMessageVersion)
	if err != nil {
		return format.Message{}, err
	}

	sm := storedMessage{}
	if err = json.Unmarshal(vo.Data, &sm); err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal stored message")
	}

	// Create message from data
	return sm, nil
}

// DeleteMessage deletes the message with the specified key from the key value
// store.
func (cmh *cmixMessageHandler) DeleteMessage(kv versioned.KV, key string) error {
	return kv.Delete(key, currentCmixMessageVersion)
}

// HashMessage generates a hash of the message.
func (cmh *cmixMessageHandler) HashMessage(m interface{}) utility.MessageHash {
	msg := m.(storedMessage)

	h, _ := blake2b.New256(nil)
	h.Write(msg.Recipient.Marshal())
	h.Write(msg.Msg)
	digest := h.Sum(nil)

	var messageHash utility.MessageHash
	copy(messageHash[:], digest)

	jww.TRACE.Printf("HashMessage Results: %s -> %s",
		base64.StdEncoding.EncodeToString(digest),
		base64.StdEncoding.EncodeToString(messageHash[:]))

	return messageHash
}

// CmixMessageBuffer wraps the message buffer to store and load raw cMix
// messages.
type CmixMessageBuffer struct {
	mb *utility.MessageBuffer
}

func NewOrLoadCmixMessageBuffer(kv versioned.KV, key string) (
	*CmixMessageBuffer, error) {

	cmb, err := LoadCmixMessageBuffer(kv, key)
	if err != nil {
		mb, err := utility.NewMessageBuffer(kv, &cmixMessageHandler{}, key)
		if err != nil {
			return nil, err
		}

		return &CmixMessageBuffer{mb: mb}, nil
	}

	return cmb, nil
}

func LoadCmixMessageBuffer(kv versioned.KV, key string) (*CmixMessageBuffer, error) {
	mb, err := utility.LoadMessageBuffer(kv, &cmixMessageHandler{}, key)
	if err != nil {
		return nil, err
	}

	return &CmixMessageBuffer{mb: mb}, nil
}

func (cmb *CmixMessageBuffer) Add(msg format.Message, recipient *id.ID,
	params CMIXParams) {
	sm := storedMessage{
		Msg:       msg.MarshalImmutable(),
		Recipient: recipient,
		Params:    params,
	}

	cmb.mb.Add(sm)
}

func (cmb *CmixMessageBuffer) AddProcessing(msg format.Message, recipient *id.ID,
	params CMIXParams) {
	sm := storedMessage{
		Msg:       msg.MarshalImmutable(),
		Recipient: recipient,
		Params:    params,
	}

	cmb.mb.AddProcessing(sm)
}

func (cmb *CmixMessageBuffer) Next() (format.Message, *id.ID, CMIXParams, bool) {
	m, ok := cmb.mb.Next()
	if !ok {
		return format.Message{}, nil, CMIXParams{}, false
	}

	sm := m.(storedMessage)
	msg, err := format.Unmarshal(sm.Msg)
	if err != nil {
		jww.FATAL.Panicf(
			"Could not unmarshal for stored cMix message buffer: %+v", err)
	}

	return msg, sm.Recipient, sm.Params, true
}

func (cmb *CmixMessageBuffer) Succeeded(msg format.Message, recipient *id.ID) {
	sm := storedMessage{
		Msg:       msg.MarshalImmutable(),
		Recipient: recipient,
	}

	cmb.mb.Succeeded(sm)
}

func (cmb *CmixMessageBuffer) Failed(msg format.Message, recipient *id.ID) {
	sm := storedMessage{
		Msg:       msg.MarshalImmutable(),
		Recipient: recipient,
	}

	cmb.mb.Failed(sm)
}
