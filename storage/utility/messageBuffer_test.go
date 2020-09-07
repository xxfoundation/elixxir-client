package utility

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests happy path of NewMessageBuffer.
func TestNewMessageBuffer(t *testing.T) {
	// Set up expected value
	expectedMB := &MessageBuffer{
		messages:           make(map[messageHash]struct{}),
		processingMessages: make(map[messageHash]struct{}),
		kv:                 versioned.NewKV(make(ekv.Memstore)),
		key:                "testKey",
	}

	testMB, err := NewMessageBuffer(expectedMB.kv, expectedMB.key)
	if err != nil {
		t.Errorf("NewMessageBuffer() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}

	if !reflect.DeepEqual(expectedMB, testMB) {
		t.Errorf("NewMessageBuffer() returned an incorrect MessageBuffer."+
			"\n\texpected: %v\n\treceived: %v", expectedMB, testMB)
	}
}

// Tests happy path of TestLoadMessageBuffer.
func TestLoadMessageBuffer(t *testing.T) {
	// Set up expected value
	expectedMB := &MessageBuffer{
		messages:           make(map[messageHash]struct{}),
		processingMessages: make(map[messageHash]struct{}),
		kv:                 versioned.NewKV(make(ekv.Memstore)),
		key:                "testKey",
	}
	_ = addTestMessages(expectedMB, 20)
	err := expectedMB.save()
	if err != nil {
		t.Fatalf("Error saving MessageBuffer: %v", err)
	}

	testMB, err := LoadMessageBuffer(expectedMB.kv, expectedMB.key)

	// Move all the messages into one map to match the output
	for mh := range expectedMB.processingMessages {
		expectedMB.messages[mh] = struct{}{}
	}
	expectedMB.processingMessages = make(map[messageHash]struct{})

	if err != nil {
		t.Errorf("LoadMessageBuffer() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}

	if !reflect.DeepEqual(expectedMB, testMB) {
		t.Errorf("NewMessageBuffer() returned an incorrect MessageBuffer."+
			"\n\texpected: %+v\n\treceived: %+v", expectedMB, testMB)
	}
}

// Tests happy path of save() with a new empty MessageBuffer.
func TestMessageBuffer_save_NewMB(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	key := "testKey"
	mb, err := NewMessageBuffer(kv, key)
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}

	err = mb.save()
	if err != nil {
		t.Errorf("save() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}
	obj, err := kv.Get(key)
	if err != nil {
		t.Errorf("save() did not correctly save buffer with key %+v to storage."+
			"\n\terror: %v", key, err)
	}

	var messageArr []messageHash
	err = json.Unmarshal(obj.Data, &messageArr)
	if !reflect.DeepEqual([]messageHash{}, messageArr) {
		t.Errorf("save() returned versioned object with incorrect data."+
			"\n\texpected: %#v\n\treceived: %#v",
			[]messageHash{}, messageArr)
	}
}

// Tests happy path of save().
func TestMessageBuffer_save(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	key := "testKey"
	mb, err := NewMessageBuffer(kv, key)
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}

	expectedMH := addTestMessages(mb, 20)

	err = mb.save()
	if err != nil {
		t.Errorf("save() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}
	obj, err := kv.Get(key)
	if err != nil {
		t.Errorf("save() did not correctly save buffer with key %+v to storage."+
			"\n\terror: %v", key, err)
	}

	var messageArr []messageHash
	err = json.Unmarshal(obj.Data, &messageArr)
	if !cmpMessageHash(expectedMH, messageArr) {
		t.Errorf("save() returned versioned object with incorrect data."+
			"\n\texpected: %v\n\treceived: %v",
			expectedMH, messageArr)
	}
}

// Tests happy path of MessageBuffer.Add().
func TestMessageBuffer_Add(t *testing.T) {
	// Create new MessageBuffer and fill with messages
	testMB, err := NewMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	testMsgs, expectedMessages := makeTestMessages(20)
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	if !reflect.DeepEqual(expectedMessages, testMB.messages) {
		t.Errorf("Add() failed to add messages correctly into the buffer."+
			"\n\texpected: %v\n\trecieved: %v",
			expectedMessages, testMB.messages)
	}

	// Test adding duplicates
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	if !reflect.DeepEqual(expectedMessages, testMB.messages) {
		t.Errorf("Add() failed to add messages correctly into the buffer."+
			"\n\texpected: %v\n\trecieved: %v",
			expectedMessages, testMB.messages)
	}
}

// Tests happy path of MessageBuffer.Next().
func TestMessageBuffer_Next(t *testing.T) {
	// Create new MessageBuffer and fill with messages
	testMB, err := NewMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	testMsgs, _ := makeTestMessages(20)
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	for m, exists := testMB.Next(); exists; m, exists = testMB.Next() {
		foundMsg := false
		for i := range testMsgs {
			if bytes.Equal(testMsgs[i].Marshal(), m.Marshal()) {
				foundMsg = true
				testMsgs[i] = testMsgs[len(testMsgs)-1]
				testMsgs[len(testMsgs)-1] = format.Message{}
				testMsgs = testMsgs[:len(testMsgs)-1]
				break
			}
		}
		if !foundMsg {
			t.Errorf("Next() returned the wrong message."+
				"\n\trecieved: %+v", m)
		}
	}
}

func Test_saveMessage(t *testing.T) {
	// Set up test values
	kv := versioned.NewKV(make(ekv.Memstore))
	subKey := "testKey"
	testMsgs, _ := makeTestMessages(1)
	mh := hashMessage(testMsgs[0])
	key := makeStoredMessageKey(subKey, mh)

	// Save message
	err := saveMessage(kv, testMsgs[0], key)
	if err != nil {
		t.Errorf("saveMessage() returned an error."+
			"\n\texpected: %v\n\trecieved: %v", nil, err)
	}

	// Try to get message
	obj, err := kv.Get(key)
	if err != nil {
		t.Errorf("Get() returned an error."+
			"\n\texpected: %v\n\trecieved: %v", nil, err)
	}

	if !bytes.Equal(testMsgs[0].Marshal(), obj.Data) {
		t.Errorf("saveMessage() returned versioned object with incorrect data."+
			"\n\texpected: %v\n\treceived: %v",
			testMsgs[0], obj.Data)
	}
}

// Tests happy path of MessageBuffer.Succeeded().
func TestMessageBuffer_Succeeded(t *testing.T) {
	// Create new MessageBuffer and fill with message
	testMB, err := NewMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	testMsgs, _ := makeTestMessages(1)
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	// Get message
	m, _ := testMB.Next()

	testMB.Succeeded(m)

	_, exists1 := testMB.messages[hashMessage(m)]
	_, exists2 := testMB.processingMessages[hashMessage(m)]
	if exists1 || exists2 {
		t.Errorf("Succeeded() did not remove the message from the buffer."+
			"\n\tbuffer: %+v", testMB)
	}
}

// Tests happy path of MessageBuffer.Failed().
func TestMessageBuffer_Failed(t *testing.T) {
	// Create new MessageBuffer and fill with message
	testMB, err := NewMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	testMsgs, _ := makeTestMessages(1)
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	// Get message
	m, _ := testMB.Next()

	testMB.Failed(m)

	_, exists1 := testMB.messages[hashMessage(m)]
	_, exists2 := testMB.processingMessages[hashMessage(m)]
	if !exists1 || exists2 {
		t.Errorf("Failed() did not move the message back into the \"not "+
			"processed\" state.\n\tbuffer: %+v", testMB)
	}
}

// addTestMessages adds random messages to the buffer.
func addTestMessages(mb *MessageBuffer, n int) []messageHash {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	msgs := make([]messageHash, n)
	for i := 0; i < n; i++ {
		keyData := make([]byte, 16)
		prng.Read(keyData)
		mh := messageHash{}
		copy(mh[:], keyData)

		if i%10 == 0 {
			mb.processingMessages[mh] = struct{}{}
		} else {
			mb.messages[mh] = struct{}{}
		}
		msgs[i] = mh

	}
	return msgs
}

// cmpMessageHash compares two slices of messageHash to see if they have the
// exact same elements in any order.
func cmpMessageHash(arrA, arrB []messageHash) bool {
	if len(arrA) != len(arrB) {
		return false
	}
	for _, a := range arrA {
		foundInB := false
		for _, b := range arrB {
			if a == b {
				foundInB = true
				break
			}
		}
		if !foundInB {
			return false
		}
	}
	return true
}

// makeTestMessages creates a list of messages with random data and the expected
// map after they are added to the buffer.
func makeTestMessages(n int) ([]format.Message, map[messageHash]struct{}) {
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mh := map[messageHash]struct{}{}
	msgs := make([]format.Message, n)
	for i := range msgs {
		msgs[i] = format.NewMessage(128)
		payload := make([]byte, 128)
		prng.Read(payload)
		msgs[i].SetPayloadA(payload)
		prng.Read(payload)
		msgs[i].SetPayloadB(payload)
		mh[hashMessage(msgs[i])] = struct{}{}
	}

	return msgs, mh
}
