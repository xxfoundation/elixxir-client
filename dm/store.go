////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
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
func (dc *dmClient) setBlock(senderPubKey ed25519.PublicKey) {
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

// setBlock is a helper function which unblocks the user across devices via the
// remote KV.
func (dc *dmClient) deleteBlock(senderPubKey ed25519.PublicKey) {
	elemName := base64.RawStdEncoding.EncodeToString(senderPubKey)
	dc.mux.Lock()
	defer dc.mux.Unlock()
	_, err := dc.remote.DeleteMapElement(dmMapName, elemName, dmMapVersion)
	if err != nil {
		jww.WARN.Printf("[DM] Failed to remotely store user with public "+
			"key (%s) as unblocked", elemName)
	}
}

func (dc *dmClient) getBlockedSenders() []ed25519.PublicKey {
	dc.mux.RLock()
	defer dc.mux.RUnlock()

	blockedMap, err := dc.remote.GetMap(dmMapName, dmMapVersion)
	if err != nil {
		jww.ERROR.Panicf("[DM] Failed to retrieve map from storage: %+v", err)
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

// mapUpdate acts as a listener for remote edits to the dmClient. It will
// update internal state accordingly.
func (dc *dmClient) mapUpdate(
	edits map[string]versioned.ElementEdit) {
	dc.mux.Lock()
	defer dc.mux.Unlock()

	for _, edit := range edits {
		if edit.Operation == versioned.Deleted {
			// Add logic here when needed
		} else {
			// Add logic here when needed
		}
	}
}
