////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/cloudflare/circl/dh/sidh"

	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"

	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"

	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
)

type mockCyHandlerLegacySIDH struct {
}

func (m mockCyHandlerLegacySIDH) AddKey(session.CypherLegacySIDH) {}

func (m mockCyHandlerLegacySIDH) DeleteKey(session.CypherLegacySIDH) {}

// newTestManager returns a new relationship for testing.
func newTestManagerLegacySIDH(t *testing.T) (managerLegacySIDH, *versioned.KV) {
	if t == nil {
		panic("Cannot run this outside tests")
	}

	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng.GetStream())
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng.GetStream())

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	err := partnerSIDHPrivKey.Generate(rng.GetStream())
	if err != nil {
		t.Errorf("Failed to generate private key: %+v", err)
	}
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	err = mySIDHPrivKey.Generate(rng.GetStream())
	if err != nil {
		t.Errorf("Failed to generate private key: %+v", err)
	}
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	kv := versioned.NewKV(ekv.MakeMemstore())
	partnerID := id.NewIdFromString("partner", id.User, t)

	myId := id.NewIdFromString("me", id.User, t)

	// Create new relationship
	m := NewManagerLegacySIDH(kv, myId, partnerID, myPrivKey, partnerPubKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		session.GetDefaultParams(), session.GetDefaultParams(),
		mockCyHandlerLegacySIDH{}, grp,
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG))

	newM := m.(*managerLegacySIDH)

	return *newM, kv
}

func managersEqualLegacySIDH(expected, received *managerLegacySIDH, t *testing.T) bool {
	equal := true
	if !reflect.DeepEqual(expected.cyHandler, received.cyHandler) {
		t.Errorf("Did not Receive expected Manager.cyHandler."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.cyHandler, received.cyHandler)
		equal = false
	}
	if !reflect.DeepEqual(expected.kv, received.kv) {
		t.Errorf("Did not Receive expected Manager.kv."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.kv, received.kv)
		equal = false
	}
	if !expected.partner.Cmp(received.partner) {
		t.Errorf("Did not Receive expected Manager.partner."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.partner, received.partner)
		equal = false
	}
	if !relationshipsEqualLegacySIDH(expected.receive, received.receive) {
		t.Errorf("Did not Receive expected Manager.Receive."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.receive, received.receive)
		equal = false
	}
	if !relationshipsEqualLegacySIDH(expected.send, received.send) {
		t.Errorf("Did not Receive expected Manager.Send."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.send, received.send)
		equal = false
	}

	return equal
}

// Compare certain fields of two session buffs for equality
func relationshipsEqualLegacySIDH(buff *relationshipLegacySIDH, buff2 *relationshipLegacySIDH) bool {
	if len(buff.sessionByID) != len(buff2.sessionByID) {
		return false
	}
	if len(buff.sessions) != len(buff2.sessions) {
		return false
	}

	if !bytes.Equal(buff.fingerprint, buff2.fingerprint) {
		return false
	}
	// Make sure all sessions are present
	for k := range buff.sessionByID {
		_, ok := buff2.sessionByID[k]
		if !ok {
			// key not present in other map
			return false
		}
	}
	// Comparing base key only for now
	// This should ensure that the session buffers have the same sessions in the same order
	for i := range buff.sessions {
		if buff.sessions[i].GetBaseKey().Cmp(buff2.sessions[i].GetBaseKey()) != 0 {
			return false
		}
	}
	return true
}
