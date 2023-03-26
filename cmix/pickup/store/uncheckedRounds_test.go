////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"testing"
)

// Unit test
func TestNewUncheckedStore(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	testStore := &UncheckedRoundStore{
		list: make(map[roundIdentity]UncheckedRound),
		kv:   kv,
	}

	store, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("NewUncheckedStore returned an error: %+v", err)
	}

	// Compare manually created object with UncheckedRoundStore
	if !reflect.DeepEqual(testStore, store) {
		t.Fatalf("NewUncheckedStore returned incorrect Store."+
			"\nexpected: %+v\nreceived: %+v", testStore, store)
	}

	rid := id.Round(1)
	roundInfo := &pb.RoundInfo{ID: uint64(rid)}
	recipient := id.NewIdFromString("recipientID", id.User, t)
	ephID, _, _, _ := ephemeral.GetId(
		recipient, id.ArrIDLen, netTime.Now().UnixNano())
	uncheckedRound := UncheckedRound{
		Info:      roundInfo,
		LastCheck: netTime.Now(),
		NumChecks: 0,
		Identity:  Identity{Source: recipient, EpdId: ephID},
	}

	ri := newRoundIdentity(rid, recipient, ephID)
	store.list[ri] = uncheckedRound
	if err = store.save(); err != nil {
		t.Fatalf("Could not save store: %+v", err)
	}

	// Test if round list data matches
	expectedRoundData, err := store.marshal()
	if err != nil {
		t.Fatalf("Failed to marshal UncheckedRoundStore: %+v", err)
	}
	roundData, err := store.kv.Get(uncheckedRoundPrefix+uncheckedRoundKey, uncheckedRoundVersion)
	if err != nil {
		t.Fatalf("Failed to get round list from storage: %+v", err)
	}

	if !bytes.Equal(expectedRoundData, roundData) {
		t.Fatalf("Data from store unexpected.\nexpected %+v\nreceived: %v",
			expectedRoundData, roundData)
	}
}

// Unit test.
func TestLoadUncheckedStore(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	// Add round to store
	rid := id.Round(0)
	roundInfo := &pb.RoundInfo{ID: uint64(rid)}
	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromBytes([]byte("Sauron"), t)
	err = testStore.AddRound(id.Round(roundInfo.ID), roundInfo, source, ephId)
	if err != nil {
		t.Fatalf("Failed to add round to store: %+v", err)
	}

	// Load store
	loadedStore, err := LoadUncheckedStore(kv)
	if err != nil {
		t.Fatalf("LoadUncheckedStore returned an error: %+v", err)
	}

	// Check if round is in loaded store
	ri := newRoundIdentity(rid, source, ephId)
	rnd, exists := loadedStore.list[ri]
	if !exists {
		t.Fatalf("Added round %d not found in loaded store.", rid)
	}

	// Check if set values are expected
	if !bytes.Equal(rnd.EpdId[:], ephId[:]) || !source.Cmp(rnd.Source) {
		t.Fatalf("Values in loaded round %d are not expected."+
			"\nexpected address: %d\nreceived address: %d"+
			"\nexpected source: %s\nreceived source: %s",
			rid, ephId.Int64(), rnd.EpdId.Int64(), source, rnd.Source)
	}
}

// Unit test.
func TestUncheckedRoundStore_AddRound(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	// Add round to store
	rid := id.Round(0)
	roundInfo := &pb.RoundInfo{ID: uint64(rid)}
	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromBytes([]byte("Sauron"), t)
	err = testStore.AddRound(id.Round(roundInfo.ID), roundInfo, source, ephId)
	if err != nil {
		t.Fatalf("AddRound returned an error: %+v", err)
	}

	ri := newRoundIdentity(rid, source, ephId)
	if _, exists := testStore.list[ri]; !exists {
		t.Error("Could not find added round in list")
	}
}

// Unit test.
func TestUncheckedRoundStore_GetRound(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	// Add round to store
	rid := id.Round(0)
	roundInfo := &pb.RoundInfo{ID: uint64(rid)}
	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromString("Sauron", id.User, t)
	err = testStore.AddRound(id.Round(roundInfo.ID), roundInfo, source, ephId)
	if err != nil {
		t.Fatalf("Failed to add round to store: %+v", err)
	}

	// Retrieve round that was inserted
	retrievedRound, exists := testStore.GetRound(rid, source, ephId)
	if !exists {
		t.Fatalf("GetRound error: " +
			"Could not get round from store")
	}

	if !bytes.Equal(retrievedRound.EpdId[:], ephId[:]) {
		t.Fatalf("Retrieved address ID for round %d does not match expected."+
			"\nexpected: %d\nreceived: %d", rid, ephId.Int64(),
			retrievedRound.EpdId.Int64())
	}

	if !source.Cmp(retrievedRound.Source) {
		t.Fatalf("Retrieved source ID for round %d does not match expected."+
			"\nexpected: %s\nreceived: %s", rid, source, retrievedRound.Source)
	}

	// Try to pull unknown round from store
	unknownRound := id.Round(1)
	unknownRecipient := id.NewIdFromString("invalidID", id.User, t)
	unknownEphId := ephemeral.Id{11, 12, 13, 14, 15, 16, 17, 18}
	_, exists = testStore.GetRound(unknownRound, unknownRecipient, unknownEphId)
	if exists {
		t.Fatalf("Should not find unknown round %d in store.", unknownRound)
	}
}

// Tests that two identifies for the same round can be retrieved separately.
func TestUncheckedRoundStore_GetRound_TwoIDs(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	s, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	// Add round to store for the same round but two sources
	rid := id.Round(0)
	roundInfo := &pb.RoundInfo{ID: uint64(rid)}
	ephId1 := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source1 := id.NewIdFromString("Sauron", id.User, t)
	err = s.AddRound(rid, roundInfo, source1, ephId1)
	if err != nil {
		t.Fatalf("Failed to add round for source 1 to store: %+v", err)
	}

	ephId2 := ephemeral.Id{11, 12, 13, 14, 15, 16, 17, 18}
	source2 := id.NewIdFromString("Sauron2", id.User, t)
	err = s.AddRound(rid, roundInfo, source2, ephId2)
	if err != nil {
		t.Fatalf("Failed to add round for source 2 to store: %+v", err)
	}

	// Increment each a set number of times
	incNum1, incNum2 := 3, 13
	for i := 0; i < incNum1; i++ {
		if err = s.IncrementCheck(rid, source1, ephId1); err != nil {
			t.Errorf("Failed to incremement for source 1 (%d): %+v", i, err)
		}
	}
	for i := 0; i < incNum2; i++ {
		if err = s.IncrementCheck(rid, source2, ephId2); err != nil {
			t.Errorf("Failed to incremement for source 2 (%d): %+v", i, err)
		}
	}

	// Retrieve round that was inserted
	retrievedRound, exists := s.GetRound(rid, source1, ephId1)
	if !exists {
		t.Fatalf("Could not get round for source 1 from store")
	}

	if !bytes.Equal(retrievedRound.EpdId[:], ephId1[:]) {
		t.Fatalf("Retrieved address ID for round %d does not match expected."+
			"\nexpected: %d\nreceived: %d", rid, ephId1.Int64(),
			retrievedRound.EpdId.Int64())
	}

	if !source1.Cmp(retrievedRound.Source) {
		t.Fatalf("Retrieved source ID for round %d does not match expected."+
			"\nexpected: %s\nreceived: %s", rid, source1, retrievedRound.Source)
	}

	retrievedRound, exists = s.GetRound(rid, source2, ephId2)
	if !exists {
		t.Fatalf("Could not get round for source 2 from store")
	}

	if !bytes.Equal(retrievedRound.EpdId[:], ephId2[:]) {
		t.Fatalf("Retrieved address ID for round %d does not match expected."+
			"\nexpected: %d\nreceived: %d", rid, ephId2.Int64(),
			retrievedRound.EpdId.Int64())
	}

	if !source2.Cmp(retrievedRound.Source) {
		t.Fatalf("Retrieved source ID for round %d does not match expected."+
			"\nexpected: %s\nreceived: %s", rid, source2, retrievedRound.Source)
	}
}

// Unit test
func TestUncheckedRoundStore_GetList(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	// Add rounds to store
	numRounds := 10
	for i := 0; i < numRounds; i++ {
		rid := id.Round(i)
		roundInfo := &pb.RoundInfo{ID: uint64(rid)}
		ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
		source := id.NewIdFromUInt(uint64(i), id.User, t)
		err = testStore.AddRound(id.Round(roundInfo.ID), roundInfo, source, ephId)
		if err != nil {
			t.Errorf("Failed to add round to store: %+v", err)
		}
	}

	// Retrieve list
	retrievedList := testStore.GetList(t)
	if len(retrievedList) != numRounds {
		t.Errorf("List returned is not of expected size."+
			"\nexpected: %d\nreceived: %d", numRounds, len(retrievedList))
	}

	for i := 0; i < numRounds; i++ {
		rid := id.Round(i)
		ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
		source := id.NewIdFromUInt(uint64(i), id.User, t)
		ri := newRoundIdentity(rid, source, ephId)
		if _, exists := retrievedList[ri]; !exists {
			t.Errorf("Retrieved list does not contain expected round %d.", rid)
		}
	}

}

// Unit test.
func TestUncheckedRoundStore_IncrementCheck(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	// Add rounds to store
	numRounds := 10
	for i := 0; i < numRounds; i++ {
		roundInfo := &pb.RoundInfo{ID: uint64(i)}
		ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
		source := id.NewIdFromUInt(uint64(i), id.User, t)
		err = testStore.AddRound(id.Round(roundInfo.ID), roundInfo, source, ephId)
		if err != nil {
			t.Fatalf("Failed to add round to store: %+v", err)
		}
	}

	testRound := id.Round(3)
	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromUInt(uint64(testRound), id.User, t)
	numChecks := 4
	for i := 0; i < numChecks; i++ {
		err = testStore.IncrementCheck(testRound, source, ephId)
		if err != nil {
			t.Errorf("Could not increment check for round %d: %v", testRound, err)
		}
	}

	rnd, _ := testStore.GetRound(testRound, source, ephId)
	if rnd.NumChecks != uint64(numChecks) {
		t.Errorf("Round %d did not have expected number of checks."+
			"\nexpected: %v\nreceived: %v", testRound, numChecks, rnd.NumChecks)
	}

	// Error path: check unknown round can not be incremented
	unknownRound := id.Round(numRounds + 5)
	err = testStore.IncrementCheck(unknownRound, source, ephId)
	if err == nil {
		t.Errorf("Should not find round %d which was not added to store",
			unknownRound)
	}

	// Reach max checks, ensure that round is removed
	maxRound := id.Round(7)
	source = id.NewIdFromUInt(uint64(maxRound), id.User, t)
	for i := 0; i < maxChecks+1; i++ {
		err = testStore.IncrementCheck(maxRound, source, ephId)
		if err != nil {
			t.Errorf("Could not increment check for round %d: %v", maxRound, err)
		}
	}
}

// Unit test
func TestUncheckedRoundStore_Remove(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	// Add rounds to store
	numRounds := 10
	for i := 0; i < numRounds; i++ {
		roundInfo := &pb.RoundInfo{ID: uint64(i)}
		ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
		source := id.NewIdFromUInt(uint64(i), id.User, t)
		err = testStore.AddRound(id.Round(roundInfo.ID), roundInfo, source, ephId)
		if err != nil {
			t.Fatalf("Failed to add round to store: %+v", err)
		}
	}

	// Remove round from storage
	removedRound := id.Round(1)
	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromUInt(uint64(removedRound), id.User, t)
	err = testStore.Remove(removedRound, source, ephId)
	if err != nil {
		t.Errorf("Could not have removed round %d from storage: %+v",
			removedRound, err)
	}

	// Check that round was removed
	_, exists := testStore.GetRound(removedRound, source, ephId)
	if exists {
		t.Errorf("Round %d expected to be removed from storage", removedRound)
	}

	// Error path: attempt to remove unknown round
	unknownRound := id.Round(numRounds + 5)
	err = testStore.Remove(unknownRound, source, ephId)
	if err == nil {
		t.Errorf("Should not have removed round %d which is not in storage",
			unknownRound)
	}
}
