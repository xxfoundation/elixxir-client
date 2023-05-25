////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"os"
	"testing"
)

func newPrivKeyTestManager() *manager {
	return &manager{
		channels: make(map[id.ID]*joinedChannel),
		rng:      fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		local:    versioned.NewKV(ekv.MakeMemstore()),
	}
}

// Tests that manager.IsChannelAdmin returns true to a channel private key saved
// in storage and returns false to one that is not.
func Test_manager_IsChannelAdmin(t *testing.T) {
	m := newPrivKeyTestManager()
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	if !m.IsChannelAdmin(c.ReceptionID) {
		t.Errorf(
			"User not admin of channel %s when they should be.", c.ReceptionID)
	}

	if m.IsChannelAdmin(id.NewIdFromString("invalidID", id.User, t)) {
		t.Errorf(
			"User admin of channel %s when they should not be.", c.ReceptionID)
	}
}

// Tests that a private key can get retrieved from storage using
// manager.ExportChannelAdminKey and that it matches the original channel ID and
// private key when decrypted. Also tests that it can be verified using
// manager.VerifyChannelAdminKey and imported into a new manager using
// manager.ImportChannelAdminKey.
func Test_manager_Export_Verify_Import_ChannelAdminKey(t *testing.T) {
	password := "hunter2"
	m1 := newPrivKeyTestManager()
	m2 := newPrivKeyTestManager()
	c, pk, err := m1.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	pkPacket, err := m1.ExportChannelAdminKey(c.ReceptionID, password)
	if err != nil {
		t.Fatalf("Failed to export private key: %+v", err)
	}

	b, err := broadcast.NewBroadcastChannel(c, new(mockBroadcastClient), m2.rng)
	if err != nil {
		t.Errorf("Failed to create new broadcast channel: %+v", err)
	}

	m2.channels[*c.ReceptionID] = &joinedChannel{b, true}

	valid, err := m2.VerifyChannelAdminKey(c.ReceptionID, password, pkPacket)
	if err != nil {
		t.Errorf("Failed to verify channel admin key: %+v", err)
	} else if !valid {
		t.Errorf("Channels did not match")
	}

	err = m2.ImportChannelAdminKey(c.ReceptionID, password, pkPacket)
	if err != nil {
		t.Errorf("Failed to import private key: %+v", err)
	}

	password2 := "hunter3"
	loadedPacket, err := m2.ExportChannelAdminKey(c.ReceptionID, password2)
	if err != nil {
		t.Errorf("Failed to get private key: %+v", err)
	}

	channelID, privKey, err :=
		cryptoBroadcast.ImportPrivateKey(password2, loadedPacket)
	if err != nil {
		t.Errorf("Failed to import private key: %+v", err)
	}

	if !channelID.Cmp(c.ReceptionID) {
		t.Errorf("Incorrect channel ID.\nexpected: %s\nreceived: %s",
			c.ReceptionID, channelID)
	}
	if !privKey.GetGoRSA().Equal(pk.GetGoRSA()) {
		t.Errorf("Incorrect private key.\nexpected: %s\nreceived: %s",
			pk, privKey)
	}
}

// Error path: Tests that when no private key exists for the channel ID,
// manager.ExportChannelAdminKey returns an error that the private key does not
// exist in storage, as determined by local.Exists.
func Test_manager_ExportChannelAdminKey_NoPrivateKeyError(t *testing.T) {
	m := &manager{
		rng:   fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		local: versioned.NewKV(ekv.MakeMemstore()),
	}

	invalidChannelID := id.NewIdFromString("someID", id.User, t)
	_, err := m.ExportChannelAdminKey(invalidChannelID, "password")
	if err == nil || m.local.Exists(err) {
		t.Errorf("Unexpected error when no private key exist."+
			"\nexpected: %s\nreceived: %+v", "object not found", err)
	}
}

// Error path: Tests that when the password is invalid,
// manager.ImportChannelAdminKey returns an error.
func Test_manager_ImportChannelAdminKey_WrongPasswordError(t *testing.T) {
	m := newPrivKeyTestManager()
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, "hunter2")
	if err != nil {
		t.Fatalf("Failed to export private key: %+v", err)
	}

	err = m.ImportChannelAdminKey(c.ReceptionID, "invalidPassword", pkPacket)
	if err == nil {
		t.Error("Importing private key with incorrect password did not fail.")
	}
}

// Error path: Tests that when the channel ID does not match,
// manager.ImportChannelAdminKey returns an error.
func Test_manager_ImportChannelAdminKey_WrongChannelIdError(t *testing.T) {
	m := newPrivKeyTestManager()
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	password := "hunter2"
	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, password)
	if err != nil {
		t.Fatalf("Failed to export private key: %+v", err)
	}

	err = m.ImportChannelAdminKey(&id.ID{}, password, pkPacket)
	if err == nil {
		t.Error("Importing private key with incorrect channel ID did not fail.")
	}
}

// Error path: Tests that when the password is invalid,
// manager.VerifyChannelAdminKey returns an error.
func Test_manager_VerifyChannelAdminKey_WrongPasswordError(t *testing.T) {
	m := newPrivKeyTestManager()
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, "hunter2")
	if err != nil {
		t.Fatalf("Failed to export private key: %+v", err)
	}

	_, err = m.VerifyChannelAdminKey(c.ReceptionID, "invalidPassword", pkPacket)
	if err == nil {
		t.Error("Verifying private key with incorrect password did not fail.")
	}
}

// Error path: Tests that when the channel ID does not match,
// manager.VerifyChannelAdminKey returns false.
func Test_manager_VerifyChannelAdminKey_WrongChannelIdError(t *testing.T) {

	m := newPrivKeyTestManager()
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	password := "hunter2"
	pkPacket, err := m.ExportChannelAdminKey(c.ReceptionID, password)
	if err != nil {
		t.Fatalf("Failed to export private key: %+v", err)
	}

	match, err := m.VerifyChannelAdminKey(&id.ID{}, password, pkPacket)
	if err != nil {
		t.Fatalf("Failed to decrypt private key: %+v", err)
	} else if match {
		t.Error(
			"Importing private key with incorrect channel ID returned a match.")
	}
}

// Tests that manager.DeleteChannelAdminKey deletes the channel and that
// manager.ExportChannelAdminKey returns an error.
func Test_manager_DeleteChannelAdminKey(t *testing.T) {
	m := newPrivKeyTestManager()
	c, _, err := m.generateChannel("name", "desc", cryptoBroadcast.Public, 512)
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	err = m.DeleteChannelAdminKey(c.ReceptionID)
	if err != nil {
		t.Fatalf("Failed to delete private key: %+v", err)
	}

	_, err = m.ExportChannelAdminKey(c.ReceptionID, "hunter2")
	if m.local.Exists(err) {
		t.Fatalf("Private key was not deleted: %+v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage                                                                    //
////////////////////////////////////////////////////////////////////////////////

// Tests that a rsa.PrivateKey saved to storage with saveChannelPrivateKey and
// loaded with loadChannelPrivateKey matches the original.
func Test_saveChannelPrivateKey_loadChannelPrivateKey(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	c, pk, err := cryptoBroadcast.NewChannel(
		"name", "description", cryptoBroadcast.Public, 512, &csprng.SystemRNG{})
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	err = saveChannelPrivateKey(c.ReceptionID, pk, kv)
	if err != nil {
		t.Errorf("Failed to save private key: %+v", err)
	}

	loadedPk, err := loadChannelPrivateKey(c.ReceptionID, kv)
	if err != nil {
		t.Errorf("Failed to load private key: %+v", err)
	}

	if !pk.GetGoRSA().Equal(loadedPk.GetGoRSA()) {
		t.Errorf("Loaded private key does not match original."+
			"\nexpected: %q\nreceived: %q",
			pk.MarshalPem(), loadedPk.MarshalPem())
	}
}

// Error path: Tests that loadChannelPrivateKey returns an error when there is
// nothing saved to storage for the given channel ID.
func Test_loadChannelPrivateKey_StorageError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	channelID, _ := id.NewRandomID(rand.New(rand.NewSource(654)), id.User)

	_, err := loadChannelPrivateKey(channelID, kv)
	if err == nil || kv.Exists(err) {
		t.Errorf("Failed to get expected error when nothing should exist in "+
			"storage.\nexpected: %s\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that deleteChannelPrivateKey deletes the private key for the channel
// from storage.
func Test_deleteChannelPrivateKey(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	c, pk, err := cryptoBroadcast.NewChannel(
		"name", "description", cryptoBroadcast.Public, 512, &csprng.SystemRNG{})
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	err = saveChannelPrivateKey(c.ReceptionID, pk, kv)
	if err != nil {
		t.Errorf("Failed to save private key: %+v", err)
	}

	err = deleteChannelPrivateKey(c.ReceptionID, kv)
	if err != nil {
		t.Errorf("Failed to delete private key: %+v", err)
	}

	_, err = loadChannelPrivateKey(c.ReceptionID, kv)
	if kv.Exists(err) {
		t.Errorf("Private key not deleted: %+v", err)
	}
}

// Consistency test for makeChannelPrivateKeyStoreKey.
func Test_makeChannelPrivateKeyStoreKey_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(489))

	expectedKeys := []string{
		"channelPrivateKey/1ADbtux2hJxCHcHXPRgGNhTrh6lLuTDLr4pj03hoPJ8D",
		"channelPrivateKey/Nr0uHOgSh/vxQDJhqRFszrwq+EYiDJOkgsioPqYHvYMD",
		"channelPrivateKey/C/lsX0K048zvW3HQ1YczcIb/qFQ7sA8IRQT9w5ULdfAD",
		"channelPrivateKey/v3yZLAeOLtSIxy/hLvAt/9/RMde9wCSb//IhhITEMc8D",
		"channelPrivateKey/5zp/iZqmKLwjqyPD75ynLIPCJ9/zNURScICeK794+iUD",
		"channelPrivateKey/yQnayARGeuOggCTL7MVW+YqObW4ta1dAXCPpUfxcIbsD",
		"channelPrivateKey/ErqGJcH3PieyyqX28bD33rkVLkU6s/4h+Kv1//GwTUUD",
		"channelPrivateKey/fqa59O23bbHJGGlpWp+fNvAfdwbcdnCtrnP4EudRdLwD",
		"channelPrivateKey/5USvPos1dFHkH3uiCxNXK9+BOpmpsMcL0ildB02AIIAD",
		"channelPrivateKey/TZRksydOx4OSTlFchR75XTeFXgxjDMHCqDFHoBPiLHED",
		"channelPrivateKey/YI+07Le5wFzJ0x9o2Fye3vTtqKPyJamTMpidSo7tJ0sD",
		"channelPrivateKey/ZXafGa97xwmoszAPfF4/U75ifzZ7EmugOtr2vM1TCsUD",
		"channelPrivateKey/phx3YCMSqftR/uYjQNpwbUM9/L1fde+9LTHBSJ/tQccD",
		"channelPrivateKey/5twWFIXxgchAZh0vo32Fltfw68ecB/dqwFDJt2hkKEUD",
		"channelPrivateKey/dGbmZW/9piZwF+BUj6WX9lRXix7JloMXubWUerKCpkUD",
	}

	for i, expected := range expectedKeys {
		channelID, err := id.NewRandomID(prng, id.User)
		if err != nil {
			t.Errorf("Failed to generate channel ID %d: %+v", i, err)
		}

		key := makeChannelPrivateKeyStoreKey(channelID)
		if key != expected {
			t.Errorf("Storage key for channel %s does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", channelID, i, expected, key)
		}
	}
}
