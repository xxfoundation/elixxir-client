///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"testing"
)

// Tests happy path of New().
func TestNew(t *testing.T) {
	rootKv := versioned.NewKV(make(ekv.Memstore))
	expectedStore := &Store{
		multiParts:  make(map[multiPartID]*multiPartMessage),
		activeParts: make(map[*multiPartMessage]bool),
		kv:          rootKv.Prefix(packagePrefix),
	}

	store := NewOrLoad(rootKv)

	if !reflect.DeepEqual(expectedStore, store) {
		t.Errorf("New() did not return the expecte Store."+
			"\n\texpected: %v\n\treceived: %v", expectedStore, store)
	}
}

// Tests happy path of Store.AddFirst().
func TestStore_AddFirst(t *testing.T) {
	part := []byte("Test message.")
	s := NewOrLoad(versioned.NewKV(ekv.Memstore{}))

	msg, complete := s.AddFirst(id.NewIdFromString("User", id.User, t),
		message.XxMessage, 5, 0, 1, netTime.Now(), netTime.Now(), part,
		[]byte{0})

	if !complete {
		t.Errorf("AddFirst() returned that the message was not complete.")
	}

	if !bytes.Equal(part, msg.Payload) {
		t.Errorf("AddFirst() returned message with invalid payload."+
			"\n\texpected: %v\n\treceived: %v", part, msg.Payload)
	}
}

// Tests happy path of Store.Add().
func TestStore_Add(t *testing.T) {
	part1 := []byte("Test message.")
	part2 := []byte("Second Sentence.")
	s := New(versioned.NewKV(ekv.Memstore{}))

	msg, complete := s.AddFirst(id.NewIdFromString("User", id.User, t),
		message.XxMessage, 5, 0, 2, netTime.Now(), netTime.Now(), part1,
		[]byte{0})

	if complete {
		t.Errorf("AddFirst() returned that the message was complete.")
	}

	msg, complete = s.Add(id.NewIdFromString("User", id.User, t),
		5, 1, part2, []byte{0})
	if !complete {
		t.Errorf("AddFirst() returned that the message was not complete.")
	}

	part := append(part1, part2...)
	if !bytes.Equal(part, msg.Payload) {
		t.Errorf("AddFirst() returned message with invalid payload."+
			"\n\texpected: %v\n\treceived: %v", part, msg.Payload)
	}
}

// Unit test of prune
func TestStore_ClearMessages(t *testing.T) {
	// Setup: Add 2 message to store: an old message past the threshold and a new message
	part1 := []byte("Test message.")
	part2 := []byte("Second Sentence.")
	s := New(versioned.NewKV(ekv.Memstore{}))

	partner1 := id.NewIdFromString("User", id.User, t)
	messageId1 := uint64(5)
	oldTimestamp := netTime.Now().Add(-2 * clearPartitionThreshold)
	s.AddFirst(partner1,
		message.XxMessage, messageId1, 0, 2, netTime.Now(),
		oldTimestamp, part1,
		[]byte{0})
	s.Add(partner1, messageId1, 1, part2, []byte{0})

	partner2 := id.NewIdFromString("User1", id.User, t)
	messageId2 := uint64(6)
	newTimestamp := netTime.Now()
	s.AddFirst(partner2, message.XxMessage, messageId2, 0, 2, netTime.Now(),
		newTimestamp, part1,
		[]byte{0})

	// Call clear messages
	s.prune()

	// Check if old message cleared
	mpmId := getMultiPartID(partner1, messageId1)
	if _, ok := s.multiParts[mpmId]; ok {
		t.Errorf("Prune error: " +
			"Expected old message to be cleared out of store")
	}

	// Check if new message remains
	mpmId2 := getMultiPartID(partner2, messageId2)
	if _, ok := s.multiParts[mpmId2]; !ok {
		t.Errorf("Prune error: " +
			"Expected new message to be remain in store")
	}
}
