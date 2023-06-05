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
	elemName := base64.RawStdEncoding.EncodeToString(senderPubKey)
	dc.mux.Lock()
	defer dc.mux.Unlock()
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
	elemName := base64.RawStdEncoding.EncodeToString(senderPubKey)
	dc.mux.Lock()
	defer dc.mux.Unlock()
	_, err := dc.remote.DeleteMapElement(dmMapName, elemName, dmMapVersion)
	if err != nil {
		jww.WARN.Printf("[DM] Failed to remotely store user with public "+
			"key (%s) as unblocked", elemName)
	}
}

// IsBlocked is a helper function which returns if the given sender is blocked.
// Blocking is controlled by the Receiver / EventModel
func (dc *dmClient) isBlocked(senderPubKey ed25519.PublicKey) bool {
	elemName := base64.StdEncoding.EncodeToString(senderPubKey)

	// Check remote
	dc.mux.RLock()
	defer dc.mux.RUnlock()
	_, err := dc.remote.GetMapElement(dmMapName, elemName, dmMapVersion)
	if err != nil {
		// Check locally
		conversation := dc.receiver.GetConversation(senderPubKey)

		if conversation != nil {
			return conversation.BlockedTimestamp != nil
		}
	}

	return false
}

// GetBlockedSenders is a helper function which returns all senders who are
// blocked by this user.
// Blocking is controlled by the Receiver / EventModel
func (dc *dmClient) getBlockedSenders() []ed25519.PublicKey {
	dc.mux.RLock()
	defer dc.mux.RUnlock()

	dc.mux.RLock()
	defer dc.mux.RUnlock()

	blockedMap, err := dc.remote.GetMap(dmMapName, dmMapVersion)
	if err != nil {
		allConversations := dc.receiver.GetConversations()
		blocked := make([]ed25519.PublicKey, 0)
		for i := range allConversations {
			convo := allConversations[i]
			if convo.BlockedTimestamp != nil {
				pub := convo.Pubkey
				blocked = append(blocked, ed25519.PublicKey(pub))
			}
		}
		return blocked
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
