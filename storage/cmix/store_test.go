////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"testing"
)

var testStore *Store

// Main testing function
func TestMain(m *testing.M) {

	kv := make(ekv.Memstore)
	vkv := versioned.NewKV(kv)

	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	priv := grp.NewInt(2)

	testStore, _ = NewStore(grp, vkv, priv)

	runFunc := func() int {
		return m.Run()
	}

	os.Exit(runFunc())
}

// Happy path Add/Remove test
func TestStore_AddRemove(t *testing.T) {
	nodeId := id.NewIdFromString("test", id.Node, t)
	key := testStore.grp.NewInt(5)

	err := testStore.Add(nodeId, key)
	if err != nil {
		t.Errorf("Unable to add node key: %+v", err)
		return
	}
	if _, exists := testStore.nodes[*nodeId]; !exists {
		t.Errorf("Failed to add node key")
		return
	}

	err = testStore.Remove(nodeId)
	if err != nil {
		t.Errorf("Unable to remove node key: %+v", err)
		return
	}
	if _, exists := testStore.nodes[*nodeId]; exists {
		t.Errorf("Failed to remove node key")
		return
	}
}

// Missing keys path
func TestStore_GetRoundKeys_Missing(t *testing.T) {
	var err error

	// Set up the circuit
	numIds := 10
	nodeIds := make([]*id.ID, numIds)
	for i := 0; i < numIds; i++ {
		nodeIds[i] = id.NewIdFromUInt(uint64(i)+1, id.Node, t)
		key := testStore.grp.NewInt(int64(i) + 1)

		// Only add every other node so there are missing nodes
		if i%2 == 0 {
			err = testStore.Add(nodeIds[i], key)
			if err != nil {
				t.Errorf("Unable to add node key: %+v", err)
			}
		}
	}

	circuit := connect.NewCircuit(nodeIds)
	result, missing := testStore.GetRoundKeys(circuit)
	if len(missing) != numIds/2 {
		t.Errorf("Expected to have %d missing keys, got %d", numIds/2, len(missing))
	}
	if result != nil {
		t.Errorf("Expected nil value for result due to missing keys!")
	}
}

// Happy path
func TestNewStore(t *testing.T) {
	kv := make(ekv.Memstore)
	vkv := versioned.NewKV(kv)

	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	priv := grp.NewInt(2)
	pub := diffieHellman.GeneratePublicKey(priv, grp)

	store, err := NewStore(grp, vkv, priv)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if store.nodes == nil {
		t.Errorf("Failed to initialize nodes")
	}
	if store.dhPrivateKey == nil || store.dhPrivateKey.Cmp(priv) != 0 {
		t.Errorf("Failed to set store.dhPrivateKey correctly")
	}
	if store.dhPublicKey == nil || store.dhPublicKey.Cmp(pub) != 0 {
		t.Errorf("Failed to set store.dhPublicKey correctly")
	}
	if store.grp == nil {
		t.Errorf("Failed to set store.grp")
	}
	if store.kv == nil {
		t.Errorf("Failed to set store.kv")
	}
}
