////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"bytes"
	"gitlab.com/elixxir/ekv"
	"reflect"
	"testing"
	"time"
)

// Shows that all fields can be serialized/deserialized correctly using json
func TestVersionedObject_MarshalUnmarshal(t *testing.T) {
	original := VersionedObject{
		Version:   8,
		Timestamp: time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC),
		Data:      []byte("original text"),
	}

	marshalled := original.Marshal()

	unmarshalled := VersionedObject{}
	err := unmarshalled.Unmarshal(marshalled)
	if err != nil {
		// Should never happen
		t.Fatal(err)
	}

	if !reflect.DeepEqual(original, unmarshalled) {
		t.Error("Original and deserialized objects not equal")
	}
	t.Logf("%+v", unmarshalled)
}

// VersionedKV Get should call the upgrade function when it's available
func TestVersionedKV_Get_Err(t *testing.T) {
	kv := make(ekv.Memstore)
	vkv := NewVersionedKV(kv)
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

// Test versioned KV upgrade path
func TestVersionedKV_Get_Upgrade(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := make(ekv.Memstore)
	vkv := NewVersionedKV(kv)
	key := MakeKeyWithPrefix("test", "12345")
	original := VersionedObject{
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
		t.Errorf("upgrade should have overwritten data."+
			" result data: %q", result.Data)
	}
}

// Test Get without upgrade path
func TestVersionedKV_Get(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := make(ekv.Memstore)
	vkv := NewVersionedKV(kv)
	originalVersion := uint64(1)
	key := MakeKeyWithPrefix("test", "12345")
	original := VersionedObject{
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
		t.Errorf("upgrade should not have overwritten data."+
			" result data: %q", result.Data)
	}
}

// Test that Set puts data in the store
func TestVersionedKV_Set(t *testing.T) {
	kv := make(ekv.Memstore)
	vkv := NewVersionedKV(kv)
	originalVersion := uint64(1)
	key := MakeKeyWithPrefix("test", "12345")
	original := VersionedObject{
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
