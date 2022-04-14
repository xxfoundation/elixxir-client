////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package partner

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Test implementation of the Manager interface
type testManager struct {
	partnerId                *id.ID
	grp                      *cyclic.Group
	partnerPubKey, myPrivKey *cyclic.Int
}

// NewTestManager allows creation of a Manager interface object for testing purposes
// Backwards compatibility must be maintained if you make changes here
// Currently used for: Group chat testing
func NewTestManager(partnerId *id.ID, partnerPubKey, myPrivKey *cyclic.Int, t *testing.T) Manager {
	return &testManager{partnerId: partnerId, partnerPubKey: partnerPubKey, myPrivKey: myPrivKey}
}

func (p *testManager) GetPartnerID() *id.ID {
	return p.partnerId
}

func (p *testManager) GetMyID() *id.ID {
	panic("implement me")
}

func (p *testManager) GetMyOriginPrivateKey() *cyclic.Int {
	return p.myPrivKey
}

func (p *testManager) GetPartnerOriginPublicKey() *cyclic.Int {
	return p.partnerPubKey
}

func (p *testManager) GetSendRelationshipFingerprint() []byte {
	panic("implement me")
}

func (p *testManager) GetReceiveRelationshipFingerprint() []byte {
	panic("implement me")
}

func (p *testManager) GetConnectionFingerprintBytes() []byte {
	panic("implement me")
}

func (p *testManager) GetConnectionFingerprint() string {
	panic("implement me")
}

func (p *testManager) GetContact() contact.Contact {
	panic("implement me")
}

func (p *testManager) PopSendCypher() (*session.Cypher, error) {
	panic("implement me")
}

func (p *testManager) PopRekeyCypher() (*session.Cypher, error) {
	panic("implement me")
}

func (p *testManager) NewReceiveSession(partnerPubKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params, source *session.Session) (*session.Session, bool) {
	panic("implement me")
}

func (p *testManager) NewSendSession(myDHPrivKey *cyclic.Int, mySIDHPrivateKey *sidh.PrivateKey, e2eParams session.Params, source *session.Session) *session.Session {
	panic("implement me")
}

func (p *testManager) GetSendSession(sid session.SessionID) *session.Session {
	panic("implement me")
}

func (p *testManager) GetReceiveSession(sid session.SessionID) *session.Session {
	panic("implement me")
}

func (p *testManager) Confirm(sid session.SessionID) error {
	panic("implement me")
}

func (p *testManager) TriggerNegotiations() []*session.Session {
	panic("implement me")
}

func (p *testManager) MakeService(tag string) message.Service {
	panic("implement me")
}

func (p *testManager) Delete() error {
	panic("implement me")
}
