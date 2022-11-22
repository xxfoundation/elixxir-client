////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"os"
	"testing"
)

// Tests that a private key can get retrieved from storage using
// getSavedChannelPrivateKey and that it matches the original channel ID and
// private key when decrypted.
func Test_importChannelPrivateKey_getSavedChannelPrivateKey(t *testing.T) {
	prng := rand.New(rand.NewSource(489))
	password := "hunter2"
	kv := versioned.NewKV(ekv.MakeMemstore())
	c, pk, err := cryptoBroadcast.NewChannel(
		"name", "description", cryptoBroadcast.Public, 18, &csprng.SystemRNG{})
	if err != nil {
		t.Fatalf("Failed to generate new channel: %+v", err)
	}

	pkPacket, err :=
		cryptoBroadcast.ExportPrivateKey(c.ReceptionID, pk, password, prng)
	if err != nil {
		t.Fatalf("Failed to export private key: %+v", err)
	}

	err = importChannelPrivateKey(password, pkPacket, kv)
	if err != nil {
		t.Errorf("Failed to import private key: %+v", err)
	}

	password2 := "hunter3"
	loadedPacket, err :=
		getSavedChannelPrivateKey(c.ReceptionID, password2, kv, prng)
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

////////////////////////////////////////////////////////////////////////////////
// Storage                                                                    //
////////////////////////////////////////////////////////////////////////////////

// Tests that a rsa.PrivateKey saved to storage with saveChannelPrivateKey and
// loaded with loadChannelPrivateKey matches the original.
func Test_saveChannelPrivateKey_loadChannelPrivateKey(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	c, pk, err := cryptoBroadcast.NewChannel(
		"name", "description", cryptoBroadcast.Public, 18, &csprng.SystemRNG{})
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
