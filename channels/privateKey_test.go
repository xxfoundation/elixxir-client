////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

var dummyAdminKeyUpdate = func(chID *id.ID, isAdmin bool) {}

func newPrivKeyTestManager(t *testing.T) *manager {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	akm := newAdminKeysManager(rkv, dummyAdminKeyUpdate)
	return &manager{
		channels:         make(map[id.ID]*joinedChannel),
		rng:              fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		adminKeysManager: akm,
	}
}

// Tests that manager.IsChannelAdmin returns true to a channel private key saved
// in storage and returns false to one that is not.
func Test_manager_IsChannelAdmin(t *testing.T) {
	m := newPrivKeyTestManager(t)
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	require.True(t, m.IsChannelAdmin(c.ReceptionID),
		"User not admin of channel %s when they should be.", c.ReceptionID)

	require.False(t, m.IsChannelAdmin(id.NewIdFromString("invalidID", id.User, t)),
		"User admin of channel %s when they should not be.", c.ReceptionID)
}

// Tests that a private key can get retrieved from storage using
// manager.ExportChannelAdminKey and that it matches the original channel ID and
// private key when decrypted. Also tests that it can be verified using
// manager.VerifyChannelAdminKey and imported into a new manager using
// manager.ImportChannelAdminKey.
func Test_manager_Export_Verify_Import_ChannelAdminKey(t *testing.T) {
	password := "hunter2"
	m1 := newPrivKeyTestManager(t)
	m2 := newPrivKeyTestManager(t)
	c, pk, err := m1.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	require.NoError(t, err, "Failed to generate new channel: %+v", err)

	pkPacket, err := m1.ExportChannelAdminKey(c.ReceptionID, password)
	require.NoError(t, err, "Failed to export private key: %+v", err)

	b, err := broadcast.NewBroadcastChannel(c, new(mockBroadcastClient), m2.rng)
	require.NoError(t, err, "Failed to create new broadcast channel: %+v", err)

	m2.channels[*c.ReceptionID] = &joinedChannel{b, true}

	valid, err := m2.VerifyChannelAdminKey(c.ReceptionID, password, pkPacket)
	require.NoError(t, err, "Failed to verify channel admin key: %+v", err)
	require.True(t, valid, "Channels did not match")

	err = m2.ImportChannelAdminKey(c.ReceptionID, password, pkPacket)
	require.NoError(t, err, "Failed to import private key: %+v", err)

	password2 := "hunter3"
	loadedPacket, err := m2.ExportChannelAdminKey(c.ReceptionID, password2)
	require.NoError(t, err, "Failed to get private key: %+v", err)

	channelID, privKey, err :=
		cryptoBroadcast.ImportPrivateKey(password2, loadedPacket)
	require.NoError(t, err, "Failed to import private key: %+v", err)

	require.Equal(t, channelID, c.ReceptionID,
		"Incorrect channel ID.\nexpected: %s\nreceived: %s",
		c.ReceptionID, channelID)

	require.Equal(t, privKey.GetGoRSA(), pk.GetGoRSA(),
		"Incorrect private key.\nexpected: %s\nreceived: %s",
		pk, privKey)

}

// Error path: Tests that when no private key exists for the channel ID,
// manager.ExportChannelAdminKey returns an error that the private key does not
// exist in storage, as determined by local.Exists.
func Test_manager_ExportChannelAdminKey_NoPrivateKeyError(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	akm := newAdminKeysManager(rkv, dummyAdminKeyUpdate)

	m := &manager{
		//local:            versioned.NewKV(ekv.MakeMemstore()),
		adminKeysManager: akm,
		rng:              fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		local:            versioned.NewKV(ekv.MakeMemstore()),
	}

	invalidChannelID := id.NewIdFromString("someID", id.User, t)
	_, err := m.ExportChannelAdminKey(invalidChannelID, "password")
	require.Errorf(t, err,
		"Unexpected error when no private key exist."+
			"\nexpected: %s\nreceived: %+v", "object not found", err)
}

// Error path: Tests that when the password is invalid,
// manager.ImportChannelAdminKey returns an error.
func Test_manager_ImportChannelAdminKey_WrongPasswordError(t *testing.T) {
	m := newPrivKeyTestManager(t)
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	require.NoError(t, err, "Failed to generate new channel: %+v", err)

	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, "hunter2")
	require.NoError(t, err, "Failed to export private key: %+v", err)

	err = m.ImportChannelAdminKey(c.ReceptionID, "invalidPassword", pkPacket)
	require.Error(t, err,
		"Importing private key with incorrect password did not fail.")
}

// Error path: Tests that when the channel ID does not match,
// manager.ImportChannelAdminKey returns an error.
func Test_manager_ImportChannelAdminKey_WrongChannelIdError(t *testing.T) {
	m := newPrivKeyTestManager(t)
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	require.NoError(t, err, "Failed to generate new channel: %+v", err)

	password := "hunter2"
	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, password)
	require.NoError(t, err,
		"Failed to export private key: %+v", err)

	err = m.ImportChannelAdminKey(&id.ID{}, password, pkPacket)
	require.Error(t, err,
		"Importing private key with incorrect channel ID did not fail.")
}

// Error path: Tests that when the password is invalid,
// manager.VerifyChannelAdminKey returns an error.
func Test_manager_VerifyChannelAdminKey_WrongPasswordError(t *testing.T) {
	m := newPrivKeyTestManager(t)
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	require.NoError(t, err,
		"Failed to generate new channel: %+v", err)

	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, "hunter2")
	require.NoError(t, err, "Failed to export private key: %+v", err)

	_, err = m.VerifyChannelAdminKey(c.ReceptionID, "invalidPassword", pkPacket)
	require.Error(t, err, "Verifying private key with incorrect password did not fail.")
}

// Error path: Tests that when the channel ID does not match,
// manager.VerifyChannelAdminKey returns false.
func Test_manager_VerifyChannelAdminKey_WrongChannelIdError(t *testing.T) {
	m := newPrivKeyTestManager(t)
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	require.NoError(t, err,
		"Failed to generate new channel: %+v", err)

	password := "hunter2"
	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, password)
	require.NoError(t, err,
		"Failed to export private key: %+v", err)

	match, err := m.VerifyChannelAdminKey(&id.ID{}, password, pkPacket)
	require.NoError(t, err,
		"Failed to decrypt private key: %+v", err)
	require.False(t, match,
		"Importing private key with incorrect channel ID returned a match.")
}

// Tests that manager.DeleteChannelAdminKey deletes the channel and that
// manager.ExportChannelAdminKey returns an error.
func Test_manager_DeleteChannelAdminKey(t *testing.T) {
	m := newPrivKeyTestManager(t)
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	require.NoError(t, err,
		"Failed to generate new channel: %+v", err)

	require.NoError(t, m.DeleteChannelAdminKey(c.ReceptionID),
		"Failed to delete private key: %+v", err)

	_, err = m.ExportChannelAdminKey(c.ReceptionID, "hunter2")
	require.Error(t, err)
}

////////////////////////////////////////////////////////////////////////////////
// Storage                                                                    //
////////////////////////////////////////////////////////////////////////////////

// Tests that a rsa.PrivateKey saved to storage with saveChannelPrivateKey and
// loaded with loadChannelPrivateKey matches the original.
func Test_saveChannelPrivateKey_loadChannelPrivateKey(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	akm := newAdminKeysManager(rkv, dummyAdminKeyUpdate)
	c, pk, err := cryptoBroadcast.NewChannel(
		"name", "description", cryptoBroadcast.Public, 512, &csprng.SystemRNG{})
	require.NoError(t, err, "Failed to generate new channel: %+v", err)

	require.NoError(t, akm.saveChannelPrivateKey(c.ReceptionID, pk),
		"Failed to save private key: %+v", err)

	loadedPk, err := akm.loadChannelPrivateKey(c.ReceptionID)
	require.NoError(t, err,
		"Failed to load private key: %+v", err)

	require.Equal(t, pk.GetGoRSA(), loadedPk.GetGoRSA(),
		"Loaded private key does not match original."+
			"\nexpected: %q\nreceived: %q",
		pk.MarshalPem(), loadedPk.MarshalPem())
}

// Error path: Tests that loadChannelPrivateKey returns an error when there is
// nothing saved to storage for the given channel ID.
func Test_loadChannelPrivateKey_StorageError(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	akm := newAdminKeysManager(rkv, dummyAdminKeyUpdate)

	channelID, _ := id.NewRandomID(rand.New(rand.NewSource(654)), id.User)

	_, err := akm.loadChannelPrivateKey(channelID)
	require.Error(t, err,
		"Failed to get expected error when nothing should exist in "+
			"storage.\nexpected: %s\nreceived: %+v", os.ErrNotExist, err)
}

// Tests that deleteChannelPrivateKey deletes the private key for the channel
// from storage.
func Test_deleteChannelPrivateKey(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	akm := newAdminKeysManager(rkv, dummyAdminKeyUpdate)
	c, pk, err := cryptoBroadcast.NewChannel(
		"name", "description", cryptoBroadcast.Public, 512, &csprng.SystemRNG{})
	require.NoError(t, err, "Failed to generate new channel: %+v", err)

	require.NoError(t, akm.saveChannelPrivateKey(c.ReceptionID, pk),
		"Failed to save private key: %+v", err)

	require.NoError(t, akm.deleteChannelPrivateKey(c.ReceptionID),
		"Failed to delete private key: %+v", err)

	_, err = akm.loadChannelPrivateKey(c.ReceptionID)
	require.Error(t, err)
}

func Test_mapUpdate(t *testing.T) {
	rkv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	akm := newAdminKeysManager(rkv, dummyAdminKeyUpdate)

	const numTests = 100

	wg := &sync.WaitGroup{}
	wg.Add(numTests)

	edits := make(map[string]versioned.ElementEdit, numTests)
	expectedUpdates := make(map[id.ID]bool, numTests)
	rng := rand.New(rand.NewSource(69))

	// build the input and output data
	for i := 0; i < numTests; i++ {
		cid := &id.ID{}
		cid[0] = byte(i)

		privKey, err := rsa.GetScheme().Generate(rng, 1024)
		require.NoError(t, err)

		// make 1/3 chance it will be deleted
		existsChoice := make([]byte, 1)
		rng.Read(existsChoice)
		op := versioned.KeyOperation(int(existsChoice[0]) % 3)
		data := privKey.MarshalPem()
		expected := true

		if op == versioned.Deleted {
			require.NoError(t, akm.saveChannelPrivateKey(cid, privKey))
			data = nil
			expected = false
		} else if op == versioned.Updated {
			privKeyOld, err := rsa.GetScheme().Generate(rng, 1024)
			require.NoError(t, err)
			require.NoError(t, akm.saveChannelPrivateKey(cid, privKeyOld))
		}

		expectedUpdates[*cid] = expected

		// Create the edit
		edits[marshalChID(cid)] = versioned.ElementEdit{
			OldElement: nil,
			NewElement: &versioned.Object{
				Version:   0,
				Timestamp: time.Now(),
				Data:      data,
			},
			Operation: op,
		}
	}

	akm.callback = func(chID *id.ID, isAdmin bool) {
		expectedUpdate, exists := expectedUpdates[*chID]
		require.True(t, exists)
		require.Equal(t, expectedUpdate, isAdmin)
		wg.Done()
	}

	akm.mapUpdate(adminKeysMapName, edits)
	wg.Wait()

}
