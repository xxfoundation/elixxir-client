package e2e

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// GetGroup returns the cyclic group used for end to end encruption
func (m *manager) GetGroup() *cyclic.Group {
	return m.grp
}

// GetDefaultHistoricalDHPubkey returns the default user's Historical DH Public Key
func (m *manager) GetDefaultHistoricalDHPubkey() *cyclic.Int {
	return m.Ratchet.GetDHPublicKey()
}

// GetDefaultHistoricalDHPrivkey returns the default user's Historical DH Private Key
func (m *manager) GetDefaultHistoricalDHPrivkey() *cyclic.Int {
	return m.Ratchet.GetDHPrivateKey()
}

// GetDefaultID returns the default IDs
func (m *manager) GetDefaultID() *id.ID {
	return m.myDefaultID
}
