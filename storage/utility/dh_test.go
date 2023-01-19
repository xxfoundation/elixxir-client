////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"testing"
)

// Unit test for StoreCyclicKey
func TestStoreCyclicKey(t *testing.T) {
	kv := ekv.MakeMemstore()
	vkv := versioned.NewKV(kv)
	grp := getTestGroup()
	x := grp.NewInt(77)

	err := StoreCyclicKey(vkv, x, "testKey")
	if err != nil {
		t.Error("Failed to store cyclic key")
	}
}

// Unit test for LoadCyclicKey
func TestLoadCyclicKey(t *testing.T) {
	kv := ekv.MakeMemstore()
	vkv := versioned.NewKV(kv)
	grp := getTestGroup()
	x := grp.NewInt(77)

	intKey := "testKey"
	err := StoreCyclicKey(vkv, x, intKey)
	if err != nil {
		t.Errorf("Failed to store cyclic key: %+v", err)
	}

	loaded, err := LoadCyclicKey(vkv, intKey)
	if err != nil {
		t.Errorf("Failed to load cyclic key: %+v", err)
	}
	if loaded.Cmp(x) != 0 {
		t.Errorf("Stored int did not match received.  Stored: %v, Received: %v", x, loaded)
	}
}

// Unit test for DeleteCyclicKey
func TestDeleteCyclicKey(t *testing.T) {
	kv := ekv.MakeMemstore()
	vkv := versioned.NewKV(kv)
	grp := getTestGroup()
	x := grp.NewInt(77)

	intKey := "testKey"
	err := StoreCyclicKey(vkv, x, intKey)
	if err != nil {
		t.Errorf("Failed to store cyclic key: %+v", err)
	}

	err = DeleteCyclicKey(vkv, intKey)
	if err != nil {
		t.Fatalf("DeleteCyclicKey returned an error: %v", err)
	}

	_, err = LoadCyclicKey(vkv, intKey)
	if err == nil {
		t.Errorf("DeleteCyclicKey error: Should not load deleted key: %+v", err)
	}
}
