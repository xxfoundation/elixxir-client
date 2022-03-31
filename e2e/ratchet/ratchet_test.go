///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// Tests happy path of NewStore.
func TestNewStore(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	privKey := grp.NewInt(57)
	kv := versioned.NewKV(make(ekv.Memstore))
	expectedStore := &Ratchet{
		managers:            make(map[relationshipIdentity]*partner.Manager),
		defaultDHPrivateKey: privKey,
		defaultDHPublicKey:  diffieHellman.GeneratePublicKey(privKey, grp),
		grp:                 grp,
		kv:                  kv.Prefix(packagePrefix),
	}
	expectedData, err := expectedStore.marshal()
	if err != nil {
		t.Fatalf("marshal() produced an error: %v", err)
	}

	err = New(kv, &id.ID{}, privKey, grp)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	key, err := expectedStore.kv.Get(storeKey, 0)
	if err != nil {
		t.Errorf("get() error when getting Ratchet from KV: %v", err)
	}

	if !bytes.Equal(expectedData, key.Data) {
		t.Errorf("NewStore() returned incorrect Ratchet."+
			"\n\texpected: %+v\n\treceived: %+v", expectedData,
			key.Data)
	}
}

// Tests happy path of LoadStore.
func TestLoadStore(t *testing.T) {
	expectedRatchet, kv, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	store, err := Load(kv, &id.ID{},
		expectedRatchet.grp, expectedRatchet.cyHandler, expectedRatchet.sInteface,
		expectedRatchet.rng)
	if err != nil {
		t.Errorf("LoadStore() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedRatchet, store) {
		t.Errorf("LoadStore() returned incorrect Ratchet."+
			"\n\texpected: %#v\n\treceived: %#v", expectedRatchet,
			store)
	}
}

// Tests happy path of Ratchet.AddPartner.
func TestStore_AddPartner(t *testing.T) {
	rng := csprng.NewSystemRNG()
	r, kv, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	p := session.GetDefaultE2ESessionParams()
	partnerPubKey := diffieHellman.GeneratePublicKey(r.defaultDHPrivateKey, r.grp)
	// NOTE: e2e store doesn't contain a private SIDH key, that's
	// because they're completely address as part of the
	// initiation of the connection.
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	myPrivSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)
	expectedManager := partner.NewManager(kv, r.defaultID, partnerID,
		r.defaultDHPrivateKey, partnerPubKey, myPrivSIDHKey, pubSIDHKey,
		p, p, r.cyHandler, r.grp, r.rng)

	receivedManager, err := r.AddPartner(r.defaultID, r.defaultDHPrivateKey,
		partnerID, partnerPubKey, pubSIDHKey, myPrivSIDHKey, p, p, true)
	if err != nil {
		t.Fatalf("AddPartner returned an error: %v", err)
	}

	if !managersEqual(expectedManager, receivedManager, t) {
		t.Errorf("Inconsistent data between partner.Managers")
	}

	relationshipId := makeRelationshipIdentity(partnerID, r.defaultID)

	m, exists := r.managers[relationshipId]
	if !exists {
		t.Errorf("Manager does not exist in map.\n\tmap: %+v",
			r.managers)
	}

	if !managersEqual(expectedManager, m, t) {
		t.Errorf("Inconsistent data between partner.Managers")
	}
}

// Unit test for DeletePartner
func TestStore_DeletePartner(t *testing.T) {
	rng := csprng.NewSystemRNG()
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	partnerPubKey := diffieHellman.GeneratePublicKey(r.defaultDHPrivateKey, r.grp)
	p := session.GetDefaultE2ESessionParams()
	// NOTE: e2e store doesn't contain a private SIDH key, that's
	// because they're completely address as part of the
	// initiation of the connection.
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	myPrivSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)

	_, err = r.AddPartner(r.defaultID, r.defaultDHPrivateKey,
		partnerID, partnerPubKey, pubSIDHKey, myPrivSIDHKey, p, p, true)
	if err != nil {
		t.Fatalf("AddPartner returned an error: %v", err)
	}

	err = r.DeletePartner(partnerID, r.defaultID)
	if err != nil {
		t.Fatalf("DeletePartner received an error: %v", err)
	}

	_, err = r.GetPartner(partnerID, r.defaultID)
	if err == nil {
		t.Errorf("Shouldn't be able to pull deleted partner from store")
	}

}

// Tests happy path of Ratchet.GetPartner.
func TestStore_GetPartner(t *testing.T) {
	rng := csprng.NewSystemRNG()
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	partnerPubKey := diffieHellman.GeneratePublicKey(r.defaultDHPrivateKey, r.grp)
	p := session.GetDefaultE2ESessionParams()
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	myPrivSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)
	expectedManager, err := r.AddPartner(r.defaultID, r.defaultDHPrivateKey,
		partnerID, partnerPubKey, pubSIDHKey, myPrivSIDHKey, p, p, true)
	if err != nil {
		t.Fatalf("AddPartner returned an error: %v", err)
	}

	m, err := r.GetPartner(partnerID, r.defaultID)
	if err != nil {
		t.Errorf("GetPartner() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedManager, m) {
		t.Errorf("GetPartner() returned wrong Manager."+
			"\n\texpected: %v\n\treceived: %v", expectedManager, m)
	}
}

// Tests that Ratchet.GetPartner returns an error for non existent partnerID.
func TestStore_GetPartner_Error(t *testing.T) {
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	m, err := r.GetPartner(partnerID, r.defaultID)
	if err == nil {
		t.Error("GetPartner() did not produce an error.")
	}

	if m != nil {
		t.Errorf("GetPartner() did not return a nil relationship."+
			"\n\texpected: %v\n\treceived: %v", nil, m)
	}
}

// Tests happy path of Ratchet.GetDHPrivateKey.
func TestStore_GetDHPrivateKey(t *testing.T) {
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	if r.defaultDHPrivateKey != r.GetDHPrivateKey() {
		t.Errorf("GetDHPrivateKey() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %v",
			r.defaultDHPrivateKey, r.GetDHPrivateKey())
	}
}

// Tests happy path of Ratchet.GetDHPublicKey.
func TestStore_GetDHPublicKey(t *testing.T) {
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	if r.defaultDHPublicKey != r.GetDHPublicKey() {
		t.Errorf("GetDHPublicKey() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %v",
			r.defaultDHPublicKey, r.GetDHPublicKey())
	}
}
