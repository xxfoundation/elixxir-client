////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package edge

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Tests that newPreimages returns the expected new Preimages.
func Test_newPreimages(t *testing.T) {
	identity := id.NewIdFromString("identity", id.User, t)
	expected := Preimages{{
		Data:   identity.Bytes(),
		Type:   "default",
		Source: identity.Bytes(),
	}}

	received := newPreimages(identity)

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("New Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that Preimages.add adds the expected Preimage to the list.
func TestPreimages_add(t *testing.T) {
	identity0 := id.NewIdFromString("identity0", id.User, t)
	identity1 := id.NewIdFromString("identity1", id.User, t)
	expected := Preimages{
		{identity0.Bytes(), "default", identity0.Bytes()},
		{identity0.Bytes(), "group", identity0.Bytes()},
		{identity1.Bytes(), "default", identity1.Bytes()},
	}

	pis := newPreimages(identity0)
	pis = pis.add(Preimage{identity0.Bytes(), "group", identity0.Bytes()})
	pis = pis.add(Preimage{identity1.Bytes(), "default", identity1.Bytes()})

	if !reflect.DeepEqual(expected, pis) {
		t.Errorf("Failed to add expected Preimages."+
			"\nexpected: %+v\nreceived: %+v", expected, pis)
	}
}

// Tests that Preimages.remove removes all the correct Preimage from the list.
func TestPreimages_remove(t *testing.T) {
	var pis Preimages
	var identities [][]byte

	// Add 10 Preimage to the list
	for i := 0; i < 10; i++ {
		identity := id.NewIdFromUInt(uint64(i), id.User, t)
		pisType := "default"
		if i%2 == 0 {
			pisType = "group"
		}

		pis = pis.add(Preimage{identity.Bytes(), pisType, identity.Bytes()})
		identities = append(identities, identity.Bytes())
	}

	// Remove each Preimage, check if the length of the list has changed, and
	// check that the correct Preimage was removed
	for i, identity := range identities {
		pis = pis.remove(identity)

		if len(pis) != len(identities)-(i+1) {
			t.Errorf("Length of Preimages incorrect after removing %d Premiages."+
				"\nexpected: %d\nreceived: %d", i, len(identities)-(i+1),
				len(pis))
		}

		// Check if the correct Preimage was deleted
		for _, pimg := range pis {
			if bytes.Equal(pimg.Data, identity) {
				t.Errorf("Failed to delete Preimage #%d: %+v", i, pimg)
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that the Preimages loaded via loadPreimages matches the original saved
// to storage.
func Test_loadPreimages(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	identity := id.NewIdFromString("identity", id.User, t)
	pis := Preimages{
		{[]byte("identity0"), "default", []byte("identity0")},
		{[]byte("identity0"), "group", []byte("identity0")},
		{[]byte("identity1"), "default", []byte("identity1")},
	}

	err := pis.save(kv, identity)
	if err != nil {
		t.Errorf("Failed to save Preimages to storage: %+v", err)
	}

	loaded, err := loadPreimages(kv, identity)
	if err != nil {
		t.Errorf("loadPreimages returned an error: %+v", err)
	}

	if !reflect.DeepEqual(pis, loaded) {
		t.Errorf("Loaded Preimages do not match original."+
			"\nexpected: %+v\nreceived: %+v", pis, loaded)
	}
}

// Tests that the data saved to storage via Preimages.save can be loaded and
// unmarshalled and that it matches the original.
func TestPreimages_save(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	identity := id.NewIdFromString("identity", id.User, t)
	pis := Preimages{
		{[]byte("identity0"), "default", []byte("identity0")},
		{[]byte("identity0"), "group", []byte("identity0")},
		{[]byte("identity1"), "default", []byte("identity1")},
	}

	err := pis.save(kv, identity)
	if err != nil {
		t.Errorf("save returned an error: %+v", err)
	}

	obj, err := kv.Get(preimagesKey(identity), preimageStoreVersion)
	if err != nil {
		t.Errorf("Failed to load Preimages from storage: %+v", err)
	}

	var loaded Preimages
	err = json.Unmarshal(obj.Data, &loaded)
	if err != nil {
		t.Errorf("Failed to unmarshal Preimages loaded from storage: %+v", err)
	}

	if !reflect.DeepEqual(pis, loaded) {
		t.Errorf("Loaded Preimages do not match original."+
			"\nexpected: %+v\nreceived: %+v", pis, loaded)
	}
}

// Tests that Preimages.delete deletes the Preimages saved to storage by
// attempting to load them.
func TestPreimages_delete(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	identity := id.NewIdFromString("identity", id.User, t)
	pis := Preimages{
		{[]byte("identity0"), "default", []byte("identity0")},
		{[]byte("identity0"), "group", []byte("identity0")},
		{[]byte("identity1"), "default", []byte("identity1")},
	}

	err := pis.save(kv, identity)
	if err != nil {
		t.Errorf("Failed to save Preimages to storage: %+v", err)
	}

	err = pis.delete(kv, identity)
	if err != nil {
		t.Errorf("delete returned an error: %+v", err)
	}

	loaded, err := loadPreimages(kv, identity)
	if err == nil {
		t.Errorf("loadPreimages loaded a Preimages from storage when it "+
			"should have been deleted: %+v", loaded)
	}
}

// Consistency test: tests that preimagesKey returned the expected output for a
// set input.
func Test_preimagesKey(t *testing.T) {
	expectedKeys := []string{
		"preimageStoreKey:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:ACOG8m/BAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:AEcN5N+CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:AGqU109DAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:AI4byb8EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:ALGivC7FAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:ANUprp6GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:APiwoQ5HAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:ARw3k34IAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
		"preimageStoreKey:AT++he3JAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
	}

	for i, expected := range expectedKeys {
		identity := id.NewIdFromUInt(uint64(i)*1e16, id.User, t)
		key := preimagesKey(identity)
		if key != expected {
			t.Errorf("Key #%d does not match expected."+
				"\nexpected: %q\nreceived: %q", i, expected, key)
		}
	}
}
