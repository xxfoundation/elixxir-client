////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"testing"
)

// Tests that storeKey saves the key to storage by loading it and comparing it
// to the original.
func Test_storeKey(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedKey := []byte("MyTestKey")

	// Save the key
	err := storeKey(expectedKey, kv)
	if err != nil {
		t.Errorf("storeKey returned an error: %+v", err)
	}

	// Attempt to load the key
	obj, err := kv.Get(userKeyStorageKey, userKeyVersion)
	if err != nil {
		t.Errorf("Failed to get key from storage: %+v", err)
	}

	// Check that the key matches the original
	if !bytes.Equal(expectedKey, obj.Data) {
		t.Errorf("Loaded key does not match original."+
			"\nexpected: %q\nreceived: %q", expectedKey, obj.Data)
	}
}

// Tests that loadKey restores the original key saved to stage and compares it
// to the original.
func Test_loadKey(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedKey := []byte("MyTestKey")

	// Save the key
	err := kv.Set(userKeyStorageKey, userKeyVersion, &versioned.Object{
		Version:   userKeyVersion,
		Timestamp: netTime.Now(),
		Data:      expectedKey,
	})
	if err != nil {
		t.Errorf("Failed to save key to storage: %+v", err)
	}

	// Attempt to load the key
	loadedKey, err := loadKey(kv)
	if err != nil {
		t.Errorf("loadKey returned an error: %+v", err)
	}

	// Check that the key matches the original
	if !bytes.Equal(expectedKey, loadedKey) {
		t.Errorf("Loaded key does not match original."+
			"\nexpected: %q\nreceived: %q", expectedKey, loadedKey)
	}
}

// Tests that deleteKey deletes the key from storage by trying to recover a
// deleted key.
func Test_deleteKey(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedKey := []byte("MyTestKey")

	// Save the key
	err := storeKey(expectedKey, kv)
	if err != nil {
		t.Errorf("Failed to save key to storage: %+v", err)
	}

	// Delete the key
	err = deleteKey(kv)
	if err != nil {
		t.Errorf("deleteKey returned an error: %+v", err)
	}

	// Attempt to load the key
	obj, err := loadKey(kv)
	if err == nil || obj != nil {
		t.Errorf("Loaded object from storage when it should be deleted: %+v", obj)
	}
}
