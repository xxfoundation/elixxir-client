////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receptionID

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestNewOrLoadStore_New(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &Store{
		active: make([]*registration, 0),
		kv:     kv,
	}

	s := NewOrLoadStore(kv)

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

func TestNewOrLoadStore_Load(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
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

	testStore := NewOrLoadStore(kv)
	for i, active := range testStore.active {
		if !s.active[i].Equal(active.Identity) {
			t.Errorf("Failed to generate expected Store."+
				"\nexpected: %#v\nreceived: %#v", s.active[i], active)
		}
	}
}

func TestStore_save(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
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
	s := NewOrLoadStore(versioned.NewKV(ekv.MakeMemstore()))
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

func TestStore_GetIdentities(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
	prng := rand.New(rand.NewSource(42))

	numToTest := 100

	idsGenerated := make(map[uint64]interface{})

	for i := 0; i < numToTest; i++ {
		testID, err := generateFakeIdentity(prng, 15, netTime.Now())
		if err != nil {
			t.Fatalf("Failed to generate fake ID: %+v", err)
		}
		testID.Fake = false
		if s.AddIdentity(testID.Identity) != nil {
			t.Errorf("AddIdentity() produced an error: %+v", err)
		}

		idsGenerated[getIDFp(testID.EphemeralIdentity)] = nil

	}

	//get one
	var idu []IdentityUse
	o := func(a []IdentityUse) error {
		idu = a
		return nil
	}
	err := s.ForEach(1, prng, 15, o)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	if _, exists := idsGenerated[getIDFp(idu[0].EphemeralIdentity)]; !exists ||
		idu[0].Fake {
		t.Errorf("An unknown or fake identity was returned")
	}

	//get three
	err = s.ForEach(3, prng, 15, o)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	if len(idu) != 3 {
		t.Errorf("the wrong number of identities was returned")
	}

	for i := 0; i < len(idu); i++ {
		if _, exists := idsGenerated[getIDFp(idu[i].EphemeralIdentity)]; !exists ||
			idu[i].Fake {
			t.Errorf("An unknown or fake identity was returned")
		}
	}

	//get ten
	err = s.ForEach(10, prng, 15, o)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	if len(idu) != 10 {
		t.Errorf("the wrong number of identities was returned")
	}

	for i := 0; i < len(idu); i++ {
		if _, exists := idsGenerated[getIDFp(idu[i].EphemeralIdentity)]; !exists ||
			idu[i].Fake {
			t.Errorf("An unknown or fake identity was returned")
		}
	}

	//get fifty
	err = s.ForEach(50, prng, 15, o)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	if len(idu) != 50 {
		t.Errorf("the wrong number of identities was returned")
	}

	for i := 0; i < len(idu); i++ {
		if _, exists := idsGenerated[getIDFp(idu[i].EphemeralIdentity)]; !exists ||
			idu[i].Fake {
			t.Errorf("An unknown or fake identity was returned")
		}
	}

	//get 100
	err = s.ForEach(100, prng, 15, o)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	if len(idu) != 100 {
		t.Errorf("the wrong number of identities was returned")
	}

	for i := 0; i < len(idu); i++ {
		if _, exists := idsGenerated[getIDFp(idu[i].EphemeralIdentity)]; !exists ||
			idu[i].Fake {
			t.Errorf("An unknown or fake identity was returned")
		}
	}

	//get 1000, should only return 100
	err = s.ForEach(1000, prng, 15, o)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	if len(idu) != 100 {
		t.Errorf("the wrong number of identities was returned")
	}

	for i := 0; i < len(idu); i++ {
		if _, exists := idsGenerated[getIDFp(idu[i].EphemeralIdentity)]; !exists ||
			idu[i].Fake {
			t.Errorf("An unknown or fake identity was returned")
		}
	}

	// get 100 a second time and make sure the order is not the same as a
	// smoke test that the shuffle is working
	var idu2 []IdentityUse
	o2 := func(a []IdentityUse) error {
		idu2 = a
		return nil
	}

	err = s.ForEach(1000, prng, 15, o2)
	if err != nil {
		t.Errorf("GetIdentity() produced an error: %+v", err)
	}

	diferent := false
	for i := 0; i < len(idu); i++ {
		if !idu[i].Source.Cmp(idu2[i].Source) {
			diferent = true
			break
		}
	}

	if !diferent {
		t.Errorf("The 2 100 shuffels retruned the same result, shuffling" +
			" is likley not occuring")
	}

}

func TestStore_GetIdentities_NoIdentities(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
	prng := rand.New(rand.NewSource(42))

	var idu []IdentityUse
	o := func(a []IdentityUse) error {
		idu = a
		return nil
	}

	err := s.ForEach(5, prng, 15, o)
	if err != nil {
		t.Errorf("GetIdentities() produced an error: %+v", err)
	}

	if len(idu) != 1 {
		t.Errorf("GetIdenties() did not return only one identity " +
			"when looking for a fake")
	}

	if !idu[0].Fake {
		t.Errorf("GetIdenties() did not return a fake identity " +
			"when only one is avalible")
	}
}

func TestStore_GetIdentities_BadNum(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
	prng := rand.New(rand.NewSource(42))

	o := func(a []IdentityUse) error {
		return nil
	}

	err := s.ForEach(0, prng, 15, o)
	if err == nil {
		t.Errorf("GetIdentities() shoud error with bad num value")
	}

	err = s.ForEach(-1, prng, 15, o)
	if err == nil {
		t.Errorf("GetIdentities() shoud error with bad num value")
	}

	err = s.ForEach(-100, prng, 15, o)
	if err == nil {
		t.Errorf("GetIdentities() shoud error with bad num value")
	}

	err = s.ForEach(-1000000, prng, 15, o)
	if err == nil {
		t.Errorf("GetIdentities() shoud error with bad num value")
	}

	err = s.ForEach(math.MinInt64, prng, 15, o)
	if err == nil {
		t.Errorf("GetIdentities() shoud error with bad num value")
	}
}

func getIDFp(identity EphemeralIdentity) uint64 {
	h, _ := hash.NewCMixHash()
	h.Write(identity.EphId[:])
	h.Write(identity.Source.Bytes())
	r := h.Sum(nil)
	return binary.BigEndian.Uint64(r)
}

func TestStore_AddIdentity(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
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
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
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
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
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
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewOrLoadStore(kv)
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
