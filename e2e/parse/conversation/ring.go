////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/xx_network/primitives/netTime"
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
	lookupTooOldErr     = "requested ID %d is lower than oldest ID %d"
	lookupPastRecentErr = "requested ID %d is higher than most recent ID %d"
)

// Buff is a circular buffer which containing Message's.
type Buff struct {
	buff           []*Message
	lookup         map[truncatedMessageID]*Message
	oldest, newest uint32
	mux            sync.RWMutex
	kv             versioned.KV
}

// NewBuff initializes a new ring buffer with size n.
func NewBuff(kv versioned.KV, n int) (*Buff, error) {
	kv, err := kv.Prefix(ringBuffPrefix)
	if err != nil {
		return nil, err
	}

	// Construct object
	rb := &Buff{
		buff:   make([]*Message, n),
		lookup: make(map[truncatedMessageID]*Message, n),
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
func (b *Buff) Add(id MessageID, timestamp time.Time) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.push(&Message{
		MessageId: id,
		Timestamp: timestamp,
	})

	return b.save()
}

// Get retrieves the most recent entry.
func (b *Buff) Get() *Message {
	b.mux.RLock()
	defer b.mux.RUnlock()

	mostRecentIndex := b.newest % uint32(len(b.buff))
	return b.buff[mostRecentIndex]
}

// GetByMessageID looks up and returns the message with MessageID ID from the
// lookup map. If the message does not exist, an error is returned.
func (b *Buff) GetByMessageID(id MessageID) (*Message, error) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	// Look up message
	msg, exists := b.lookup[id.truncate()]
	if !exists {
		return nil, errors.Errorf(noMessageFoundErr, id)
	}

	// Return message if found
	return msg, nil
}

// GetNextMessage looks up the Message with the next sequential MessageID in the
// ring buffer after the Message with the requested MessageID.
func (b *Buff) GetNextMessage(id MessageID) (*Message, error) {
	b.mux.RLock()
	defer b.mux.RUnlock()

	// Look up message
	msg, exists := b.lookup[id.truncate()]
	if !exists {
		return nil, errors.Errorf(noMessageFoundErr, id)
	}

	lookupId := msg.id + 1

	// Check that it is not before our first known ID
	if lookupId < b.oldest {
		return nil, errors.Errorf(lookupTooOldErr, id, b.oldest)
	}

	// Check that it is not after our last known ID
	if lookupId > b.newest {
		return nil, errors.Errorf(lookupPastRecentErr, id, b.newest)
	}

	return b.buff[(lookupId % uint32(len(b.buff)))], nil
}

// next handles incrementing the old and new markers.
func (b *Buff) next() {
	b.newest++
	if b.newest >= uint32(len(b.buff)) {
		b.oldest++
	}
}

// push adds a Message to the Buff, clearing the overwritten message from both
// the buff and the lookup structures.
func (b *Buff) push(val *Message) {
	// Update circular buffer trackers
	b.next()

	val.id = b.newest

	// Handle overwrite of the oldest message
	b.handleMessageOverwrite()

	// Set message in RAM
	b.buff[b.newest%uint32(len(b.buff))] = val
	b.lookup[val.MessageId.truncate()] = val

}

// handleMessageOverwrite deletes the message that will be overwritten by push
// from the lookup structure.
func (b *Buff) handleMessageOverwrite() {
	overwriteIndex := b.newest % uint32(len(b.buff))
	messageToOverwrite := b.buff[overwriteIndex]
	if messageToOverwrite != nil {
		delete(b.lookup, messageToOverwrite.MessageId.truncate())
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// LoadBuff loads the ring buffer from storage. It loads all messages from
// storage and repopulates the buffer.
func LoadBuff(kv versioned.KV) (*Buff, error) {
	kv, err := kv.Prefix(ringBuffPrefix)
	if err != nil {
		return nil, err
	}

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
		lookup: make(map[truncatedMessageID]*Message, len(list)),
		oldest: oldest,
		newest: newest,
		mux:    sync.RWMutex{},
		kv:     kv,
	}

	// Load each message from storage
	for i, tmID := range list {
		msg, err := loadMessage(tmID, kv)
		if err != nil {
			return nil, err
		}

		// Place message into reconstructed buffer in memory
		rb.lookup[tmID] = msg
		rb.buff[i] = msg
	}

	return rb, nil
}

// save stores the ring buffer and its elements to storage.
// NOTE: This function is not thread-safe; a lock should be held by the caller.
func (b *Buff) save() error {

	// Save each message individually to storage
	for _, msg := range b.buff {
		if msg != nil {
			if err := b.saveMessage(msg); err != nil {
				return errors.Errorf(saveMessageErr, msg.MessageId, err)
			}
		}
	}

	return b.saveBuff()
}

// saveBuff saves the marshalled Buff to storage.
func (b *Buff) saveBuff() error {
	obj := &versioned.Object{
		Version:   ringBuffVersion,
		Timestamp: netTime.Now(),
		Data:      b.marshal(),
	}

	return b.kv.Set(ringBuffKey, obj)
}

// marshal creates a byte buffer containing serialized information on the Buff.
func (b *Buff) marshal() []byte {
	// Create buffer of proper size
	// (newest (4) + oldest (4) + (TruncatedMessageIdLen * length of buffer))
	buff := bytes.NewBuffer(nil)
	buff.Grow(4 + 4 + (TruncatedMessageIdLen * len(b.lookup)))

	// Write newest index into buffer
	bb := make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, b.newest)
	buff.Write(bb)

	// Write oldest index into buffer
	bb = make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, b.oldest)
	buff.Write(bb)

	// Write the truncated message IDs into buffer
	for _, msg := range b.buff {
		if msg != nil {
			buff.Write(msg.MessageId.truncate().Bytes())
		}
	}

	return buff.Bytes()
}

// unmarshalBuffer unmarshalls a byte slice into Buff information.
func unmarshalBuffer(b []byte) (
	newest, oldest uint32, list []truncatedMessageID) {
	buff := bytes.NewBuffer(b)

	// Read the newest index from the buffer
	newest = binary.LittleEndian.Uint32(buff.Next(4))

	// Read the oldest index from the buffer
	oldest = binary.LittleEndian.Uint32(buff.Next(4))

	// Initialize list to the number of truncated IDs
	list = make([]truncatedMessageID, 0, buff.Len()/TruncatedMessageIdLen)

	// Read each truncatedMessageID and save into list
	const n = TruncatedMessageIdLen
	for next := buff.Next(n); len(next) == n; next = buff.Next(n) {
		list = append(list, newTruncatedMessageID(next))
	}

	return
}

// saveMessage saves a Message to storage.
func (b *Buff) saveMessage(msg *Message) error {
	obj := &versioned.Object{
		Version:   messageVersion,
		Timestamp: netTime.Now(),
		Data:      msg.marshal(),
	}

	return b.kv.Set(
		makeMessageKey(msg.MessageId.truncate()), obj)

}

// loadMessage loads a message given truncatedMessageID from storage.
func loadMessage(tmID truncatedMessageID, kv versioned.KV) (*Message, error) {
	// Load message from storage
	vo, err := kv.Get(makeMessageKey(tmID), messageVersion)
	if err != nil {
		return nil, errors.Errorf(loadMessageErr, tmID, err)
	}

	// Unmarshal message
	return unmarshalMessage(vo.Data), nil
}

// makeMessageKey generates te key used to save a message to storage.
func makeMessageKey(tmID truncatedMessageID) string {
	return messageKey + tmID.String()
}
