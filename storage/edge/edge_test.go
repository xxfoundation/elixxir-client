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
	"sync"
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

	var wg sync.WaitGroup

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

	wg.Add(1)
	wg.Add(1)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(50 * time.Millisecond).C:
				t.Errorf("Timed out waiting for callback (%d).", i)
			case r := <-id0Chan:
				if !identities[0].Cmp(r.identity) {
					t.Errorf("Received wrong identity (%d).\nexpected: %s"+
						"\nreceived: %s", i, identities[0], r.identity)
				} else if r.deleted == true {
					t.Errorf("Received wrong value for deleted (%d)."+
						"\nexpected: %t\nreceived: %t", i, true, r.deleted)
				}
			}
			wg.Done()
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

	wg.Add(1)
	go func() {
		select {
		case <-time.NewTimer(10 * time.Millisecond).C:
			t.Errorf("Timed out waiting for callback.")
		case r := <-id1Chan:
			if !identities[1].Cmp(r.identity) {
				t.Errorf("Received wrong identity.\nexpected: %s\nreceived: %s",
					identities[1], r.identity)
			} else if r.deleted == true {
				t.Errorf("Received wrong value for deleted."+
					"\nexpected: %t\nreceived: %t", true, r.deleted)
			}
		}
		wg.Done()
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

	expected = preimages[0]
	if !reflect.DeepEqual(pis[expected.key()], expected) {
		t.Errorf("Second Preimage of first Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	}

	expected = preimages[2]
	if !reflect.DeepEqual(pis[expected.key()], expected) {
		t.Errorf("Third Preimage of first Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
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

	expected = preimages[1]
	if !reflect.DeepEqual(pis[expected.key()], expected) {
		t.Errorf("Second Preimage of second Preimages does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	}

	wg.Wait()
}

func TestStore_Remove(t *testing.T) {
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

	s.Add(preimages[0], identities[0])
	s.Add(preimages[1], identities[1])
	s.Add(preimages[2], identities[0])

	var wg sync.WaitGroup

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

	wg.Add(1)
	wg.Add(1)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(50 * time.Millisecond).C:
				t.Errorf("Timed out waiting for callback (%d).", i)
			case r := <-id0Chan:
				if !identities[0].Cmp(r.identity) {
					t.Errorf("Received wrong identity (%d).\nexpected: %s"+
						"\nreceived: %s", i, identities[0], r.identity)
				} else if r.deleted == true {
					t.Errorf("Received wrong value for deleted (%d)."+
						"\nexpected: %t\nreceived: %t", i, true, r.deleted)
				}
			}
			wg.Done()
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

	wg.Add(1)
	go func() {
		select {
		case <-time.NewTimer(10 * time.Millisecond).C:
			t.Errorf("Timed out waiting for callback.")
		case r := <-id1Chan:
			if !identities[1].Cmp(r.identity) {
				t.Errorf("Received wrong identity.\nexpected: %s\nreceived: %s",
					identities[1], r.identity)
			} else if r.deleted == true {
				t.Errorf("Received wrong value for deleted."+
					"\nexpected: %t\nreceived: %t", true, r.deleted)
			}
		}
		wg.Done()
	}()

	err := s.Remove(preimages[0], identities[0])
	if err != nil {
		t.Errorf("Remove returned an error: %+v", err)
	}

	err = s.Remove(preimages[1], identities[1])
	if err != nil {
		t.Errorf("Remove returned an error: %+v", err)
	}

	err = s.Remove(preimages[2], identities[0])
	if err != nil {
		t.Errorf("Remove returned an error: %+v", err)
	}

	if len(s.edge) != 3 {
		t.Errorf("Length of edge incorrect.\nexpected: %d\nreceived: %d",
			3, len(s.edge))
	}

	// pis := s.edge[*identities[0]]
	//
	// if len(pis) != 3 {
	// 	t.Errorf("Length of preimages for identity %s inocrrect."+
	// 		"\nexpected: %d\nreceived: %d", identities[0], 3, len(pis))
	// }
	//
	// expected := Preimage{identities[0].Bytes(), "default", identities[0].Bytes()}
	// if !reflect.DeepEqual(pis[expected.key()], expected) {
	// 	t.Errorf("First Preimage of first Preimages does not match expected."+
	// 		"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	// }
	//
	// expected = preimages[0]
	// if !reflect.DeepEqual(pis[expected.key()], expected) {
	// 	t.Errorf("Second Preimage of first Preimages does not match expected."+
	// 		"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	// }
	//
	// expected = preimages[2]
	// if !reflect.DeepEqual(pis[expected.key()], expected) {
	// 	t.Errorf("Third Preimage of first Preimages does not match expected."+
	// 		"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	// }
	//
	// pis = s.edge[*identities[1]]
	//
	// if len(pis) != 2 {
	// 	t.Errorf("Length of preimages for identity %s inocrrect."+
	// 		"\nexpected: %d\nreceived: %d", identities[1], 2, len(pis))
	// }
	//
	// expected = Preimage{identities[1].Bytes(), "default", identities[1].Bytes()}
	// if !reflect.DeepEqual(pis[expected.key()], expected) {
	// 	t.Errorf("First Preimage of second Preimages does not match expected."+
	// 		"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	// }
	//
	// expected = preimages[1]
	// if !reflect.DeepEqual(pis[expected.key()], expected) {
	// 	t.Errorf("Second Preimage of second Preimages does not match expected."+
	// 		"\nexpected: %+v\nreceived: %+v", expected, pis[expected.key()])
	// }

	wg.Wait()
}

func TestStore_Get_Josh(t *testing.T) {
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

	s.Add(preimages[0], identities[0])
	s.Add(preimages[1], identities[1])
	s.Add(preimages[2], identities[0])

	// retrieve for first identity (has two preimages)
	receivedPreimages, ok := s.Get(identities[0])
	if !ok { // Check that identity exists in map
		t.Errorf("Could not retrieve preimages for identity %s", identities[0])
	}

	// Check for first preimage for first identity
	preimageKey := preimages[0].key()
	preimage, ok := receivedPreimages[preimageKey]
	if !ok {
		t.Errorf("Could not retrieve preimage with key %s for identity %s",
			preimageKey, identities[0])
	}

	// Check that retrieved value matches
	if !reflect.DeepEqual(preimages[0], preimage) {
		t.Errorf("Unexpected preimage received." +
			"\n\tExpected %s\n\tReceived: %v", preimages[0], preimage)
	}

	// Check for second preimage for first identity
	preimageKey = preimages[2].key()
	preimage, ok = receivedPreimages[preimageKey]
	if !ok {
		t.Errorf("Could not retrieve preimage with key %s for identity %s",
			preimageKey, identities[0])
	}

	// Check that retrieved value matches
	if !reflect.DeepEqual(preimages[2], preimage) {
		t.Errorf("Unexpected preimage received." +
			"\n\tExpected %s\n\tReceived: %v", preimages[2], preimage)
	}

	// Check second identity (has one preimage)
	receivedPreimages, ok = s.Get(identities[1])
	if !ok { // Check that identity exists in map
		t.Errorf("Could not retrieve preimages for identity %s", identities[1])
	}

	// Check for first preimage for first identity
	preimageKey = preimages[1].key()
	preimage, ok = receivedPreimages[preimageKey]
	if !ok {
		t.Errorf("Could not retrieve preimage with key %s for identity %s",
			preimageKey, identities[1])
	}

	// Check that retrieved value matches
	if !reflect.DeepEqual(preimages[1], preimage) {
		t.Errorf("Unexpected preimage received." +
			"\n\tExpected %s\n\tReceived: %v", preimages[0], preimage)
	}

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
