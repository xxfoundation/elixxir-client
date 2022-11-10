////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupStore

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Unit test of NewStore.
func TestNewStore(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	expectedStore := &Store{
		list: make(map[id.ID]Group),
		user: user,
		kv:   kv.Prefix(groupStoragePrefix),
	}

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("NewStore returned an error: %+v", err)
	}

	// Compare manually created object with NewUnknownRoundsStore
	if !reflect.DeepEqual(expectedStore, store) {
		t.Errorf("NewStore returned incorrect Store."+
			"\nexpected: %+v\nreceived: %+v", expectedStore, store)
	}

	// Add information in store
	testGroup := createTestGroup(prng, t)

	store.list[*testGroup.ID] = testGroup

	if err := store.save(); err != nil {
		t.Fatalf("save() could not write to disk: %+v", err)
	}

	groupIds := make([]id.ID, 0, len(store.list))
	for grpId := range store.list {
		groupIds = append(groupIds, grpId)
	}

	// Check that stored group ID list is expected value
	expectedData := serializeGroupIdList(store.list)

	obj, err := store.kv.Get(groupListStorageKey, groupListVersion)
	if err != nil {
		t.Errorf("Could not get group list: %+v", err)
	}

	// Check that the stored data is the data outputted by marshal
	if !bytes.Equal(expectedData, obj.Data) {
		t.Errorf("NewStore() returned incorrect Store."+
			"\nexpected: %+v\nreceived: %+v", expectedData, obj.Data)
	}

	obj, err = store.kv.Get(groupStoreKey(testGroup.ID), groupListVersion)
	if err != nil {
		t.Errorf("Could not get group: %+v", err)
	}

	newGrp, err := DeserializeGroup(obj.Data)
	if err != nil {
		t.Errorf("Failed to deserialize group: %+v", err)
	}

	if !reflect.DeepEqual(testGroup, newGrp) {
		t.Errorf("NewStore() returned incorrect Store."+
			"\nexpected: %#v\nreceived: %#v", testGroup, newGrp)
	}
}

func TestNewOrLoadStore(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	store, err := NewOrLoadStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to create new store: %+v", err)
	}

	// Add group to store
	testGroup := createTestGroup(prng, t)
	if err = store.Add(testGroup); err != nil {
		t.Fatalf("Failed to add test group: %+v", err)
	}

	// Load the store from kv
	receivedStore, err := NewOrLoadStore(kv, user)
	if err != nil {
		t.Fatalf("LoadStore returned an error: %+v", err)
	}

	// Check that state in loaded store matches store that was saved
	if len(receivedStore.list) != len(store.list) {
		t.Errorf("LoadStore returned Store with incorrect number of groups."+
			"\nexpected len: %d\nreceived len: %d",
			len(store.list), len(receivedStore.list))
	}

	if _, exists := receivedStore.list[*testGroup.ID]; !exists {
		t.Fatalf("Failed to get group from loaded group map."+
			"\nexpected: %#v\nreceived: %#v", testGroup, receivedStore.list)
	}
}

// Unit test of LoadStore.
func TestLoadStore(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to create new store: %+v", err)
	}

	// Add group to store
	testGroup := createTestGroup(prng, t)
	if err = store.Add(testGroup); err != nil {
		t.Fatalf("Failed to add test group: %+v", err)
	}

	// Load the store from kv
	receivedStore, err := LoadStore(kv, user)
	if err != nil {
		t.Fatalf("LoadStore returned an error: %+v", err)
	}

	// Check that state in loaded store matches store that was saved
	if len(receivedStore.list) != len(store.list) {
		t.Errorf("LoadStore returned Store with incorrect number of groups."+
			"\nexpected len: %d\nreceived len: %d",
			len(store.list), len(receivedStore.list))
	}

	if _, exists := receivedStore.list[*testGroup.ID]; !exists {
		t.Fatalf("Failed to get group from loaded group map."+
			"\nexpected: %#v\nreceived: %#v", testGroup, receivedStore.list)
	}
}

// Error path: show that LoadStore returns an error when no group store can be
// found in storage.
func TestLoadStore_GetError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(rand.New(rand.NewSource(42)))
	expectedErr := strings.SplitN(kvGetGroupListErr, "%", 2)[0]

	// Load the store from kv
	_, err := LoadStore(kv, user)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LoadStore did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: show that loadStore returns an error when no group can be found
// in storage.
func Test_loadStore_GetGroupError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(rand.New(rand.NewSource(42)))
	var idList []byte
	for i := 0; i < 10; i++ {
		idList = append(idList, id.NewIdFromUInt(uint64(i), id.Group, t).Marshal()...)
	}
	expectedErr := strings.SplitN(groupLoadErr, "%", 2)[0]

	// Load the groups from kv
	_, err := loadStore(idList, kv, user)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadStore did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

}

// Tests that a map of groups can be serialized and deserialized into a list
// that has the same group IDs.
func Test_serializeGroupIdList_deserializeGroupIdList(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	n := 10
	testMap := make(map[id.ID]Group, n)
	expected := make([]*id.ID, n)
	for i := 0; i < n; i++ {
		grp := createTestGroup(prng, t)
		expected[i] = grp.ID
		testMap[*grp.ID] = grp
	}

	// Serialize and deserialize map
	data := serializeGroupIdList(testMap)
	newList := deserializeGroupIdList(data)

	// Sort expected and received lists so that they are in the same order
	sort.Slice(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].Bytes(), expected[j].Bytes()) == -1
	})
	sort.Slice(newList, func(i, j int) bool {
		return bytes.Compare(newList[i].Bytes(), newList[j].Bytes()) == -1
	})

	// Check if they match
	if !reflect.DeepEqual(expected, newList) {
		t.Errorf("Failed to serialize and deserilize group map into list."+
			"\nexpected: %+v\nreceived: %+v", expected, newList)
	}
}

// Unit test of Store.Len.
func TestStore_Len(t *testing.T) {
	s := Store{list: make(map[id.ID]Group)}

	if s.Len() != 0 {
		t.Errorf("Len returned the wrong length.\nexpected: %d\nreceived: %d",
			0, s.Len())
	}

	n := 10
	for i := 0; i < n; i++ {
		s.list[*id.NewIdFromUInt(uint64(i), id.Group, t)] = Group{}
	}

	if s.Len() != n {
		t.Errorf("Len returned the wrong length.\nexpected: %d\nreceived: %d",
			n, s.Len())
	}
}

// Unit test of Store.Add.
func TestStore_Add(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to create store: %+v", err)
	}

	// Add maximum number of groups allowed
	for i := 0; i < MaxGroupChats; i++ {
		// Add group to store
		grp := createTestGroup(prng, t)
		err = store.Add(grp)
		if err != nil {
			t.Errorf("Add returned an error (%d): %v", i, err)
		}

		if _, exists := store.list[*grp.ID]; !exists {
			t.Errorf("Group %s was not added to the map (%d)", grp.ID, i)
		}
	}

	if len(store.list) != MaxGroupChats {
		t.Errorf("Length of group map does not match number of groups added."+
			"\nexpected: %d\nreceived: %d", MaxGroupChats, len(store.list))
	}
}

// Error path: shows that an error is returned when trying to add too many
// groups.
func TestStore_Add_MapFullError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)
	expectedErr := strings.SplitN(maxGroupsErr, "%", 2)[0]

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to create store: %+v", err)
	}

	// Add maximum number of groups allowed
	for i := 0; i < MaxGroupChats; i++ {
		err = store.Add(createTestGroup(prng, t))
		if err != nil {
			t.Errorf("Add returned an error (%d): %v", i, err)
		}
	}

	err = store.Add(createTestGroup(prng, t))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Add did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: show Store.Add returns an error when attempting to add a group
// that is already in the map.
func TestStore_Add_GroupExistsError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)
	expectedErr := strings.SplitN(groupExistsErr, "%", 2)[0]

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to create store: %+v", err)
	}

	grp := createTestGroup(prng, t)
	err = store.Add(grp)
	if err != nil {
		t.Errorf("Add returned an error: %+v", err)
	}

	err = store.Add(grp)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Add did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of Store.Remove.
func TestStore_Remove(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to create store: %+v", err)
	}

	// Add maximum number of groups allowed
	groups := make([]Group, MaxGroupChats)
	for i := 0; i < MaxGroupChats; i++ {
		groups[i] = createTestGroup(prng, t)
		if err = store.Add(groups[i]); err != nil {
			t.Errorf("Failed to add group (%d): %v", i, err)
		}
	}

	// Remove all groups
	for i, grp := range groups {
		err = store.Remove(grp.ID)
		if err != nil {
			t.Errorf("Remove returned an error (%d): %+v", i, err)
		}

		if _, exists := store.list[*grp.ID]; exists {
			t.Fatalf("Group %s still exists in map (%d).", grp.ID, i)
		}
	}

	// Check that the list is empty now
	if len(store.list) != 0 {
		t.Fatalf("Remove failed to remove all groups.."+
			"\nexpected: %d\nreceived: %d", 0, len(store.list))
	}
}

// Error path: shows that Store.Remove returns an error when no group with the
// given ID is found in the map.
func TestStore_Remove_RemoveGroupNotInMemoryError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)
	expectedErr := strings.SplitN(groupRemoveErr, "%", 2)[0]

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to create store: %+v", err)
	}

	grp := createTestGroup(prng, t)
	err = store.Remove(grp.ID)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Remove did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of Store.GroupIDs.
func TestStore_GroupIDs(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	n := 10
	store := Store{list: make(map[id.ID]Group, n)}
	expected := make([]*id.ID, n)
	for i := 0; i < n; i++ {
		grp := createTestGroup(prng, t)
		expected[i] = grp.ID
		store.list[*grp.ID] = grp
	}

	newList := store.GroupIDs()

	// Sort expected and received lists so that they are in the same order
	sort.Slice(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].Bytes(), expected[j].Bytes()) == -1
	})
	sort.Slice(newList, func(i, j int) bool {
		return bytes.Compare(newList[i].Bytes(), newList[j].Bytes()) == -1
	})

	// Check if they match
	if !reflect.DeepEqual(expected, newList) {
		t.Errorf("GroupIDs did not return the expected list."+
			"\nexpected: %+v\nreceived: %+v", expected, newList)
	}
}

// Tests that Store.Groups returns a list with all the groups.
func TestStore_Groups(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	store := Store{list: make(map[id.ID]Group, 10)}
	expected := make([]Group, len(store.list))
	for i := range expected {
		grp := createTestGroup(prng, t)
		expected[i] = grp
		store.list[*grp.ID] = grp
	}

	groups := store.Groups()

	sort.Slice(expected, func(i, j int) bool {
		return bytes.Compare(expected[i].ID[:], expected[j].ID[:]) == -1
	})
	sort.Slice(groups, func(i, j int) bool {
		return bytes.Compare(groups[i].ID[:], groups[j].ID[:]) == -1
	})

	if !reflect.DeepEqual(expected, groups) {
		t.Errorf("List of Groups does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, groups)
	}
}

// Unit test of Store.Get.
func TestStore_Get(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to make new Store: %+v", err)
	}

	// Add group to store
	grp := createTestGroup(prng, t)
	if err = store.Add(grp); err != nil {
		t.Errorf("Failed to add group to store: %+v", err)
	}

	// Attempt to get group
	retrieved, exists := store.Get(grp.ID)
	if !exists {
		t.Errorf("get failed to return the expected group: %#v", grp)
	}

	if !reflect.DeepEqual(grp, retrieved) {
		t.Errorf("get did not return the expected group."+
			"\nexpected: %#v\nreceived: %#v", grp, retrieved)
	}
}

// Error path: shows that Store.Get return false if no group is found.
func TestStore_Get_NoGroupError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(rand.New(rand.NewSource(42)))

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to make new Store: %+v", err)
	}

	// Attempt to get group
	retrieved, exists := store.Get(id.NewIdFromString("testID", id.Group, t))
	if exists {
		t.Errorf("get returned a group that should not exist: %#v", retrieved)
	}
}

// Unit test of Store.GetByKeyFp.
func TestStore_GetByKeyFp(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to make new Store: %+v", err)
	}

	// Add group to store
	grp := createTestGroup(prng, t)
	if err = store.Add(grp); err != nil {
		t.Fatalf("Failed to add group: %+v", err)
	}

	// get group by fingerprint
	salt := newSalt(groupSalt)
	generatedFP := group.NewKeyFingerprint(grp.Key, salt, store.user.ID)
	retrieved, exists := store.GetByKeyFp(generatedFP, salt)
	if !exists {
		t.Errorf("GetByKeyFp failed to find a group with the matching key "+
			"fingerprint: %#v", grp)
	}

	// check that retrieved value match
	if !reflect.DeepEqual(grp, retrieved) {
		t.Errorf("GetByKeyFp failed to return the expected group."+
			"\nexpected: %#v\nreceived: %#v", grp, retrieved)
	}
}

// Error path: shows that Store.GetByKeyFp return false if no group is found.
func TestStore_GetByKeyFp_NoGroupError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(prng)

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to make new Store: %+v", err)
	}

	// get group by fingerprint
	grp := createTestGroup(prng, t)
	salt := newSalt(groupSalt)
	generatedFP := group.NewKeyFingerprint(grp.Key, salt, store.user.ID)
	retrieved, exists := store.GetByKeyFp(generatedFP, salt)
	if exists {
		t.Errorf("GetByKeyFp found a group when none should exist: %#v",
			retrieved)
	}
}

// Unit test of Store.GetUser.
func TestStore_GetUser(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := randMember(rand.New(rand.NewSource(42)))

	store, err := NewStore(kv, user)
	if err != nil {
		t.Fatalf("Failed to make new Store: %+v", err)
	}

	if !user.Equal(store.GetUser()) {
		t.Errorf("GetTransmissionIdentity() failed to return the expected member."+
			"\nexpected: %#v\nreceived: %#v", user, store.GetUser())
	}
}

// Unit test of Store.SetUser.
func TestStore_SetUser(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	prng := rand.New(rand.NewSource(42))
	oldUser := randMember(prng)
	newUser := randMember(prng)

	store, err := NewStore(kv, oldUser)
	if err != nil {
		t.Fatalf("Failed to make new Store: %+v", err)
	}

	store.SetUser(newUser, t)

	if !newUser.Equal(store.user) {
		t.Errorf("SetUser() failed to set the correct user."+
			"\nexpected: %#v\nreceived: %#v", newUser, store.user)
	}
}

// Panic path: show that Store.SetUser panics when the interface is not of a
// testing type.
func TestStore_SetUser_NonTestingInterfacePanic(t *testing.T) {
	user := randMember(rand.New(rand.NewSource(42)))
	store := &Store{}
	nonTestingInterface := struct{}{}
	expectedErr := fmt.Sprintf(setUserPanic, nonTestingInterface)

	defer func() {
		if r := recover(); r == nil || r.(string) != expectedErr {
			t.Errorf("SetUser failed to panic with the expected message."+
				"\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	}()

	store.SetUser(user, nonTestingInterface)
}
