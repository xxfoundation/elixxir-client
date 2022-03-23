///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/interfaces/params"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"math/rand"
	"reflect"
	"testing"
)

// Tests happy path of NewStore.
func TestNewStore(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	privKey := grp.NewInt(57)
	kv := versioned.NewKV(make(ekv.Memstore))
	fingerprints := newFingerprints()
	rng := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)
	e2eP := params.GetDefaultE2ESessionParams()
	expectedStore := &Store{
		managers:     make(map[id.ID]*Manager),
		dhPrivateKey: privKey,
		dhPublicKey:  diffieHellman.GeneratePublicKey(privKey, grp),
		grp:          grp,
		kv:           kv.Prefix(packagePrefix),
		fingerprints: &fingerprints,
		context: &context{
			fa:   &fingerprints,
			grp:  grp,
			rng:  rng,
			myID: &id.ID{},
		},
		e2eParams: e2eP,
	}
	expectedData, err := expectedStore.marshal()
	if err != nil {
		t.Fatalf("marshal() produced an error: %v", err)
	}

	store, err := NewStore(grp, kv, privKey, &id.ID{}, rng)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedStore, store) {
		t.Errorf("NewStore() returned incorrect Store."+
			"\n\texpected: %+v\n\treceived: %+v", expectedStore,
			store)
	}

	key, err := expectedStore.kv.Get(storeKey, 0)
	if err != nil {
		t.Errorf("get() error when getting Store from KV: %v", err)
	}

	if !bytes.Equal(expectedData, key.Data) {
		t.Errorf("NewStore() returned incorrect Store."+
			"\n\texpected: %+v\n\treceived: %+v", expectedData,
			key.Data)
	}
}

// Tests happy path of LoadStore.
func TestLoadStore(t *testing.T) {
	expectedStore, kv, rng := makeTestStore()

	store, err := LoadStore(kv, &id.ID{}, rng)
	if err != nil {
		t.Errorf("LoadStore() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedStore, store) {
		t.Errorf("LoadStore() returned incorrect Store."+
			"\n\texpected: %#v\n\treceived: %#v", expectedStore,
			store)
	}
}

// Tests happy path of Store.AddPartner.
func TestStore_AddPartner(t *testing.T) {
	rng := csprng.NewSystemRNG()
	s, _, _ := makeTestStore()
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	pubKey := diffieHellman.GeneratePublicKey(s.dhPrivateKey, s.grp)
	p := params.GetDefaultE2ESessionParams()
	// NOTE: e2e store doesn't contain a private SIDH key, that's
	// because they're completely address as part of the
	// initiation of the connection.
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	privSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)
	expectedManager := newManager(s.context, s.kv, partnerID,
		s.dhPrivateKey, pubKey,
		privSIDHKey, pubSIDHKey,
		p, p)

	err := s.AddPartner(partnerID, pubKey, s.dhPrivateKey, pubSIDHKey,
		privSIDHKey, p, p)
	if err != nil {
		t.Fatalf("AddPartner returned an error: %v", err)
	}

	m, exists := s.managers[*partnerID]
	if !exists {
		t.Errorf("Manager does not exist in map.\n\tmap: %+v",
			s.managers)
	}

	if !reflect.DeepEqual(expectedManager, m) {
		t.Errorf("Added Manager not expected.\n\texpected: "+
			"%v\n\treceived: %v", expectedManager, m)
	}
}

// Unit test for DeletePartner
func TestStore_DeletePartner(t *testing.T) {
	rng := csprng.NewSystemRNG()
	s, _, _ := makeTestStore()
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	pubKey := diffieHellman.GeneratePublicKey(s.dhPrivateKey, s.grp)
	p := params.GetDefaultE2ESessionParams()
	// NOTE: e2e store doesn't contain a private SIDH key, that's
	// because they're completely address as part of the
	// initiation of the connection.
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	privSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)

	err := s.AddPartner(partnerID, pubKey, s.dhPrivateKey, pubSIDHKey,
		privSIDHKey, p, p)
	if err != nil {
		t.Fatalf("Could not add partner in set up: %v", err)
	}

	err = s.DeletePartner(partnerID)
	if err != nil {
		t.Fatalf("DeletePartner received an error: %v", err)
	}

	_, err = s.GetPartner(partnerID)
	if err == nil {
		t.Errorf("Shouldn't be able to pull deleted partner from store")
	}

}

// Tests happy path of Store.GetPartner.
func TestStore_GetPartner(t *testing.T) {
	rng := csprng.NewSystemRNG()
	s, _, _ := makeTestStore()
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	pubKey := diffieHellman.GeneratePublicKey(s.dhPrivateKey, s.grp)
	p := params.GetDefaultE2ESessionParams()
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	privSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)
	expectedManager := newManager(s.context, s.kv, partnerID,
		s.dhPrivateKey, pubKey, privSIDHKey, pubSIDHKey, p, p)
	_ = s.AddPartner(partnerID, pubKey, s.dhPrivateKey, pubSIDHKey,
		privSIDHKey, p, p)

	m, err := s.GetPartner(partnerID)
	if err != nil {
		t.Errorf("GetPartner() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedManager, m) {
		t.Errorf("GetPartner() returned wrong Manager."+
			"\n\texpected: %v\n\treceived: %v", expectedManager, m)
	}
}

// Tests that Store.GetPartner returns an error for non existent partnerID.
func TestStore_GetPartner_Error(t *testing.T) {
	s, _, _ := makeTestStore()
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	m, err := s.GetPartner(partnerID)
	if err == nil {
		t.Error("GetPartner() did not produce an error.")
	}

	if m != nil {
		t.Errorf("GetPartner() did not return a nil relationship."+
			"\n\texpected: %v\n\treceived: %v", nil, m)
	}
}

// Tests happy path of Store.GetPartnerContact.
func TestStore_GetPartnerContact(t *testing.T) {
	s, _, _ := makeTestStore()
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	pubKey := diffieHellman.GeneratePublicKey(s.dhPrivateKey, s.grp)
	p := params.GetDefaultE2ESessionParams()
	expected := contact.Contact{
		ID:       partnerID,
		DhPubKey: pubKey,
	}
	rng := csprng.NewSystemRNG()
	_, pubSIDHKey := genSidhKeys(rng, sidh.KeyVariantSidhA)
	privSIDHKey, _ := genSidhKeys(rng, sidh.KeyVariantSidhB)

	_ = s.AddPartner(partnerID, pubKey, s.dhPrivateKey, pubSIDHKey,
		privSIDHKey, p, p)

	c, err := s.GetPartnerContact(partnerID)
	if err != nil {
		t.Errorf("GetPartnerContact() produced an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, c) {
		t.Errorf("GetPartnerContact() returned wrong Contact."+
			"\nexpected: %s\nreceived: %s", expected, c)
	}
}

// Tests that Store.GetPartnerContact returns an error for non existent partnerID.
func TestStore_GetPartnerContact_Error(t *testing.T) {
	s, _, _ := makeTestStore()
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	_, err := s.GetPartnerContact(partnerID)
	if err == nil || err.Error() != NoPartnerErrorStr {
		t.Errorf("GetPartnerContact() did not produce the expected error."+
			"\nexpected: %s\nreceived: %+v", NoPartnerErrorStr, err)
	}
}

// Tests happy path of Store.PopKey.
func TestStore_PopKey(t *testing.T) {
	s, _, _ := makeTestStore()
	se, _ := makeTestSession()

	// Pop Key that does not exist
	fp := format.Fingerprint{0xF, 0x6, 0x2}
	key, exists := s.PopKey(fp)
	if exists {
		t.Errorf("PopKey() popped a Key with fingerprint %v that should not "+
			"exist.", fp)
	}
	if key != nil {
		t.Errorf("PopKey() did not return a nil Key when it should not exist."+
			"\n\texpected: %+v\n\treceived: %+v", nil, key)
	}

	// Add a Key
	keys := []*Key{newKey(se, 0), newKey(se, 1), newKey(se, 2)}
	s.add(keys)
	fp = keys[0].Fingerprint()

	// Pop a Key that does exist
	key, exists = s.PopKey(fp)
	if !exists {
		t.Errorf("PopKey() could not find Key with fingerprint %v.", fp)
	}

	if !reflect.DeepEqual(keys[0], key) {
		t.Errorf("PopKey() did not return the correct Key."+
			"\n\texpected: %+v\n\trecieved: %+v", keys[0], key)
	}
}

// Tests happy path of Store.CheckKey.
func TestStore_CheckKey(t *testing.T) {
	s, _, _ := makeTestStore()
	se, _ := makeTestSession()

	// Check for a Key that does not exist
	fp := format.Fingerprint{0xF, 0x6, 0x2}
	exists := s.CheckKey(fp)
	if exists {
		t.Errorf("CheckKey() found a Key with fingerprint %v.", fp)
	}

	// Add Keys
	keys := []*Key{newKey(se, 0), newKey(se, 1), newKey(se, 2)}
	s.add(keys)
	fp = keys[0].Fingerprint()

	// Check for a Key that does exist
	exists = s.CheckKey(fp)
	if !exists {
		t.Errorf("CheckKey() could not find Key with fingerprint %v.", fp)
	}
}

// Tests happy path of Store.GetDHPrivateKey.
func TestStore_GetDHPrivateKey(t *testing.T) {
	s, _, _ := makeTestStore()

	if s.dhPrivateKey != s.GetDHPrivateKey() {
		t.Errorf("GetDHPrivateKey() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %v",
			s.dhPrivateKey, s.GetDHPrivateKey())
	}
}

// Tests happy path of Store.GetDHPublicKey.
func TestStore_GetDHPublicKey(t *testing.T) {
	s, _, _ := makeTestStore()

	if s.dhPublicKey != s.GetDHPublicKey() {
		t.Errorf("GetDHPublicKey() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %v",
			s.dhPublicKey, s.GetDHPublicKey())
	}
}

// Tests happy path of Store.GetGroup.
func TestStore_GetGroup(t *testing.T) {
	s, _, _ := makeTestStore()

	if s.grp != s.GetGroup() {
		t.Errorf("GetGroup() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %v",
			s.grp, s.GetGroup())
	}
}

// Tests happy path of newFingerprints.
func Test_newFingerprints(t *testing.T) {
	expectedFp := fingerprints{toKey: make(map[format.Fingerprint]*Key)}
	fp := newFingerprints()

	if !reflect.DeepEqual(&expectedFp, &fp) {
		t.Errorf("newFingerprints() returned incorrect fingerprints."+
			"\n\texpected: %+v\n\treceived: %+v", &expectedFp, &fp)
	}
}

// Tests happy path of fingerprints.add.
func TestFingerprints_add(t *testing.T) {
	se, _ := makeTestSession()
	keys := []*Key{newKey(se, 0), newKey(se, 1), newKey(se, 2)}
	fps := newFingerprints()
	fps.add(keys)

	for i, key := range keys {
		testKey, exists := fps.toKey[key.Fingerprint()]
		if !exists {
			t.Errorf("add() failed to add key with fingerprint %v (round %d).",
				key.Fingerprint(), i)
		}

		if !reflect.DeepEqual(key, testKey) {
			t.Errorf("add() did not add the correct Key for fingerprint %v "+
				"(round %d).\n\texpected: %v\n\treceived: %v",
				key.Fingerprint(), i, key, testKey)
		}
	}
}

// Tests happy path of fingerprints.remove.
func TestFingerprints_remove(t *testing.T) {
	se, _ := makeTestSession()
	keys := []*Key{newKey(se, 0), newKey(se, 1), newKey(se, 2)}
	fps := newFingerprints()
	fps.add(keys)
	fps.remove(keys)

	if len(fps.toKey) != 0 {
		t.Errorf("remove() failed to remove all the keys.\n\tmap: %v", fps.toKey)
	}
}

func makeTestStore() (*Store, *versioned.KV, *fastRNG.StreamGenerator) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	privKey := grp.NewInt(57)
	kv := versioned.NewKV(make(ekv.Memstore))
	rng := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)
	s, err := NewStore(grp, kv, privKey, &id.ID{}, rng)
	if err != nil {
		panic("NewStore() produced an error: " + err.Error())
	}
	return s, kv, rng
}

func genSidhKeys(rng io.Reader, variant sidh.KeyVariant) (*sidh.PrivateKey, *sidh.PublicKey) {
	sidHPrivKey := util.NewSIDHPrivateKey(variant)
	sidHPubKey := util.NewSIDHPublicKey(variant)

	if err := sidHPrivKey.Generate(rng); err != nil {
		panic("failure to generate SidH A private key")
	}
	sidHPrivKey.GeneratePublicKey(sidHPubKey)

	return sidHPrivKey, sidHPubKey
}
