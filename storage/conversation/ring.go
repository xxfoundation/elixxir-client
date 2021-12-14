///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

// Storage keys and versions.
const (
	ringBuffPrefix  = "ringBuffPrefix"
	ringBuffKey     = "ringBuffKey"
	ringBuffVersion = 0
	messageKey      = "ringBuffMessageKey"
	messageVersion  = 0
)

// Error messages.
const (
	saveMessageErr = "failed to save message with truncated ID %s to storage: %+v"
	loadMessageErr = "failed to load message with truncated ID %s to storage: %+v"
	loadBuffErr    = "failed to load ring buffer from storage: %+v"
)

// Buff is a circular buffer which containing Message's.
type Buff struct {
	buff           []*Message
	lookup         map[TruncatedMessageId]*Message
	oldest, newest int
	mux            sync.RWMutex
	kv             *versioned.KV
}

// NewBuff initializes a new ring buffer with size n.
func NewBuff(kv *versioned.KV, n int) (*Buff, error) {
	kv = kv.Prefix(ringBuffPrefix)
	rb := &Buff{
		buff:   make([]*Message, n),
		lookup: make(map[TruncatedMessageId]*Message, n),
		oldest: 0,
		newest: -1,
		kv:     kv,
	}

	return rb, rb.save()
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

func LoadBuff(kv *versioned.KV) (*Buff, error) {
	vo, err := kv.Get(ringBuffKey, ringBuffVersion)
	if err != nil {
		return nil, errors.Errorf(loadBuffErr, err)
	}

	newest, oldest, list := unmarshal(vo.Data)

	rb := &Buff{
		buff:   make([]*Message, len(list)),
		lookup: make(map[TruncatedMessageId]*Message, len(list)),
		oldest: oldest,
		newest: newest,
		mux:    sync.RWMutex{},
		kv:     kv,
	}

	for i, tmid := range list {
		// Load message from storage
		vo, err = kv.Get(makeMessageKey(tmid), messageVersion)
		if err != nil {
			return nil, errors.Errorf(loadMessageErr, tmid, err)
		}

		// Unmarshal message
		msg := unmarshalMessage(vo.Data)

		// Place message into reconstructed buffer (RAM)
		rb.lookup[tmid] = msg
		rb.buff[i] = msg
	}

	return rb, nil
}

// save stores the ring buffer and its elements to storage.
func (rb *Buff) save() error {
	rb.mux.Lock()
	defer rb.mux.Unlock()

	// Save each message individually to storage
	for _, msg := range rb.buff {
		if err := rb.saveMessage(msg); err != nil {
			return errors.Errorf(saveMessageErr,
				msg.MessageId.Truncate().String(), err)
		}
	}

	return rb.saveBuff()
}

// saveBuff is a function which saves the marshalled Buff.
func (rb *Buff) saveBuff() error {
	obj := &versioned.Object{
		Version:   ringBuffVersion,
		Timestamp: netTime.Now(),
		Data:      rb.marshal(),
	}

	return rb.kv.Set(ringBuffKey, ringBuffVersion, obj)

}

// marshal creates a byte buffer containing serialized information
// on the Buff.
func (rb *Buff) marshal() []byte {
	// Create buffer of proper size
	// (newest (4 bytes) + oldest (4 bytes) +
	// (TruncatedMessageIdLen * length of buffer)
	buff := bytes.NewBuffer(nil)
	buff.Grow(4 + 4 + (TruncatedMessageIdLen * len(rb.lookup)))

	// Write newest index into buffer
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(rb.newest))
	buff.Write(b)

	// Write oldest index into buffer
	b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(rb.oldest))
	buff.Write(b)

	// Write the truncated message IDs into buffer
	for _, msg := range rb.buff {
		buff.Write(msg.MessageId.Truncate().Bytes())
	}

	return buff.Bytes()
}

func (rb *Buff) saveMessage(msg *Message) error {
	obj := &versioned.Object{
		Version:   messageVersion,
		Timestamp: netTime.Now(),
		Data:      rb.marshal(),
	}

	return rb.kv.Set(
		makeMessageKey(msg.MessageId.Truncate()), messageVersion, obj)

}

// unmarshal unmarshalls a byte slice into Buff information.
func unmarshal(b []byte) (newest, oldest int,
	list []TruncatedMessageId) {
	buff := bytes.NewBuffer(b)

	// Read the newest index from the buffer
	newest = int(binary.LittleEndian.Uint32(buff.Next(4)))

	// Read the oldest index from the buffer
	oldest = int(binary.LittleEndian.Uint32(buff.Next(4)))

	// Initialize list to the number of truncated IDs
	list = make([]TruncatedMessageId, 0, buff.Len()/TruncatedMessageIdLen)

	// Read each TruncatedMessageId and save into list
	for next := buff.Next(TruncatedMessageIdLen); len(next) == TruncatedMessageIdLen; next = buff.Next(TruncatedMessageIdLen) {
		tmid := TruncatedMessageId{}
		copy(tmid[:], next)
		list = append(list, tmid)
	}

	return
}

// makeMessageKey generates te key used to save a message to storage.
func makeMessageKey(tmid TruncatedMessageId) string {
	return messageKey + tmid.String()
}
