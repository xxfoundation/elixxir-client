////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receptionID

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestIdentity_store_loadIdentity(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	r := Identity{
		EphemeralIdentity: EphemeralIdentity{
			EphId:  ephemeral.Id{},
			Source: &id.Permissioning,
		},
		AddressSize: 15,
		End:         netTime.Now().Round(0),
		ExtraChecks: 12,
		StartValid:  netTime.Now().Round(0),
		EndValid:    netTime.Now().Round(0),
		Ephemeral:   false,
	}

	err := r.store(kv, "")
	if err != nil {
		t.Errorf("Failed to store: %+v", err)
	}

	rLoad, err := loadIdentity(kv, "")
	if err != nil {
		t.Errorf("Failed to load: %+v", err)
	}

	if !r.Equal(rLoad) {
		t.Errorf("Registrations are not the same.\nsaved:  %+v\nloaded: %+v",
			r, rLoad)
	}
}

func TestIdentity_delete(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	r := Identity{
		EphemeralIdentity: EphemeralIdentity{
			EphId:  ephemeral.Id{},
			Source: &id.Permissioning,
		},
		AddressSize: 15,
		End:         netTime.Now().Round(0),
		ExtraChecks: 12,
		StartValid:  netTime.Now().Round(0),
		EndValid:    netTime.Now().Round(0),
		Ephemeral:   false,
	}

	err := r.store(kv, "")
	if err != nil {
		t.Errorf("Failed to store: %s", err)
	}

	err = r.delete(kv, "")
	if err != nil {
		t.Errorf("Failed to delete: %s", err)
	}

	_, err = loadIdentity(kv, "")
	if err == nil {
		t.Error("Load after delete succeeded.")
	}
}

func TestIdentity_String(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	received, _ := generateFakeIdentity(rng, 15, timestamp)
	expected := "-1763 U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID"

	s := received.String()
	if s != expected {
		t.Errorf("String did not return the correct value."+
			"\nexpected: %s\nreceived: %s", expected, s)
	}
}

func TestIdentity_Equal(t *testing.T) {
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	a, _ := generateFakeIdentity(rand.New(rand.NewSource(42)), 15, timestamp)
	b, _ := generateFakeIdentity(rand.New(rand.NewSource(42)), 15, timestamp)
	c, _ := generateFakeIdentity(rand.New(rand.NewSource(42)), 15, netTime.Now())

	if !a.Identity.Equal(b.Identity) {
		t.Errorf("Equal() found two equal identities as unequal."+
			"\na: %s\nb: %s", a, b)
	}

	if a.Identity.Equal(c.Identity) {
		t.Errorf("Equal() found two unequal identities as equal."+
			"\na: %s\nc: %s", a, c)
	}
}

// TestIdentity_store_loadProcessNext tests that when an Identity is stored,
// Identity.ProcessNext is loaded. This test is exhaustive by making a reasonably
// long Identity.ProcessNext linked list, and checking if all Identity's are loaded.
func TestIdentity_store_loadProcessNext(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	const numTests = 10

	// Construct the first identity, which will be stored
	ephId := ephemeral.Id{}
	copy(ephId[:], []byte{0})
	first := &Identity{
		EphemeralIdentity: EphemeralIdentity{
			EphId:  ephId,
			Source: &id.Permissioning,
		},
		AddressSize: 0,
		End:         netTime.Now().Round(0),
		ExtraChecks: 0,
		StartValid:  netTime.Now().Round(0),
		EndValid:    netTime.Now().Round(0),
		Ephemeral:   false,
	}

	// Build the linked list with unique identities. Use temp as a temporary
	// head, such that the previous node in the linked list can have its next
	// set with the next identity, and then set temp to the next identity for
	// the next iteration
	temp := first
	for i := 1; i < numTests; i++ {
		// Ensure uniqueness of every identity by having the ephemeral ID
		// contain the number of the iteration (ie the value of i)
		ephId = ephemeral.Id{}
		copy(ephId[:], []byte{byte(i)})

		next := &Identity{
			EphemeralIdentity: EphemeralIdentity{
				EphId:  ephId,
				Source: &id.Permissioning,
			},
			AddressSize: 25,
			End:         netTime.Now().Round(0),
			ExtraChecks: 16,
			StartValid:  netTime.Now().Round(0),
			EndValid:    netTime.Now().Round(0),
			Ephemeral:   false,
		}

		temp.ProcessNext = next
		temp = next
	}

	// Save the first identity. This should be the head of
	// the created linked list, and thus all nodes (Identity's) should be saved.
	err := first.store(kv, "")
	if err != nil {
		t.Errorf("Failed to store: %s", err)
	}

	// Load the identity
	loadedIdentity, err := loadIdentity(kv, "")
	if err != nil {
		t.Errorf("Failed to load: %+v", err)
	}

	// Smoke test: Check that there is a next element in the linked list
	if loadedIdentity.ProcessNext == nil {
		t.Fatalf("Failed to load processNext for identity!")
	}

	// Serialize the linked list, such that we can iterate over
	// it to check for expected values later
	temp = &loadedIdentity
	serializedList := make([]*Identity, 0)
	for temp != nil {
		serializedList = append(serializedList, temp)
		temp = temp.ProcessNext
	}

	// Smoke test: Check that the number of loaded identities is of the
	// expected quantity
	if len(serializedList) != numTests {
		t.Fatalf("Bad number of identities from Identity.ProcessNext."+
			"\nExpected: %d"+
			"\nReceived: %d", numTests, len(serializedList))
	}

	// Go through every identity to make sure it has the expected value
	for i := 0; i < len(serializedList); i++ {

		// The loaded identity should have an ephemeral ID
		// contain the number of the iteration (ie the value of i)
		ephId = ephemeral.Id{}
		copy(ephId[:], []byte{byte(i)})

		received := serializedList[i]

		if !reflect.DeepEqual(ephId, received.EphId) {
			t.Errorf("Identity #%d loaded is not expected."+
				"\nExpected: %+v"+
				"\nReceived: %+v", i, ephId, received.EphId)
		}

	}

}
