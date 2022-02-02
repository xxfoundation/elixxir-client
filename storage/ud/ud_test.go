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
	"strconv"
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

func TestStore_AddFact(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	expected := fact.Fact{
		Fact: "josh",
		T:    fact.Username,
	}

	err = expectedStore.StoreFact(expected)
	if err != nil {
		t.Fatalf("StoreFact() produced an error: %v", err)
	}

	_, exists := expectedStore.registeredFacts[expected]
	if !exists {
		t.Fatalf("Fact %s does not exist in map", expected)
	}

}

func TestStore_GetFacts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	expectedFacts := make([]fact.Fact, 0)
	for i := 0; i < 100; i++ {

		f := fact.Fact{
			Fact: strconv.Itoa(i),
			T:    fact.Username,
		}

		err = testStore.StoreFact(f)
		if err != nil {
			t.Fatalf("Faild to add fact %v: %v", f, err)
		}
		expectedFacts = append(expectedFacts, f)

	}

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

	expectedFacts := make([]string, 0)
	for i := 0; i < 100; i++ {

		f := fact.Fact{
			Fact: strconv.Itoa(i),
			T:    fact.Username,
		}

		err = testStore.StoreFact(f)
		if err != nil {
			t.Fatalf("Faild to add fact %v: %v", f, err)
		}
		expectedFacts = append(expectedFacts, f.Stringify())
	}

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
