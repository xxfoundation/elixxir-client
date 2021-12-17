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
	"testing"
	"time"
)

// TestNewBuff tests the creation of a Buff object.
func TestNewBuff(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	if len(testBuff.buff) != buffLen {
		t.Fatalf("NewBuff did not produce buffer of "+
			"expected size. "+
			"\n\tExpected: %d"+
			"\n\tReceived slice size: %v",
			buffLen, len(testBuff.lookup))
	}

	_, err = kv.Prefix(ringBuffPrefix).Get(ringBuffKey, ringBuffVersion)
	if err != nil {
		t.Fatalf("Could not pull Buff from KV: %v", err)
	}

}

func TestBuff_Add(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	mid := NewMessageIdFromBytes([]byte("test"))
	err = testBuff.Add(mid, timestamp)
	if err != nil {
		t.Fatalf("Add error: %v", err)
	}

	if len(testBuff.lookup) != 1 {
		t.Fatalf("Message was not added to buffer's map")
	}

	received, exists := testBuff.lookup[mid.truncate()]
	if !exists {
		t.Fatalf("Message does not exist in buffer after add.")
	}

	expected := &Message{
		MessageId: mid,
		Timestamp: timestamp,
		id:        0,
	}

	if !reflect.DeepEqual(expected, received) {
		t.Fatalf("Expected Message not found in map."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expected, received)
	}

	f

}
