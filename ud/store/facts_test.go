///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/fact"
	"reflect"
	"sort"
	"testing"
)

func TestNewStore(t *testing.T) {

	kv := versioned.NewKV(ekv.MakeMemstore())

	_, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

}

// Unit test
func TestStore_RestoreFromBackUp(t *testing.T) {

	kv := versioned.NewKV(ekv.MakeMemstore())

	s, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	expected := fact.Fact{
		Fact: "josh",
		T:    fact.Username,
	}

	fl := fact.FactList{expected}

	err = s.RestoreFromBackUp(fl)
	if err != nil {
		t.Fatalf("RestoreFromBackup err: %v", err)
	}

	_, exists := s.confirmedFacts[expected]
	if !exists {
		t.Fatalf("Fact %s does not exist in map", expected)
	}

}

// Error case.
func TestStore_RestoreFromBackUp_StatefulStore(t *testing.T) {

	kv := versioned.NewKV(ekv.MakeMemstore())

	s, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	confirmId := "confirm"
	expected := fact.Fact{
		Fact: "josh",
		T:    fact.Username,
	}

	err = s.StoreUnconfirmedFact(confirmId, expected)
	if err != nil {
		t.Fatalf("StoreUnconfirmedFact error: %v", err)
	}

	// Expected error: should error when restoring on
	// a stateful store.
	fl := fact.FactList{expected}
	err = s.RestoreFromBackUp(fl)
	if err == nil {
		t.Fatalf("RestoreFromBackup err: %v", err)
	}

}

func TestStore_ConfirmFact(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	confirmId := "confirm"

	expected := fact.Fact{
		Fact: "josh",
		T:    fact.Username,
	}

	err = expectedStore.StoreUnconfirmedFact(confirmId, expected)
	if err != nil {
		t.Fatalf("StoreUnconfirmedFact error: %v", err)
	}

	err = expectedStore.ConfirmFact(confirmId)
	if err != nil {
		t.Fatalf("ConfirmFact() produced an error: %v", err)
	}

	_, exists := expectedStore.confirmedFacts[expected]
	if !exists {
		t.Fatalf("Fact %s does not exist in map", expected)
	}

	// Check that fact was removed from unconfirmed
	_, exists = expectedStore.unconfirmedFacts[confirmId]
	if exists {
		t.Fatalf("Confirmed fact %v should be removed from unconfirmed"+
			" map", expected)
	}
}

func TestStore_StoreUnconfirmedFact(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	confirmId := "confirm"

	expected := fact.Fact{
		Fact: "josh",
		T:    fact.Username,
	}

	err = expectedStore.StoreUnconfirmedFact(confirmId, expected)
	if err != nil {
		t.Fatalf("StoreUnconfirmedFact error: %v", err)
	}

	// Check that fact exists in unconfirmed
	_, exists := expectedStore.unconfirmedFacts[confirmId]
	if !exists {
		t.Fatalf("Confirmed fact %v should be removed from unconfirmed"+
			" map", expected)
	}
}

func TestStore_DeleteFact(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	expected := fact.Fact{
		Fact: "josh",
		T:    fact.Username,
	}

	expectedStore.confirmedFacts[expected] = struct{}{}

	_, exists := expectedStore.confirmedFacts[expected]
	if !exists {
		t.Fatalf("Fact %s does not exist in map", expected)
	}

	err = expectedStore.DeleteFact(expected)
	if err != nil {
		t.Fatalf("DeleteFact() produced an error: %v", err)
	}

	err = expectedStore.DeleteFact(expected)
	if err == nil {
		t.Fatalf("DeleteFact should produce an error when deleting a fact not in store")
	}

}

func TestStore_BackUpMissingFacts(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	email := fact.Fact{
		Fact: "josh@elixxir.io",
		T:    fact.Email,
	}

	phone := fact.Fact{
		Fact: "6175555678",
		T:    fact.Phone,
	}

	username := fact.Fact{
		Fact: "admin",
		T:    fact.Username,
	}

	err = expectedStore.BackUpMissingFacts(username, email, phone)
	if err != nil {
		t.Fatalf("BackUpMissingFacts() produced an error: %v", err)
	}

	_, exists := expectedStore.confirmedFacts[email]
	if !exists {
		t.Fatalf("Fact %v not found in store.", email)
	}

	_, exists = expectedStore.confirmedFacts[phone]
	if !exists {
		t.Fatalf("Fact %v not found in store.", phone)
	}

}

func TestStore_BackUpMissingFacts_DuplicateFactType(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	email := fact.Fact{
		Fact: "josh@elixxir.io",
		T:    fact.Email,
	}

	phone := fact.Fact{
		Fact: "6175555678",
		T:    fact.Phone,
	}

	username := fact.Fact{
		Fact: "admin",
		T:    fact.Username,
	}

	err = expectedStore.BackUpMissingFacts(username, email, phone)
	if err != nil {
		t.Fatalf("BackUpMissingFacts() produced an error: %v", err)
	}

	err = expectedStore.BackUpMissingFacts(username, email, fact.Fact{})
	if err == nil {
		t.Fatalf("BackUpMissingFacts() should not allow backing up an "+
			"email when an email has already been backed up: %v", err)
	}

	err = expectedStore.BackUpMissingFacts(username, fact.Fact{}, phone)
	if err == nil {
		t.Fatalf("BackUpMissingFacts() should not allow backing up a "+
			"phone number when a phone number has already been backed up: %v", err)
	}

}

func TestStore_GetFacts(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	testStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	emailFact := fact.Fact{
		Fact: "josh@elixxir.io",
		T:    fact.Email,
	}

	emptyFact := fact.Fact{}

	username := fact.Fact{
		Fact: "admin",
		T:    fact.Username,
	}

	err = testStore.BackUpMissingFacts(username, emailFact, emptyFact)
	if err != nil {
		t.Fatalf("Faild to add fact %v: %v", emailFact, err)
	}

	phoneFact := fact.Fact{
		Fact: "6175555212",
		T:    fact.Phone,
	}

	err = testStore.BackUpMissingFacts(username, emptyFact, phoneFact)
	if err != nil {
		t.Fatalf("Faild to add fact %v: %v", phoneFact, err)
	}

	expectedFacts := []fact.Fact{emailFact, phoneFact}

	receivedFacts := testStore.GetFacts()

	sort.SliceStable(receivedFacts, func(i, j int) bool {
		return receivedFacts[i].Fact > receivedFacts[j].Fact
	})

	sort.SliceStable(expectedFacts, func(i, j int) bool {
		return expectedFacts[i].Fact > expectedFacts[j].Fact
	})

	if !reflect.DeepEqual(expectedFacts, receivedFacts) {
		t.Fatalf("GetFacts() did not return expected fact list."+
			"\nExpected: %v"+
			"\nReceived: %v", expectedFacts, receivedFacts)
	}
}

func TestStore_GetFactStrings(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	testStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	emailFact := fact.Fact{
		Fact: "josh@elixxir.io",
		T:    fact.Email,
	}
	username := fact.Fact{
		Fact: "admin",
		T:    fact.Username,
	}

	emptyFact := fact.Fact{}

	err = testStore.BackUpMissingFacts(username, emailFact, emptyFact)
	if err != nil {
		t.Fatalf("Faild to add fact %v: %v", emailFact, err)
	}

	phoneFact := fact.Fact{
		Fact: "6175555212",
		T:    fact.Phone,
	}

	err = testStore.BackUpMissingFacts(emptyFact, emptyFact, phoneFact)
	if err != nil {
		t.Fatalf("Faild to add fact %v: %v", phoneFact, err)
	}

	expectedFacts := []string{username.Stringify(), emailFact.Stringify(), phoneFact.Stringify()}

	receivedFacts := testStore.GetStringifiedFacts()
	sort.SliceStable(receivedFacts, func(i, j int) bool {
		return receivedFacts[i] > receivedFacts[j]
	})

	sort.SliceStable(expectedFacts, func(i, j int) bool {
		return expectedFacts[i] > expectedFacts[j]
	})

	if !reflect.DeepEqual(expectedFacts, receivedFacts) {
		t.Fatalf("GetStringifiedFacts() did not return expected fact list."+
			"\nExpected: %v"+
			"\nReceived: %v", expectedFacts, receivedFacts)
	}

}
