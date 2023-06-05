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
