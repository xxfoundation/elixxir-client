////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"bytes"
	"gitlab.com/elixxir/client/v5/cmix/gateway"
	"gitlab.com/elixxir/client/v5/storage"
	commNetwork "gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Tests that LoadRegistrar returns a new Registrar when none exists
// in the KV.
func TestLoadRegistrar_New(t *testing.T) {
	connect.TestingOnlyDisableTLS = true
	session := storage.InitTestingSession(t)
	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	sender, err := gateway.NewSender(gateway.DefaultPoolParams(), rngGen,
		getNDF(), newMockManager(), session, nil)
	if err != nil {
		t.Fatalf("Failed to create new sender: %+v", err)
	}
	nodeChan := make(chan commNetwork.NodeGateway, InputChanLen)

	r, err := LoadRegistrar(session, sender, &MockClientComms{},
		rngGen, nodeChan, func() int { return 100 })
	if err != nil {
		t.Fatalf("Failed to create new registrar: %+v", err)
	}

	if r.(*registrar).nodes == nil {
		t.Errorf("Failed to initialize nodes")
	}
	if r.(*registrar).kv == nil {
		t.Errorf("Failed to set store.kv")
	}
}

func TestLoadRegistrar_Load(t *testing.T) {
	testR := makeTestRegistrar(&MockClientComms{}, t)
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	// Add a test nodes key
	nodeId := id.NewIdFromString("test", id.Node, t)
	k := grp.NewInt(5)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("Could not parse precanned time: %v", err.Error())
	}
	expectedValid := uint64(testTime.UnixNano())

	expectedKeyId := []byte("expectedKeyID")

	testR.add(nodeId, k, expectedValid, expectedKeyId)

	// Load the store and check its attributes
	r, err := LoadRegistrar(
		testR.session, testR.sender, testR.comms, testR.rng, testR.c, func() int { return 100 })
	if err != nil {
		t.Fatalf("Unable to load store: %+v", err)
	}
	if len(r.(*registrar).nodes) != len(testR.nodes) {
		t.Errorf("LoadStore failed to load nodes keys")
	}

	circuit := connect.NewCircuit([]*id.ID{nodeId})
	keys, _ := r.GetNodeKeys(circuit)
	if keys.(*mixCypher).keys[0].validUntil != expectedValid {
		t.Errorf("Unexpected valid until value loaded from store."+
			"\nexpected: %v\nreceived: %v",
			expectedValid, keys.(*mixCypher).keys[0].validUntil)
	}
	if !bytes.Equal(keys.(*mixCypher).keys[0].keyId, expectedKeyId) {
		t.Errorf("Unexpected keyID value loaded from store."+
			"\nexpected: %v\nreceived: %v",
			expectedKeyId, keys.(*mixCypher).keys[0].keyId)
	}
}

func Test_registrar_GetNodeKeys(t *testing.T) {
	r := makeTestRegistrar(&MockClientComms{}, t)
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	// Set up the circuit
	numIds := 10
	nodeIds := make([]*id.ID, numIds)
	for i := 0; i < numIds; i++ {
		nodeIds[i] = id.NewIdFromUInt(uint64(i)+1, id.Node, t)
		k := grp.NewInt(int64(i) + 1)
		r.add(nodeIds[i], k, 0, nil)

		// This is wack but it cleans up after the test
		defer r.remove(nodeIds[i])
	}

	circuit := connect.NewCircuit(nodeIds)
	result, err := r.GetNodeKeys(circuit)
	if err != nil {
		t.Errorf("GetNodeKeys returrned an error: %+v", err)
	}
	if result == nil || len(result.(*mixCypher).keys) != numIds {
		t.Errorf("Expected to have %d nodes keys", numIds)
	}
}

func Test_registrar_GetNodeKeys_Missing(t *testing.T) {
	r := makeTestRegistrar(&MockClientComms{}, t)
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	// Set up the circuit
	numIds := 10
	nodeIds := make([]*id.ID, numIds)
	for i := 0; i < numIds; i++ {
		nodeIds[i] = id.NewIdFromUInt(uint64(i)+1, id.Node, t)
		k := grp.NewInt(int64(i) + 1)

		// Only add every other nodes so there are missing nodes
		if i%2 == 0 {
			r.add(nodeIds[i], k, 0, nil)
			r.add(nodeIds[i], k, 0, nil)

			// This is wack but it cleans up after the test
			defer r.remove(nodeIds[i])
		}
	}

	circuit := connect.NewCircuit(nodeIds)
	result, err := r.GetNodeKeys(circuit)
	if err == nil {
		t.Error("GetNodeKeys did not return an error when keys " +
			"should be missing.")
	}
	if result != nil {
		t.Errorf("Expected nil value for result due to " +
			"missing keys!")
	}
}

func Test_registrar_HasNode(t *testing.T) {
	r := makeTestRegistrar(&MockClientComms{}, t)
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	nodeId := id.NewIdFromString("test", id.Node, t)
	k := grp.NewInt(5)

	r.add(nodeId, k, 0, nil)
	if _, exists := r.nodes[*nodeId]; !exists {
		t.Fatal("Failed to add node's key.")
	}

	if !r.HasNode(nodeId) {
		t.Fatal("Cannot find the node's ID that that was added.")
	}
}

// Tests that Has returns false when it does not have.
func Test_registrar_Has_Not(t *testing.T) {
	r := makeTestRegistrar(&MockClientComms{}, t)

	nodeId := id.NewIdFromString("test", id.Node, t)

	if r.HasNode(nodeId) {
		t.Fatal("Found the node when it should not have been found.")
	}
}

func Test_registrar_NumRegistered(t *testing.T) {
	r := makeTestRegistrar(&MockClientComms{}, t)
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	if r.NumRegisteredNodes() != 0 {
		t.Errorf("Unexpected NumRegisteredNodes for a new Registrar."+
			"\nexpected: %d\nreceived: %d", 0, r.NumRegisteredNodes())
	}

	count := 50
	for i := 0; i < count; i++ {
		r.add(id.NewIdFromUInt(uint64(i), id.Node, t),
			grp.NewInt(int64(42+i)), 0, nil)
	}

	if r.NumRegisteredNodes() != count {
		t.Errorf("Unexpected NumRegisteredNodes."+
			"\nexpected: %d\nreceived: %d", count, r.NumRegisteredNodes())
	}
}
