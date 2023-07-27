////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/fact"
	"reflect"
	"testing"
)

// Test it loads a Store from storage if it exists.
func TestNewOrLoadStore_LoadStore(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := newStore(kv)
	if err != nil {
		t.Errorf("newStore() produced an error: %v", err)
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
	kv := versioned.NewKV(ekv.MakeMemstore())
	expectedKv, err := kv.Prefix(prefix)
	require.NoError(t, err)

	receivedStore, err := NewOrLoadStore(kv)
	if err != nil {
		t.Fatalf("NewOrLoadStore() produced an error: %v", err)
	}

	expectedStore := &Store{
		confirmedFacts:   make(map[fact.Fact]struct{}, 0),
		unconfirmedFacts: make(map[string]fact.Fact, 0),
		kv:               expectedKv,
	}

	require.Equal(t, expectedStore, receivedStore)

}

func TestStore_MarshalUnmarshal_ConfirmedFacts(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := newStore(kv)
	if err != nil {
		t.Errorf("newStore() produced an error: %v", err)
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
		t.Errorf("newStore() returned incorrect Store."+
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
	kv := versioned.NewKV(ekv.MakeMemstore())

	expectedStore, err := newStore(kv)
	if err != nil {
		t.Errorf("newStore() produced an error: %v", err)
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
		t.Errorf("newStore() returned incorrect Store."+
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
