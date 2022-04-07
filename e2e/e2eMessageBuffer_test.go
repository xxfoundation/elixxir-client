///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Test happy path of e2eMessageHandler.SaveMessage().
func TestE2EMessageHandler_SaveMessage(t *testing.T) {
	// Set up test values
	emg := &e2eMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs := makeTestE2EMessages(10, t)

	for _, msg := range testMsgs {
		key := utility.MakeStoredMessageKey("testKey", emg.HashMessage(msg))

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
		testMsg := e2eMessage{}
		if err := json.Unmarshal(obj.Data, &testMsg); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
		}

		if !e2eMessagesEqual(msg, testMsg, t) {
			t.Errorf("SaveMessage() returned versioned object with incorrect data."+
				"\n\texpected: %v\n\treceived: %v",
				msg, testMsg)
		}
	}
}

// Test happy path of e2eMessageHandler.LoadMessage().
func TestE2EMessageHandler_LoadMessage(t *testing.T) {
	// Set up test values
	cmh := &e2eMessageHandler{}
	kv := versioned.NewKV(make(ekv.Memstore))
	testMsgs := makeTestE2EMessages(10, t)

	for _, msg := range testMsgs {
		key := utility.MakeStoredMessageKey("testKey", cmh.HashMessage(msg))

		// Save message
		if err := cmh.SaveMessage(kv, msg, key); err != nil {
			t.Errorf("SaveMessage() returned an error: %v", err)
		}

		// Try to load message
		face, err := cmh.LoadMessage(kv, key)
		if err != nil {
			t.Errorf("LoadMessage() returned an error."+
				"\n\texpected: %v\n\trecieved: %v", nil, err)
		}

		testMsg, ok := face.(e2eMessage)
		if !ok {
			t.Fatalf("Unexpected message type from LoadMessage")
		}

		// Test if message loaded matches expected
		if !e2eMessagesEqual(msg, testMsg, t) {
			t.Errorf("LoadMessage() returned an unexpected object."+
				"\n\texpected: %v\n\treceived: %v",
				msg, testMsg)
		}
	}
}

// Smoke test of e2eMessageHandler.
func TestE2EMessageHandler_Smoke(t *testing.T) {
	// Set up test messages
	testMsgs := makeTestE2EMessages(2, t)
	kv := versioned.NewKV(make(ekv.Memstore))
	// Create new buffer
	cmb, err := NewOrLoadE2eMessageBuffer(kv, "testKey")
	if err != nil {
		t.Errorf("NewE2eMessageBuffer() returned an error."+
			"\n\texpected: %v\n\trecieved: %v", nil, err)
	}

	// Parse message 0
	msg0 := testMsgs[0]
	recipient0, err := id.Unmarshal(msg0.Recipient)
	if err != nil {
		t.Fatalf("bad data in test message: %v", err)
	}

	// Parse message 1
	msg1 := testMsgs[1]
	recipient1, err := id.Unmarshal(msg1.Recipient)
	if err != nil {
		t.Fatalf("bad data in test message: %v", err)
	}
	// Add two messages
	cmb.Add(catalog.MessageType(msg0.MessageType), recipient0,
		msg0.Payload, msg0.Params)
	cmb.Add(catalog.MessageType(msg1.MessageType), recipient1,
		msg1.Payload, msg1.Params)

	if len(cmb.mb.GetMessages()) != 2 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			2, len(cmb.mb.GetMessages()))
	}

	msgType, recipient, payload, _, exists := cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	cmb.Succeeded(msgType, recipient, payload)

	if len(cmb.mb.GetMessages()) != 1 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			1, cmb.mb.GetMessages())
	}

	msgType, recipient, payload, _, exists = cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	if len(cmb.mb.GetMessages()) != 0 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			0, len(cmb.mb.GetMessages()))
	}
	cmb.Failed(msgType, recipient, payload)

	if len(cmb.mb.GetMessages()) != 1 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			1, len(cmb.mb.GetMessages()))
	}

	msgType, recipient, payload, _, exists = cmb.Next()
	if !exists {
		t.Error("Next() did not find any messages in buffer.")
	}
	cmb.Succeeded(msgType, recipient, payload)

	msgType, recipient, payload, _, exists = cmb.Next()
	if exists {
		t.Error("Next() found a message in the buffer when it should be empty.")
	}

	if len(cmb.mb.GetMessages()) != 0 {
		t.Errorf("Unexpected length of buffer.\n\texpected: %d\n\trecieved: %d",
			0, len(cmb.mb.GetMessages()))
	}

}

func TestE2EParamMarshalUnmarshal(t *testing.T) {
	msg := &e2eMessage{
		Recipient:   id.DummyUser[:],
		Payload:     []byte{1, 2, 3, 4, 5, 6, 7, 8, 9},
		MessageType: 42,
		Params: Params{
			CMIX: cmix.CMIXParams{
				RoundTries:       6,
				Timeout:          99,
				RetryDelay:       -4,
				BlacklistedNodes: map[id.ID]bool{},
			},
		},
	}

	t.Logf("msg1: %#v\n", msg)

	b, err := json.Marshal(&msg)

	if err != nil {
		t.Errorf("Failed to Marshal E2eMessage")
	}

	t.Logf("json: %s\n", string(b))

	msg2 := &e2eMessage{}

	err = json.Unmarshal(b, &msg2)

	if err != nil {
		t.Errorf("Failed to Unmarshal E2eMessage")
	}

	t.Logf("msg2: %#v\n", msg2)

	if !reflect.DeepEqual(msg, msg2) {
		t.Errorf("Unmarshaled message is not the same")
	}

}
