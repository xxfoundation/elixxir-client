package utility

import (
	"crypto/md5"
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/format"
	"sync"
	"time"
)

// messageHash stores the key for each message stored in the buffer.
type messageHash [16]byte

// Sub key used in building keys for saving the message to the key value store
const messageSubKey = "bufferedMessage"

// Version of the file saved to the key value store
const currentMessageBufferVersion = 0

// MessageBuffer holds a list of messages in the "not processed" or "processing"
// state both in memory. Messages in the "not processed" state are held in the
// messages map and messages in the "processing" state are moved into the
// processingMessages map. When the message is done being processed, it is
// removed from the buffer. The actual messages are saved in the key value store
// along with a copy of the buffer that is held in memory.
type MessageBuffer struct {
	messages           map[messageHash]struct{}
	processingMessages map[messageHash]struct{}
	kv                 *versioned.KV
	key                string
	mux                sync.RWMutex
}

// NewMessageBuffer creates a new empty buffer and saves it to the passed in key
// value store at the specified key. An error is returned on an unsuccessful
// save.
func NewMessageBuffer(kv *versioned.KV, key string) (*MessageBuffer, error) {
	// Create new empty buffer
	mb := &MessageBuffer{
		messages:           make(map[messageHash]struct{}),
		processingMessages: make(map[messageHash]struct{}),
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
func LoadMessageBuffer(kv *versioned.KV, key string) (*MessageBuffer, error) {
	// Create new empty buffer
	mb := &MessageBuffer{
		messages:           make(map[messageHash]struct{}),
		processingMessages: make(map[messageHash]struct{}),
		kv:                 kv,
		key:                key,
	}

	// Load rounds into buffer
	err := mb.load()

	// Return the filled buffer or an error if loading failed
	return mb, err
}

// save saves the buffer as a versioned object. All messages, regardless if they
// are in the "not processed" or "processing" state are stored together and
// considered "not processed".
func (mb *MessageBuffer) save() error {
	now := time.Now()

	// Build a combined list of message hashes in messages + processingMessages
	allMessages := mb.getMessageList()

	// Marshal list of message hashes into byte slice
	data, err := json.Marshal(allMessages)
	if err != nil {
		return err
	}

	// Create versioned object with data
	obj := versioned.Object{
		Version:   currentMessageBufferVersion,
		Timestamp: now,
		Data:      data,
	}

	// Save versioned object
	return mb.kv.Set(mb.key, &obj)
}

// getMessageList returns a list of all message hashes stored in messages and
// processingMessages in a random order.
func (mb *MessageBuffer) getMessageList() []messageHash {
	// Create new slice with a length to fit all messages in either list
	msgs := make([]messageHash, len(mb.messages)+len(mb.processingMessages))

	i := 0
	// Add messages from the "not processed" list
	for msg := range mb.messages {
		msgs[i] = msg
		i++
	}

	// Add messages from the "processing" list
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
	vo, err := mb.kv.Get(mb.key)
	if err != nil {
		return err
	}

	// Create slice of message hashes from data
	var msgs []messageHash
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
func (mb *MessageBuffer) Add(m format.Message) {
	h := hashMessage(m)

	mb.mux.Lock()
	defer mb.mux.Unlock()

	// Ensure message does not already exist in buffer
	_, exists1 := mb.messages[h]
	_, exists2 := mb.processingMessages[h]
	if exists1 || exists2 {
		return
	}

	// Save message as versioned object
	err := saveMessage(mb.kv, m, makeStoredMessageKey(mb.key, h))
	if err != nil {
		jww.FATAL.Panicf("Error saving message: %v", err)
	}

	// Add message to the buffer
	mb.messages[h] = struct{}{}

	// Save buffer
	err = mb.save()
	if err != nil {
		jww.FATAL.Panicf("Error whilse saving buffer: %v", err)
	}
}

// Next gets the next message from the buffer whose state is "not processing".
// The returned messages are moved to the processing state. If there are no
// messages remaining, then false is returned.
func (mb *MessageBuffer) Next() (format.Message, bool) {
	mb.mux.Lock()
	defer mb.mux.Unlock()

	if len(mb.messages) == 0 {
		return format.Message{}, false
	}

	// Pop the next messageHash from the "not processing" list
	h := next(mb.messages)
	delete(mb.messages, h)

	// Add message to list of processing messages
	mb.processingMessages[h] = struct{}{}

	// Retrieve the message for storage
	m, err := loadMessage(mb.kv, makeStoredMessageKey(mb.key, h))
	if err != nil {
		jww.FATAL.Panicf("Could not load message: %v", err)
	}
	return m, true
}

// next returns the first messageHash in the map returned by range.
func next(msgMap map[messageHash]struct{}) messageHash {
	for h := range msgMap {
		return h
	}
	return messageHash{}
}

// Succeeded sets a messaged as processed and removed it from the buffer.
func (mb *MessageBuffer) Succeeded(m format.Message) {
	h := hashMessage(m)

	mb.mux.Lock()
	defer mb.mux.Unlock()

	delete(mb.processingMessages, h)
	err := mb.save()
	if err != nil {
		jww.FATAL.Fatalf("Failed to save: %v", err)
	}
}

// Failed sets a message as failed to process. It changes the message back to
// the "not processed" state.
func (mb *MessageBuffer) Failed(m format.Message) {
	h := hashMessage(m)

	mb.mux.Lock()
	defer mb.mux.Unlock()

	// Remove from "processing" state
	delete(mb.processingMessages, h)

	// Add to "not processed" state
	mb.messages[h] = struct{}{}
}

// saveMessage saves the message as a versioned object.
func saveMessage(kv *versioned.KV, m format.Message, key string) error {
	now := time.Now()

	// Create versioned object
	obj := versioned.Object{
		Version:   currentMessageBufferVersion,
		Timestamp: now,
		Data:      m.Marshal(),
	}

	// Save versioned object
	return kv.Set(key, &obj)
}

// loadMessage loads the message with the specified key.
func loadMessage(kv *versioned.KV, key string) (format.Message, error) {
	// Load the versioned object
	vo, err := kv.Get(key)
	if err != nil {
		return format.Message{}, err
	}

	// Create message from data
	return format.Unmarshal(vo.Data), err
}

// hashMessage generates a hash of the message.
func hashMessage(m format.Message) messageHash {
	// Sum returns a array that is the exact same size as the messageHash and Go
	// apparently automatically casts it
	return md5.Sum(m.Marshal())
}

// makeStoredMessageKey generates a new key for the message based on its has.
func makeStoredMessageKey(key string, h messageHash) string {
	return key + messageSubKey + string(h[:])
}
