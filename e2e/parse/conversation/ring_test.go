///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// TestNewBuff tests the creation of a Buff object.
func TestNewBuff(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	/// Check buffer was initialized to expected length
	if len(testBuff.buff) != buffLen {
		t.Fatalf("NewBuff did not produce buffer of "+
			"expected size. "+
			"\n\tExpected: %d"+
			"\n\tReceived slice size: %v",
			buffLen, len(testBuff.lookup))
	}

	// Check that buffer exists in KV
	_, err = kv.Prefix(ringBuffPrefix).Get(ringBuffKey, ringBuffVersion)
	if err != nil {
		t.Fatalf("Could not pull Buff from KV: %v", err)
	}

}

// TestBuff_Add tests whether Buff.Add properly adds to the Buff object.
// This includes modifying the Buff.buff, buff.lookup and proper index updates.
func TestBuff_Add(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	mid := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(mid, timestamp)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	// Check that map entries exist
	if len(testBuff.lookup) != 1 {
		t.Fatalf("Message was not added to buffer's map")
	}

	// Check that expected entry exists in the map
	received, exists := testBuff.lookup[mid.truncate()]
	if !exists {
		t.Fatalf("Message does not exist in buffer after add.")
	}

	// Reconstruct added message
	expected := &Message{
		MessageId: mid,
		Timestamp: timestamp,
		id:        0,
	}

	// Check map for inserted Message
	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("Expected Message not found in map."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
	}

	// Check buffer for inserted Message
	if !reflect.DeepEqual(testBuff.buff[0], expected) {
		t.Fatalf("Expected message not found in buffer."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, testBuff.buff[0])
	}

	// Check that newest index was updated
	if testBuff.newest != 0 {
		t.Fatalf("Buffer's newest index was not updated to expected value."+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", 0, testBuff.newest)
	}
}

// TestBuff_Add_Overflow inserts buffer length + 1 Message's to the buffer
// and ensures the oldest value is overwritten.
func TestBuff_Add_Overflow(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	// Insert message to be overwritten
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	oldest := NewMessageIdFromBytes([]byte("will be overwritten"))
	err = testBuff.Add(oldest, timestamp)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	// Insert buffLen elements to overwrite element inserted above
	for i := 0; i < buffLen; i++ {
		timestamp = time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		mid := NewMessageIdFromBytes([]byte(strconv.Itoa(i)))
		err = testBuff.Add(mid, timestamp)
		if err != nil {
			t.Fatalf("Add error: %v", err)
		}

		if testBuff.newest != uint32(i+1) {
			t.Fatalf("Buffer's newest index was not updated for insert."+
				"\n\tExpected: %d"+
				"\n\tReceived: %d", i+1, testBuff.newest)
		}
	}

	// Test that the oldest index has been updated
	if testBuff.oldest != 1 {
		t.Fatalf("Buffer's oldest index was not updated to expected value."+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", 1, testBuff.oldest)
	}

	// Check that oldest value no longer exists in map
	_, exists := testBuff.lookup[oldest.truncate()]
	if exists {
		t.Fatalf("Oldest value expected to be overwritten in map!")
	}

}

// TestBuff_Get tests that Buff.Get returns the latest inserted Message.
func TestBuff_Get(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	mid := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(mid, timestamp)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	// Reconstruct expected message
	expected := &Message{
		MessageId: mid,
		Timestamp: timestamp,
		id:        0,
	}

	// Retrieve newly inserted value using get()
	received := testBuff.Get()

	// Check that retrieved value is expected
	if !reflect.DeepEqual(received, expected) {
		t.Fatalf("get() did not retrieve expected value."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
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
		t.Fatalf("Add error: %v", err)
	}

	// Ensure newly inserted message is returned by get()
	if !reflect.DeepEqual(testBuff.Get(), newlyInserted) {
		t.Fatalf("get() did not retrieve expected value."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
	}

}

// TestBuff_GetByMessageId tests that Buff.GetByMessageId returns the Message with
// the requested MessageId.
func TestBuff_GetByMessageId(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	mid := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(mid, timestamp)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	// Reconstruct expected message
	expected := &Message{
		MessageId: mid,
		Timestamp: timestamp,
		id:        0,
	}

	// Retrieve message using getter
	received, err := testBuff.GetByMessageId(mid)
	if err != nil {
		t.Fatalf("GetMessageId error: %v", err)
	}

	// Check retrieved value matches expected
	if !reflect.DeepEqual(received, expected) {
		t.Fatalf("GetByMessageId retrieved unexpected value."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
	}

}

// TestBuff_GetByMessageId_Error tests that Buff.GetByMessageId returns an error
// when requesting a MessageId that does not exist in Buff.
func TestBuff_GetByMessageId_Error(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	uninsertedMid := NewMessageIdFromBytes([]byte("test"))

	// Un-inserted MessageId should not exist in Buff, causing an error
	_, err = testBuff.GetByMessageId(uninsertedMid)
	if err == nil {
		t.Fatalf("GetByMessageId should error when requesting a " +
			"MessageId not in the buffer")
	}

}

// TestBuff_GetNextMessage tests whether
func TestBuff_GetNextMessage(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	// Insert initial message
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	oldMsgId := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(oldMsgId, timestamp)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	// Insert next message
	nextMsgId := NewMessageIdFromBytes([]byte("test2"))
	err = testBuff.Add(nextMsgId, timestamp)
	if err != nil {
		t.Fatalf("Add error: %v", err)
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
		t.Fatalf("GetNextMessage error: %v", err)
	}

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("GetNextMessage did not retrieve expected value."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
	}

}

// TestBuff_marshalUnmarshal tests that the Buff's marshal and unmarshalBuffer functionality
// are inverse methods.
func TestLoadBuff(t *testing.T) {
	// Initialize buffer
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	// Insert buffLen elements to overwrite element inserted above
	for i := 0; i < buffLen; i++ {
		timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		mid := NewMessageIdFromBytes([]byte(strconv.Itoa(i)))
		err = testBuff.Add(mid, timestamp)
		if err != nil {
			t.Fatalf("Add error: %v", err)
		}
	}

	// Load buffer from storage
	received, err := LoadBuff(kv)
	if err != nil {
		t.Fatalf("LoadBuff error: %v", err)
	}

	if reflect.DeepEqual(testBuff, received) {
		t.Fatalf("Loaded buffer does not match stored.")
	}

}
