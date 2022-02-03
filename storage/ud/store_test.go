///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/fact"
	"reflect"
	"sort"
	"testing"
)

func TestNewStore(t *testing.T) {

	kv := versioned.NewKV(make(ekv.Memstore))
	expectedStore := &Store{
		kv: kv.Prefix(prefix),
	}
	expectedData, err := expectedStore.marshal()
	if err != nil {
		t.Fatalf("marshal() produced an error: %v", err)
	}

	_, err = NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	data, err := expectedStore.kv.Get(key, version)
	if err != nil {
		t.Errorf("Get() error when getting Store from KV: %v", err)
	}

	if !bytes.Equal(expectedData, data.Data) {
		t.Errorf("NewStore() returned incorrect Store."+
			"\nexpected: %+v\nreceived: %+v", expectedData,
			data.Data)
	}
}

func TestLoadStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	receivedStore, err := LoadStore(kv)
	if err != nil {
		t.Fatalf("LoadStore() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedStore, receivedStore) {
		t.Errorf("LoadStore() returned incorrect Store."+
			"\nexpected: %#v\nreceived: %#v", expectedStore,
			receivedStore)

	}

}

func TestStore_BackUpMissingFacts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

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

	err = expectedStore.BackUpMissingFacts(email, phone)
	if err != nil {
		t.Fatalf("BackUpMissingFacts() produced an error: %v", err)
	}

	f := expectedStore.registeredFacts[emailIndex]
	if !reflect.DeepEqual(f, email) {
		t.Fatalf("Fact in store does not match expected value."+
			"\nExpected: %v"+
			"\nReceived: %v", email, f)
	}

	f = expectedStore.registeredFacts[phoneIndex]
	if !reflect.DeepEqual(f, phone) {
		t.Fatalf("Fact in store does not match expected value."+
			"\nExpected: %v"+
			"\nReceived: %v", phone, f)
	}

}

func TestStore_BackUpMissingFacts_DuplicateFactType(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

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

	err = expectedStore.BackUpMissingFacts(email, phone)
	if err != nil {
		t.Fatalf("BackUpMissingFacts() produced an error: %v", err)
	}

	err = expectedStore.BackUpMissingFacts(email, fact.Fact{})
	if err == nil {
		t.Fatalf("BackUpMissingFacts() should not allow backing up an "+
			"email when an email has already been backed up: %v", err)
	}

	err = expectedStore.BackUpMissingFacts(fact.Fact{}, phone)
	if err == nil {
		t.Fatalf("BackUpMissingFacts() should not allow backing up a "+
			"phone number when a phone number has already been backed up: %v", err)
	}

}

func TestStore_GetFacts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	emailFact := fact.Fact{
		Fact: "josh@elixxir.io",
		T:    fact.Email,
	}

	emptyFact := fact.Fact{}

	err = testStore.BackUpMissingFacts(emailFact, emptyFact)
	if err != nil {
		t.Fatalf("Faild to add fact %v: %v", emailFact, err)
	}

	phoneFact := fact.Fact{
		Fact: "6175555212",
		T:    fact.Phone,
	}

	err = testStore.BackUpMissingFacts(emptyFact, phoneFact)
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
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	emailFact := fact.Fact{
		Fact: "josh@elixxir.io",
		T:    fact.Email,
	}

	emptyFact := fact.Fact{}

	err = testStore.BackUpMissingFacts(emailFact, emptyFact)
	if err != nil {
		t.Fatalf("Faild to add fact %v: %v", emailFact, err)
	}

	phoneFact := fact.Fact{
		Fact: "6175555212",
		T:    fact.Phone,
	}

	err = testStore.BackUpMissingFacts(emptyFact, phoneFact)
	if err != nil {
		t.Fatalf("Faild to add fact %v: %v", phoneFact, err)
	}

	expectedFacts := []string{emailFact.Stringify(), phoneFact.Stringify()}

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
