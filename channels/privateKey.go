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
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

// IsChannelAdmin returns true if the user is an admin of the channel.
func (m *manager) IsChannelAdmin(channelID *id.ID) bool {
	jww.INFO.Printf("[CH] IsChannelAdmin in channel %s", channelID)
	if _, err := m.adminKeysManager.loadChannelPrivateKey(
		channelID); err != nil {
		if m.adminKeysManager.remote.Exists(err) {
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
	privKey, err := m.adminKeysManager.loadChannelPrivateKey(channelID)
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

	return m.adminKeysManager.saveChannelPrivateKey(channelID, pk)
}

// DeleteChannelAdminKey deletes the private key for the given channel.
//
// CAUTION: This will remove admin access. This cannot be undone. If the private
// key is deleted, it cannot be recovered and the channel can never have another
// admin.
func (m *manager) DeleteChannelAdminKey(channelID *id.ID) error {
	jww.INFO.Printf("[CH] DeleteChannelAdminKey for channel %s", channelID)
	return m.adminKeysManager.deleteChannelPrivateKey(channelID)
}

////////////////////////////////////////////////////////////////////////////////
// Storage                                                                    //
////////////////////////////////////////////////////////////////////////////////

const (
	adminKeyPrefix      = "adminKeys"
	adminKeysMapName    = "adminKeysMap"
	adminKeysMapVersion = 0
)

// adminKeysManager is responsible for handling admin key modifications for any
// channel. This is embedded within the [manager].
type adminKeysManager struct {
	callback func(ch *id.ID, isAdmin bool)
	remote   versioned.KV
	mux      sync.RWMutex
}

// newAdminKeysManager is a constructor for the adminKeysManager.
func newAdminKeysManager(
	kv versioned.KV, cb func(ch *id.ID, isAdmin bool)) *adminKeysManager {

	kvRemote, err := kv.Prefix(adminKeyPrefix)
	if err != nil {
		jww.FATAL.Panicf("[CH] Admin keys failed to prefix KV: %+v", err)
	}

	adminMan := &adminKeysManager{remote: kvRemote, callback: cb}

	err = adminMan.remote.ListenOnRemoteMap(
		adminKeysMapName, adminKeysMapVersion, adminMan.mapUpdate)
	if err != nil && adminMan.remote.Exists(err) {
		jww.FATAL.Panicf("[CH] Failed to load and listen to remote "+
			"updates on adminKeysManager: %+v", err)
	}

	return adminMan
}

// saveChannelPrivateKey saves the [rsa.PrivateKey] for the given channel ID to
// storage. This is called everytime a user generates a channel so that they can
// access the channel's private key.
//
// The private key can retrieved from storage via loadChannelPrivateKey.
func (akm *adminKeysManager) saveChannelPrivateKey(
	channelID *id.ID, pk rsa.PrivateKey) error {

	elementName := marshalChID(channelID)
	err := akm.remote.StoreMapElement(adminKeysMapName, elementName,
		&versioned.Object{
			Version:   adminKeysMapVersion,
			Timestamp: netTime.Now(),
			Data:      pk.MarshalPem(),
		}, adminKeysMapVersion)
	if err != nil {
		return err
	}

	go akm.callback(channelID, true)
	return nil
}

// loadChannelPrivateKey retrieves the [rsa.PrivateKey] for the given channel ID
// from storage.
//
// The private key is saved to storage via saveChannelPrivateKey.
func (akm *adminKeysManager) loadChannelPrivateKey(
	channelID *id.ID) (rsa.PrivateKey, error) {
	elementName := marshalChID(channelID)
	obj, err := akm.remote.GetMapElement(
		adminKeysMapName, elementName, adminKeysMapVersion)
	if err != nil {
		return nil, err
	}

	return rsa.GetScheme().UnmarshalPrivateKeyPEM(obj.Data)
}

// deleteChannelPrivateKey deletes the private key from storage for the given
// channel ID.
func (akm *adminKeysManager) deleteChannelPrivateKey(
	channelID *id.ID) error {
	elementName := marshalChID(channelID)
	_, err := akm.remote.DeleteMapElement(adminKeysMapName, elementName, adminKeysMapVersion)
	if err != nil {
		return err
	}

	go akm.callback(channelID, false)
	return nil
}

// mapUpdate handles map updates, handles by versioned.KV's ListenOnRemoteMap
// method.
func (akm *adminKeysManager) mapUpdate(edits map[string]versioned.ElementEdit) {
	akm.mux.Lock()
	defer akm.mux.Unlock()

	for elementName, edit := range edits {
		// unmarshal element name
		chanId, err := unmarshalChID(elementName)
		if err != nil {
			jww.WARN.Printf("Failed to unmarshal id in admin key "+
				"update %s on operation %s , skipping: %+v", elementName,
				edit.Operation, err)
			continue
		}

		if edit.Operation == versioned.Deleted {
			akm.callback(chanId, false)
		} else if edit.Operation == versioned.Created ||
			edit.Operation == versioned.Updated ||
			edit.Operation == versioned.Loaded {
			akm.callback(chanId, true)
		} else {
			jww.WARN.Printf("Failed to handle admin key update %s, "+
				"bad operation: %s, skipping", elementName, edit.Operation)
		}
	}
}
