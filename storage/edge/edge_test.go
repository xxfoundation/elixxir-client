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
	"encoding/json"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/storage/versioned"
	fingerprint2 "gitlab.com/elixxir/crypto/fingerprint"
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
// Store.Add adds all three exist and that the length of the list is correct.
// Also checks that the appropriate callbacks are called.
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
	s.callbacks[*identities[0]] = []ListUpdateCallBack{
		func(identity *id.ID, deleted bool) {
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
	s.callbacks[*identities[1]] = []ListUpdateCallBack{
		func(identity *id.ID, deleted bool) {
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

	expected := Preimage{preimage.MakeDefault(identities[0]), catalog.Default, identities[0].Bytes()}
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

	expected = Preimage{preimage.MakeDefault(identities[1]), catalog.Default, identities[1].Bytes()}
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

// Adds three Preimage to two identities and tests that Store.Remove removes all
// three blue the default preimage for the second identity and checks that all
// Preimage have been deleted, that the Preimages for the second identity has
// been deleted and that the callbacks are called with the expected values.
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
	s.callbacks[*identities[0]] = []ListUpdateCallBack{
		func(identity *id.ID, deleted bool) {
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
	s.callbacks[*identities[1]] = []ListUpdateCallBack{
		func(identity *id.ID, deleted bool) {
			id1Chan <- struct {
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
			case r := <-id1Chan:
				if !identities[1].Cmp(r.identity) {
					t.Errorf("Received wrong identity (%d).\nexpected: %s"+
						"\nreceived: %s", i, identities[1], r.identity)
				}
			}
			wg.Done()
		}
	}()

	err := s.Remove(preimages[0], identities[0])
	if err != nil {
		t.Errorf("Remove returned an error: %+v", err)
	}

	err = s.Remove(preimages[1], identities[1])
	if err != nil {
		t.Errorf("Remove returned an error: %+v", err)
	}

	err = s.Remove(Preimage{Data: identities[1].Bytes()}, identities[1])
	if err != nil {
		t.Errorf("Remove returned an error: %+v", err)
	}

	err = s.Remove(preimages[2], identities[0])
	if err != nil {
		t.Errorf("Remove returned an error: %+v", err)
	}

	if len(s.edge) != 2 {
		t.Errorf("Length of edge incorrect.\nexpected: %d\nreceived: %d",
			2, len(s.edge))
	}

	pis := s.edge[*identities[0]]

	if len(pis) != 1 {
		t.Errorf("Length of preimages for identity %s inocrrect."+
			"\nexpected: %d\nreceived: %d", identities[0], 1, len(pis))
	}

	expected := preimages[0]
	if _, exists := pis[expected.key()]; exists {
		t.Errorf("Second Preimage of first Preimages exists when it should " +
			"have been deleted.")
	}

	expected = preimages[2]
	if _, exists := pis[expected.key()]; exists {
		t.Errorf("Third Preimage of first Preimages exists when it should " +
			"have been deleted.")
	}

	pis = s.edge[*identities[1]]

	if len(pis) != 0 {
		t.Errorf("Length of preimages for identity %s inocrrect."+
			"\nexpected: %d\nreceived: %d", identities[1], 0, len(pis))
	}

	wg.Wait()
}

// Tests that Store.Get returns the expected Preimages.
func TestStore_Get(t *testing.T) {
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

	pis, exists := s.Get(identities[0])
	if !exists {
		t.Errorf("No Preimages found for identity %s.", identities[0])
	}

	expected := []Preimage{
		{preimage.MakeDefault(identities[0]), catalog.Default, identities[0].Bytes()},
		preimages[0],
		preimages[2],
	}

	if len(expected) != len(pis) {
		t.Errorf("First Preimages for identity %s does not match expected, difrent lengths of %d and %d"+
			"\nexpected: %+v\nreceived: %+v", identities[0], len(expected), len(pis), expected, pis)
	}

top:
	for i, lookup := range expected {
		for _, checked := range pis {
			if reflect.DeepEqual(lookup, checked) {
				continue top
			}
		}
		t.Errorf("Entree %d in expected %v not found in received %v", i, lookup, pis)
	}

	pis, exists = s.Get(identities[1])
	if !exists {
		t.Errorf("No Preimages found for identity %s.", identities[1])
	}

	expected = []Preimage{
		{preimage.MakeDefault(identities[1]), catalog.Default, identities[1].Bytes()},
		preimages[1],
	}

	if len(expected) != len(pis) {
		t.Errorf("First Preimages for identity %s does not match expected, difrent lengths of %d and %d"+
			"\nexpected: %+v\nreceived: %+v", identities[0], len(expected), len(pis), expected, pis)
	}

top2:
	for i, lookup := range expected {
		for _, checked := range pis {
			if reflect.DeepEqual(lookup, checked) {
				continue top2
			}
		}
		t.Errorf("Entree %d in expected %v not found in received %v", i, lookup, pis)
	}
}

// Tests that Store.AddUpdateCallback adds all the appropriate callbacks for
// each identity by calling each callback and checking if the received identity
// is correct.
func TestStore_AddUpdateCallback(t *testing.T) {
	s, _, _ := newTestStore(t)
	// Create list of n identities, each with one more callback than the last
	// with the first having one
	n := 3
	chans := make(map[id.ID][]chan *id.ID, n)
	for i := 0; i < n; i++ {
		identity := id.NewIdFromUInt(uint64(i), id.User, t)
		chans[*identity] = make([]chan *id.ID, i+1)
		for j := range chans[*identity] {
			cbChan := make(chan *id.ID, 2)
			cb := func(cbIdentity *id.ID, _ bool) { cbChan <- cbIdentity }
			chans[*identity][j] = cbChan
			s.AddUpdateCallback(identity, cb)
		}
	}

	var wg sync.WaitGroup
	for identity, chanList := range chans {
		for i := range chanList {
			wg.Add(1)
			go func(identity *id.ID, i int) {
				select {
				case <-time.NewTimer(150 * time.Millisecond).C:
					t.Errorf("Timed out waiting on callback %d/%d for "+
						"identity %s.", i+1, len(chans[*identity]), identity)
				case r := <-chans[*identity][i]:
					if !identity.Cmp(r) {
						t.Errorf("Identity received from callback %d/%d does "+
							"not match expected.\nexpected: %s\nreceived: %s",
							i+1, len(chans[*identity]), identity, r)
					}
				}
				wg.Done()
			}(identity.DeepCopy(), i)
		}
	}

	for identity, cbs := range chans {
		for i := range cbs {
			go s.callbacks[identity][i](identity.DeepCopy(), false)
		}
	}

	wg.Wait()
}

func TestLoadStore(t *testing.T) {
	// Initialize store
	s, kv, _ := newTestStore(t)
	identities := []*id.ID{
		id.NewIdFromString("identity0", id.User, t),
		id.NewIdFromString("identity1", id.User, t),
	}
	preimages := []Preimage{
		{[]byte("ID0"), "default0", []byte("ID0")},
		{[]byte("ID1"), "default1", []byte("ID1")},
		{[]byte("ID2"), "default2", []byte("ID2")},
	}

	// Add preimages
	s.Add(preimages[0], identities[0])
	s.Add(preimages[1], identities[1])
	s.Add(preimages[2], identities[0])

	err := s.save()
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	receivedStore, err := LoadStore(kv)
	if err != nil {
		t.Fatalf("LoadStore error: %v", err)
	}

	expectedPis := [][]Preimage{
		{
			Preimage{preimage.MakeDefault(identities[0]), catalog.Default, identities[0].Bytes()},
			preimages[0],
			preimages[2],
		},
		{
			Preimage{preimage.MakeDefault(identities[1]), catalog.Default, identities[1].Bytes()},
			preimages[1],
		},
	}

	for i, identity := range identities {
		pis, exists := receivedStore.Get(identity)
		if !exists {
			t.Errorf("Identity %s does not exist in loaded store", identity)
		}

		if len(expectedPis[i]) != len(pis) {
			t.Errorf("First Preimages for identity %s does not match expected, difrent lengths of %d and %d"+
				"\nexpected: %+v\nreceived: %+v", identities[0], len(expectedPis[i]), len(pis), expectedPis[i], pis)
		}

	top:
		for idx, lookup := range expectedPis[i] {
			for _, checked := range pis {
				if reflect.DeepEqual(lookup, checked) {
					continue top
				}
			}
			t.Errorf("Entree %d in expected %v not found in received %v", idx, lookup, pis)
		}

	}
}

func TestStore_Check(t *testing.T) {
	// Initialize store
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

	// Add preimages
	s.Add(preimages[0], identities[0])
	s.Add(preimages[1], identities[1])
	s.Add(preimages[2], identities[0])

	testMsg := []byte("test message 123")
	preImageData := preimages[0].Data
	testFp := fingerprint2.IdentityFP(testMsg, preImageData)

	has, forMe, receivedPreImage := s.Check(identities[0], testFp, testMsg)

	if !has || !forMe || !reflect.DeepEqual(receivedPreImage, preimages[0]) {
		t.Errorf("Unexpected result from Check()."+
			"\nExpected results: (has: %v) "+
			"\n\t(forMe: %v)"+
			"\n\t(Preimage: %v)"+
			"\nReceived results: (has: %v) "+
			"\n\t(forME: %v)"+
			"\n\t(Preimage: %v)", true, true, preimages[0],
			has, forMe, receivedPreImage)
	}

	// Check with wrong identity (has should be true, for me false)
	has, forMe, _ = s.Check(identities[1], testFp, testMsg)
	if !has || forMe {
		t.Errorf("Unexpected results from check."+
			"\nExpected results: (has: %v)"+
			"\n\t(ForMe %v)"+
			"\nReceived results: "+
			"has: %v"+
			"\n\t(ForMe: %v)", true, false, has, forMe)
	}

}

func TestStore_save(t *testing.T) {
	// Initialize store
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

	// Save data to KV
	err := s.save()
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Manually pull from KV
	vo, err := s.kv.Get(edgeStoreKey, preimageStoreVersion)
	if err != nil {
		t.Fatalf("Failed to retrieve from KV: %v", err)
	}

	receivedIdentities := make([]id.ID, 0)
	err = json.Unmarshal(vo.Data, &receivedIdentities)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	for _, receivedId := range receivedIdentities {
		_, exists := s.Get(&receivedId)
		if !exists {
			t.Fatalf("Identity retrieved from store does not match " +
				"identity stored in")
		}
	}
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
