////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// TestNewBuff tests the creation of a Buff object.
func TestNewBuff(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(ekv.MakeMemstore())
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	// Check buffer was initialized to expected length
	if len(testBuff.buff) != buffLen {
		t.Errorf("New Buff has incorrect length.\nexpected: %d\nreceived: %d",
			buffLen, len(testBuff.lookup))
	}

	// Check that buffer exists in KV
	kv, err = kv.Prefix(ringBuffPrefix)
	require.NoError(t, err)
	_, err = kv.Get(ringBuffKey, ringBuffVersion)
	if err != nil {
		t.Errorf("Failed to load Buff from KV: %+v", err)
	}
}

// Tests that Buff.Add properly adds to the Buff object. This includes modifying
// the Buff.buff, buff.lookup, and proper index updates.
func TestBuff_Add(t *testing.T) {
	// Initialize buffer
	testBuff, err := NewBuff(versioned.NewKV(ekv.MakeMemstore()), 20)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	mid := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(mid, timestamp)
	if err != nil {
		t.Errorf("Add returned an error: %+v", err)
	}

	// Check that map entries exist
	if len(testBuff.lookup) != 1 {
		t.Errorf("Incorrect length: message was not added to buffer's map."+
			"\nexpected: %d\nreceived: %d", 1, len(testBuff.lookup))
	}

	// Check that expected entry exists in the map
	received, exists := testBuff.lookup[mid.truncate()]
	if !exists {
		t.Error("Message does not exist in buffer after add.")
	}

	// Reconstruct added message
	expected := &Message{
		MessageId: mid,
		Timestamp: timestamp,
		id:        0,
	}

	// Check map for inserted Message
	if !reflect.DeepEqual(expected, received) {
		t.Errorf("Expected message not found in map."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}

	// Check buffer for inserted Message
	if !reflect.DeepEqual(testBuff.buff[0], expected) {
		t.Errorf("Expected message not found in buffer."+
			"\nexpected: %+v\nreceived: %+v", expected, testBuff.buff[0])
	}

	// Check that newest index was updated
	if testBuff.newest != 0 {
		t.Errorf("Buffer's newest index was not updated to expected value."+
			"\nexpected: %d\nreceived: %d", 0, testBuff.newest)
	}
}

// Inserts buffer length + 1 Message's to the buffer and ensures the oldest
// value is overwritten.
func TestBuff_Add_Overflow(t *testing.T) {
	buffLen := 20
	testBuff, err := NewBuff(versioned.NewKV(ekv.MakeMemstore()), buffLen)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	// Insert message to be overwritten
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	oldest := NewMessageIdFromBytes([]byte("will be overwritten"))
	err = testBuff.Add(oldest, timestamp)
	if err != nil {
		t.Errorf("Failed to add message to buffer: %+v", err)
	}

	// Insert buffLen elements to overwrite element inserted above
	for i := 0; i < buffLen; i++ {
		timestamp = time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		mid := NewMessageIdFromBytes([]byte(strconv.Itoa(i)))
		err = testBuff.Add(mid, timestamp)
		if err != nil {
			t.Errorf("Failed to add message to buffer: %+v", err)
		}

		if testBuff.newest != uint32(i+1) {
			t.Errorf("Buffer's newest index was not updated for insert."+
				"\nexpected: %d\nreceived: %d", i+1, testBuff.newest)
		}
	}

	// Test that the oldest index has been updated
	if testBuff.oldest != 1 {
		t.Errorf("Buffer's oldest index was not updated to expected value."+
			"\nexpected: %d\nreceived: %d", 1, testBuff.oldest)
	}

	// Check that oldest value no longer exists in map
	_, exists := testBuff.lookup[oldest.truncate()]
	if exists {
		t.Errorf("Oldest value expected to be overwritten in map!")
	}
}

// Tests that Buff.Get returns the latest inserted Message.
func TestBuff_Get(t *testing.T) {
	// Initialize buffer
	testBuff, err := NewBuff(versioned.NewKV(ekv.MakeMemstore()), 20)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	mid := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(mid, timestamp)
	if err != nil {
		t.Errorf("Failed to add message to buffer: %+v", err)
	}

	// Reconstruct expected message
	expected := &Message{
		MessageId: mid,
		Timestamp: timestamp,
		id:        0,
	}

	// Retrieve newly inserted value using get
	received := testBuff.Get()

	// Check that retrieved value is expected
	if !reflect.DeepEqual(received, expected) {
		t.Errorf("Get did not retrieve expected value."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}

	// Construct new message to insert
	newlyInsertedMid := NewMessageIdFromBytes([]byte("test2"))
	newlyInserted := &Message{
		MessageId: newlyInsertedMid,
		Timestamp: timestamp,
		id:        1,
	}

	// Add new message to buffer
	err = testBuff.Add(newlyInsertedMid, timestamp)
	if err != nil {
		t.Errorf("Failed to add message to buffer: %+v", err)
	}

	// Ensure newly inserted message is returned by get
	if !reflect.DeepEqual(testBuff.Get(), newlyInserted) {
		t.Errorf("Get did not retrieve expected value."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that Buff.GetByMessageID returns the Message with the requested
// MessageID.
func TestBuff_GetByMessageID(t *testing.T) {
	// Initialize buffer
	testBuff, err := NewBuff(versioned.NewKV(ekv.MakeMemstore()), 20)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	mid := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(mid, timestamp)
	if err != nil {
		t.Errorf("Failed to add message to buffer: %+v", err)
	}

	// Reconstruct expected message
	expected := &Message{
		MessageId: mid,
		Timestamp: timestamp,
		id:        0,
	}

	// Retrieve message using getter
	received, err := testBuff.GetByMessageID(mid)
	if err != nil {
		t.Errorf("GetByMessageID error: %+v", err)
	}

	// Check retrieved value matches expected
	if !reflect.DeepEqual(received, expected) {
		t.Errorf("GetByMessageID retrieved unexpected value."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}

}

// Tests that Buff.GetByMessageID returns an error when requesting a MessageID
// that does not exist in Buff.
func TestBuff_GetByMessageID_Error(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(ekv.MakeMemstore())
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	unInsertedMid := NewMessageIdFromBytes([]byte("test"))

	// Un-inserted MessageID should not exist in Buff, causing an error
	_, err = testBuff.GetByMessageID(unInsertedMid)
	if err == nil {
		t.Errorf("GetByMessageID should error when requesting a MessageID " +
			"not in the buffer.")
	}

}

func TestBuff_GetNextMessage(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(ekv.MakeMemstore())
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	oldMsgId := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(oldMsgId, timestamp)
	if err != nil {
		t.Errorf("Failed to add message to buffer: %+v", err)
	}

	// Insert next message
	nextMsgId := NewMessageIdFromBytes([]byte("test2"))
	err = testBuff.Add(nextMsgId, timestamp)
	if err != nil {
		t.Errorf("Failed to add message to buffer: %+v", err)
	}

	// Construct expected message (the newest message)
	expected := &Message{
		MessageId: nextMsgId,
		Timestamp: timestamp,
		id:        1,
	}

	// Retrieve message after the old message
	received, err := testBuff.GetNextMessage(oldMsgId)
	if err != nil {
		t.Errorf("GetNextMessage returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("GetNextMessage did not retrieve expected value."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}

}

func TestLoadBuff(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(ekv.MakeMemstore())
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Errorf("Failed to make new Buff: %+v", err)
	}

	// Insert buffLen elements to overwrite element inserted above
	for i := 0; i < buffLen; i++ {
		timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		mid := NewMessageIdFromBytes([]byte(strconv.Itoa(i)))
		err = testBuff.Add(mid, timestamp)
		if err != nil {
			t.Errorf("Failed to add message to buffer: %+v", err)
		}
	}

	// Load buffer from storage
	received, err := LoadBuff(kv)
	if err != nil {
		t.Errorf("LoadBuff returned an error: %+v", err)
	}

	if reflect.DeepEqual(testBuff, received) {
		t.Errorf("Loaded buffer does not match stored."+
			"\nexpected: %+v\nreceived: %+v", testBuff, received)
	}
}
