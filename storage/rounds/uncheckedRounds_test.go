///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"sync"
	"testing"
)

func TestNewUncheckedStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	testStore := &UncheckedRoundStore{
		list: make(map[id.Round]UncheckedRound),
		kv:   kv.Prefix(uncheckedRoundPrefix),
	}

	store, err := NewUncheckedStore(kv)
	if err != nil {
		t.Fatalf("NewUncheckedStore error: " +
			"Could not create unchecked stor: %v", err)
	}

	// Compare manually created object with NewUnknownRoundsStore
	if !reflect.DeepEqual(testStore, store) {
		t.Fatalf("NewUncheckedStore error: " +
			"Returned incorrect Store."+
			"\n\texpected: %+v\n\treceived: %+v", testStore, store)
	}

	UnknownRounds{
		rounds: nil,
		params: UnknownRoundsParams{},
		kv:     nil,
		mux:    sync.Mutex{},
	}
}