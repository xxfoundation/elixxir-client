////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
)

// Tests that a confirmation for different partners and sentByFingerprints can be
// saved and loaded from storage via Store.StoreConfirmation and
// Store.LoadConfirmation.
func TestStore_StoreConfirmation_LoadConfirmation(t *testing.T) {
	s := &Store{kv: versioned.NewKV(ekv.MakeMemstore())}
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	testValues := make([]struct {
		partner           *id.ID
		fingerprint       format.Fingerprint
		confirmation, mac []byte
	}, 10)

	var partner *id.ID
	for i := range testValues {
		partner, _ = id.NewRandomID(prng, id.User)

		// Generate original fingerprint

		var fpBytes []byte
		var fp format.Fingerprint
		if i%2 == 1 {
			dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
			_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
			fpBytes = auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
			fp = format.NewFingerprint(fpBytes)
		}

		// Generate confirmation
		confirmation := make([]byte, 32)
		mac := make([]byte, 32)
		prng.Read(confirmation)
		prng.Read(mac)
		mac[0] = 0

		testValues[i] = struct {
			partner           *id.ID
			fingerprint       format.Fingerprint
			confirmation, mac []byte
		}{partner: partner, fingerprint: fp, confirmation: confirmation, mac: mac}

		err := s.StoreConfirmation(partner, confirmation, mac, fp)
		if err != nil {
			t.Errorf("StoreConfirmation returned an error (%d): %+v", i, err)
		}
	}

	for i, val := range testValues {
		loadedConfirmation, mac, fp, err := s.LoadConfirmation(val.partner)
		if err != nil {
			t.Errorf("LoadConfirmation returned an error (%d): %+v", i, err)
		}

		if !reflect.DeepEqual(val.confirmation, loadedConfirmation) {
			t.Errorf("Loaded confirmation does not match original (%d)."+
				"\n\texpected: %v\n\treceived: %v\n", i, val.confirmation,
				loadedConfirmation)
		}
		if !reflect.DeepEqual(val.mac, mac) {
			t.Errorf("Loaded mac does not match original (%d)."+
				"\n\texpected: %v\n\treceived: %v\n", i, val.mac,
				mac)
		}
		if !reflect.DeepEqual(val.fingerprint, fp) {
			t.Errorf("Loaded fingerprint does not match original (%d)."+
				"\n\texpected: %v\n\treceived: %v\n", i, val.fingerprint,
				fp)
		}
	}
}

// Tests that Store.DeleteConfirmation deletes the correct confirmation from
// storage and that it cannot be loaded from storage.
func TestStore_deleteConfirmation(t *testing.T) {
	s := &Store{kv: versioned.NewKV(ekv.MakeMemstore())}
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	testValues := make([]struct {
		partner           *id.ID
		fingerprint       format.Fingerprint
		confirmation, mac []byte
	}, 10)

	for i := range testValues {
		partner, _ := id.NewRandomID(prng, id.User)
		//if i%2 == 0 {
		//	partner, _ = id.NewRandomID(prng, id.User)
		//}

		// Generate original fingerprint
		var fpBytes []byte
		var fp format.Fingerprint
		if i%2 == 1 {
			dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
			_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
			fpBytes = auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
			fp = format.NewFingerprint(fpBytes)
		}

		// Generate confirmation
		confirmation := make([]byte, 32)
		mac := make([]byte, 32)
		prng.Read(confirmation)
		prng.Read(mac)
		mac[0] = 0

		testValues[i] = struct {
			partner           *id.ID
			fingerprint       format.Fingerprint
			confirmation, mac []byte
		}{partner: partner, fingerprint: fp, confirmation: confirmation, mac: mac}

		err := s.StoreConfirmation(partner, confirmation, mac, fp)
		if err != nil {
			t.Errorf("StoreConfirmation returned an error (%d): %+v", i, err)
		}
	}

	for i, val := range testValues {
		err := s.DeleteConfirmation(val.partner)
		if err != nil {
			t.Errorf("DeleteConfirmation returned an error (%d): %+v", i, err)
		}

		loadedConfirmation, mac, _, err := s.LoadConfirmation(val.partner)
		if err == nil || loadedConfirmation != nil || mac != nil {
			t.Errorf("LoadConfirmation returned a confirmation for partner "+
				"%s and fingerprint %v (%d)", val.partner, val.fingerprint, i)
		}
	}
}

// Consistency test of makeConfirmationKey.
func Test_makeConfirmationKey_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	expectedKeys := []string{
		"Confirmation/U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
		"Confirmation/P2HTdbwCtB30+Rkp4Y/anm+C5U50joHnnku9b+NM3LoD",
		"Confirmation/r66IG4KnURCKQu08kDyqQ0ZaeGIGFpeK7QzjxsTzrnsD",
		"Confirmation/otwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGFJiUcD",
		"Confirmation/lk39x56NU0NzZhz9ZtdP7B4biUkatyNuS3UhYpDPK+sD",
		"Confirmation/l4KD1KCaNvlsIJQXRuPtTaZGqa6LT6e0/Doguvoade0D",
		"Confirmation/HPCdo54Okp0CSry8sWk5e7c05+8KbgHxhU3rX+Qk/vcD",
		"Confirmation/Ud9mj4dOLJ8c4JyoYBfn4hdIMD/0HBsj4RxI7RdTnWgD",
		"Confirmation/Ximg3KRqw6DVcBM7whVx9fVKZDEFUT/YQpsZSuG6nyoD",
		"Confirmation/ZxkHLWcvYfqgvob0V5Iew3wORgzw1wPQfcX1ZhpFATMD",
		"Confirmation/IpwYPBkzqRZYXhg7twkZLbDmyNcJudc4O5k8aUmZRbAD",
		"Confirmation/Rc0b8Lz8GjRsQ08RzwBBb6YWlbkgLmg2Ohx4f0eE4K4D",
		"Confirmation/1ieMn3yHL4QPnZTZ/e2uk9sklXGPWAuMjyvsxqp2w7AD",
		"Confirmation/FER0v9N80ga1Gs4FCrYZnsezltYY/eDhopmabz2fi3oD",
		"Confirmation/KRnCqHpJlPweQB4RxaScfo6p5l1sxARl/TUvLELsPT4D",
		"Confirmation/Q9EGMwNtPUa4GRauRv8T1qay+tkHnW3zRAWQKWZ7LrQD",
	}

	for i, expected := range expectedKeys {
		partner, _ := id.NewRandomID(prng, id.User)
		dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
		_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
		fp := auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)

		key := makeConfirmationKey(partner)
		if expected != key {
			t.Errorf("Confirmation key does not match expected for partner "+
				"%s and fingerprint %v (%d).\nexpected: %q\nreceived: %q",
				partner, fp, i, expected, key)
		}

		// fmt.Printf("\"%s\",\n", key)
	}
}
