///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"testing"
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
