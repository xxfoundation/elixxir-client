///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"bytes"
	"gitlab.com/elixxir/ekv"
	"testing"
	"time"
)

// KV Get should call the Upgrade function when it's available
func TestVersionedKV_Get_Err(t *testing.T) {
	kv := make(ekv.Memstore)
	vkv := NewKV(kv)
	key := MakeKeyWithPrefix("test", "12345")
	result, err := vkv.Get(key)
	if err == nil {
		t.Error("Getting a key that didn't exist should have" +
			" returned an error")
	}
	if result != nil {
		t.Error("Getting a key that didn't exist shouldn't " +
			"have returned data")
	}
}

// Test versioned KV Upgrade path
func TestVersionedKV_Get_Upgrade(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := make(ekv.Memstore)
	vkv := NewKV(kv)
	key := MakeKeyWithPrefix("test", "12345")
	original := Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      []byte("not upgraded"),
	}
	originalSerialized := original.Marshal()
	kv[key] = originalSerialized

	result, err := vkv.Get(key)
	if err != nil {
		t.Fatalf("Error getting something that should have been in: %v",
			err)
	}
	if !bytes.Equal(result.Data,
		[]byte("this object was upgraded from v0 to v1")) {
		t.Errorf("Upgrade should have overwritten data."+
			" result data: %q", result.Data)
	}
}

// Test Get without Upgrade path
func TestVersionedKV_Get(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := make(ekv.Memstore)
	vkv := NewKV(kv)
	originalVersion := uint64(1)
	key := MakeKeyWithPrefix("test", "12345")
	original := Object{
		Version:   originalVersion,
		Timestamp: time.Now(),
		Data:      []byte("not upgraded"),
	}
	originalSerialized := original.Marshal()
	kv[key] = originalSerialized

	result, err := vkv.Get(key)
	if err != nil {
		t.Fatalf("Error getting something that should have been in: %v",
			err)
	}
	if !bytes.Equal(result.Data, []byte("not upgraded")) {
		t.Errorf("Upgrade should not have overwritten data."+
			" result data: %q", result.Data)
	}
}

// Test that Set puts data in the store
func TestVersionedKV_Set(t *testing.T) {
	kv := make(ekv.Memstore)
	vkv := NewKV(kv)
	originalVersion := uint64(1)
	key := MakeKeyWithPrefix("test", "12345")
	original := Object{
		Version:   originalVersion,
		Timestamp: time.Now(),
		Data:      []byte("not upgraded"),
	}
	err := vkv.Set(key, &original)
	if err != nil {
		t.Fatal(err)
	}

	// Store should now have data in it at that key
	_, ok := kv[key]
	if !ok {
		t.Error("data store didn't have anything in the key")
	}
}
