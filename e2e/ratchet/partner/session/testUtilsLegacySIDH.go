////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"testing"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"

	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"

	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

func CreateTestSessionLegacySIDH(numKeys, keysAvailable, rekeyThreshold uint32, status Negotiation, t *testing.T) (*SessionLegacySIDH, *versioned.KV) {
	if t == nil {
		panic("Cannot run this outside tests")
	}
	s, kv := makeTestSessionLegacySIDH()
	if rekeyThreshold > 0 {
		s.rekeyThreshold = rekeyThreshold
	}
	if numKeys > 0 {
		s.keyState.SetNumKeysTEST(numKeys, t)
	}
	if keysAvailable > 0 {
		s.keyState.SetNumAvailableTEST(keysAvailable, t)
	}

	s.negotiationStatus = status

	return s, kv
}

// Make a default test session with some things populated
func makeTestSessionLegacySIDH() (*SessionLegacySIDH, *versioned.KV) {
	grp := getGroup()
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	baseKey := GenerateE2ESessionBaseKeyLegacySIDH(myPrivKey, partnerPubKey, grp,
		mySIDHPrivKey, partnerSIDHPubKey)
	kv := versioned.NewKV(ekv.MakeMemstore())
	sid := GetSessionIDFromBaseKey(baseKey)

	s := &SessionLegacySIDH{
		baseKey:           baseKey,
		myPrivKey:         myPrivKey,
		partnerPubKey:     partnerPubKey,
		mySIDHPrivKey:     mySIDHPrivKey,
		partnerSIDHPubKey: partnerSIDHPubKey,
		e2eParams:         GetDefaultParams(),
		sID:               sid,
		kv:                kv.Prefix(MakeSessionPrefix(sid)),
		t:                 Receive,
		negotiationStatus: Confirmed,
		rekeyThreshold:    5,
		partner:           &id.ID{},
		grp:               grp,
		cyHandler:         &mockCyHandlerLegacySIDH{},
		rng:               fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
	}
	var err error
	s.keyState, err = util.NewStateVector(s.kv,
		"", 1024)
	if err != nil {
		panic(err)
	}
	return s, kv
}

// compare fields also represented in SessionDisk
// fields not represented in SessionDisk shouldn't be expected to be populated by Unmarshal
func cmpSerializedFieldsLegacySIDH(a *SessionLegacySIDH, b *SessionLegacySIDH) error {
	if a.negotiationStatus != b.negotiationStatus {
		return errors.New("confirmed differed")
	}
	if a.t != b.t {
		return errors.New("t differed")
	}
	if a.e2eParams.MaxKeys != b.e2eParams.MaxKeys {
		return errors.New("maxKeys differed")
	}
	if a.e2eParams.MinKeys != b.e2eParams.MinKeys {
		return errors.New("minKeys differed")
	}
	if a.e2eParams.NumRekeys != b.e2eParams.NumRekeys {
		return errors.New("NumRekeys differed")
	}
	if a.baseKey.Cmp(b.baseKey) != 0 {
		return errors.New("baseKey differed")
	}
	if a.myPrivKey.Cmp(b.myPrivKey) != 0 {
		return errors.New("myPrivKey differed")
	}
	if a.partnerPubKey.Cmp(b.partnerPubKey) != 0 {
		return errors.New("partnerPubKey differed")
	}
	return nil
}

type mockCyHandlerLegacySIDH struct{}

func (m *mockCyHandlerLegacySIDH) AddKey(CypherLegacySIDH)    {}
func (m *mockCyHandlerLegacySIDH) DeleteKey(CypherLegacySIDH) {}
