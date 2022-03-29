///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"testing"
)

// Test happy path of cmixMessageHandler.SaveMessage.
func Test_cmixMessageHandler_SaveMessage(t *testing.T) {
	// Set up test values
	cmh := &cmixMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, ids, _ := makeTestCmixMessages(10)

	for i := range testMsgs {
		msg := storedMessage{
			Msg:       testMsgs[i].Marshal(),
			Recipient: ids[i].Marshal(),
		}

		key := utility.MakeStoredMessageKey("testKey", cmh.HashMessage(msg))

		// Save message
		err := cmh.SaveMessage(kv, msg, key)
		if err != nil {
			t.Errorf("SaveMessage returned an error: %+v", err)
		}

		// Try to get message
		obj, err := kv.Get(key, 0)
		if err != nil {
			t.Errorf("Failed to get message: %v", err)
		}

		// Test if message retrieved matches expected
		if !bytes.Equal(msg.Marshal(), obj.Data) {
			t.Errorf("Failed to get expected message from storage."+
				"\nexpected: %v\nreceived: %v", msg, obj.Data)
		}
	}
}

// Test happy path of cmixMessageHandler.LoadMessage.
func Test_cmixMessageHandler_LoadMessage(t *testing.T) {
	// Set up test values
	cmh := &cmixMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, ids, _ := makeTestCmixMessages(10)

	for i := range testMsgs {
		msg := storedMessage{
			Msg:       testMsgs[i].Marshal(),
			Recipient: ids[i].Marshal(),
		}
		key := utility.MakeStoredMessageKey("testKey", cmh.HashMessage(msg))

		// Save message
		if err := cmh.SaveMessage(kv, msg, key); err != nil {
			t.Errorf("Failed to save message to storage: %v", err)
		}

		// Try to load message
		testMsg, err := cmh.LoadMessage(kv, key)
		if err != nil {
			t.Errorf("LoadMessage returned an error: %v", err)
		}

		// Test if message loaded matches expected
		if !reflect.DeepEqual(msg, testMsg) {
			t.Errorf("Failed to load expected message from storage."+
				"\nexpected: %v\nreceived: %v", msg, testMsg)
		}
	}
}

// Smoke test of cmixMessageHandler.
func Test_cmixMessageBuffer_Smoke(t *testing.T) {
	// Set up test messages
	testMsgs, ids, _ := makeTestCmixMessages(2)

	// Create new buffer
	cmb, err := NewOrLoadCmixMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Errorf("Failed to make new cmixMessageHandler: %+v", err)
	}

	// Add two messages
	cmb.Add(testMsgs[0], ids[0], GetDefaultCMIXParams())
	cmb.Add(testMsgs[1], ids[1], GetDefaultCMIXParams())

	msg, rid, _, exists := cmb.Next()
	if !exists {
		t.Error("Next did not find any messages in buffer.")
	}
	cmb.Succeeded(msg, rid)

	msg, rid, _, exists = cmb.Next()
	if !exists {
		t.Error("Next did not find any messages in buffer.")
	}

	cmb.Failed(msg, rid)

	msg, rid, _, exists = cmb.Next()
	if !exists {
		t.Error("Next did not find any messages in buffer.")
	}
	cmb.Succeeded(msg, rid)

	msg, rid, _, exists = cmb.Next()
	if exists {
		t.Error("Next found a message in the buffer when it should be empty.")
	}

}

// makeTestCmixMessages creates a list of messages with random data and the
// expected map after they are added to the buffer.
func makeTestCmixMessages(n int) (
	[]format.Message, []*id.ID, map[utility.MessageHash]struct{}) {
	cmh := &cmixMessageHandler{}
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	mh := map[utility.MessageHash]struct{}{}
	msgs := make([]format.Message, n)
	ids := make([]*id.ID, n)
	for i := range msgs {
		msgs[i] = format.NewMessage(128)
		payload := make([]byte, 128)
		prng.Read(payload)
		msgs[i].SetPayloadA(payload)
		prng.Read(payload)
		msgs[i].SetPayloadB(payload)

		rid := id.ID{}
		prng.Read(rid[:32])
		rid[32] = byte(id.User)
		ids[i] = &rid
		sm := storedMessage{
			Msg:       msgs[i].Marshal(),
			Recipient: ids[i].Marshal(),
		}
		mh[cmh.HashMessage(sm)] = struct{}{}
	}

	return msgs, ids, mh
}
