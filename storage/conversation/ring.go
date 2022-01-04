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
	"math"
	"sync"
	"time"
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
	saveMessageErr      = "failed to save message with message ID %s to storage: %+v"
	loadMessageErr      = "failed to load message with truncated ID %s from storage: %+v"
	loadBuffErr         = "failed to load ring buffer from storage: %+v"
	noMessageFoundErr   = "failed to find message with message ID %s"
	lookupTooOldErr     = "requested ID %d is lower than oldest id %d"
	lookupPastRecentErr = "requested id %d is higher than most recent id %d"
)

// Buff is a circular buffer which containing Message's.
type Buff struct {
	buff           []*Message
	lookup         map[truncatedMessageId]*Message
	oldest, newest uint32
	mux            sync.RWMutex
	kv             *versioned.KV
}

// NewBuff initializes a new ring buffer with size n.
func NewBuff(kv *versioned.KV, n int) (*Buff, error) {
	kv = kv.Prefix(ringBuffPrefix)

	// Construct object
	rb := &Buff{
		buff:   make([]*Message, n),
		lookup: make(map[truncatedMessageId]*Message, n),
		oldest: 0,
		// Set to max int since index is unsigned.
		// Upon first insert, index will overflow back to zero.
		newest: math.MaxUint32,
		kv:     kv,
	}

	// Save to storage and return
	return rb, rb.save()
}

// Add pushes a message to the circular buffer Buff.
func (rb *Buff) Add(id MessageId, timestamp time.Time) error {
	rb.mux.Lock()
	defer rb.mux.Unlock()
	rb.push(&Message{
		MessageId: id,
		Timestamp: timestamp,
	})

	return rb.save()
}

// Get retrieves the most recent entry.
func (rb *Buff) Get() *Message {
	rb.mux.RLock()
	defer rb.mux.RUnlock()

	mostRecentIndex := rb.newest % uint32(len(rb.buff))
	return rb.buff[mostRecentIndex]

}

// GetByMessageId looks up and returns the message with MessageId id from
// Buff.lookup. If the message does not exist, an error is returned.
func (rb *Buff) GetByMessageId(id MessageId) (*Message, error) {
	rb.mux.RLock()
	defer rb.mux.RUnlock()

	// Look up message
	msg, exists := rb.lookup[id.truncate()]
	if !exists { // If message not found, return an error
		return nil, errors.Errorf(noMessageFoundErr, id)
	}

	// Return message if found
	return msg, nil
}

// GetNextMessage looks up the Message with the next sequential Message.id
// in the ring buffer after the Message with the requested MessageId.
func (rb *Buff) GetNextMessage(id MessageId) (*Message, error) {
	rb.mux.RLock()
	defer rb.mux.RUnlock()

	// Look up message
	msg, exists := rb.lookup[id.truncate()]
	if !exists { // If message not found, return an error
		return nil, errors.Errorf(noMessageFoundErr, id)
	}

	lookupId := msg.id + 1

	// Check it's not before our first known id
	if lookupId < rb.oldest {
		return nil, errors.Errorf(lookupTooOldErr, id, rb.oldest)
	}

	// Check it's not after our last known id
	if lookupId > rb.newest {
		return nil, errors.Errorf(lookupPastRecentErr, id, rb.newest)
	}

	return rb.buff[(lookupId % uint32(len(rb.buff)))], nil
}

// next is a helper function for Buff, which handles incrementing
// the old & new markers.
func (rb *Buff) next() {
	rb.newest++
	if rb.newest >= uint32(len(rb.buff)) {
		rb.oldest++
	}
}

// push adds a Message to the Buff, clearing the overwritten message from
// both the buff and the lookup structures.
func (rb *Buff) push(val *Message) {
	// Update circular buffer trackers
	rb.next()

	val.id = rb.newest

	// Handle overwrite of the oldest message
	rb.handleMessageOverwrite()

	// Set message in RAM
	rb.buff[rb.newest%uint32(len(rb.buff))] = val
	rb.lookup[val.MessageId.truncate()] = val

}

// handleMessageOverwrite is a helper function which deletes the message
// that will be overwritten by push from the lookup structure.
func (rb *Buff) handleMessageOverwrite() {
	overwriteIndex := rb.newest % uint32(len(rb.buff))
	messageToOverwrite := rb.buff[overwriteIndex]
	if messageToOverwrite != nil {
		delete(rb.lookup, messageToOverwrite.MessageId.truncate())
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadBuff loads the ring buffer from storage. It loads all
// messages from storage and repopulates the buffer.
func LoadBuff(kv *versioned.KV) (*Buff, error) {
	kv = kv.Prefix(ringBuffPrefix)

	// Extract ring buffer from storage
	vo, err := kv.Get(ringBuffKey, ringBuffVersion)
	if err != nil {
		return nil, errors.Errorf(loadBuffErr, err)
	}

	// Unmarshal ring buffer from data
	newest, oldest, list := unmarshalBuffer(vo.Data)

	// Construct buffer
	rb := &Buff{
		buff:   make([]*Message, len(list)),
		lookup: make(map[truncatedMessageId]*Message, len(list)),
		oldest: oldest,
		newest: newest,
		mux:    sync.RWMutex{},
		kv:     kv,
	}

	// Load each message from storage
	for i, tmid := range list {
		msg, err := loadMessage(tmid, kv)
		if err != nil {
			return nil, err
		}

		// Place message into reconstructed buffer (RAM)
		rb.lookup[tmid] = msg
		rb.buff[i] = msg
	}

	return rb, nil
}

// save stores the ring buffer and its elements to storage.
// NOTE: save is unsafe, a lock should be held by the caller.
func (rb *Buff) save() error {

	// Save each message individually to storage
	for _, msg := range rb.buff {
		if msg != nil {
			if err := rb.saveMessage(msg); err != nil {
				return errors.Errorf(saveMessageErr,
					msg.MessageId, err)
			}
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
		if msg != nil {
			buff.Write(msg.MessageId.truncate().Bytes())
		}
	}

	return buff.Bytes()
}

// unmarshalBuffer unmarshalls a byte slice into Buff information.
func unmarshalBuffer(b []byte) (newest, oldest uint32,
	list []truncatedMessageId) {
	buff := bytes.NewBuffer(b)

	// Read the newest index from the buffer
	newest = binary.LittleEndian.Uint32(buff.Next(4))

	// Read the oldest index from the buffer
	oldest = binary.LittleEndian.Uint32(buff.Next(4))

	// Initialize list to the number of truncated IDs
	list = make([]truncatedMessageId, 0, buff.Len()/TruncatedMessageIdLen)

	// Read each truncatedMessageId and save into list
	for next := buff.Next(TruncatedMessageIdLen); len(next) == TruncatedMessageIdLen; next = buff.Next(TruncatedMessageIdLen) {
		list = append(list, newTruncatedMessageId(next))
	}

	return
}

// saveMessage saves a Message to storage, using the truncatedMessageId
// as the KV key.
func (rb *Buff) saveMessage(msg *Message) error {
	obj := &versioned.Object{
		Version:   messageVersion,
		Timestamp: netTime.Now(),
		Data:      msg.marshal(),
	}

	return rb.kv.Set(
		makeMessageKey(msg.MessageId.truncate()), messageVersion, obj)

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

// makeMessageKey generates te key used to save a message to storage.
func makeMessageKey(tmid truncatedMessageId) string {
	return messageKey + tmid.String()
}
