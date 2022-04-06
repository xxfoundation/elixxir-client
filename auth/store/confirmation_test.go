////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// Tests that a confirmation for different partners and sentByFingerprints can be
// saved and loaded from storage via Store.StoreConfirmation and
// Store.LoadConfirmation.
func TestStore_StoreConfirmation_LoadConfirmation(t *testing.T) {
	s := &Store{kv: versioned.NewKV(make(ekv.Memstore))}
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	testValues := make([]struct {
		partner                   *id.ID
		fingerprint, confirmation []byte
	}, 10)

	partner, _ := id.NewRandomID(prng, id.User)
	for i := range testValues {
		if i%2 == 0 {
			partner, _ = id.NewRandomID(prng, id.User)
		}

		// Generate original fingerprint
		var fp []byte
		if i%2 == 1 {
			dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
			_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
			fp = auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
		}

		// Generate confirmation
		confirmation := make([]byte, 32)
		prng.Read(confirmation)

		testValues[i] = struct {
			partner                   *id.ID
			fingerprint, confirmation []byte
		}{partner: partner, fingerprint: fp, confirmation: confirmation}

		err := s.StoreConfirmation(partner, fp, confirmation)
		if err != nil {
			t.Errorf("StoreConfirmation returned an error (%d): %+v", i, err)
		}
	}

	for i, val := range testValues {
		loadedConfirmation, err := s.LoadConfirmation(val.partner, val.fingerprint)
		if err != nil {
			t.Errorf("LoadConfirmation returned an error (%d): %+v", i, err)
		}

		if !reflect.DeepEqual(val.confirmation, loadedConfirmation) {
			t.Errorf("Loaded confirmation does not match original (%d)."+
				"\nexpected: %v\nreceived: %v", i, val.confirmation,
				loadedConfirmation)
		}
	}
}

// Tests that Store.DeleteConfirmation deletes the correct confirmation from
// storage and that it cannot be loaded from storage.
func TestStore_deleteConfirmation(t *testing.T) {
	s := &Store{kv: versioned.NewKV(make(ekv.Memstore))}
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	testValues := make([]struct {
		partner                   *id.ID
		fingerprint, confirmation []byte
	}, 10)

	partner, _ := id.NewRandomID(prng, id.User)
	for i := range testValues {
		if i%2 == 0 {
			partner, _ = id.NewRandomID(prng, id.User)
		}

		// Generate original fingerprint
		var fp []byte
		if i%2 == 1 {
			dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
			_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
			fp = auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
		}

		// Generate confirmation
		confirmation := make([]byte, 32)
		prng.Read(confirmation)

		testValues[i] = struct {
			partner                   *id.ID
			fingerprint, confirmation []byte
		}{partner: partner, fingerprint: fp, confirmation: confirmation}

		err := s.StoreConfirmation(partner, fp, confirmation)
		if err != nil {
			t.Errorf("StoreConfirmation returned an error (%d): %+v", i, err)
		}
	}

	for i, val := range testValues {
		err := s.DeleteConfirmation(val.partner, val.fingerprint)
		if err != nil {
			t.Errorf("DeleteConfirmation returned an error (%d): %+v", i, err)
		}

		loadedConfirmation, err := s.LoadConfirmation(val.partner, val.fingerprint)
		if err == nil || loadedConfirmation != nil {
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
		"Confirmation/U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID/VzgXG/mlQA68iq1eCgEMoew1rnuVG6mA2x2U34PYiOs=",
		"Confirmation/P2HTdbwCtB30+Rkp4Y/anm+C5U50joHnnku9b+NM3LoD/DT1RkZJUbdDqNLQv+Pp+Ilx7ZvOX5zBzl8gseeRLu1w=",
		"Confirmation/r66IG4KnURCKQu08kDyqQ0ZaeGIGFpeK7QzjxsTzrnsD/BVkxRTRPx5+16fRHsq5bYkpZDJyVJaon0roLGsOBSmI=",
		"Confirmation/otwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGFJiUcD/jKSgdUKni0rsIDDutHlO1fiss+BiNd1vxSGxJL0u2e8=",
		"Confirmation/lk39x56NU0NzZhz9ZtdP7B4biUkatyNuS3UhYpDPK+sD/prNQTXAQjkTRhltOQuhU8XagwwWP0RfwJe6yrtI3aaY=",
		"Confirmation/l4KD1KCaNvlsIJQXRuPtTaZGqa6LT6e0/Doguvoade0D/D+xEPt5A44s0BD5u/fz1iiPFoCnOR52PefTFOehdkbU=",
		"Confirmation/HPCdo54Okp0CSry8sWk5e7c05+8KbgHxhU3rX+Qk/vcD/cPDqZ3S1mqVxRTQ1p7Gwg7cEc34Xz/fUsIpghGiJygg=",
		"Confirmation/Ud9mj4dOLJ8c4JyoYBfn4hdIMD/0HBsj4RxI7RdTnWgD/minVwOqyN3l4zy7A4dvJDQ5ZLUcM2NmNdAWhR5/NTDc=",
		"Confirmation/Ximg3KRqw6DVcBM7whVx9fVKZDEFUT/YQpsZSuG6nyoD/dK0ZnuwEmyeXqjQj5mri5f8ChTHOVgTgUKkOGjUfPyQ=",
		"Confirmation/ZxkHLWcvYfqgvob0V5Iew3wORgzw1wPQfcX1ZhpFATMD/r0Nylw9Bd+eol1+4UWwWD8SBchPbjtnLYJx1zX1htEo=",
		"Confirmation/IpwYPBkzqRZYXhg7twkZLbDmyNcJudc4O5k8aUmZRbAD/eszeUU8yAglf5TrE5U4L8SVqKOPqypt9RbVjworRBbk=",
		"Confirmation/Rc0b8Lz8GjRsQ08RzwBBb6YWlbkgLmg2Ohx4f0eE4K4D/jhddD9Kqk6rcSJAB/Jy88cwhozR43M1nL+VTyl34SEk=",
		"Confirmation/1ieMn3yHL4QPnZTZ/e2uk9sklXGPWAuMjyvsxqp2w7AD/aaMF2inM08M9FdFOHPfGKMnoqqEJ4MiXxDhY2J84cE8=",
		"Confirmation/FER0v9N80ga1Gs4FCrYZnsezltYY/eDhopmabz2fi3oD/TJ5e0/2ji9eZSYa78RIP2ZvDW/PxP685D3xZAqHkGHY=",
		"Confirmation/KRnCqHpJlPweQB4RxaScfo6p5l1sxARl/TUvLELsPT4D/mlbwi77z/XUw/LfzX8L67k0/0dAIDHAYicLd2RukYO0=",
		"Confirmation/Q9EGMwNtPUa4GRauRv8T1qay+tkHnW3zRAWQKWZ7LrQD/0J3tuOL9xxfZdFQ73YEktXkeoFY6sAJIcgzlyDl3BxQ=",
	}

	for i, expected := range expectedKeys {
		partner, _ := id.NewRandomID(prng, id.User)
		dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
		_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
		fp := auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)

		key := makeConfirmationKey(partner, fp)
		if expected != key {
			t.Errorf("Confirmation key does not match expected for partner "+
				"%s and fingerprint %v (%d).\nexpected: %q\nreceived: %q",
				partner, fp, i, expected, key)
		}

		// fmt.Printf("\"%s\",\n", key)
	}
}
