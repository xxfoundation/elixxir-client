////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

// Error messages.
const (
	// mutedUserManager.save
	storeMutedUsersErr    = "could not store muted users for channel %s: %+v"
	storeMutedChannelsErr = "could not store muted channel IDs: %+v"

	// mutedUserManager.load
	loadMutedChannelsErr = "could not load list of muted channels: %+v"
	loadMutedUsersErr    = "could not load muted users for channel %s"
)

// mutedUserKey identifies a user in the muted user list for each channel. It is
// derived from a user's ed25519.PublicKey.
type mutedUserKey string

// mutedUserManager manages the list of muted users in each channel.
type mutedUserManager struct {
	// List of muted users in each channel. The internal map keys on
	// mutedUserKey (which is a string) because json.Marshal (which is used for
	// storage) requires the key be a string.
	list map[id.ID]map[mutedUserKey]struct{}

	mux sync.RWMutex
	kv  versioned.KV
}

// newOrLoadMutedUserManager loads an existing mutedUserManager from storage, if
// it exists. Otherwise, it initialises a new empty mutedUserManager.
func newOrLoadMutedUserManager(kv versioned.KV) (*mutedUserManager, error) {
	mum := newMutedUserManager(kv)

	err := mum.load()
	if err != nil && kv.Exists(err) {
		return nil, err
	}

	return mum, nil
}

// newMutedUserManager initializes a new and empty mutedUserManager.
func newMutedUserManager(kv versioned.KV) *mutedUserManager {
	return &mutedUserManager{
		list: make(map[id.ID]map[mutedUserKey]struct{}),
		kv:   kv,
	}
}

// muteUser adds the user to the muted list for the given channel.
func (mum *mutedUserManager) muteUser(
	channelID *id.ID, userPubKey ed25519.PublicKey) {
	mum.mux.Lock()
	defer mum.mux.Unlock()

	// Add the channel to the list if it does not exist
	var channelIdUpdate bool
	if _, exists := mum.list[*channelID]; !exists {
		mum.list[*channelID] = make(map[mutedUserKey]struct{})
		channelIdUpdate = true
	}

	// Add user to channel's mute list
	mum.list[*channelID][makeMutedUserKey(userPubKey)] = struct{}{}

	// Save to storage
	if err := mum.save(channelID, channelIdUpdate); err != nil {
		jww.FATAL.Panicf("[CH] Failed to save muted users: %+v", err)
	}
}

// unmuteUser removes the user from the muted list for the given channel.
func (mum *mutedUserManager) unmuteUser(
	channelID *id.ID, userPubKey ed25519.PublicKey) {
	mum.mux.Lock()
	defer mum.mux.Unlock()

	// Do nothing if the channel is not in the list
	mutedUsers, exists := mum.list[*channelID]
	if !exists {
		return
	}

	// Delete the user from the muted user list
	delete(mutedUsers, makeMutedUserKey(userPubKey))

	// If no more muted users exist for the channel, then delete the channel
	var channelIdUpdate bool
	if len(mutedUsers) == 0 {
		delete(mum.list, *channelID)
		channelIdUpdate = true
	}

	// Save to storage
	if err := mum.save(channelID, channelIdUpdate); err != nil {
		jww.FATAL.Panicf("[CH] Failed to save muted users: %+v", err)
	}
}

// isMuted returns true if the user is muted in the specified channel. Returns
// false if the user is not muted in the given channel.
func (mum *mutedUserManager) isMuted(
	channelID *id.ID, userPubKey ed25519.PublicKey) bool {
	mum.mux.RLock()
	defer mum.mux.RUnlock()

	// Return false if the channel is not in the list
	mutedUsers, exists := mum.list[*channelID]
	if !exists {
		return false
	}

	// Check if the user is in the list
	_, exists = mutedUsers[makeMutedUserKey(userPubKey)]
	return exists
}

// getMutedUsers returns a list of muted user's public keys for the given
// channel ID.
func (mum *mutedUserManager) getMutedUsers(channelID *id.ID) []ed25519.PublicKey {
	mum.mux.RLock()
	defer mum.mux.RUnlock()

	// Return false if the channel is not in the list
	mutedUsers, exists := mum.list[*channelID]
	if !exists {
		return []ed25519.PublicKey{}
	}

	userList := make([]ed25519.PublicKey, len(mutedUsers))
	i := 0
	for user := range mutedUsers {
		pubKey, err := user.decode()
		if err != nil {
			jww.ERROR.Printf("[CH] Could not decode user public key %d of %d "+
				"in channel %s: %+v", i+1, len(mutedUsers), channelID, err)
			continue
		}
		userList[i] = pubKey
		i++
	}

	// Return the list of muted users and truncate its length and capacity to
	// exclude users that could not be decoded
	return userList[:i:i]
}

// removeChannel deletes the muted user list for the given channel. This should
// only be called when leaving a channel
func (mum *mutedUserManager) removeChannel(channelID *id.ID) error {
	mum.mux.Lock()
	defer mum.mux.Unlock()

	if _, exists := mum.list[*channelID]; !exists {
		return nil
	}

	delete(mum.list, *channelID)

	err := mum.saveChannelList()
	if err != nil {
		return err
	}
	return mum.deleteMutedUsers(channelID)
}

// len returns the number of muted users in the specified channel.
func (mum *mutedUserManager) len(channelID *id.ID) int {
	mum.mux.RLock()
	defer mum.mux.RUnlock()
	return len(mum.list[*channelID])
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Storage values.
const (
	mutedChannelListStoreVer = 0
	mutedChannelListStoreKey = "mutedChannelList"
	mutedUserListStoreVer    = 0
	mutedUserListStorePrefix = "mutedUserList/"
)

// save stores the muted user list for the given channel ID to storage. If
// channelIdUpdate is true, then the main list of channel IDs is also updated.
func (mum *mutedUserManager) save(channelID *id.ID, channelIdUpdate bool) error {
	if err := mum.saveMutedUsers(channelID); err != nil {
		return errors.Errorf(storeMutedUsersErr, channelID, err)
	} else if channelIdUpdate {
		if err = mum.saveChannelList(); err != nil {
			return errors.Errorf(storeMutedChannelsErr, err)
		}
	}
	return nil
}

// load gets all the muted users from storage and loads them into the muted user
// list.
func (mum *mutedUserManager) load() error {
	// Get list of channel IDs
	channelIDs, err := mum.loadChannelList()
	if err != nil {
		return errors.Wrap(err, loadMutedChannelsErr)
	}

	// Get list of muted users for each channel and load them into the map
	for _, channelID := range channelIDs {
		channelList, err2 := mum.loadMutedUsers(channelID)
		if err2 != nil {
			return errors.Wrapf(err2, loadMutedUsersErr, channelID)
		}
		mum.list[*channelID] = channelList
	}

	return nil
}

// saveChannelList stores the list of channel IDs with muted users to storage.
func (mum *mutedUserManager) saveChannelList() error {
	channelIdList := make([]*id.ID, 0, len(mum.list))
	for channelID := range mum.list {
		chID := channelID
		channelIdList = append(channelIdList, &chID)
	}

	data, err := json.Marshal(channelIdList)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   mutedChannelListStoreVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return mum.kv.Set(mutedChannelListStoreKey, obj)
}

// loadChannelList retrieves the list of channel IDs with muted users from
// storage.
func (mum *mutedUserManager) loadChannelList() ([]*id.ID, error) {
	obj, err := mum.kv.Get(mutedChannelListStoreKey, mutedChannelListStoreVer)
	if err != nil {
		return nil, err
	}

	var channelIdList []*id.ID
	return channelIdList, json.Unmarshal(obj.Data, &channelIdList)
}

// saveMutedUsers stores the muted user list for the given channel to storage.
func (mum *mutedUserManager) saveMutedUsers(channelID *id.ID) error {
	// If the list is empty, then delete it from storage
	if len(mum.list[*channelID]) == 0 {
		return mum.deleteMutedUsers(channelID)
	}

	data, err := json.Marshal(mum.list[*channelID])
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   mutedUserListStoreVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return mum.kv.Set(makeMutedChannelStoreKey(channelID), obj)
}

// loadMutedUsers retrieves the muted user list for the given channel from
// storage.
func (mum *mutedUserManager) loadMutedUsers(
	channelID *id.ID) (map[mutedUserKey]struct{}, error) {
	obj, err := mum.kv.Get(
		makeMutedChannelStoreKey(channelID), mutedChannelListStoreVer)
	if err != nil {
		return nil, err
	}

	var list map[mutedUserKey]struct{}
	return list, json.Unmarshal(obj.Data, &list)
}

// deleteMutedUsers deletes the muted user file for this channel ID from
// storage.
func (mum *mutedUserManager) deleteMutedUsers(channelID *id.ID) error {
	return mum.kv.Delete(
		makeMutedChannelStoreKey(channelID), mutedChannelListStoreVer)
}

// makeMutedUserKey generates a mutedUserKey from a user's [ed25519.PublicKey].
func makeMutedUserKey(pubKey ed25519.PublicKey) mutedUserKey {
	return mutedUserKey(hex.EncodeToString(pubKey[:]))
}

// decode decodes the mutedUserKey into an ed25519.PublicKey.
func (k mutedUserKey) decode() (ed25519.PublicKey, error) {
	data, err := hex.DecodeString(string(k))
	if err != nil {
		return nil, err
	}

	if len(data) != ed25519.PublicKeySize {
		return nil, errors.Errorf(
			"data must be %d bytes; received data of %d bytes",
			ed25519.PublicKeySize, len(data))
	}

	return data, nil
}

// makeMutedChannelStoreKey generates the key used to save and load a list of
// muted users for a specific channel from storage.
func makeMutedChannelStoreKey(channelID *id.ID) string {
	return mutedUserListStorePrefix +
		base64.StdEncoding.EncodeToString(channelID[:])
}
