////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// IsChannelAdmin returns true if the user is an admin of the channel.
func (m *manager) IsChannelAdmin(channelID *id.ID) bool {
	jww.INFO.Printf("[CH] IsChannelAdmin in channel %s", channelID)
	if _, err := loadChannelPrivateKey(channelID, m.kv); err != nil {
		if m.kv.Exists(err) {
			jww.WARN.Printf("[CH] Private key for channel ID %s found in "+
				"storage, but an error was encountered while accessing it: %+v",
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
	jww.INFO.Printf("[CH] ExportChannelAdminKey in channel %s", channelID)
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
// key does not belong to the given channel.
//
// Returns the error WrongPasswordErr for an invalid password. Returns the error
// ChannelDoesNotExistsErr if the channel has not already been joined.
func (m *manager) VerifyChannelAdminKey(channelID *id.ID,
	encryptionPassword string, encryptedPrivKey []byte) (bool, error) {
	jww.INFO.Printf("[CH] VerifyChannelAdminKey in channel %s", channelID)
	decryptedChannelID, pk, err :=
		cryptoBroadcast.ImportPrivateKey(encryptionPassword, encryptedPrivKey)
	if err != nil {
		return false, WrongPasswordErr
	}

	// Compare channel ID
	if !channelID.Cmp(decryptedChannelID) {
		return false, nil
	}

	c, err := m.getChannel(decryptedChannelID)
	if err != nil {
		return false, err
	}

	// Compare public keys
	if !bytes.Equal(cryptoBroadcast.HashPubKey(pk.Public()),
		c.broadcast.Get().RsaPubKeyHash) {
		return false, nil
	}

	return true, nil
}

// ImportChannelAdminKey decrypts and imports the given encrypted private key
// and grants the user admin access to the channel the private key belongs to.
// Returns an error if the private key cannot be decrypted or if the private key
// is for the wrong channel.
//
// Returns the error WrongPasswordErr for an invalid password. Returns the error
// ChannelDoesNotExistsErr if the channel has not already been joined. Returns
// the error WrongPrivateKeyErr if the private key does not belong to the
// channel.
func (m *manager) ImportChannelAdminKey(
	channelID *id.ID, encryptionPassword string, encryptedPrivKey []byte) error {
	jww.INFO.Printf("[CH] ImportChannelAdminKey for channel %s", channelID)
	decryptedChannelID, pk, err :=
		cryptoBroadcast.ImportPrivateKey(encryptionPassword, encryptedPrivKey)
	if err != nil {
		return WrongPasswordErr
	}

	// Compare channel IDs
	if !channelID.Cmp(decryptedChannelID) {
		return WrongPrivateKeyErr
	}

	c, err := m.getChannel(decryptedChannelID)
	if err != nil {
		return err
	}

	// Compare public keys
	if !bytes.Equal(cryptoBroadcast.HashPubKey(pk.Public()),
		c.broadcast.Get().RsaPubKeyHash) {
		return WrongPrivateKeyErr
	}

	return saveChannelPrivateKey(channelID, pk, m.kv)
}

// DeleteChannelAdminKey deletes the private key for the given channel.
//
// CAUTION: This will remove admin access. This cannot be undone. If the private
// key is deleted, it cannot be recovered and the channel can never have another
// admin.
func (m *manager) DeleteChannelAdminKey(channelID *id.ID) error {
	jww.INFO.Printf("[CH] DeleteChannelAdminKey for channel %s", channelID)
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
	channelID *id.ID, pk rsa.PrivateKey, kv *utility.KV) error {
	obj := &versioned.Object{
		Version:   channelPrivateKeyStoreVersion,
		Timestamp: netTime.Now(),
		Data:      pk.MarshalPem(),
	}
	return kv.Set(makeChannelPrivateKeyStoreKey(channelID), obj.Marshal())
}

// loadChannelPrivateKey retrieves the [rsa.PrivateKey] for the given channel ID
// from storage.
//
// The private key is saved to storage via saveChannelPrivateKey.
func loadChannelPrivateKey(
	channelID *id.ID, kv *utility.KV) (rsa.PrivateKey, error) {
	data, err := kv.Get(
		makeChannelPrivateKeyStoreKey(channelID), channelPrivateKeyStoreVersion)
	if err != nil {
		return nil, err
	}

	return rsa.GetScheme().UnmarshalPrivateKeyPEM(data)
}

// deleteChannelPrivateKey deletes the private key from storage for the given
// channel ID.
func deleteChannelPrivateKey(channelID *id.ID, kv *utility.KV) error {
	return kv.Delete(
		makeChannelPrivateKeyStoreKey(channelID), channelPrivateKeyStoreVersion)
}

// makeChannelPrivateKeyStoreKey generates a unique storage key for the given
// channel that is used to save and load the channel's private key from storage.
func makeChannelPrivateKeyStoreKey(channelID *id.ID) string {
	return channelPrivateKeyStoreKey + channelID.String()
}
