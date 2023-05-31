////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"encoding/base64"
	"encoding/json"
	"sync"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/netTime"
)

// MessageHash stores the hash of a message, which is used as the key for each
// message stored in the buffer.
type MessageHash [16]byte

func (m MessageHash) String() string {
	return base64.StdEncoding.EncodeToString(m[:])
}

// Sub key used in building keys for saving the message to the key value store
const messageSubKey = "bufferedMessage"

// Version of the file saved to the key value store
const CurrentMessageBufferVersion = 0

// MessageHandler interface used to handle the passed in message type so the
// buffer can be used at different layers of the stack.
type MessageHandler interface {
	// SaveMessage saves the message as a versioned object at the specified key
	// in the key value store.
	SaveMessage(kv versioned.KV, m interface{}, key string) error

	// LoadMessage returns the message with the specified key from the key value
	// store.
	LoadMessage(kv versioned.KV, key string) (interface{}, error)

	// DeleteMessage deletes the message with the specified key from the key
	// value store.
	DeleteMessage(kv versioned.KV, key string) error

	// HashMessage generates a hash of the message.
	HashMessage(m interface{}) MessageHash
}

// MessageBuffer holds a list of messages in the "not processed" or "processing"
// state both in memory. Messages in the "not processed" state are held in the
// messages map and messages in the "processing" state are moved into the
// processingMessages map. When the message is done being processed, it is
// removed from the buffer. The actual messages are saved in the key value store
// along with a copy of the buffer that is held in memory.
type MessageBuffer struct {
	messages           map[MessageHash]struct{}
	processingMessages map[MessageHash]struct{}
	kv                 versioned.KV
	handler            MessageHandler
	key                string
	mux                sync.RWMutex
}

// NewMessageBuffer creates a new empty buffer and saves it to the passed in key
// value store at the specified key. An error is returned on an unsuccessful
// save.
func NewMessageBuffer(kv versioned.KV, handler MessageHandler,
	key string) (*MessageBuffer, error) {
	// Create new empty buffer
	mb := &MessageBuffer{
		messages:           make(map[MessageHash]struct{}),
		processingMessages: make(map[MessageHash]struct{}),
		handler:            handler,
		kv:                 kv,
		key:                key,
	}

	// Save the buffer
	err := mb.save()

	// Return the new buffer or an error if saving failed
	return mb, err
}

// LoadMessageBuffer loads an existing message buffer from the key value store
// into memory at the given key. Returns an error if buffer cannot be loaded.
func LoadMessageBuffer(kv versioned.KV, handler MessageHandler,
	key string) (*MessageBuffer, error) {
	// Create new empty buffer
	mb := &MessageBuffer{
		messages:           make(map[MessageHash]struct{}),
		processingMessages: make(map[MessageHash]struct{}),
		handler:            handler,
		kv:                 kv,
		key:                key,
	}

	// Load rounds into buffer
	err := mb.load()

	// Return the filled buffer or an error if loading failed
	return mb, err
}

// GetMessages is a getter function which retrieves the
// MessageBuffer.messages map.
func (mb *MessageBuffer) GetMessages() map[MessageHash]struct{} {
	mb.mux.RLock()
	defer mb.mux.RUnlock()
	return mb.messages
}

// GetProcessingMessages is a getter function which retrieves the
// MessageBuffer.processingMessages map.
func (mb *MessageBuffer) GetProcessingMessages() map[MessageHash]struct{} {
	mb.mux.RLock()
	defer mb.mux.RUnlock()
	return mb.processingMessages
}

// save saves the buffer as a versioned object. All messages, regardless if they
// are in the "not processed" or "processing" state are stored together and
// considered "not processed".
func (mb *MessageBuffer) save() error {
	now := netTime.Now()

	// Build a combined list of message hashes in messages + processingMessages
	allMessages := mb.getMessageList()

	// Marshal list of message hashes into byte slice
	data, err := json.Marshal(allMessages)
	if err != nil {
		return err
	}

	// Create versioned object with data
	obj := versioned.Object{
		Version:   CurrentMessageBufferVersion,
		Timestamp: now,
		Data:      data,
	}

	// Save versioned object
	return mb.kv.Set(mb.key, &obj)
}

// getMessageList returns a list of all message hashes stored in messages and
// processingMessages in a random order.
func (mb *MessageBuffer) getMessageList() []MessageHash {
	// Create new slice with a length to fit all messages in either list
	msgs := make([]MessageHash, len(mb.messages)+len(mb.processingMessages))

	i := 0
	// Add messages from the "not processed" list
	for msg := range mb.messages {
		msgs[i] = msg
		i++
	}

	// AddFingerprint messages from the "processing" list
	for msg := range mb.processingMessages {
		msgs[i] = msg
		i++
	}

	return msgs
}

// load retrieves all the messages from the versioned object and stores them as
// unprocessed messages.
func (mb *MessageBuffer) load() error {

	// Load the versioned object
	vo, err := mb.kv.Get(mb.key, CurrentMessageBufferVersion)
	if err != nil {
		return err
	}

	// Create slice of message hashes from data
	var msgs []MessageHash
	err = json.Unmarshal(vo.Data, &msgs)
	if err != nil {
		return err
	}

	// Convert slice to map and save all rounds as unprocessed
	for _, m := range msgs {
		mb.messages[m] = struct{}{}
	}

	return nil
}

// Add adds a message to the buffer in "not processing" state.
func (mb *MessageBuffer) Add(m interface{}) interface{} {
	h := mb.handler.HashMessage(m)
	jww.TRACE.Printf("Critical Messages Add(%s)",
		base64.StdEncoding.EncodeToString(h[:]))

	mb.mux.Lock()
	defer mb.mux.Unlock()

	// Ensure message does not already exist in buffer
	if _, exists1 := mb.messages[h]; exists1 {
		msg, err := mb.handler.LoadMessage(mb.kv, MakeStoredMessageKey(mb.key, h))
		if err != nil {
			jww.FATAL.Panicf("Error loading message %s: %v", h, err)
		}
		return msg
	}
	if _, exists2 := mb.processingMessages[h]; exists2 {
		msg, err := mb.handler.LoadMessage(mb.kv, MakeStoredMessageKey(mb.key, h))
		if err != nil {
			jww.FATAL.Panicf("Error loading processing message %s: %v", h, err)
		}
		return msg
	}

	// Save message as versioned object
	err := mb.handler.SaveMessage(mb.kv, m, MakeStoredMessageKey(mb.key, h))
	if err != nil {
		jww.FATAL.Panicf("Error saving message: %v", err)
	}

	// AddFingerprint message to the buffer
	mb.messages[h] = struct{}{}

	// Save buffer
	err = mb.save()
	if err != nil {
		jww.FATAL.Panicf("Error while saving buffer: %v", err)
	}

	return m
}

// Add adds a message to the buffer in "processing" state.
func (mb *MessageBuffer) AddProcessing(m interface{}) interface{} {
	h := mb.handler.HashMessage(m)
	jww.TRACE.Printf("Critical Messages AddProcessing(%s)",
		base64.StdEncoding.EncodeToString(h[:]))

	mb.mux.Lock()
	defer mb.mux.Unlock()

	// Ensure message does not already exist in buffer
	if face1, exists1 := mb.messages[h]; exists1 {
		return face1
	}
	if face2, exists2 := mb.processingMessages[h]; exists2 {
		return face2
	}

	// Save message as versioned object
	err := mb.handler.SaveMessage(mb.kv, m, MakeStoredMessageKey(mb.key, h))
	if err != nil {
		jww.FATAL.Panicf("Error saving message: %v", err)
	}

	// AddFingerprint message to the buffer
	mb.processingMessages[h] = struct{}{}

	// Save buffer
	err = mb.save()
	if err != nil {
		jww.FATAL.Panicf("Error whilse saving buffer: %v", err)
	}

	return m
}

// Next gets the next message from the buffer whose state is "not processing".
// The returned messages are moved to the processing state. If there are no
// messages remaining, then false is returned.
func (mb *MessageBuffer) Next() (interface{}, bool) {
	mb.mux.Lock()
	defer mb.mux.Unlock()

	if len(mb.messages) == 0 {
		return format.Message{}, false
	}

	var m interface{}
	var err error

	//run until empty or a valid message is
	for m == nil && len(mb.messages) > 0 {
		// Pop the next MessageHash from the "not processing" list
		h := next(mb.messages)
		jww.TRACE.Printf("Critical Messages Next returned %s",
			base64.StdEncoding.EncodeToString(h[:]))

		delete(mb.messages, h)

		// AddFingerprint message to list of processing messages
		mb.processingMessages[h] = struct{}{}

		// Retrieve the message for storage
		m, err = mb.handler.LoadMessage(mb.kv, MakeStoredMessageKey(mb.key, h))
		if err != nil {
			m = nil
			jww.ERROR.Printf("Failed to load message %s from store, "+
				"this may happen on occasion due to replays to increase "+
				"reliability: %v", h, err)
		}

		if m != nil && h != mb.handler.HashMessage(m) {
			jww.WARN.Printf("MessageHash mismatch, possible"+
				" deserialization failure: %v != %v",
				mb.handler.HashMessage(m), h)
		}
	}

	return m, m != nil
}

// next returns the first MessageHash in the map returned by range.
func next(msgMap map[MessageHash]struct{}) MessageHash {
	for h := range msgMap {
		return h
	}
	return MessageHash{}
}

// Remove sets a messaged as processed and removed it from the buffer.
func (mb *MessageBuffer) Succeeded(m interface{}) {
	h := mb.handler.HashMessage(m)
	jww.TRACE.Printf("Critical Messages Succeeded(%s)",
		base64.StdEncoding.EncodeToString(h[:]))

	mb.mux.Lock()
	defer mb.mux.Unlock()

	// Done message from buffer
	delete(mb.processingMessages, h)
	delete(mb.messages, h)

	// Done message from key value store
	err := mb.handler.DeleteMessage(mb.kv, MakeStoredMessageKey(mb.key, h))
	if err != nil {
		jww.ERROR.Printf("Failed to delete message from store, "+
			"this may happen on occasion due to replays to increase "+
			"reliability: %v", err)
	}

	// Save modified buffer to key value store
	err = mb.save()
	if err != nil {
		jww.FATAL.Fatalf("Failed to save: %v", err)
	}
}

// Failed sets a message as failed to process. It changes the message back to
// the "not processed" state.
func (mb *MessageBuffer) Failed(m interface{}) {
	h := mb.handler.HashMessage(m)
	jww.TRACE.Printf("Critical Messages Failed(%s)",
		base64.StdEncoding.EncodeToString(h[:]))

	mb.mux.Lock()
	defer mb.mux.Unlock()

	// Done from "processing" state
	delete(mb.processingMessages, h)

	// Save message as versioned object
	err := mb.handler.SaveMessage(mb.kv, m, MakeStoredMessageKey(mb.key, h))
	if err != nil {
		jww.FATAL.Panicf("Error saving message: %v", err)
	}

	// AddFingerprint to "not processed" state
	mb.messages[h] = struct{}{}

	// Save buffer
	err = mb.save()
	if err != nil {
		jww.FATAL.Panicf("Error whilse saving buffer: %v", err)
	}
}

// MakeStoredMessageKey generates a new key for the message based on its has.
func MakeStoredMessageKey(key string, h MessageHash) string {
	return key + messageSubKey + base64.StdEncoding.EncodeToString(h[:])
}
