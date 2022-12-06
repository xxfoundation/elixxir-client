////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// IsChannelAdmin returns true if the user is an admin of the channel.
func (m *manager) IsChannelAdmin(channelID *id.ID) bool {
	jww.INFO.Printf("[CH] IsChannelAdmin %s", channelID)
	if _, err := loadChannelPrivateKey(channelID, m.kv); err != nil {
		if m.kv.Exists(err) {
			jww.WARN.Printf("Private key for channel ID %s found in storage, "+
				"but an error was encountered while accessing it: %+v",
				channelID, err)
		}
		return false
	}
	return true
}

// ExportChannelAdminKey loads the private key from storage and returns it
// encrypted with the given encryptionPassword.
func (m *manager) ExportChannelAdminKey(
	channelID *id.ID, encryptionPassword string) ([]byte, error) {
	jww.INFO.Printf("[CH] ExportChannelAdminKey %s", channelID)
	privKey, err := loadChannelPrivateKey(channelID, m.kv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load private key from storage")
	}

	stream := m.rng.GetStream()
	pkPacket, err := cryptoBroadcast.ExportPrivateKey(
		channelID, privKey, encryptionPassword, stream)
	stream.Close()
	if err != nil {
		return nil, errors.Errorf("failed to export private key: %+v", err)
	}

	return pkPacket, nil
}

// VerifyChannelAdminKey verifies that the encrypted private key can be
// decrypted and that it matches the expected channel. Returns false if private
// key does not belong to the given channel ID. Returns an error for an invalid
// password.
func (m *manager) VerifyChannelAdminKey(channelID *id.ID,
	encryptionPassword string, encryptedPrivKey []byte) (bool, error) {
	jww.INFO.Printf("[CH] VerifyChannelAdminKey %s", channelID)
	decryptedChannelID, _, err :=
		cryptoBroadcast.ImportPrivateKey(encryptionPassword, encryptedPrivKey)
	if err != nil {
		return false,
			errors.Errorf("failed to decrypt private channel key: %+v", err)
	}

	if !channelID.Cmp(decryptedChannelID) {
		return false, nil
	}

	return true, nil
}

// ImportChannelAdminKey decrypts the given private channel ID and saves it to
// storage.
func (m *manager) ImportChannelAdminKey(
	channelID *id.ID, encryptionPassword string, encryptedPrivKey []byte) error {
	jww.INFO.Printf("[CH] ImportChannelAdminKey %s", channelID)
	decryptedChannelID, privKey, err :=
		cryptoBroadcast.ImportPrivateKey(encryptionPassword, encryptedPrivKey)
	if err != nil {
		return errors.Errorf("failed to decrypt private channel key: %+v", err)
	}

	if !channelID.Cmp(decryptedChannelID) {
		return errors.New("private key belongs to a different channel")
	}

	return saveChannelPrivateKey(channelID, privKey, m.kv)
}

// DeleteChannelAdminKey deletes the private key for the given channel.
//
// CAUTION: This will remove admin access. This cannot be undone. If the private
// key is deleted, it cannot be recovered and the channel can never have another
// admin.
func (m *manager) DeleteChannelAdminKey(channelID *id.ID) error {
	jww.INFO.Printf("[CH] DeleteChannelAdminKey %s", channelID)
	return deleteChannelPrivateKey(channelID, m.kv)
}

////////////////////////////////////////////////////////////////////////////////
// Storage                                                                    //
////////////////////////////////////////////////////////////////////////////////

// Storage values.
const (
	channelPrivateKeyStoreVersion = 0
	channelPrivateKeyStoreKey     = "channelPrivateKey/"
)

// saveChannelPrivateKey saves the [rsa.PrivateKey] for the given channel ID to
// storage. This is called everytime a user generates a channel so that they can
// access the channel's private key.
//
// The private key can retrieved from storage via loadChannelPrivateKey.
func saveChannelPrivateKey(
	channelID *id.ID, pk rsa.PrivateKey, kv *versioned.KV) error {
	return kv.Set(makeChannelPrivateKeyStoreKey(channelID),
		&versioned.Object{
			Version:   channelPrivateKeyStoreVersion,
			Timestamp: netTime.Now(),
			Data:      pk.MarshalPem(),
		},
	)
}

// loadChannelPrivateKey retrieves the [rsa.PrivateKey] for the given channel ID
// from storage.
//
// The private key is saved to storage via saveChannelPrivateKey.
func loadChannelPrivateKey(
	channelID *id.ID, kv *versioned.KV) (rsa.PrivateKey, error) {
	obj, err := kv.Get(
		makeChannelPrivateKeyStoreKey(channelID), channelPrivateKeyStoreVersion)
	if err != nil {
		return nil, err
	}

	return rsa.GetScheme().UnmarshalPrivateKeyPEM(obj.Data)
}

// deleteChannelPrivateKey deletes the private key from storage for the given
// channel ID.
func deleteChannelPrivateKey(channelID *id.ID, kv *versioned.KV) error {
	return kv.Delete(
		makeChannelPrivateKeyStoreKey(channelID), channelPrivateKeyStoreVersion)
}

// makeChannelPrivateKeyStoreKey generates a unique storage key for the given
// channel that is used to save and load the channel's private key from storage.
func makeChannelPrivateKeyStoreKey(channelID *id.ID) string {
	return channelPrivateKeyStoreKey + channelID.String()
}
