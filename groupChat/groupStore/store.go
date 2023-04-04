////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupStore

import (
	"bytes"
	"sync"
	"testing"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage values.
const (
	// Key used to identify the list of Groups in storage.
	groupStoragePrefix  = "GroupChatListStore"
	groupListStorageKey = "GroupChatList"
	groupListVersion    = 0
)

// Error messages.
const (
	kvGetGroupListErr = "failed to get list of group IDs from storage: %+v"
	groupLoadErr      = "failed to load group %d/%d: %+v"
	groupSaveErr      = "failed to save group %s to storage: %+v"
	maxGroupsErr      = "failed to add new group, max number of groups (%d) reached"
	groupExistsErr    = "group with ID %s already exists"
	groupRemoveErr    = "failed to remove group with ID %s, group not found in memory"
	saveListRemoveErr = "failed to save new group ID list after removing group %s"
	setUserPanic      = "Store.SetUser is for testing only. Got %T"
)

// MaxGroupChats is the maximum number of group chats that a user can be a part
// of at once.
const MaxGroupChats = 64

// Store stores the list of Groups that a user is a part of.
type Store struct {
	list map[id.ID]Group
	user group.Member
	kv   versioned.KV
	mux  sync.RWMutex
}

// NewStore constructs a new Store object for the user and saves it to storage.
func NewStore(kv versioned.KV, user group.Member) (*Store, error) {
	kv, err := kv.Prefix(groupStoragePrefix)
	if err != nil {
		return nil, err
	}
	s := &Store{
		list: make(map[id.ID]Group),
		user: user.DeepCopy(),
		kv:   kv,
	}

	return s, s.save()
}

// NewOrLoadStore loads the group store from storage or makes a new one if it
// does not exist.
func NewOrLoadStore(kv versioned.KV, user group.Member) (*Store, error) {
	prefixKv, err := kv.Prefix(groupStoragePrefix)
	if err != nil {
		return nil, err
	}

	// Load the list of group IDs from file if they exist
	vo, err := prefixKv.Get(groupListStorageKey, groupListVersion)
	if err == nil {
		return loadStore(vo.Data, prefixKv, user)
	}

	// If there is no group list saved, then make a new one
	return NewStore(kv, user)
}

// LoadStore loads all the Groups from storage into memory and return them in
// a Store object.
func LoadStore(kv versioned.KV, user group.Member) (*Store, error) {
	kv, err := kv.Prefix(groupStoragePrefix)
	if err != nil {
		return nil, err
	}

	// Load the list of group IDs from file
	vo, err := kv.Get(groupListStorageKey, groupListVersion)
	if err != nil {
		return nil, errors.Errorf(kvGetGroupListErr, err)
	}

	return loadStore(vo.Data, kv, user)
}

// loadStore builds the list of group IDs and loads the groups from storage.
func loadStore(data []byte, kv versioned.KV, user group.Member) (*Store, error) {
	// Deserialize list of group IDs
	groupIDs := deserializeGroupIdList(data)

	// Initialize the Store
	s := &Store{
		list: make(map[id.ID]Group, len(groupIDs)),
		user: user.DeepCopy(),
		kv:   kv,
	}

	// Load each Group from storage into the map
	for i, grpID := range groupIDs {
		grp, err := loadGroup(grpID, kv)
		if err != nil {
			return nil, errors.Errorf(groupLoadErr, i, len(grpID), err)
		}
		s.list[*grpID] = grp
	}

	return s, nil
}

// saveGroupList saves a list of group IDs to storage.
func (s *Store) saveGroupList() error {
	// Create the versioned object
	obj := &versioned.Object{
		Version:   groupListVersion,
		Timestamp: netTime.Now(),
		Data:      serializeGroupIdList(s.list),
	}

	// Save to storage
	return s.kv.Set(groupListStorageKey, obj)
}

// serializeGroupIdList serializes the list of group IDs.
func serializeGroupIdList(list map[id.ID]Group) []byte {
	buff := bytes.NewBuffer(nil)
	buff.Grow(id.ArrIDLen * len(list))

	// Create list of IDs from map
	for grpId := range list {
		buff.Write(grpId.Marshal())
	}

	return buff.Bytes()
}

// deserializeGroupIdList deserializes data into a list of group IDs.
func deserializeGroupIdList(data []byte) []*id.ID {
	idLen := id.ArrIDLen
	groupIDs := make([]*id.ID, 0, len(data)/idLen)
	buff := bytes.NewBuffer(data)

	// Copy each set of data into a new ID and append to list
	for n := buff.Next(idLen); len(n) == idLen; n = buff.Next(idLen) {
		var newID id.ID
		copy(newID[:], n)
		groupIDs = append(groupIDs, &newID)
	}

	return groupIDs
}

// save saves the group ID list and each group individually to storage.
func (s *Store) save() error {
	// Store group ID list
	err := s.saveGroupList()
	if err != nil {
		return err
	}

	// Store individual groups
	for grpID, grp := range s.list {
		if err := grp.store(s.kv); err != nil {
			return errors.Errorf(groupSaveErr, grpID, err)
		}
	}

	return nil
}

// Len returns the number of groups stored.
func (s *Store) Len() int {
	s.mux.RLock()
	defer s.mux.RUnlock()

	return len(s.list)
}

// Add adds a new group to the group list and saves it to storage. An error is
// returned if the user has the max number of groups (MaxGroupChats).
func (s *Store) Add(g Group) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Check if the group list is full.
	if len(s.list) >= MaxGroupChats {
		return errors.Errorf(maxGroupsErr, MaxGroupChats)
	}

	// Return an error if the group already exists in the map
	if _, exists := s.list[*g.ID]; exists {
		return errors.Errorf(groupExistsErr, g.ID)
	}

	// Add the group to the map
	s.list[*g.ID] = g.DeepCopy()

	// Update the group list in storage
	err := s.saveGroupList()
	if err != nil {
		return err
	}

	// Store the group to storage
	return g.store(s.kv)
}

// Remove removes the group with the corresponding ID from memory and storage.
// An error is returned if the group cannot be found in memory or storage.
func (s *Store) Remove(groupID *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Exit if the Group does not exist in memory
	if _, exists := s.list[*groupID]; !exists {
		return errors.Errorf(groupRemoveErr, groupID)
	}

	// Delete Group from memory
	delete(s.list, *groupID)

	// Remove group ID from list in memory
	err := s.saveGroupList()
	if err != nil {
		return errors.Errorf(saveListRemoveErr, groupID)
	}

	// Delete Group from storage
	return removeGroup(groupID, s.kv)
}

// GroupIDs returns a list of all group IDs.
func (s *Store) GroupIDs() []*id.ID {
	s.mux.RLock()
	defer s.mux.RUnlock()

	idList := make([]*id.ID, 0, len(s.list))
	for gid := range s.list {
		idList = append(idList, gid.DeepCopy())
	}

	return idList
}

// Groups returns a list of all groups.
func (s *Store) Groups() []Group {
	s.mux.RLock()
	defer s.mux.RUnlock()

	groupList := make([]Group, 0, len(s.list))
	for _, g := range s.list {
		groupList = append(groupList, g)
	}

	return groupList
}

// Get returns the Group for the given group ID. Returns false if no Group is
// found.
func (s *Store) Get(groupID *id.ID) (Group, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	grp, exists := s.list[*groupID]
	if !exists {
		return Group{}, false
	}

	return grp.DeepCopy(), exists
}

// GetByKeyFp returns the group with the matching key fingerprint and salt.
// Returns false if no group is found.
func (s *Store) GetByKeyFp(keyFp format.Fingerprint, salt [group.SaltLen]byte) (
	Group, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Iterate through each group to check if the key fingerprint matches
	for _, grp := range s.list {
		if group.CheckKeyFingerprint(keyFp, grp.Key, salt, s.user.ID) {
			return grp.DeepCopy(), true
		}
	}

	return Group{}, false
}

// GetUser returns the group member for the current user.
func (s *Store) GetUser() group.Member {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.user.DeepCopy()
}

// SetUser allows a user to be set. This function is for testing purposes only.
// It panics if the interface is not of a testing type.
func (s *Store) SetUser(user group.Member, x interface{}) {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf(setUserPanic, x)
	}

	s.mux.Lock()
	defer s.mux.Unlock()
	s.user = user.DeepCopy()
}
