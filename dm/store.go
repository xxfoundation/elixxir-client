////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"encoding/base64"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

// setBlock is a helper function which blocks the user across devices via the
// remote KV.
func (dc *dmClient) setBlocked(senderPubKey ed25519.PublicKey) {
	elemName := getElementName(senderPubKey)
	err := dc.remote.StoreMapElement(dmMapName, elemName,
		&versioned.Object{
			Version:   dmStoreVersion,
			Timestamp: netTime.Now(),
			Data:      senderPubKey,
		}, dmMapVersion)
	if err != nil {
		jww.WARN.Printf("[DM] Failed to remotely store user with public "+
			"key (%s) as blocked", elemName)
	}
}

// setBlock is a helper function which blocks the user across devices via the
// remote KV.
func (dc *dmClient) deleteBlocked(senderPubKey ed25519.PublicKey) {
	elemName := getElementName(senderPubKey)
	_, err := dc.remote.DeleteMapElement(dmMapName, elemName, dmMapVersion)
	if err != nil {
		jww.WARN.Printf("[DM] Failed to remotely store user with public "+
			"key (%s) as unblocked", elemName)
	}
}

// IsBlocked is a helper function which returns if the given sender is blocked.
// Blocking is controlled by the remote KV.
//
// NOTE: In the internal remote structure, a blocked user is defined as a user
// which has an entry in storage. A non-blocked user is defined as a user which
// has no entry.
//
// In practical terms, when calling GetMapElement on a user's public key, a
// successful call implies this user has been blocked, while an unsuccessful
// call implies the user has not been blocked.
//
// Please note that the current construction assumes a local KV operation,
// so network failure should not cause false negatives. If this assumption is
// changed, this construction will need modification.
func (dc *dmClient) isBlocked(senderPubKey ed25519.PublicKey) bool {
	elemName := base64.RawStdEncoding.EncodeToString(senderPubKey)

	// Check remote
	_, err := dc.remote.GetMapElement(dmMapName, elemName, dmMapVersion)

	return err == nil
}

// GetBlockedSenders is a helper function which returns all senders who are
// blocked by this user.
// Blocking is controlled by the remote KV.
func (dc *dmClient) getBlockedSenders() []ed25519.PublicKey {
	blockedMap, err := dc.remote.GetMap(dmMapName, dmMapVersion)
	if err != nil {
		jww.WARN.Panicf("[DM] Failed to retrieve map from storage: %+v", err)
		return nil
	}

	blockedSenders := make([]ed25519.PublicKey, 0)
	for elemName, elem := range blockedMap {
		pubKeyBytes, err := base64.RawStdEncoding.DecodeString(elemName)
		if err != nil {
			jww.WARN.Printf("[DM] Failed to parse blocked user %s: %+v",
				elemName, elem)
			continue
		}

		blockedSenders = append(blockedSenders, pubKeyBytes)
	}

	return blockedSenders

}

// getElementName is a helper function which retrieves the element name for the
// sender public key.
func getElementName(senderPubKey ed25519.PublicKey) string {
	return base64.RawStdEncoding.EncodeToString(senderPubKey)
}