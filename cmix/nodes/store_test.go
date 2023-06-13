////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"bytes"
	"encoding/base64"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"testing"
	"time"
)

// Happy path add/remove test.
func Test_registrar_add_remove(t *testing.T) {
	r := makeTestRegistrar(&MockClientComms{}, t)
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	nodeId := id.NewIdFromString("test", id.Node, t)
	k := grp.NewInt(5)
	keyId := []byte("keyId")
	r.add(nodeId, k, 0, keyId)
	if _, exists := r.nodes[*nodeId]; !exists {
		t.Fatal("Failed to add node's key.")
	}

	r.remove(nodeId)
	if _, exists := r.nodes[*nodeId]; exists {
		t.Fatal("Failed to remove node's key.")
	}
}

func TestRegistrar_MapUpdate(t *testing.T) {
	r := makeTestRegistrar(&MockClientComms{}, t)
	grp := cyclic.NewGroup(large.NewInt(5000), large.NewInt(2))

	numtests := 100

	edits := make(map[string]versioned.ElementEdit, numtests)
	expectedResults := make(map[id.ID]*key)

	for i := 0; i < numtests; i++ {
		nodeId := id.NewIdFromUInt(uint64(i), id.Node, t)
		k := grp.NewInt(int64(i + 1))
		keyId := []byte("keyId_" + strconv.Itoa(i))
		nodeKey := newKey(k, 0, keyId)
		keyBytes, _ := nodeKey.marshal()

		k2 := grp.NewInt(int64(1000 + i))
		keyId2 := []byte("keyId_" + strconv.Itoa(1000+i))
		nodeKey2 := newKey(k2, 0, keyId2)

		keyOP := versioned.KeyOperation(i % 3)

		var newObject *versioned.Object

		newObject = &versioned.Object{
			Version:   currentStoreMapVersion,
			Timestamp: time.Now(),
			Data:      keyBytes,
		}

		switch keyOP {
		case versioned.Created:
			expectedResults[*nodeId] = nodeKey
		case versioned.Updated:
			expectedResults[*nodeId] = nodeKey
			r.nodes[*nodeId] = nodeKey2
		case versioned.Deleted:
			newObject = nil
			r.nodes[*nodeId] = nodeKey
			expectedResults[*nodeId] = nil
		}

		elementName := base64.StdEncoding.EncodeToString(nodeId[:])

		edits[elementName] = versioned.ElementEdit{
			OldElement: nil,
			NewElement: newObject,
			Operation:  keyOP,
		}
	}

	r.mapUpdate(edits)

	for nid, expectedKey := range expectedResults {

		receivedKey, exists := r.nodes[nid]
		if expectedKey == nil {
			if exists {
				t.Errorf("Key %s not deleted", nid)
			}
			continue
		}
		if !bytes.Equal(expectedKey.keyId, receivedKey.keyId) {
			t.Errorf("key ids dont match: %s vs %s", string(expectedKey.keyId), string(receivedKey.keyId))
		}
		if receivedKey.validUntil != expectedKey.validUntil {
			t.Errorf("valid until does not match")
		}

		if receivedKey.k.Cmp(expectedKey.k) != 0 {
			t.Errorf("keys do not match")
		}
	}
}
