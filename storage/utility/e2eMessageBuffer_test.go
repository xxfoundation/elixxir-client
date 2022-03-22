///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

import (
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"testing"
)

// Test happy path of e2eMessageHandler.SaveMessage().
func TestE2EMessageHandler_SaveMessage(t *testing.T) {
	// Set up test values
	emg := &e2eMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, _ := makeTestE2EMessages(10, t)

	for _, msg := range testMsgs {
		key := MakeStoredMessageKey("testKey", emg.HashMessage(msg))

		// Save message
		err := emg.SaveMessage(kv, msg, key)
		if err != nil {
			t.Errorf("SaveMessage() returned an error."+
				"\n\texpected: %v\n\trecieved: %v", nil, err)
		}

		// Try to get message
		obj, err := kv.Get(key, 0)
		if err != nil {
			t.Errorf("get() returned an error: %v", err)
		}

		// Test if message retrieved matches expected
		testMsg := &e2eMessage{}
		if err := json.Unmarshal(obj.Data, testMsg); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}
		if !reflect.DeepEqual(msg, *testMsg) {
			t.Errorf("SaveMessage() returned versioned object with incorrect data."+
				"\n\texpected: %v\n\treceived: %v",
				msg, *testMsg)
		}
	}
}

// Test happy path of e2eMessageHandler.LoadMessage().
func TestE2EMessageHandler_LoadMessage(t *testing.T) {
	// Set up test values
	cmh := &e2eMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs, _ := makeTestE2EMessages(10, t)

	for _, msg := range testMsgs {
		key := MakeStoredMessageKey("testKey", cmh.HashMessage(msg))

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

// Smoke test of e2eMessageHandler.
func TestE2EMessageHandler_Smoke(t *testing.T) {
	// Set up test messages
	_, testMsgs := makeTestE2EMessages(2, t)

	// Create new buffer
	cmb, err := NewE2eMessageBuffer(versioned.NewKV(make(ekv.Memstore)), "testKey")
	if err != nil {
		t.Errorf("NewE2eMessageBuffer() returned an error."+
			"\n\texpected: %v\n\trecieved: %v", nil, err)
	}

	// Add two messages
	cmb.Add(testMsgs[0], params.E2E{})
	cmb.Add(testMsgs[1], params.E2E{})

	if len(cmb.mb.messages) != 2 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			2, len(cmb.mb.messages))
	}

	msg, _, exists := cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	cmb.Succeeded(msg, params.E2E{})

	if len(cmb.mb.messages) != 1 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			1, len(cmb.mb.messages))
	}

	msg, _, exists = cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	if len(cmb.mb.messages) != 0 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			0, len(cmb.mb.messages))
	}
	cmb.Failed(msg, params.E2E{})

	if len(cmb.mb.messages) != 1 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			1, len(cmb.mb.messages))
	}

	msg, _, exists = cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	cmb.Succeeded(msg, params.E2E{})

	msg, _, exists = cmb.Next()
	if exists {
		t.Error("Next() found a message in the buffer when it should be empty.")
	}

	if len(cmb.mb.messages) != 0 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			0, len(cmb.mb.messages))
	}

}

// makeTestE2EMessages creates a list of messages with random data and the
// expected map after they are added to the buffer.
func makeTestE2EMessages(n int, t *testing.T) ([]e2eMessage, []message.Send) {
	prng := rand.New(rand.NewSource(netTime.Now().UnixNano()))
	msgs := make([]e2eMessage, n)
	send := make([]message.Send, n)
	for i := range msgs {
		rngBytes := make([]byte, 128)
		prng.Read(rngBytes)
		msgs[i].Recipient = rngBytes
		prng.Read(rngBytes)
		msgs[i].Payload = rngBytes
		prng.Read(rngBytes)
		msgs[i].MessageType = uint32(rngBytes[0])

		send[i].Recipient = id.NewIdFromString(string(msgs[i].Recipient), id.User, t)
		send[i].Payload = msgs[i].Payload
		send[i].MessageType = message.Type(msgs[i].MessageType)
	}

	return msgs, send
}

func TestE2EParamMarshalUnmarshal(t *testing.T) {
	msg := &e2eMessage{
		Recipient:   id.DummyUser[:],
		Payload:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
		MessageType: 42,
		Params: params.E2E{
			Type:       1,
			RetryCount: 7,
			CMIX: params.CMIX{
				RoundTries: 6,
				Timeout:    99,
				RetryDelay: -4,
			},
		},
	}

	fmt.Printf("msg1: %#v\n", msg)

	b, err := json.Marshal(&msg)

	if err != nil {
		t.Errorf("Failed to Marshal E2eMessage")
	}

	fmt.Printf("json: %s\n", string(b))

	msg2 := &e2eMessage{}

	err = json.Unmarshal(b, &msg2)

	if err != nil {
		t.Errorf("Failed to Unmarshal E2eMessage")
	}

	fmt.Printf("msg2: %#v\n", msg2)

	if !reflect.DeepEqual(msg, msg2) {
		t.Errorf("Unmarshaled message is not the same")
	}

}
