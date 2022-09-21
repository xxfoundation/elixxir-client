////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"bytes"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/cloudflare/circl/dh/sidh"

	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/crypto/diffieHellman"
)

// Tests happy path of Ratchet.AddPartner.
func TestStore_AddPartnerLegacySIDH(t *testing.T) {
	rng := csprng.NewSystemRNG()
	r, kv, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	p := session.GetDefaultParams()
	partnerPubKey := diffieHellman.GeneratePublicKey(r.advertisedDHPrivateKey, r.grp)
	// NOTE: e2e store doesn't contain a private SIDH key, that's
	// because they're completely address as part of the
	// initiation of the connection.
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	myPrivSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)
	expectedManager := partner.NewManagerLegacySIDH(kv, r.myID, partnerID,
		r.advertisedDHPrivateKey, partnerPubKey, myPrivSIDHKey, pubSIDHKey,
		p, p, r.cyHandlerLegacySIDH, r.grp, r.rng)

	receivedManager, err := r.AddPartnerLegacySIDH(
		partnerID,
		partnerPubKey, r.advertisedDHPrivateKey,
		pubSIDHKey, myPrivSIDHKey, p, p)
	if err != nil {
		t.Fatalf("AddPartner returned an error: %v", err)
	}

	if !managersEqualLegacySIDH(expectedManager, receivedManager, t) {
		t.Errorf("Inconsistent data between partner.Managers")
	}

	relationshipId := *partnerID

	m, exists := r.managersLegacySIDH[relationshipId]
	if !exists {
		t.Errorf("Manager does not exist in map.\n\tmap: %+v",
			r.managers)
	}

	if !managersEqualLegacySIDH(expectedManager, m, t) {
		t.Errorf("Inconsistent data between partner.Managers")
	}
}

// Unit test for DeletePartner
func TestStore_DeletePartnerLegacySIDH(t *testing.T) {
	rng := csprng.NewSystemRNG()
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	partnerPubKey := diffieHellman.GeneratePublicKey(r.advertisedDHPrivateKey, r.grp)
	p := session.GetDefaultParams()
	// NOTE: e2e store doesn't contain a private SIDH key, that's
	// because they're completely address as part of the
	// initiation of the connection.
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	myPrivSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)

	_, err = r.AddPartnerLegacySIDH(partnerID, r.advertisedDHPrivateKey,
		partnerPubKey, pubSIDHKey, myPrivSIDHKey, p, p)
	if err != nil {
		t.Fatalf("AddPartner returned an error: %v", err)
	}

	err = r.DeletePartnerLegacySIDH(partnerID)
	if err != nil {
		t.Fatalf("DeletePartner received an error: %v", err)
	}

	_, err = r.GetPartnerLegacySIDH(partnerID)
	if err == nil {
		t.Errorf("Shouldn't be able to pull deleted partner from store")
	}

}

// Tests happy path of Ratchet.GetPartner.
func TestStore_GetPartnerLegacySIDH(t *testing.T) {
	rng := csprng.NewSystemRNG()
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	partnerPubKey := diffieHellman.GeneratePublicKey(r.advertisedDHPrivateKey, r.grp)
	p := session.GetDefaultParams()
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	myPrivSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)
	expectedManager, err := r.AddPartnerLegacySIDH(partnerID, r.advertisedDHPrivateKey,
		partnerPubKey, pubSIDHKey, myPrivSIDHKey, p, p)
	if err != nil {
		t.Fatalf("AddPartner returned an error: %v", err)
	}

	m, err := r.GetPartnerLegacySIDH(partnerID)
	if err != nil {
		t.Errorf("GetPartner() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedManager, m) {
		t.Errorf("GetPartner() returned wrong Manager."+
			"\n\texpected: %v\n\treceived: %v", expectedManager, m)
	}
}

// Ratchet.GetAllPartnerIDs unit test.
func TestRatchet_GetAllPartnerIDsLegacySIDH(t *testing.T) {
	// Setup
	numTests := 100
	expectedPartners := make([]*id.ID, 0, numTests)
	rng := csprng.NewSystemRNG()
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	// Generate partners and add them ot the manager
	for i := 0; i < numTests; i++ {
		partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
		partnerPubKey := diffieHellman.GeneratePublicKey(r.advertisedDHPrivateKey, r.grp)
		p := session.GetDefaultParams()
		_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
		myPrivSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)
		_, err := r.AddPartnerLegacySIDH(partnerID, r.advertisedDHPrivateKey,
			partnerPubKey, pubSIDHKey, myPrivSIDHKey, p, p)
		if err != nil {
			t.Fatalf("AddPartner returned an error: %v", err)
		}

		expectedPartners = append(expectedPartners, partnerID)
	}

	receivedPartners := r.GetAllPartnerIDsLegacySIDH()

	// Sort these slices as GetAllPartnerIDs iterates over a map, which indices
	// at random in Go
	sort.SliceStable(receivedPartners, func(i, j int) bool {
		return bytes.Compare(receivedPartners[i].Bytes(), receivedPartners[j].Bytes()) == -1
	})

	sort.SliceStable(expectedPartners, func(i, j int) bool {
		return bytes.Compare(expectedPartners[i].Bytes(), expectedPartners[j].Bytes()) == -1
	})

	if !reflect.DeepEqual(receivedPartners, expectedPartners) {
		t.Fatalf("Unexpected data retrieved from GetAllPartnerIDs."+
			"\nExpected: %v"+
			"\nReceived: %v", expectedPartners, receivedPartners)
	}

}

// Tests that Ratchet.GetPartner returns an error for non existent partnerID.
func TestStore_GetPartner_ErrorLegacySIDH(t *testing.T) {
	r, _, err := makeTestRatchet()
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	m, err := r.GetPartnerLegacySIDH(partnerID)
	if err == nil {
		t.Error("GetPartner() did not produce an error.")
	}

	if m != nil {
		t.Errorf("GetPartner() did not return a nil relationship."+
			"\n\texpected: %v\n\treceived: %v", nil, m)
	}
}
