package utility

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"testing"
)

// Unit test for StoreCyclicKey
func TestStoreCyclicKey(t *testing.T) {
	kv := make(ekv.Memstore)
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
	kv := make(ekv.Memstore)
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
