////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Tests that newOrLoadMutedUserManager initialises a new empty mutedUserManager
// when called for the first time and that it loads the mutedUserManager from
// storage after the original has been saved.
func Test_newOrLoadMutedUserManager(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := newMutedUserManager(kv)

	mum, err := newOrLoadMutedUserManager(kv)
	if err != nil {
		t.Errorf("Failed to create new mutedUserManager: %+v", err)
	}

	if !reflect.DeepEqual(expected, mum) {
		t.Errorf("New mutedUserManager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, mum)
	}

	mum.muteUser(randChannelID(prng, t), makeEd25519PubKey(prng, t))

	loadedMum, err := newOrLoadMutedUserManager(kv)
	if err != nil {
		t.Errorf("Failed to load mutedUserManager: %+v", err)
	}

	if !reflect.DeepEqual(mum, loadedMum) {
		t.Errorf("Loaded mutedUserManager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", mum, loadedMum)
	}
}

// Tests that newMutedUserManager returns the new expected mutedUserManager.
func Test_newMutedUserManager(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &mutedUserManager{
		list: make(map[id.ID]map[mutedUserKey]struct{}),
		kv:   kv,
	}

	mum := newMutedUserManager(kv)

	if !reflect.DeepEqual(expected, mum) {
		t.Errorf("New mutedUserManager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, mum)
	}
}

// Tests that mutedUserManager.muteUser adds all the users to the list and that
// all the users are saved to storage.
func Test_mutedUserManager_muteUser(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	kv := versioned.NewKV(ekv.MakeMemstore())
	mum := newMutedUserManager(kv)

	expected := make(map[id.ID]map[mutedUserKey]struct{})

	for i := 0; i < 20; i++ {
		channelID := randChannelID(prng, t)
		expected[*channelID] = make(map[mutedUserKey]struct{})
		for j := 0; j < 50; j++ {
			pubKey := makeEd25519PubKey(prng, t)
			expected[*channelID][makeMutedUserKey(pubKey)] = struct{}{}
			mum.muteUser(channelID, pubKey)
		}
	}

	if !reflect.DeepEqual(expected, mum.list) {
		t.Errorf("User list does not match expected."+
			"\nexpected: %s\nreceived: %s", expected, mum.list)
	}

	newMum := newMutedUserManager(mum.kv)
	if err := newMum.load(); err != nil {
		t.Fatalf("Failed to load user list: %+v", err)
	}

	if !reflect.DeepEqual(expected, newMum.list) {
		t.Errorf("Loaded mutedUserManager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, newMum.list)
	}
}

// Tests that mutedUserManager.unmuteUser removes all muted users from the list
// and that all the users are removed from storage.
func Test_mutedUserManager_unmuteUser(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	kv := versioned.NewKV(ekv.MakeMemstore())
	mum := newMutedUserManager(kv)

	expected := make(map[id.ID]map[mutedUserKey]ed25519.PublicKey)

	for i := 0; i < 20; i++ {
		channelID := randChannelID(prng, t)
		expected[*channelID] = make(map[mutedUserKey]ed25519.PublicKey)
		for j := 0; j < 50; j++ {
			pubKey := makeEd25519PubKey(prng, t)
			expected[*channelID][makeMutedUserKey(pubKey)] = pubKey
			mum.muteUser(channelID, pubKey)
		}
	}
	for channelID, mutedUsers := range expected {
		for key, pubKey := range mutedUsers {
			mum.unmuteUser(&channelID, pubKey)

			if _, exists := mum.list[channelID][key]; exists {
				t.Errorf("User %s not removed from list.", key)
			}
		}
	}

	if len(mum.list) != 0 {
		t.Errorf(
			"%d not removed from list: %v", len(mum.list), mum.list)
	}

	newMum := newMutedUserManager(mum.kv)
	if err := newMum.load(); err != nil {
		t.Fatalf("Failed to load user list: %+v", err)
	}

	if len(newMum.list) != 0 {
		t.Errorf("%d not removed from loaded list: %v",
			len(newMum.list), newMum.list)
	}

	// Check that unmuteUser does nothing for a nonexistent channel
	mum.unmuteUser(randChannelID(prng, t), makeEd25519PubKey(prng, t))
}

// Tests that mutedUserManager.isMuted only returns true for users in the list.
func Test_mutedUserManager_isMuted(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	kv := versioned.NewKV(ekv.MakeMemstore())
	mum := newMutedUserManager(kv)

	expected := make(map[id.ID][]ed25519.PublicKey)

	for i := 0; i < 20; i++ {
		channelID := randChannelID(prng, t)
		expected[*channelID] = make([]ed25519.PublicKey, 50)
		for j := range expected[*channelID] {
			pubKey := makeEd25519PubKey(prng, t)
			expected[*channelID][j] = pubKey
			if j%2 == 0 {
				mum.muteUser(channelID, pubKey)
			}
		}
	}

	for channelID, pubKeys := range expected {
		for i, pubKey := range pubKeys {
			if i%2 == 0 && !mum.isMuted(&channelID, pubKey) {
				t.Errorf("User %x in channel %s is not muted when they should "+
					"be (%d).", pubKey, channelID, i)
			} else if i%2 != 0 && mum.isMuted(&channelID, pubKey) {
				t.Errorf("User %x in channel %s is muted when they should not "+
					"be (%d).", pubKey, channelID, i)
			}
		}
	}

	// Check that isMuted returns false for a nonexistent channel
	if mum.isMuted(randChannelID(prng, t), makeEd25519PubKey(prng, t)) {
		t.Errorf("User muted in channel that does not exist.")
	}
}

// Tests that mutedUserManager.getMutedUsers returns the expected list of public
// keys.
func Test_mutedUserManager_getMutedUsers(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	kv := versioned.NewKV(ekv.MakeMemstore())
	mum := newMutedUserManager(kv)

	expected := make(map[id.ID][]ed25519.PublicKey)

	const numChannels = 20
	const numUsers = 50

	for i := 0; i < numChannels; i++ {
		channelID := randChannelID(prng, t)
		expected[*channelID] = make([]ed25519.PublicKey, numUsers)
		for j := range expected[*channelID] {
			pubKey := makeEd25519PubKey(prng, t)
			expected[*channelID][j] = pubKey
			mum.muteUser(channelID, pubKey)
		}
	}

	// Insert a blank public key into the list that cannot be decoded
	for channelID := range mum.list {
		mum.list[channelID][""] = struct{}{}
		break
	}

	for channelID, pubKeys := range expected {
		mutedUsers := mum.getMutedUsers(&channelID)

		// Check that both the length and capacity are correct when decoding one
		// of the channels should fail
		if len(mutedUsers) != numUsers {
			t.Errorf("Incorrect length of list.\nexpected: %d\nreceived: %d",
				numUsers, len(mutedUsers))
		}
		if cap(mutedUsers) != numUsers {
			t.Errorf("Incorrect capacity of list.\nexpected: %d\nreceived: %d",
				numUsers, cap(mutedUsers))
		}

		sort.SliceStable(pubKeys, func(i, j int) bool {
			return bytes.Compare(pubKeys[i], pubKeys[j]) == -1
		})
		sort.SliceStable(mutedUsers, func(i, j int) bool {
			return bytes.Compare(mutedUsers[i], mutedUsers[j]) == -1
		})

		if !reflect.DeepEqual(pubKeys, mutedUsers) {
			t.Errorf("List of muted users does not match expected for "+
				"channel %s.\nexpected: %x\nreceived: %x",
				&channelID, pubKeys, mutedUsers)
		}
	}
}

// Tests that mutedUserManager.getMutedUsers returns an empty list when there
// are no valid users to return.
func Test_mutedUserManager_getMutedUsers_Empty(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	kv := versioned.NewKV(ekv.MakeMemstore())
	mum := newMutedUserManager(kv)

	// Test getting list for channel that does not exist
	channelID := randChannelID(prng, t)
	mutedUsers := mum.getMutedUsers(channelID)
	if !reflect.DeepEqual(mutedUsers, []ed25519.PublicKey{}) {
		t.Errorf("Did not get expected empty list for unregistered channel ID."+
			"\nexpected: %+v\nreceived: %+v", []ed25519.PublicKey{}, mutedUsers)
	}

	// Test getting list for channel that exists but is empty
	mum.list[*channelID] = make(map[mutedUserKey]struct{})
	mutedUsers = mum.getMutedUsers(channelID)
	if !reflect.DeepEqual(mutedUsers, []ed25519.PublicKey{}) {
		t.Errorf("Did not get expected empty list for unregistered channel ID."+
			"\nexpected: %+v\nreceived: %+v", []ed25519.PublicKey{}, mutedUsers)
	}

	// Test getting list for channel that exists but with only one invalid user
	mum.list[*channelID] = map[mutedUserKey]struct{}{"": {}}
	mutedUsers = mum.getMutedUsers(channelID)
	if len(mutedUsers) != 0 {
		t.Errorf("Incorrect length of list.\nexpected: %d\nreceived: %d",
			0, len(mutedUsers))
	}
	if cap(mutedUsers) != 0 {
		t.Errorf("Incorrect capacity of list.\nexpected: %d\nreceived: %d",
			0, cap(mutedUsers))
	}
	if !reflect.DeepEqual(mutedUsers, []ed25519.PublicKey{}) {
		t.Errorf("Did not get expected empty list for unregistered channel ID."+
			"\nexpected: %+v\nreceived: %+v", []ed25519.PublicKey{}, mutedUsers)
	}

}

// Tests that mutedUserManager.removeChannel removes the channel from the list
// and from storage.
func Test_mutedUserManager_removeChannel(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	kv := versioned.NewKV(ekv.MakeMemstore())
	mum := newMutedUserManager(kv)

	channelID := &id.ID{}
	for i := 0; i < 20; i++ {
		channelID = randChannelID(prng, t)
		for j := 0; j < 50; j++ {
			pubKey := makeEd25519PubKey(prng, t)
			mum.muteUser(channelID, pubKey)
		}
	}

	err := mum.removeChannel(channelID)
	if err != nil {
		t.Fatalf("Failed to remove channel: %+v", err)
	}

	if _, exists := mum.list[*channelID]; exists {
		t.Errorf("Channel not removed from list.")
	}

	_, err = mum.loadMutedUsers(channelID)
	if err == nil || mum.kv.Exists(err) {
		t.Fatalf("Failed to delete muted user list: %+v", err)
	}
}

// Tests that mutedUserManager.len returns the correct length for an empty user
// list and a user list with users added.
func TestIsNicknameValid_mutedUserManager_len(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	kv := versioned.NewKV(ekv.MakeMemstore())
	mum := newMutedUserManager(kv)

	channelID := randChannelID(prng, t)
	if mum.len(channelID) != 0 {
		t.Errorf("New mutedUserManager has incorrect length."+
			"\nexpected: %d\nreceived: %d", 0, mum.len(channelID))
	}

	mum.muteUser(channelID, makeEd25519PubKey(prng, t))
	mum.muteUser(channelID, makeEd25519PubKey(prng, t))
	mum.muteUser(channelID, makeEd25519PubKey(prng, t))

	if mum.len(channelID) != 3 {
		t.Errorf("mutedUserManager has incorrect length."+
			"\nexpected: %d\nreceived: %d", 3, mum.len(channelID))
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that the mutedUserManager can be saved and loaded from storage using
// mutedUserManager.save and mutedUserManager.load.
func Test_mutedUserManager_save_load(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	mum := &mutedUserManager{
		list: map[id.ID]map[mutedUserKey]struct{}{
			*randChannelID(prng, t): {
				makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
			},
			*randChannelID(prng, t): {
				makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
				makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
				makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
			},
			*randChannelID(prng, t): {
				makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
				makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
			},
		},
		kv: versioned.NewKV(ekv.MakeMemstore()),
	}

	for channelID := range mum.list {
		err := mum.save(&channelID, true)
		if err != nil {
			t.Fatalf("Failed to save user list: %+v", err)
		}
	}

	newMum := newMutedUserManager(mum.kv)
	err := newMum.load()
	if err != nil {
		t.Fatalf("Failed to load user list: %+v", err)
	}

	if !reflect.DeepEqual(mum, newMum) {
		t.Errorf("Loaded mutedUserManager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", mum, newMum)
	}
}

// Error path: Tests that mutedUserManager.load returns an error when there is
// no channel list to load.
func Test_mutedUserManager_load_LoadChannelListError(t *testing.T) {
	mum := newMutedUserManager(versioned.NewKV(ekv.MakeMemstore()))
	expectedErr := loadMutedChannelsErr

	err := mum.load()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Did not get expected error when loading a channel list that "+
			"does not exist.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: Tests that mutedUserManager.load returns an error when there are
// no users saved to storage for the channel list.
func Test_mutedUserManager_load_LoadUserListError(t *testing.T) {
	prng := rand.New(rand.NewSource(953))
	mum := newMutedUserManager(versioned.NewKV(ekv.MakeMemstore()))

	channelID := randChannelID(prng, t)
	mum.list[*channelID] = make(map[mutedUserKey]struct{})
	expectedErr := fmt.Sprintf(loadMutedUsersErr, channelID)

	if err := mum.saveChannelList(); err != nil {
		t.Fatalf("Failed to save channel list to storage: %+v", err)
	}

	err := mum.load()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Did not get expected error when loading a user list that "+
			"does not exist.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the list of channels IDs can be saved and loaded from storage
// using mutedUserManager.saveChannelList and mutedUserManager.loadChannelList.
func Test_mutedUserManager_saveChannelList_loadChannelList(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	mum := newMutedUserManager(versioned.NewKV(ekv.MakeMemstore()))

	expected := []*id.ID{
		randChannelID(prng, t), randChannelID(prng, t),
		randChannelID(prng, t), randChannelID(prng, t),
		randChannelID(prng, t), randChannelID(prng, t),
		randChannelID(prng, t), randChannelID(prng, t),
		randChannelID(prng, t), randChannelID(prng, t),
	}

	for _, channelID := range expected {
		mum.list[*channelID] = make(map[mutedUserKey]struct{})
	}

	err := mum.saveChannelList()
	if err != nil {
		t.Fatalf("Failed to save channel list: %+v", err)
	}

	loaded, err := mum.loadChannelList()
	if err != nil {
		t.Fatalf("Failed to load channel list: %+v", err)
	}

	sort.SliceStable(expected, func(i, j int) bool {
		return bytes.Compare(expected[i][:], expected[j][:]) == -1
	})
	sort.SliceStable(loaded, func(i, j int) bool {
		return bytes.Compare(loaded[i][:], loaded[j][:]) == -1
	})

	if !reflect.DeepEqual(expected, loaded) {
		t.Errorf("Loaded channel list does not match expected."+
			"\nexpected: %s\nreceived: %s", expected, loaded)
	}
}

// Tests that a list of muted users for a specific channel can be saved and
// loaded from storage using mutedUserManager.saveMutedUsers and
// mutedUserManager.loadMutedUsers.
func Test_mutedUserManager_saveMutedUsers_loadMutedUsers(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	mum := newMutedUserManager(versioned.NewKV(ekv.MakeMemstore()))

	channelID := randChannelID(prng, t)
	mum.list[*channelID] = map[mutedUserKey]struct{}{
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
	}

	err := mum.saveMutedUsers(channelID)
	if err != nil {
		t.Fatalf("Failed to save muted user list: %+v", err)
	}

	loaded, err := mum.loadMutedUsers(channelID)
	if err != nil {
		t.Fatalf("Failed to load muted user list: %+v", err)
	}

	if !reflect.DeepEqual(mum.list[*channelID], loaded) {
		t.Errorf("Loaded muted user list does not match expected."+
			"\nexpected: %s\nreceived: %s", mum.list[*channelID], loaded)
	}
}

// Tests that mutedUserManager.saveMutedUsers deletes the user list from storage
// when it is empty.
func Test_mutedUserManager_saveMutedUsers_EmptyList(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	mum := newMutedUserManager(versioned.NewKV(ekv.MakeMemstore()))

	channelID := randChannelID(prng, t)
	mum.list[*channelID] = map[mutedUserKey]struct{}{
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
	}

	err := mum.saveMutedUsers(channelID)
	if err != nil {
		t.Fatalf("Failed to save muted user list: %+v", err)
	}

	mum.list[*channelID] = map[mutedUserKey]struct{}{}
	err = mum.saveMutedUsers(channelID)
	if err != nil {
		t.Fatalf("Failed to save muted user list: %+v", err)
	}

	_, err = mum.loadMutedUsers(channelID)
	if err == nil || mum.kv.Exists(err) {
		t.Fatalf("Failed to delete muted user list: %+v", err)
	}
}

// Tests that mutedUserManager.deleteMutedUsers deletes the user list for the
// given channel from storage.
func Test_mutedUserManager_deleteMutedUsers(t *testing.T) {
	prng := rand.New(rand.NewSource(189))
	mum := newMutedUserManager(versioned.NewKV(ekv.MakeMemstore()))

	channelID := randChannelID(prng, t)
	mum.list[*channelID] = map[mutedUserKey]struct{}{
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
		makeMutedUserKey(makeEd25519PubKey(prng, t)): {},
	}

	err := mum.saveMutedUsers(channelID)
	if err != nil {
		t.Fatalf("Failed to save muted user list: %+v", err)
	}

	err = mum.deleteMutedUsers(channelID)
	if err != nil {
		t.Fatalf("Failed to delete muted user list: %+v", err)
	}

	_, err = mum.loadMutedUsers(channelID)
	if err == nil || mum.kv.Exists(err) {
		t.Fatalf("Failed to delete muted user list: %+v", err)
	}
}

// Consistency test of makeMutedUserKey.
func Test_makeMutedUserKey_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(953))
	expectedKeys := []mutedUserKey{
		"c5c110ded852439379bb28e01f8d8d0355c5795c27a4d8900a4e56334fe9f501",
		"a86958de4e9e8c1f4f1dc9c236ad8b799899823a8f9da8ba0c5e190e96c7221c",
		"7da41b27cbd8c5008d7fa40077bcbbffb34805f8be45556506da0f00d9621e01",
		"a2e7062f6d50ca8a2bce840ac0b654ad9ba3dfdf2094a5e5255f3cdfaeb4a1f4",
		"605f307875c0889bb0495c5c4f743f5cd41cf9384a60cea2336443bc28f2c084",
		"ec0e906d3617294907694e7b7c121bafe7b802d6c6103f4481a408d8a5c2c81c",
		"e1b4cd55c9c3e9bee635e89151f93ea6cad9fc4c340460d426773a043a98fb31",
		"91fb296f961cddb189e13cd60e4fc83910944d10e3adc07e8615611feaf2ce64",
		"b967cb95a305c991910006139c27c8d455ee8dfdb4d3b1bf4b2ae1a4866020c7",
		"d7849701f641d0265df39f7716c209b8aa0cf24308cc89a14d4afd0012581996",
	}

	for i, expected := range expectedKeys {
		pubKey := makeEd25519PubKey(prng, t)
		key := makeMutedUserKey(pubKey)

		if key != expected {
			t.Errorf("mutedUserKey does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}
	}
}

// Tests that mutedUserKey.decode can decode each mutedUserKey to its original
// ed25519.PublicKey.
func Test_mutedUserKey_decode(t *testing.T) {
	prng := rand.New(rand.NewSource(953))
	for i := 0; i < 0; i++ {
		expected := makeEd25519PubKey(prng, t)
		key := makeMutedUserKey(expected)
		decoded, err := key.decode()
		if err != nil {
			t.Errorf("Failed to decode key: %+v", err)
		}

		if !expected.Equal(decoded) {
			t.Errorf("Decoded key does not match original (%d)."+
				"\nexpected: %x\nreceived: %x", i, expected, decoded)
		}
	}
}

// Consistency test of makeMutedChannelStoreKey.
func Test_makeMutedChannelStoreKey_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(953))
	expectedKeys := []string{
		"mutedUserList/5PzSqhi03EclS1sS3tT7EbcfDlulBr4D0jaBUqpGZ70D",
		"mutedUserList/DZcUgjcB7RdnrP9Bf8ln1d+qpjpB98219pf/qjvNzXkD",
		"mutedUserList/1CmZlD5GikWxCT2+JW4ky1PC5Kn9wkaTN5jEj9P6HoUD",
		"mutedUserList/542TKbYMnXcct0OBT5TnkNmOzAZkc/yFe7Zx6vqHrSUD",
		"mutedUserList/kLMXJSKdy+O2sef63PJDi+7J5kGTUVsbo0ij1e5bahgD",
		"mutedUserList/kdvqU9Iyy+njMpEz98qYk3C/A89aO/NYKzUjRcVdUQcD",
		"mutedUserList/8sS5xmNb0lisRMFCy11ZGd881FVvEBQ+NDtEsHBrn7sD",
		"mutedUserList/r28DbAJgmKZuFgJ2Smuw1EsZFN9i2PA+DWqjaUic688D",
		"mutedUserList/C+CV6LgfADe54mAz12STU633Y4YX7FEs5h/9UO4gbdMD",
		"mutedUserList/JlvsHupsyqukZhGKx3a1wTLYJmkYuClbSy97Cl2fKIYD",
	}

	for i, expected := range expectedKeys {
		channelID := randChannelID(prng, t)
		key := makeMutedChannelStoreKey(channelID)

		if key != expected {
			t.Errorf("Storage key does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}
	}
}

// makeEd25519PubKey generates an ed25519.PublicKey for testing.
func makeEd25519PubKey(rng io.Reader, t *testing.T) ed25519.PublicKey {
	pubKey, _, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 keys: %+v", err)
	}
	return pubKey
}
