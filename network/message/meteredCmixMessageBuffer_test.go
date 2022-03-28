///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
	"time"
)

// Test happy path of meteredCmixMessage.SaveMessage().
func Test_meteredCmixMessageHandler_SaveMessage(t *testing.T) {
	// Set up test values
	mcmh := &meteredCmixMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, _ := makeTestMeteredCmixMessage(10)

	for _, msg := range testMsgs {
		key := utility.MakeStoredMessageKey("testKey", mcmh.HashMessage(msg))

		// Save message
		err := mcmh.SaveMessage(kv, msg, key)
		if err != nil {
			t.Errorf("SaveMessage() returned an error."+
				"\n\texpected: %v\n\trecieved: %v", nil, err)
		}

		// Try to get message
		obj, err := kv.Get(key, 0)
		if err != nil {
			t.Errorf("get() returned an error: %v", err)
		}

		msgData, err := json.Marshal(&msg)
		if err != nil {
			t.Fatalf("Could not marshal message: %v", err)
		}

		// Test if message retrieved matches expected
		if !bytes.Equal(msgData, obj.Data) {
			t.Errorf("SaveMessage() returned versioned object with incorrect data."+
				"\n\texpected: %v\n\treceived: %v",
				msg, obj.Data)
		}
	}
}

// Test happy path of meteredCmixMessage.LoadMessage().
func Test_meteredCmixMessageHandler_LoadMessage(t *testing.T) {
	// Set up test values
	mcmh := &meteredCmixMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, _ := makeTestMeteredCmixMessage(10)

	for i, msg := range testMsgs {
		key := utility.MakeStoredMessageKey("testKey", mcmh.HashMessage(msg))

		// Save message
		if err := mcmh.SaveMessage(kv, msg, key); err != nil {
			t.Errorf("SaveMessage() returned an error: %v", err)
		}

		// Try to load message
		testMsg, err := mcmh.LoadMessage(kv, key)
		if err != nil {
			t.Errorf("LoadMessage() returned an error."+
				"\n\texpected: %v\n\trecieved: %v", nil, err)
		}

		testMcm := testMsg.(meteredCmixMessage)

		// Test if message loaded matches expected
		if !bytes.Equal(msg.M, testMcm.M) || msg.Count != testMcm.Count || !msg.Timestamp.Equal(testMcm.Timestamp) {
			t.Errorf("LoadMessage() returned an unexpected object (round %d)."+
				"\n\texpected: %+v\n\treceived: %+v", i, msg, testMsg.(meteredCmixMessage))
		}
	}
}

// Test happy path of meteredCmixMessage.DeleteMessage().
func Test_meteredCmixMessageHandler_DeleteMessage(t *testing.T) {
	// Set up test values
	mcmh := &meteredCmixMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, _ := makeTestMeteredCmixMessage(10)

	for _, msg := range testMsgs {
		key := utility.MakeStoredMessageKey("testKey", mcmh.HashMessage(msg))

		// Save message
		err := mcmh.SaveMessage(kv, msg, key)
		if err != nil {
			t.Errorf("SaveMessage() returned an error."+
				"\n\texpected: %v\n\trecieved: %v", nil, err)
		}

		err = mcmh.DeleteMessage(kv, key)
		if err != nil {
			t.Errorf("DeleteMessage() produced an error: %v", err)
		}

		// Try to get message
		_, err = kv.Get(key, 0)
		if err == nil {
			t.Error("get() did not return an error.")
		}
	}
}

// Smoke test of meteredCmixMessageHandler.
func Test_meteredCmixMessageHandler_Smoke(t *testing.T) {
	// Set up test messages
	testMsgs := makeTestFormatMessages(2)

	// Create new buffer
	mcmb, err := NewMeteredCmixMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Errorf("NewMeteredCmixMessageBuffer() returned an error."+
			"\nexpected: %v\nrecieved: %v", nil, err)
	}

	// AddFingerprint two messages
	mcmb.Add(testMsgs[0],
		&pb.RoundInfo{ID: 1, Timestamps: []uint64{0, 1, 2, 3}},
		receptionID.EphemeralIdentity{Source: id.NewIdFromString("user1", id.User, t)})
	mcmb.Add(testMsgs[1],
		&pb.RoundInfo{ID: 2, Timestamps: []uint64{0, 1, 2, 3}},
		receptionID.EphemeralIdentity{Source: id.NewIdFromString("user2", id.User, t)})

	msg, ri, identity, exists := mcmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	mcmb.Remove(msg, ri, identity)

	msg, ri, identity, exists = mcmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	mcmb.Failed(msg, ri, identity)

	msg, ri, identity, exists = mcmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	mcmb.Remove(msg, ri, identity)

	msg, _, _, exists = mcmb.Next()
	if exists {
		t.Error("Next() found a message in the buffer when it should be empty.")
	}
}

// makeTestMeteredCmixMessage creates a list of messages with random data and the
// expected map after they are added to the buffer.
func makeTestMeteredCmixMessage(n int) ([]meteredCmixMessage, map[utility.MessageHash]struct{}) {
	mcmh := &meteredCmixMessageHandler{}
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	mh := map[utility.MessageHash]struct{}{}
	msgs := make([]meteredCmixMessage, n)
	for i := range msgs {
		payload := make([]byte, 128)
		prng.Read(payload)
		msgs[i] = meteredCmixMessage{
			M:         payload,
			Count:     uint(prng.Uint64()),
			Timestamp: time.Unix(0, 0),
		}
		mh[mcmh.HashMessage(msgs[i])] = struct{}{}
	}

	return msgs, mh
}

// makeTestFormatMessages creates a list of messages with random data.
func makeTestFormatMessages(n int) []format.Message {
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	msgs := make([]format.Message, n)
	for i := range msgs {
		msgs[i] = format.NewMessage(128)
		payload := make([]byte, 128)
		prng.Read(payload)
		msgs[i].SetPayloadA(payload)
		prng.Read(payload)
		msgs[i].SetPayloadB(payload)
	}

	return msgs
}
