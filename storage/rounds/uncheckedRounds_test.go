///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
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
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore := &UncheckedRoundStore{
		list: make(map[id.Round]UncheckedRound),
		kv:   kv.Prefix(uncheckedRoundPrefix),
	}

	store, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("NewUncheckedStore error: "+
			"Could not create unchecked stor: %v", err)
	}

	// Compare manually created object with NewUnknownRoundsStore
	if !reflect.DeepEqual(testStore, store) {
		t.Fatalf("NewUncheckedStore error: "+
			"Returned incorrect Store."+
			"\n\texpected: %+v\n\treceived: %+v", testStore, store)
	}

	rid := id.Round(1)
	roundInfo := &pb.RoundInfo{
		ID: uint64(rid),
	}
	uncheckedRound := UncheckedRound{
		Info:      roundInfo,
		LastCheck: netTime.Now(),
		NumChecks: 0,
	}

	store.list[rid] = uncheckedRound
	if err = store.save(); err != nil {
		t.Fatalf("NewUncheckedStore error: "+
			"Could not save store: %v", err)
	}

	// Test if round list data matches
	expectedRoundData, err := store.marshal()
	if err != nil {
		t.Fatalf("NewUncheckedStore error: "+
			"Could not marshal data: %v", err)
	}
	roundData, err := store.kv.Get(uncheckedRoundKey, uncheckedRoundVersion)
	if err != nil {
		t.Fatalf("NewUncheckedStore error: "+
			"Could not retrieve round list form storage: %v", err)
	}

	if !bytes.Equal(expectedRoundData, roundData.Data) {
		t.Fatalf("NewUncheckedStore error: "+
			"Data from store was not expected"+
			"\n\tExpected %v\n\tReceived: %v", expectedRoundData, roundData.Data)
	}

}

// Unit test
func TestLoadUncheckedStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("LoadUncheckedStore error: "+
			"Could not call constructor NewUncheckedStore: %v", err)
	}

	// Add round to store
	rid := id.Round(0)
	roundInfo := &pb.RoundInfo{
		ID: uint64(rid),
	}

	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromBytes([]byte("Sauron"), t)
	err = testStore.AddRound(roundInfo, ephId, source)
	if err != nil {
		t.Fatalf("LoadUncheckedStore error: "+
			"Could not add round to store: %v", err)
	}

	// Load store
	loadedStore, err := LoadUncheckedStore(kv)
	if err != nil {
		t.Fatalf("LoadUncheckedStore error: "+
			"Could not call LoadUncheckedStore: %v", err)
	}

	// Check if round is in loaded store
	rnd, exists := loadedStore.list[rid]
	if !exists {
		t.Fatalf("LoadUncheckedStore error: "+
			"Added round %d not found in loaded store", rid)
	}

	// Check if set values are expected
	if !bytes.Equal(rnd.EpdId[:], ephId[:]) ||
		!source.Cmp(rnd.Source) {
		t.Fatalf("LoadUncheckedStore error: "+
			"Values in loaded round %d are not expected."+
			"\n\tExpected ephemeral: %v"+
			"\n\tReceived ephemeral: %v"+
			"\n\tExpected source: %v"+
			"\n\tReceived source: %v", rid,
			ephId, rnd.EpdId,
			source, rnd.Source)
	}

}

// Unit test
func TestUncheckedRoundStore_AddRound(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("AddRound error: "+
			"Could not call constructor NewUncheckedStore: %v", err)
	}

	// Add round to store
	rid := id.Round(0)
	roundInfo := &pb.RoundInfo{
		ID: uint64(rid),
	}
	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromBytes([]byte("Sauron"), t)
	err = testStore.AddRound(roundInfo, ephId, source)
	if err != nil {
		t.Fatalf("AddRound error: "+
			"Could not add round to store: %v", err)
	}

	if _, exists := testStore.list[rid]; !exists {
		t.Errorf("AddRound error: " +
			"Could not find added round in list")
	}

}

// Unit test
func TestUncheckedRoundStore_GetRound(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("GetRound error: "+
			"Could not call constructor NewUncheckedStore: %v", err)
	}

	// Add round to store
	rid := id.Round(0)
	roundInfo := &pb.RoundInfo{
		ID: uint64(rid),
	}
	ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	source := id.NewIdFromBytes([]byte("Sauron"), t)
	err = testStore.AddRound(roundInfo, ephId, source)
	if err != nil {
		t.Fatalf("GetRound error: "+
			"Could not add round to store: %v", err)
	}

	// Retrieve round that was inserted
	retrievedRound, exists := testStore.GetRound(rid)
	if !exists {
		t.Fatalf("GetRound error: " +
			"Could not get round from store")
	}

	if !bytes.Equal(retrievedRound.EpdId[:], ephId[:]) ||
		!source.Cmp(retrievedRound.Source) {
		t.Fatalf("GetRound error: "+
			"Values in loaded round %d are not expected."+
			"\n\tExpected ephemeral: %v"+
			"\n\tReceived ephemeral: %v"+
			"\n\tExpected source: %v"+
			"\n\tReceived source: %v", rid,
			ephId, retrievedRound.EpdId,
			source, retrievedRound.Source)
	}

	// Try to pull unknown round from store
	unknownRound := id.Round(1)
	_, exists = testStore.GetRound(unknownRound)
	if exists {
		t.Fatalf("GetRound error: " +
			"Should not find unknown round in store.")
	}

}

// Unit test
func TestUncheckedRoundStore_GetList(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("GetList error: "+
			"Could not call constructor NewUncheckedStore: %v", err)
	}

	// Add rounds to store
	numRounds := 10
	for i := 0; i < numRounds; i++ {
		rid := id.Round(i)
		roundInfo := &pb.RoundInfo{
			ID: uint64(rid),
		}
		ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
		source := id.NewIdFromUInt(uint64(i), id.User, t)
		err = testStore.AddRound(roundInfo, ephId, source)
		if err != nil {
			t.Errorf("GetList error: "+
				"Could not add round to store: %v", err)
		}
	}

	// Retrieve list
	retrievedList := testStore.GetList()
	if len(retrievedList) != numRounds {
		t.Errorf("GetList error: "+
			"List returned is not of expected size."+
			"\n\tExpected: %v\n\tReceived: %v", numRounds, len(retrievedList))
	}

	for i := 0; i < numRounds; i++ {
		rid := id.Round(i)
		if _, exists := retrievedList[rid]; !exists {
			t.Errorf("GetList error: "+
				"Retrieved list does not contain expected round %d.", rid)
		}
	}

}

// Unit test
func TestUncheckedRoundStore_IncrementCheck(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("IncrementCheck error: "+
			"Could not call constructor NewUncheckedStore: %v", err)
	}

	// Add rounds to store
	numRounds := 10
	for i := 0; i < numRounds; i++ {
		rid := id.Round(i)
		roundInfo := &pb.RoundInfo{
			ID: uint64(rid),
		}
		ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
		source := id.NewIdFromUInt(uint64(i), id.User, t)
		err = testStore.AddRound(roundInfo, ephId, source)
		if err != nil {
			t.Errorf("IncrementCheck error: "+
				"Could not add round to store: %v", err)
		}
	}

	testRound := id.Round(3)
	numChecks := 4
	for i := 0; i < numChecks; i++ {
		err = testStore.IncrementCheck(testRound)
		if err != nil {
			t.Errorf("IncrementCheck error: "+
				"Could not increment check for round %d: %v", testRound, err)
		}
	}

	rnd, _ := testStore.GetRound(testRound)
	if rnd.NumChecks != uint64(numChecks) {
		t.Errorf("IncrementCheck error: "+
			"Round %d did not have expected number of checks."+
			"\n\tExpected: %v\n\tReceived: %v", testRound, numChecks, rnd.NumChecks)
	}

	// Error path: check unknown round can not be incremented
	unknownRound := id.Round(numRounds + 5)
	err = testStore.IncrementCheck(unknownRound)
	if err == nil {
		t.Errorf("IncrementCheck error: "+
			"Should not find round %d which was not added to store", unknownRound)
	}

}

// Unit test
func TestUncheckedRoundStore_Remove(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	testStore, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("Remove error: "+
			"Could not call constructor NewUncheckedStore: %v", err)
	}

	// Add rounds to store
	numRounds := 10
	for i := 0; i < numRounds; i++ {
		rid := id.Round(i)
		roundInfo := &pb.RoundInfo{
			ID: uint64(rid),
		}
		ephId := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
		source := id.NewIdFromUInt(uint64(i), id.User, t)
		err = testStore.AddRound(roundInfo, ephId, source)
		if err != nil {
			t.Errorf("Remove error: "+
				"Could not add round to store: %v", err)
		}
	}

	// Remove round from storage
	removedRound := id.Round(1)
	err = testStore.Remove(removedRound)
	if err != nil {
		t.Errorf("Remove error: "+
			"Could not removed round %d from storage: %v", removedRound, err)
	}

	// Check that round was removed
	_, exists := testStore.GetRound(removedRound)
	if exists {
		t.Errorf("Remove error: "+
			"Round %d expected to be removed from storage", removedRound)
	}

	// Error path: attempt to remove unknown round
	unknownRound := id.Round(numRounds + 5)
	err = testStore.Remove(unknownRound)
	if err == nil {
		t.Errorf("Remove error: "+
			"Should not removed round %d which is not in storage", unknownRound)
	}

}
