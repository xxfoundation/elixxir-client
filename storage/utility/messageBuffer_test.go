////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

type testHandler struct {
	messages map[string][]byte
}

func (th *testHandler) SaveMessage(kv *versioned.KV, m interface{}, key string) error {
	mBytes := m.([]byte)
	th.messages[key] = mBytes
	return nil
}

func (th *testHandler) LoadMessage(kv *versioned.KV, key string) (interface{}, error) {
	m, ok := th.messages[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return m, nil
}

func (th *testHandler) DeleteMessage(kv *versioned.KV, key string) error {
	_, ok := th.messages[key]
	if !ok {
		return os.ErrNotExist
	}
	delete(th.messages, key)
	return nil
}

func (th *testHandler) HashMessage(m interface{}) MessageHash {
	h, _ := blake2b.New256(nil)

	h.Write(m.([]byte))

	var messageHash MessageHash
	copy(messageHash[:], h.Sum(nil))

	return messageHash
}

func newTestHandler() *testHandler {
	return &testHandler{messages: make(map[string][]byte)}
}

// Tests happy path of NewMessageBuffer.
func TestNewMessageBuffer(t *testing.T) {
	// Set up expected value
	th := newTestHandler()
	expectedMB := &MessageBuffer{
		messages:           make(map[MessageHash]struct{}),
		processingMessages: make(map[MessageHash]struct{}),
		handler:            th,
		kv:                 versioned.NewKV(ekv.MakeMemstore()),
		key:                "testKey",
	}

	testMB, err := NewMessageBuffer(expectedMB.kv, th, expectedMB.key)
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
	th := newTestHandler()
	// Set up expected value
	expectedMB := &MessageBuffer{
		messages:           make(map[MessageHash]struct{}),
		processingMessages: make(map[MessageHash]struct{}),
		handler:            th,
		kv:                 versioned.NewKV(ekv.MakeMemstore()),
		key:                "testKey",
	}
	_ = addTestMessages(expectedMB, 20)
	err := expectedMB.save()
	if err != nil {
		t.Fatalf("Error saving MessageBuffer: %v", err)
	}

	testMB, err := LoadMessageBuffer(expectedMB.kv, th, expectedMB.key)

	// Move all the messages into one map to match the output
	for mh := range expectedMB.processingMessages {
		expectedMB.messages[mh] = struct{}{}
	}
	expectedMB.processingMessages = make(map[MessageHash]struct{})

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
	kv := versioned.NewKV(ekv.MakeMemstore())
	key := "testKey"

	mb, err := NewMessageBuffer(kv, newTestHandler(), key)
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}

	err = mb.save()
	if err != nil {
		t.Errorf("save() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}
	obj, err := kv.Get(key, 0)
	if err != nil {
		t.Errorf("save() did not correctly save buffer with key %+v to storage."+
			"\n\terror: %v", key, err)
	}

	var messageArr []MessageHash
	err = json.Unmarshal(obj.Data, &messageArr)
	if !reflect.DeepEqual([]MessageHash{}, messageArr) {
		t.Errorf("save() returned versioned object with incorrect data."+
			"\n\texpected: %#v\n\treceived: %#v",
			[]MessageHash{}, messageArr)
	}
}

// Tests happy path of save().
func TestMessageBuffer_save(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	key := "testKey"
	mb, err := NewMessageBuffer(kv, newTestHandler(), key)
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}

	expectedMH := addTestMessages(mb, 20)

	err = mb.save()
	if err != nil {
		t.Errorf("save() returned an error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}
	obj, err := kv.Get(key, 0)
	if err != nil {
		t.Errorf("save() did not correctly save buffer with key %+v to storage."+
			"\n\terror: %v", key, err)
	}

	var messageArr []MessageHash
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
	testMB, err := NewMessageBuffer(versioned.NewKV(ekv.MakeMemstore()), newTestHandler(), "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	testMsgs, expectedMessages := makeTestMessages(20)
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	if !reflect.DeepEqual(expectedMessages, testMB.messages) {
		t.Errorf("AddFingerprint() failed to add messages correctly into the buffer."+
			"\n\texpected: %v\n\trecieved: %v",
			expectedMessages, testMB.messages)
	}

	// Test adding duplicates
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	if !reflect.DeepEqual(expectedMessages, testMB.messages) {
		t.Errorf("AddFingerprint() failed to add messages correctly into the buffer."+
			"\n\texpected: %v\n\trecieved: %v",
			expectedMessages, testMB.messages)
	}
}

// Tests happy path of MessageBuffer.Next().
func TestMessageBuffer_Next(t *testing.T) {
	// Create new MessageBuffer and fill with messages
	testMB, err := NewMessageBuffer(versioned.NewKV(ekv.MakeMemstore()), newTestHandler(), "testKey")
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
			mBytes := m.([]byte)
			if bytes.Equal(testMsgs[i], mBytes) {
				foundMsg = true
				testMsgs[i] = testMsgs[len(testMsgs)-1]
				testMsgs[len(testMsgs)-1] = []byte{}
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

func TestMessageBuffer_InvalidNext(t *testing.T) {
	// Create new MessageBuffer and fill with messages
	testMB, err := NewMessageBuffer(versioned.NewKV(ekv.MakeMemstore()), newTestHandler(), "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	m := []byte("This is a message that should fail")
	h := testMB.handler.HashMessage(m)
	testMB.Add(m)
	err = testMB.handler.DeleteMessage(testMB.kv, MakeStoredMessageKey(testMB.key, h))
	if err != nil {
		t.Fatalf("Failed to set up test (delete from kv failed): %+v", err)
	}
	msg, exists := testMB.Next()
	if msg != nil || exists {
		t.Fatalf("This should fail with an invalid message, instead got: %+v, %+v", m, exists)
	}
}

// Tests happy path of MessageBuffer.Remove().
func TestMessageBuffer_Succeeded(t *testing.T) {
	th := newTestHandler()
	// Create new MessageBuffer and fill with message
	testMB, err := NewMessageBuffer(versioned.NewKV(ekv.MakeMemstore()), th, "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	testMsgs, _ := makeTestMessages(1)
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	// get message
	m, _ := testMB.Next()

	testMB.Succeeded(m)

	_, exists1 := testMB.messages[th.HashMessage(m)]
	_, exists2 := testMB.processingMessages[th.HashMessage(m)]
	if exists1 || exists2 {
		t.Errorf("Remove() did not remove the message from the buffer."+
			"\n\tbuffer: %+v", testMB)
	}
}

// Tests happy path of MessageBuffer.Failed().
func TestMessageBuffer_Failed(t *testing.T) {
	th := newTestHandler()
	// Create new MessageBuffer and fill with message
	testMB, err := NewMessageBuffer(versioned.NewKV(ekv.MakeMemstore()), th, "testKey")
	if err != nil {
		t.Fatalf("Failed to create new MessageBuffer: %v", err)
	}
	testMsgs, _ := makeTestMessages(1)
	for _, m := range testMsgs {
		testMB.Add(m)
	}

	// get message
	m, _ := testMB.Next()

	testMB.Failed(m)

	_, exists1 := testMB.messages[th.HashMessage(m)]
	_, exists2 := testMB.processingMessages[th.HashMessage(m)]
	if !exists1 || exists2 {
		t.Errorf("Failed() did not move the message back into the \"not "+
			"processed\" state.\n\tbuffer: %+v", testMB)
	}
}

// addTestMessages adds random messages to the buffer.
func addTestMessages(mb *MessageBuffer, n int) []MessageHash {
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	msgs := make([]MessageHash, n)
	for i := 0; i < n; i++ {
		keyData := make([]byte, 16)
		prng.Read(keyData)
		mh := MessageHash{}
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

// cmpMessageHash compares two slices of MessageHash to see if they have the
// exact same elements in any order.
func cmpMessageHash(arrA, arrB []MessageHash) bool {
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
func makeTestMessages(n int) ([][]byte, map[MessageHash]struct{}) {
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	mh := map[MessageHash]struct{}{}
	msgs := make([][]byte, n)
	for i := range msgs {
		msgs[i] = make([]byte, 256)
		prng.Read(msgs[i])

		h, _ := blake2b.New256(nil)
		h.Write(msgs[i])
		var messageHash MessageHash
		copy(messageHash[:], h.Sum(nil))

		mh[messageHash] = struct{}{}
	}

	return msgs, mh
}
