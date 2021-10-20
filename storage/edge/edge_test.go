////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package edge

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests that NewStore returns the expected new Store and that it can be loaded
// from storage.
func TestNewStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	baseIdentity := id.NewIdFromString("baseIdentity", id.User, t)
	expected := &Store{
		kv:        kv.Prefix(edgeStorePrefix),
		edge:      map[id.ID]Preimages{*baseIdentity: newPreimages(baseIdentity)},
		callbacks: make(map[id.ID][]ListUpdateCallBack),
	}

	received, err := NewStore(kv, baseIdentity)
	if err != nil {
		t.Errorf("NewStore returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("New Store does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}

	_, err = expected.kv.Get(preimagesKey(baseIdentity), preimageStoreVersion)
	if err != nil {
		t.Errorf("Failed to load Store from storage: %+v", err)
	}
}

// Adds three Preimage to the store, two with the same identity. It checks that
// all three exist and that the length of the list is correct. Also checks that
// the appropriate callbacks are called.
func TestStore_Add(t *testing.T) {
	s, _, _ := newTestStore(t)
	identities := []*id.ID{
		id.NewIdFromString("identity0", id.User, t),
		id.NewIdFromString("identity1", id.User, t),
	}
	preimages := []Preimage{
		{[]byte("ID0"), "default0", []byte("ID0")},
		{[]byte("ID1"), "default1", []byte("ID1")},
		{[]byte("ID2"), "default2", []byte("ID2")},
	}

	id0Chan := make(chan struct {
		identity *id.ID
		deleted  bool
	}, 2)
	s.callbacks[*identities[0]] = []ListUpdateCallBack{func(identity *id.ID, deleted bool) {
		id0Chan <- struct {
			identity *id.ID
			deleted  bool
		}{identity: identity, deleted: deleted}
	}}

	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(10 * time.Millisecond).C:
				t.Errorf("Timed out waiting for callback (%d).", i)
			case r := <-id0Chan:
				if !identities[0].Cmp(r.identity) {
					t.Errorf("Received wrong identity (%d).\nexpected: %s"+
						"\nreceived: %s", i, identities[0], r.identity)
				}
			}
		}
	}()

	id1Chan := make(chan struct {
		identity *id.ID
		deleted  bool
	})
	s.callbacks[*identities[1]] = []ListUpdateCallBack{func(identity *id.ID, deleted bool) {
		id1Chan <- struct {
			identity *id.ID
			deleted  bool
		}{identity: identity, deleted: deleted}
	}}

	go func() {
		select {
		case <-time.NewTimer(10 * time.Millisecond).C:
			t.Errorf("Timed out waiting for callback.")
		case r := <-id0Chan:
			if !identities[0].Cmp(r.identity) {
				t.Errorf("Received wrong identity.\nexpected: %s\nreceived: %s",
					identities[0], r.identity)
			}
		}
	}()

	s.Add(preimages[0], identities[0])
	s.Add(preimages[1], identities[1])
	s.Add(preimages[2], identities[0])

	if len(s.edge) != 3 {
		t.Errorf("Length of edge incorrect.\nexpected: %d\nreceived: %d",
			3, len(s.edge))
	}

	pis := s.edge[*identities[0]]

	if len(pis) != 3 {
		t.Errorf("Length of preimages for identity %s inocrrect."+
			"\nexpected: %d\nreceived: %d", identities[0], 3, len(pis))
	}

	expected := Preimage{identities[0].Bytes(), "default", identities[0].Bytes()}
	if !reflect.DeepEqual(pis[expected.key()], expected) {
		t.Errorf("First Preimage of first Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	}

	if !reflect.DeepEqual(pis[preimages[0].key()], preimages[0]) {
		t.Errorf("Second Preimage of first Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", preimages[0], pis[preimages[0].key()])
	}

	if !reflect.DeepEqual(pis[preimages[2].key()], preimages[2]) {
		t.Errorf("Third Preimage of first Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", preimages[2], pis[preimages[2].key()])
	}

	pis = s.edge[*identities[1]]

	if len(pis) != 2 {
		t.Errorf("Length of preimages for identity %s inocrrect."+
			"\nexpected: %d\nreceived: %d", identities[1], 2, len(pis))
	}

	expected = Preimage{identities[1].Bytes(), "default", identities[1].Bytes()}
	if !reflect.DeepEqual(pis[expected.key()], expected) {
		t.Errorf("First Preimage of second Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	}

	if !reflect.DeepEqual(pis[preimages[1].key()], preimages[1]) {
		t.Errorf("Second Preimage of second Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", preimages[1], pis[preimages[1].key()])
	}
}

func TestStore_Remove(t *testing.T) {
}

func TestStore_Get(t *testing.T) {
}

func TestStore_AddUpdateCallback(t *testing.T) {
}

func TestLoadStore(t *testing.T) {
}

func TestStore_save(t *testing.T) {
}

// newTestStore creates a new Store with a random base identity. Returns the
// Store, KV, and base identity.
func newTestStore(t *testing.T) (*Store, *versioned.KV, *id.ID) {
	kv := versioned.NewKV(make(ekv.Memstore))
	baseIdentity, err := id.NewRandomID(
		rand.New(rand.NewSource(time.Now().Unix())), id.User)
	if err != nil {
		t.Fatalf("Failed to generate random base identity: %+v", err)
	}

	s, err := NewStore(kv, baseIdentity)
	if err != nil {
		t.Fatalf("Failed to create new test Store: %+v", err)
	}

	return s, kv, baseIdentity
}
