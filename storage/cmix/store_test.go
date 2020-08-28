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
	"testing"
)

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
