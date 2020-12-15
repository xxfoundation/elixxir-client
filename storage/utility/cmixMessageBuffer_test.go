///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Test happy path of cmixMessageHandler.SaveMessage().
func TestCmixMessageHandler_SaveMessage(t *testing.T) {
	// Set up test values
	cmh := &cmixMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, _ := makeTestCmixMessages(10)

	for _, msg := range testMsgs {
		key := makeStoredMessageKey("testKey", cmh.HashMessage(msg))

		// Save message
		err := cmh.SaveMessage(kv, msg, key)
		if err != nil {
			t.Errorf("SaveMessage() returned an error."+
				"\n\texpected: %v\n\trecieved: %v", nil, err)
		}

		// Try to get message
		obj, err := kv.Get(key)
		if err != nil {
			t.Errorf("Get() returned an error: %v", err)
		}

		// Test if message retrieved matches expected
		if !bytes.Equal(msg.Marshal(), obj.Data) {
			t.Errorf("SaveMessage() returned versioned object with incorrect data."+
				"\n\texpected: %v\n\treceived: %v",
				msg, obj.Data)
		}
	}
}

// Test happy path of cmixMessageHandler.LoadMessage().
func TestCmixMessageHandler_LoadMessage(t *testing.T) {
	// Set up test values
	cmh := &cmixMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, _ := makeTestCmixMessages(10)

	for _, msg := range testMsgs {
		key := makeStoredMessageKey("testKey", cmh.HashMessage(msg))

		// Save message
		if err := cmh.SaveMessage(kv, msg, key); err != nil {
			t.Errorf("SaveMessage() returned an error: %v", err)
		}

		// Try to load message
		testMsg, err := cmh.LoadMessage(kv, key)
		if err != nil {
			t.Errorf("LoadMessage() returned an error."+
				"\n\texpected: %v\n\trecieved: %v", nil, err)
		}

		// Test if message loaded matches expected
		if !reflect.DeepEqual(msg, testMsg) {
			t.Errorf("LoadMessage() returned an unexpected object."+
				"\n\texpected: %v\n\treceived: %v",
				msg, testMsg)
		}
	}
}

// Smoke test of cmixMessageHandler.
func TestCmixMessageBuffer_Smoke(t *testing.T) {
	// Set up test messages
	testMsgs, _ := makeTestCmixMessages(2)

	// Create new buffer
	cmb, err := NewCmixMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Errorf("NewCmixMessageBuffer() returned an error."+
			"\n\texpected: %v\n\trecieved: %v", nil, err)
	}

	// Add two messages
	cmb.Add(testMsgs[0])
	cmb.Add(testMsgs[1])

	if len(cmb.mb.messages) != 2 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			2, len(cmb.mb.messages))
	}

	msg, exists := cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	cmb.Succeeded(msg)

	if len(cmb.mb.messages) != 1 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			1, len(cmb.mb.messages))
	}

	msg, exists = cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	if len(cmb.mb.messages) != 0 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			0, len(cmb.mb.messages))
	}
	cmb.Failed(msg)

	if len(cmb.mb.messages) != 1 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			1, len(cmb.mb.messages))
	}

	msg, exists = cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	cmb.Succeeded(msg)

	msg, exists = cmb.Next()
	if exists {
		t.Error("Next() found a message in the buffer when it should be empty.")
	}
	cmb.Succeeded(msg)

	if len(cmb.mb.messages) != 0 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			0, len(cmb.mb.messages))
	}

}

// makeTestCmixMessages creates a list of messages with random data and the
// expected map after they are added to the buffer.
func makeTestCmixMessages(n int) ([]format.Message, map[MessageHash]struct{}) {
	cmh := &cmixMessageHandler{}
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mh := map[MessageHash]struct{}{}
	msgs := make([]format.Message, n)
	for i := range msgs {
		msgs[i] = format.NewMessage(128)
		payload := make([]byte, 128)
		prng.Read(payload)
		msgs[i].SetPayloadA(payload)
		prng.Read(payload)
		msgs[i].SetPayloadB(payload)
		mh[cmh.HashMessage(msgs[i])] = struct{}{}
	}

	return msgs, mh
}
