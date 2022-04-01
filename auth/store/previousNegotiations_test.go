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
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// Tests the four possible cases of Store.AddIfNew:
//  1. If the partner does not exist, add partner with the new fingerprint.
//		Returns newFingerprint = true, latest = true.
//	2. If the partner exists and the fingerprint does not, add the fingerprint.
//		Returns newFingerprint = true, latest = true.
//	3. If the partner exists and the fingerprint exists, do nothing.
//		Return newFingerprint = false, latest = false.
//	4. If the partner exists, the fingerprint exists, and the fingerprint is the
//     latest, do nothing.
//      Return newFingerprint = false, latest = true.
func TestStore_AddIfNew(t *testing.T) {
	s := &Store{
		kv:                   versioned.NewKV(make(ekv.Memstore)),
		previousNegotiations: make(map[id.ID]struct{}),
	}
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	newPartner := func() *id.ID {
		partner, _ := id.NewRandomID(prng, id.User)
		return partner
	}
	newFps := func() []byte {
		dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
		_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
		return auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
	}

	type test struct {
		name string

		addPartner bool     // If true, partner is added to list first
		addFp      bool     // If true, fingerprint is added to list first
		latestFp   bool     // If true, fingerprint is added as latest
		otherFps   [][]byte // Other sentByFingerprints to add first

		// Inputs
		partner *id.ID
		fp      []byte

		// Expected values
		newFingerprint bool
		latest         bool
	}

	tests := []test{
		{
			name:           "Case 1: partner does not exist",
			addPartner:     false,
			addFp:          false,
			latestFp:       false,
			partner:        newPartner(),
			fp:             newFps(),
			newFingerprint: true,
			latest:         true,
		}, {
			name:           "Case 2: partner exists, fingerprint does not",
			addPartner:     true,
			addFp:          false,
			latestFp:       false,
			otherFps:       [][]byte{newFps(), newFps(), newFps()},
			partner:        newPartner(),
			fp:             newFps(),
			newFingerprint: true,
			latest:         true,
		}, {
			name:           "Case 3: partner and fingerprint exist",
			addPartner:     true,
			addFp:          true,
			latestFp:       false,
			otherFps:       [][]byte{newFps(), newFps(), newFps()},
			partner:        newPartner(),
			fp:             newFps(),
			newFingerprint: false,
			latest:         false,
		}, {
			name:           "Case 4: partner and fingerprint exist, fingerprint latest",
			addPartner:     true,
			addFp:          true,
			latestFp:       true,
			otherFps:       [][]byte{newFps(), newFps(), newFps()},
			partner:        newPartner(),
			fp:             newFps(),
			newFingerprint: false,
			latest:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.addPartner {
				s.previousNegotiations[*tt.partner] = struct{}{}
				err := s.savePreviousNegotiations()
				if err != nil {
					t.Errorf(
						"savePreviousNegotiations returned an error: %+v", err)
				}

				var fps [][]byte
				if tt.addFp {
					fps, _ = s.loadNegotiationFingerprints(tt.partner)

					for _, fp := range tt.otherFps {
						fps = append(fps, fp)
					}

					if tt.latestFp {
						fps = append(fps, tt.fp)
					} else {
						fps = append([][]byte{tt.fp}, fps...)
					}
				}
				err = s.saveNegotiationFingerprints(tt.partner, fps...)
				if err != nil {
					t.Errorf("saveNegotiationFingerprints returned an "+
						"error: %+v", err)
				}
			}

			newFingerprint, latest := s.AddIfNew(tt.partner, tt.fp)

			if newFingerprint != tt.newFingerprint {
				t.Errorf("Unexpected value for newFingerprint."+
					"\nexpected: %t\nreceived: %t",
					tt.newFingerprint, newFingerprint)
			}
			if latest != tt.latest {
				t.Errorf("Unexpected value for latest."+
					"\nexpected: %t\nreceived: %t", tt.latest, latest)
			}
		})
	}
}

// Tests that Store.deletePreviousNegotiationPartner deletes the partner from
// previousNegotiations in memory, previousNegotiations in storage, sentByFingerprints
// in storage, and any confirmations in storage.
func TestStore_deletePreviousNegotiationPartner(t *testing.T) {
	s := &Store{
		kv:                   versioned.NewKV(make(ekv.Memstore)),
		previousNegotiations: make(map[id.ID]struct{}),
	}
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	type values struct {
		partner *id.ID
		fps     [][]byte
	}

	testValues := make([]values, 16)

	for i := range testValues {
		partner, _ := id.NewRandomID(prng, id.User)
		s.previousNegotiations[*partner] = struct{}{}

		err := s.savePreviousNegotiations()
		if err != nil {
			t.Errorf("savePreviousNegotiations returned an error (%d): %+v",
				i, err)
		}

		// Generate sentByFingerprints
		fingerprints := make([][]byte, i+1)
		for j := range fingerprints {
			dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
			_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, prng)
			fingerprints[j] = auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
		}

		err = s.saveNegotiationFingerprints(partner, fingerprints...)
		if err != nil {
			t.Errorf("saveNegotiationFingerprints returned an error (%d): %+v",
				i, err)
		}

		testValues[i] = values{partner, fingerprints}

		// Generate confirmation
		confirmation := make([]byte, 32)
		prng.Read(confirmation)

		err = s.StoreConfirmation(partner, fingerprints[0], confirmation)
		if err != nil {
			t.Errorf("StoreConfirmation returned an error (%d): %+v", i, err)
		}
	}

	// Add partner that is not in list
	partner, _ := id.NewRandomID(prng, id.User)
	testValues = append(testValues, values{partner, [][]byte{}})

	for i, v := range testValues {
		err := s.deletePreviousNegotiationPartner(v.partner)
		if err != nil {
			t.Errorf("deletePreviousNegotiationPartner returned an error "+
				"(%d): %+v", i, err)
		}

		// Check previousNegotiations in memory
		_, exists := s.previousNegotiations[*v.partner]
		if exists {
			t.Errorf("Parter %s exists in previousNegotiations (%d).",
				v.partner, i)
		}

		// Check previousNegotiations in storage
		previousNegotiations, err := s.newOrLoadPreviousNegotiations()
		if err != nil {
			t.Errorf("newOrLoadPreviousNegotiations returned an error (%d): %+v",
				i, err)
		}
		_, exists = previousNegotiations[*v.partner]
		if exists {
			t.Errorf("Parter %s exists in previousNegotiations in storage (%d).",
				v.partner, i)
		}

		// Check negotiation sentByFingerprints in storage
		fps, err := s.loadNegotiationFingerprints(v.partner)
		if err == nil || fps != nil {
			t.Errorf("Loaded sentByFingerprints for partner %s (%d): %v",
				v.partner, i, fps)
		}

		// Check all possible confirmations in storage
		for j, fp := range v.fps {
			confirmation, err := s.LoadConfirmation(v.partner, fp)
			if err == nil || fps != nil {
				t.Errorf("Loaded confirmation for partner %s and "+
					"fingerprint %v (%d, %d): %v",
					v.partner, fp, i, j, confirmation)
			}
		}
	}
}

// Tests that Store.previousNegotiations can be saved and loaded from storage
// via Store.savePreviousNegotiations andStore.newOrLoadPreviousNegotiations.
func TestStore_savePreviousNegotiations_newOrLoadPreviousNegotiations(t *testing.T) {
	s := &Store{
		kv:                   versioned.NewKV(make(ekv.Memstore)),
		previousNegotiations: make(map[id.ID]struct{}),
	}
	prng := rand.New(rand.NewSource(42))
	expected := make(map[id.ID]struct{})

	for i := 0; i < 16; i++ {
		partner, _ := id.NewRandomID(prng, id.User)
		s.previousNegotiations[*partner] = struct{}{}
		expected[*partner] = struct{}{}

		err := s.savePreviousNegotiations()
		if err != nil {
			t.Errorf("savePreviousNegotiations returned an error (%d): %+v",
				i, err)
		}

		s.previousNegotiations, err = s.newOrLoadPreviousNegotiations()
		if err != nil {
			t.Errorf("newOrLoadPreviousNegotiations returned an error (%d): %+v",
				i, err)
		}

		if !reflect.DeepEqual(expected, s.previousNegotiations) {
			t.Errorf("Loaded previousNegotiations does not match expected (%d)."+
				"\nexpected: %v\nreceived: %v", i, expected, s.previousNegotiations)
		}
	}
}

// Tests that Store.newOrLoadPreviousNegotiations returns blank negotiations if
// they do not exist.
func TestStore_newOrLoadPreviousNegotiations_noNegotiations(t *testing.T) {
	s := &Store{
		kv:                   versioned.NewKV(make(ekv.Memstore)),
		previousNegotiations: make(map[id.ID]struct{}),
	}
	expected := make(map[id.ID]struct{})

	blankNegotations, err := s.newOrLoadPreviousNegotiations()
	if err != nil {
		t.Errorf("newOrLoadPreviousNegotiations returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, blankNegotations) {
		t.Errorf("Loaded previousNegotiations does not match expected."+
			"\nexpected: %v\nreceived: %v", expected, blankNegotations)
	}
}

// Tests that a list of partner IDs that is marshalled and unmarshalled via
// marshalPreviousNegotiations and unmarshalPreviousNegotiations matches the
// original list
func Test_marshalPreviousNegotiations_unmarshalPreviousNegotiations(t *testing.T) {
	prng := rand.New(rand.NewSource(42))

	// Create original map of partner IDs
	originalPartners := make(map[id.ID]struct{}, 50)
	for i := 0; i < 50; i++ {
		partner, _ := id.NewRandomID(prng, id.User)
		originalPartners[*partner] = struct{}{}
	}

	// Marshal and unmarshal the partner list
	marshalledPartners := marshalPreviousNegotiations(originalPartners)
	unmarshalledPartners := unmarshalPreviousNegotiations(marshalledPartners)

	// Check that the original matches the unmarshalled
	if !reflect.DeepEqual(originalPartners, unmarshalledPartners) {
		t.Errorf("Unmarshalled partner list does not match original."+
			"\nexpected: %v\nreceived: %v",
			originalPartners, unmarshalledPartners)
	}
}

// Tests that a list of sentByFingerprints for different partners can be saved and
// loaded from storage via Store.saveNegotiationFingerprints and
// Store.loadNegotiationFingerprints.
func TestStore_saveNegotiationFingerprints_loadNegotiationFingerprints(t *testing.T) {
	s := &Store{kv: versioned.NewKV(make(ekv.Memstore))}
	rng := csprng.NewSystemRNG()
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	testValues := make([]struct {
		partner *id.ID
		fps     [][]byte
	}, 10)

	for i := range testValues {
		partner, _ := id.NewRandomID(rng, id.User)

		// Generate original sentByFingerprints to marshal
		originalFps := make([][]byte, 50)
		for j := range originalFps {
			dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
			_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
			originalFps[j] = auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
		}

		testValues[i] = struct {
			partner *id.ID
			fps     [][]byte
		}{partner: partner, fps: originalFps}

		err := s.saveNegotiationFingerprints(partner, originalFps...)
		if err != nil {
			t.Errorf("saveNegotiationFingerprints returned an error (%d): %+v",
				i, err)
		}
	}

	for i, val := range testValues {
		loadedFps, err := s.loadNegotiationFingerprints(val.partner)
		if err != nil {
			t.Errorf("loadNegotiationFingerprints returned an error (%d): %+v",
				i, err)
		}

		if !reflect.DeepEqual(val.fps, loadedFps) {
			t.Errorf("Loaded sentByFingerprints do not match original (%d)."+
				"\nexpected: %v\nreceived: %v", i, val.fps, loadedFps)
		}
	}
}

// Tests that a list of sentByFingerprints that is marshalled and unmarshalled via
// marshalNegotiationFingerprints and unmarshalNegotiationFingerprints matches
// the original list
func Test_marshalNegotiationFingerprints_unmarshalNegotiationFingerprints(t *testing.T) {
	rng := csprng.NewSystemRNG()
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))

	// Generate original sentByFingerprints to marshal
	originalFps := make([][]byte, 50)
	for i := range originalFps {
		dhPubKey := diffieHellman.GeneratePublicKey(grp.NewInt(42), grp)
		_, sidhPubkey := utility.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
		originalFps[i] = auth.CreateNegotiationFingerprint(dhPubKey, sidhPubkey)
	}

	// Marshal and unmarshal the fingerprint list
	marshalledFingerprints := marshalNegotiationFingerprints(originalFps...)
	unmarshalledFps := unmarshalNegotiationFingerprints(marshalledFingerprints)

	// Check that the original matches the unmarshalled
	if !reflect.DeepEqual(originalFps, unmarshalledFps) {
		t.Errorf("Unmarshalled sentByFingerprints do not match original."+
			"\nexpected: %v\nreceived: %v", originalFps, unmarshalledFps)
	}
}

// Consistency test of makeOldNegotiationFingerprintsKey.
func Test_makeNegotiationFingerprintsKey_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedKeys := []string{
		"NegotiationFingerprints/U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
		"NegotiationFingerprints/15tNdkKbYXoMn58NO6VbDMDWFEyIhTWEGsvgcJsHWAgD",
		"NegotiationFingerprints/YdN1vAK0HfT5GSnhj9qeb4LlTnSOgeeeS71v40zcuoQD",
		"NegotiationFingerprints/6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44bC68D",
		"NegotiationFingerprints/iBuCp1EQikLtPJA8qkNGWnhiBhaXiu0M48bE8657w+AD",
		"NegotiationFingerprints/W1cS/v2+DBAoh+EA2s0tiF9pLLYH2gChHBxwceeWotwD",
		"NegotiationFingerprints/wlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGFJiUf980QD",
		"NegotiationFingerprints/DtTBFgI/qONXa2/tJ/+JdLrAyv2a0FaSsTYZ5ziWTf0D",
		"NegotiationFingerprints/no1TQ3NmHP1m10/sHhuJSRq3I25LdSFikM8r60LDyicD",
		"NegotiationFingerprints/hWDxqsBnzqbov0bUqytGgEAsX7KCDohdMmDx3peCg9QD",
		"NegotiationFingerprints/mjb5bCCUF0bj7U2mRqmui0+ntPw6ILr6GnXtMnqGuLAD",
		"NegotiationFingerprints/mvHP0rO1EhnqeVM6v0SNLEedMmB1M5BZFMjMHPCdo54D",
		"NegotiationFingerprints/kp0CSry8sWk5e7c05+8KbgHxhU3rX+Qk/vesIQiR9ZcD",
		"NegotiationFingerprints/KSqiuKoEfGHNszNz6+csJ6CYwCGX2ua3MsNR32aPh04D",
		"NegotiationFingerprints/nxzgnKhgF+fiF0gwP/QcGyPhHEjtF1OdaF928qeYvGQD",
		"NegotiationFingerprints/Dl2yhksq08Js5jgjQnZaE9aW5S33YPbDRl4poNykasMD",
	}

	for i, expected := range expectedKeys {
		partner, _ := id.NewRandomID(prng, id.User)

		key := makeOldNegotiationFingerprintsKey(partner)
		if expected != key {
			t.Errorf("Negotiation sentByFingerprints key does not match expected "+
				"for partner %s (%d).\nexpected: %q\nreceived: %q", partner, i,
				expected, key)
		}

		// fmt.Printf("\"%s\",\n", key)
	}
}
