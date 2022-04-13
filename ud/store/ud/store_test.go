package store

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/fact"
	"reflect"
	"testing"
)

// Test it loads a Store from storage if it exists.
func TestNewOrLoadStore_LoadStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	receivedStore, err := NewOrLoadStore(kv)
	if err != nil {
		t.Fatalf("NewOrLoadStore() produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedStore, receivedStore) {
		t.Errorf("NewOrLoadStore() returned incorrect Store."+
			"\nexpected: %#v\nreceived: %#v", expectedStore,
			receivedStore)

	}

}

// Test that it creates a new store if an old one is not in storage.
func TestNewOrLoadStore_NewStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	receivedStore, err := NewOrLoadStore(kv)
	if err != nil {
		t.Fatalf("NewOrLoadStore() produced an error: %v", err)
	}

	expectedStore := &Store{
		confirmedFacts:   make(map[fact.Fact]struct{}, 0),
		unconfirmedFacts: make(map[string]fact.Fact, 0),
		kv:               kv.Prefix(prefix),
	}

	if !reflect.DeepEqual(expectedStore, receivedStore) {
		t.Errorf("NewOrLoadStore() returned incorrect Store."+
			"\nexpected: %#v\nreceived: %#v", expectedStore,
			receivedStore)

	}

}

func TestStore_MarshalUnmarshal_ConfirmedFacts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	data, err := expectedStore.kv.Get(confirmedFactKey, version)
	if err != nil {
		t.Errorf("get() error when getting Store from KV: %v", err)
	}

	expectedData, err := expectedStore.marshalConfirmedFacts()
	if err != nil {
		t.Fatalf("marshalConfirmedFact error: %+v", err)
	}

	if !bytes.Equal(expectedData, data.Data) {
		t.Errorf("NewStore() returned incorrect Store."+
			"\nexpected: %+v\nreceived: %+v", expectedData,
			data.Data)
	}

	recieved, err := expectedStore.unmarshalConfirmedFacts(data.Data)
	if err != nil {
		t.Fatalf("unmarshalUnconfirmedFacts error: %v", err)
	}

	if !reflect.DeepEqual(recieved, expectedStore.confirmedFacts) {
		t.Fatalf("Marshal/Unmarshal did not produce identical data"+
			"\nExpected: %v "+
			"\nReceived: %v", expectedStore.confirmedFacts, recieved)
	}
}

func TestStore_MarshalUnmarshal_UnconfirmedFacts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	data, err := expectedStore.kv.Get(unconfirmedFactKey, version)
	if err != nil {
		t.Errorf("get() error when getting Store from KV: %v", err)
	}

	expectedData, err := expectedStore.marshalUnconfirmedFacts()
	if err != nil {
		t.Fatalf("marshalConfirmedFact error: %+v", err)
	}

	if !bytes.Equal(expectedData, data.Data) {
		t.Errorf("NewStore() returned incorrect Store."+
			"\nexpected: %+v\nreceived: %+v", expectedData,
			data.Data)
	}

	recieved, err := expectedStore.unmarshalUnconfirmedFacts(data.Data)
	if err != nil {
		t.Fatalf("unmarshalUnconfirmedFacts error: %v", err)
	}

	if !reflect.DeepEqual(recieved, expectedStore.unconfirmedFacts) {
		t.Fatalf("Marshal/Unmarshal did not produce identical data"+
			"\nExpected: %v "+
			"\nReceived: %v", expectedStore.unconfirmedFacts, recieved)
	}
}
