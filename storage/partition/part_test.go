///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"math/rand"
	"testing"
	"time"
)

// Tests happy path of savePart().
func Test_savePart(t *testing.T) {
	// Set up test values
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	kv := versioned.NewKV(make(ekv.Memstore))
	partNum := uint8(prng.Uint32())
	part := make([]byte, prng.Int31n(500))
	prng.Read(part)
	key := makeMultiPartMessagePartKey(partNum)

	// Save part
	err := savePart(kv, partNum, part)
	if err != nil {
		t.Errorf("savePart() produced an error: %v", err)
	}

	// Attempt to get from key value store
	obj, err := kv.Get(key)
	if err != nil {
		t.Errorf("Get() produced an error: %v", err)
	}

	// Check if the data is correct
	if !bytes.Equal(part, obj.Data) {
		t.Errorf("Part retrieved from key value store is not expected."+
			"\n\texpected: %v\n\treceived: %v", part, obj.Data)
	}
}

// Tests happy path of loadPart().
func Test_loadPart(t *testing.T) {
	// Set up test values
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rootKv := versioned.NewKV(make(ekv.Memstore))
	partNum := uint8(prng.Uint32())
	part := make([]byte, prng.Int31n(500))
	prng.Read(part)
	key := makeMultiPartMessagePartKey(partNum)

	// Save part to key value store
	err := rootKv.Set(key, &versioned.Object{Timestamp: time.Now(), Data: part})
	if err != nil {
		t.Fatalf("Failed to set object: %v", err)
	}

	// Load part from key value store
	data, err := loadPart(rootKv, partNum)
	if err != nil {
		t.Errorf("loadPart() produced an error: %v", err)
	}

	// Check if the data is correct
	if !bytes.Equal(part, data) {
		t.Errorf("Part loaded from key value store is not expected."+
			"\n\texpected: %v\n\treceived: %v", part, data)
	}
}

// Tests that loadPart() returns an error that an item was not found for unsaved
// key.
func Test_loadPart_NotFoundError(t *testing.T) {
	// Set up test values
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	kv := versioned.NewKV(make(ekv.Memstore))
	partNum := uint8(prng.Uint32())
	part := make([]byte, prng.Int31n(500))
	prng.Read(part)

	// Load part from key value store
	data, err := loadPart(kv, partNum)
	if ekv.Exists(err) {
		t.Errorf("loadPart() found an item for the key: %v", err)
	}

	// Check if the data is correct
	if !bytes.Equal([]byte{}, data) {
		t.Errorf("Part loaded from key value store is not expected."+
			"\n\texpected: %v\n\treceived: %v", []byte{}, data)
	}
}

// Test happy path of deletePart().
func TestDeletePart(t *testing.T) {
	// Set up test values
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	kv := versioned.NewKV(make(ekv.Memstore))
	partNum := uint8(prng.Uint32())
	part := make([]byte, prng.Int31n(500))
	prng.Read(part)

	// Save part
	err := savePart(kv, partNum, part)
	if err != nil {
		t.Fatalf("savePart() produced an error: %v", err)
	}

	// Attempt to delete part
	err = deletePart(kv, partNum)
	if err != nil {
		t.Errorf("deletePart() produced an error: %v", err)
	}

	// Check if part was deleted
	_, err = loadPart(kv, partNum)
	if ekv.Exists(err) {
		t.Errorf("part was found in key value store: %v", err)
	}
}
