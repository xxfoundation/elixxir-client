package ud

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"reflect"
	"testing"
)

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

func TestStore_MarshalUnmarshal_ConfirmedFacts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	data, err := expectedStore.kv.Get(confirmedFactKey, version)
	if err != nil {
		t.Errorf("Get() error when getting Store from KV: %v", err)
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

}

func TestStore_MarshalUnmarshal_UnconfirmedFacts(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	expectedStore, err := NewStore(kv)
	if err != nil {
		t.Errorf("NewStore() produced an error: %v", err)
	}

	data, err := expectedStore.kv.Get(confirmedFactKey, version)
	if err != nil {
		t.Errorf("Get() error when getting Store from KV: %v", err)
	}

	data, err = expectedStore.kv.Get(unconfirmedFactKey, version)
	if err != nil {
		t.Errorf("Get() error when getting Store from KV: %v", err)
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

}
