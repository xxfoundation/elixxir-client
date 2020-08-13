package key

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

type Manager struct {
	params Params

	partner id.ID

	receive *SessionBuff
	send    *SessionBuff
}

// generator
func NewManager(params Params, partnerID *id.ID, myPrivKey *cyclic.Int, partnerPubKey *cyclic.Int) (*Manager, error) {
	return nil, nil
}

// ekv functions
func (m *Manager) Marshal() ([]byte, error) { return nil, nil }
func (m *Manager) Unmarshal([]byte) error   { return nil }

//gets a copy of the ID of the partner
func (m *Manager) GetPartner() *id.ID {
	p := m.partner
	return &p
}

// creates a new receive session using the latest private key this user has sent
// and the new public key received from the partner
func (m *Manager) NewReceiveSession(partnerPubKey *cyclic.Int) {}

// creates a new receive session using the latest public key received from the
// partner and a mew private key for the user
func (m *Manager) NewSendSession(myPrivKey *cyclic.Int) {}

// gets the session buffer for message reception
func (m *Manager) GetReceiveSessionBuff() *SessionBuff { return nil }

// gets the session buffer for message reception
func (m *Manager) LatestReceiveSession() *Session { return nil }
