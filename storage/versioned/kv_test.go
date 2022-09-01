////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"bytes"
	"errors"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"testing"
)

// KV get should call the Upgrade function when it's available
func TestVersionedKV_Get_Err(t *testing.T) {
	kv := ekv.MakeMemstore()
	vkv := NewKV(kv)
	key := vkv.GetFullKey("test", 0)
	result, err := vkv.Get(key, 0)
	if err == nil {
		t.Error("Getting a key that didn't exist should have" +
			" returned an error")
	}
	if result != nil {
		t.Error("Getting a key that didn't exist shouldn't " +
			"have returned data")
	}
}

// Test versioned KV happy path
func TestVersionedKV_GetUpgrade(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := ekv.MakeMemstore()
	vkv := NewKV(kv)
	key := vkv.GetFullKey("test", 0)
	original := Object{
		Version:   0,
		Timestamp: netTime.Now(),
		Data:      []byte("not upgraded"),
	}
	err := kv.Set(key, &original)
	if err != nil {
		t.Errorf("Failed to set original: %+v", err)
	}

	upgrade := []Upgrade{func(oldObject *Object) (*Object, error) {
		return &Object{
			Version:   1,
			Timestamp: netTime.Now(),
			Data:      []byte("this object was upgraded from v0 to v1"),
		}, nil
	}}

	result, err := vkv.GetAndUpgrade("test", UpgradeTable{CurrentVersion: 1,
		Table: upgrade})
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

// Test versioned KV key not found path
func TestVersionedKV_GetUpgrade_KeyNotFound(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := ekv.MakeMemstore()
	vkv := NewKV(kv)
	key := "test"

	upgrade := []Upgrade{func(oldObject *Object) (*Object, error) {
		return &Object{
			Version:   1,
			Timestamp: netTime.Now(),
			Data:      []byte("this object was upgraded from v0 to v1"),
		}, nil
	}}

	_, err := vkv.GetAndUpgrade(key, UpgradeTable{CurrentVersion: 1,
		Table: upgrade})
	if err == nil {
		t.Fatalf("Error getting something that shouldn't be there!")
	}
}

// Test versioned KV upgrade func returns error path
func TestVersionedKV_GetUpgrade_UpgradeReturnsError(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := ekv.MakeMemstore()
	vkv := NewKV(kv)
	key := vkv.GetFullKey("test", 0)
	original := Object{
		Version:   0,
		Timestamp: netTime.Now(),
		Data:      []byte("not upgraded"),
	}
	err := kv.Set(key, &original)
	if err != nil {
		t.Errorf("Failed to set original: %+v", err)
	}

	upgrade := []Upgrade{func(oldObject *Object) (*Object, error) {
		return &Object{}, errors.New("test error")
	}}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	_, _ = vkv.GetAndUpgrade("test", UpgradeTable{CurrentVersion: 1,
		Table: upgrade})
}

// Test delete key happy path
func TestVersionedKV_Delete(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := ekv.MakeMemstore()
	vkv := NewKV(kv)
	key := vkv.GetFullKey("test", 0)
	original := Object{
		Version:   0,
		Timestamp: netTime.Now(),
		Data:      []byte("not upgraded"),
	}
	err := kv.Set(key, &original)
	if err != nil {
		t.Errorf("Failed to set original: %+v", err)
	}

	err = vkv.Delete("test", 0)
	if err != nil {
		t.Fatalf("Error getting something that should have been in: %v",
			err)
	}

	o := &Object{}
	err = kv.Get(key, o)
	if err == nil {
		t.Fatal("Key still exists in kv map")
	}
}

// Test get without Upgrade path
func TestVersionedKV_Get(t *testing.T) {
	// Set up a dummy KV with the required data
	kv := ekv.MakeMemstore()
	vkv := NewKV(kv)
	originalVersion := uint64(0)
	key := vkv.GetFullKey("test", originalVersion)
	original := Object{
		Version:   originalVersion,
		Timestamp: netTime.Now(),
		Data:      []byte("not upgraded"),
	}
	err := kv.Set(key, &original)
	if err != nil {
		t.Errorf("Failed to set original in kv: %+v", err)
	}

	result, err := vkv.Get("test", originalVersion)
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
	kv := ekv.MakeMemstore()
	vkv := NewKV(kv)
	originalVersion := uint64(1)
	key := vkv.GetFullKey("test", originalVersion)
	original := Object{
		Version:   originalVersion,
		Timestamp: netTime.Now(),
		Data:      []byte("not upgraded"),
	}
	err := vkv.Set("test", originalVersion, &original)
	if err != nil {
		t.Fatal(err)
	}

	// Store should now have data in it at that key
	o := &Object{}
	err = kv.Get(key, o)
	if err != nil {
		t.Errorf("data store didn't have anything in the key: %+v", err)
	}
}
