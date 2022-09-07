////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// GetGroup returns the cyclic group used for end to end encruption
func (m *manager) GetGroup() *cyclic.Group {
	return m.grp
}

// GetHistoricalDHPubkey returns the default user's Historical
// DH Public Key
func (m *manager) GetHistoricalDHPubkey() *cyclic.Int {
	return m.Ratchet.GetDHPublicKey()
}

// GetHistoricalDHPrivkey returns the default user's Historical DH
// Private Key
func (m *manager) GetHistoricalDHPrivkey() *cyclic.Int {
	return m.Ratchet.GetDHPrivateKey()
}

// GetDefaultID returns the default IDs
func (m *manager) GetReceptionID() *id.ID {
	return m.myID
}
