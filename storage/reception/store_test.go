package reception

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expected := &Store{
		active: make([]*registration, 0),
		kv:     kv,
	}

	s := NewStore(kv)

	if !reflect.DeepEqual([]*registration{}, s.active) {
		t.Errorf("NewStore() failed to return the expected Store."+
			"\nexpected: %+v\nreceived: %+v", expected, s)
	}

	obj, err := s.kv.Get(receptionStoreStorageKey, 0)
	if err != nil {
		t.Fatalf("Failed to load store from KV: %+v", err)
	}

	var testStoredReference []storedReference
	err = json.Unmarshal(obj.Data, &testStoredReference)
	if err != nil {
		t.Errorf("Failed to unmarshal []storedReference: %+v", err)
	}
	if !reflect.DeepEqual([]storedReference{}, testStoredReference) {
		t.Errorf("Failed to retreive expected storedReference from KV."+
			"\nexpected: %#v\nreceived: %#v", []storedReference{}, testStoredReference)
	}
}

func TestLoadStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	s := NewStore(kv)
	prng := rand.New(rand.NewSource(42))

	// Fill active registration with fake identities
	for i := 0; i < 10; i++ {
		testID, err := generateFakeIdentity(prng, 15, netTime.Now())
		if err != nil {
			t.Fatalf("Failed to generate fake ID: %+v", err)
		}
		testID.Ephemeral = false
		if s.AddIdentity(testID.Identity) != nil {
			t.Fatalf("Failed to AddIdentity: %+v", err)
		}
	}

	err := s.save()
	if err != nil {
		t.Errorf("save() produced an error: %+v", err)
	}

	testStore := LoadStore(kv)
	for i, active := range testStore.active {
		if !s.active[i].Equal(active.Identity) {
			t.Errorf("Failed to generate expected Store."+
				"\nexpected: %#v\nreceived: %#v", s.active[i], active)
		}
	}
}

func TestStore_save(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	s := NewStore(kv)
	prng := rand.New(rand.NewSource(42))

	// Fill active registration with fake identities
	for i := 0; i < 10; i++ {
		testID, err := generateFakeIdentity(prng, 15, netTime.Now())
		if err != nil {
			t.Fatalf("Failed to generate fake ID: %+v", err)
		}
		testID.Ephemeral = false
		s.active = append(s.active, &registration{Identity: testID.Identity})
	}

	expected := s.makeStoredReferences()

	err := s.save()
	if err != nil {
		t.Errorf("save() produced an error: %+v", err)
	}

	obj, err := kv.Prefix(receptionPrefix).Get(receptionStoreStorageKey, 0)
	if err != nil {
		t.Errorf("get() produced an error: %+v", err)
	}

	expectedData, err := json.Marshal(expected)
	if obj.Version != receptionStoreStorageVersion {
		t.Errorf("Rectrieved version incorrect.\nexpected: %d\nreceived: %d",
			receptionStoreStorageVersion, obj.Version)
	}

	if !bytes.Equal(expectedData, obj.Data) {
		t.Errorf("Rectrieved data incorrect.\nexpected: %s\nreceived: %s",
			expectedData, obj.Data)
	}
}

func TestStore_makeStoredReferences(t *testing.T) {
	s := NewStore(versioned.NewKV(make(ekv.Memstore)))
	prng := rand.New(rand.NewSource(42))
	expected := make([]storedReference, 0)

	// Fill active registration with fake identities
	for i := 0; i < 10; i++ {
		testID, err := generateFakeIdentity(prng, 15, netTime.Now())
		if err != nil {
			t.Fatalf("Failed to generate fake ID: %+v", err)
		}
		if i%2 == 0 {
			testID.Ephemeral = false
			expected = append(expected, storedReference{
				Eph:        testID.EphId,
				Source:     testID.Source,
				StartValid: testID.StartValid.Round(0),
			})
		}
		s.active = append(s.active, &registration{Identity: testID.Identity})
	}

	sr := s.makeStoredReferences()
	if !reflect.DeepEqual(expected, sr) {
		t.Errorf("Failed to generate expected list of identities."+
			"\nexpected: %+v\nreceived: %+v", expected, sr)
	}
}

func TestStore_GetIdentity(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	s := NewStore(kv)
	prng := rand.New(rand.NewSource(42))
	testID, err := generateFakeIdentity(prng, 15, netTime.Now())
	if err != nil {
		t.Fatalf("Failed to generate fake ID: %+v", err)
	}
	if s.AddIdentity(testID.Identity) != nil {
		t.Errorf("AddIdentity() produced an error: %+v", err)
	}

	idu, err := s.GetIdentity(prng, 15)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	if !testID.Equal(idu.Identity) {
		t.Errorf("GetIdentity() did not return the expected Identity."+
			"\nexpected: %s\nreceived: %s", testID, idu)
	}
}

func TestStore_AddIdentity(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	s := NewStore(kv)
	prng := rand.New(rand.NewSource(42))
	testID, err := generateFakeIdentity(prng, 15, netTime.Now())
	if err != nil {
		t.Fatalf("Failed to generate fake ID: %+v", err)
	}

	err = s.AddIdentity(testID.Identity)
	if err != nil {
		t.Errorf("AddIdentity() produced an error: %+v", err)
	}

	if !s.active[0].Identity.Equal(testID.Identity) {
		t.Errorf("Failed to get expected Identity.\nexpected: %s\nreceived: %s",
			testID.Identity, s.active[0])
	}
}

func TestStore_RemoveIdentity(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	s := NewStore(kv)
	prng := rand.New(rand.NewSource(42))
	testID, err := generateFakeIdentity(prng, 15, netTime.Now())
	if err != nil {
		t.Fatalf("Failed to generate fake ID: %+v", err)
	}
	err = s.AddIdentity(testID.Identity)
	if err != nil {
		t.Fatalf("AddIdentity() produced an error: %+v", err)
	}
	s.RemoveIdentity(testID.EphId)

	if len(s.active) != 0 {
		t.Errorf("RemoveIdentity() failed to remove: %+v", s.active)
	}
}

func TestStore_prune(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	s := NewStore(kv)
	prng := rand.New(rand.NewSource(42))
	runs := 10
	expected := make([]*registration, runs/2)

	for i := 0; i < runs; i++ {
		timestamp := netTime.Now()
		if i%2 == 0 {
			timestamp = timestamp.Add(24 * time.Hour)
		}
		testID, err := generateFakeIdentity(prng, 15, timestamp)
		if err != nil {
			t.Fatalf("Failed to generate fake ID: %+v", err)
		}
		err = s.AddIdentity(testID.Identity)
		if err != nil {
			t.Fatalf("AddIdentity() produced an error: %+v", err)
		}
		if i%2 == 0 {
			expected[i/2] = s.active[i]
		}
	}

	s.prune(netTime.Now().Add(24 * time.Hour))

	for i, reg := range s.active {
		if !reg.Equal(expected[i].Identity) {
			t.Errorf("Unexpected identity (%d).\nexpected: %+v\nreceived: %+v",
				i, expected[i], reg)
		}
	}
}

func TestStore_selectIdentity(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	s := NewStore(kv)
	prng := rand.New(rand.NewSource(42))
	runs := 10
	expectedReg := make([]*registration, runs)

	for i := 0; i < runs; i++ {
		testID, err := generateFakeIdentity(prng, 15, netTime.Now())
		if err != nil {
			t.Fatalf("Failed to generate fake ID: %+v", err)
		}
		err = s.AddIdentity(testID.Identity)
		if err != nil {
			t.Fatalf("AddIdentity() produced an error: %+v", err)
		}
		expectedReg[i] = s.active[i]
	}

	for i := 0; i < runs; i++ {
		idu, err := s.selectIdentity(prng, netTime.Now())
		if err != nil {
			t.Errorf("selectIdentity() produced an error: %+v", err)
		}

		var found bool
		for _, expected := range expectedReg {
			if idu.Equal(expected.Identity) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected Identity returned.\nreceived: %+v", idu)
		}
	}
}
