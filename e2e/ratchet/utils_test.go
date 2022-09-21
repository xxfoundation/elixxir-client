////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
	"io"
	"reflect"
	"testing"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
)

// Constructor for a mock ratchet
func makeTestRatchet() (*Ratchet, *versioned.KV, error) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	privKey := grp.NewInt(57)
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)
	err := New(kv, &id.ID{}, privKey, grp)
	if err != nil {
		return nil, nil,
			errors.Errorf("NewStore() produced an error: %v", err)
	}

	cyHanlder := mockCyHandler{}
	cyHanlderLegacySIDH := mockCyHandlerLegacySIDH{}
	service := mockServices{}

	if err != nil {
		panic("NewStore() produced an error: " + err.Error())
	}

	r, err := Load(kv, &id.ID{}, grp, cyHanlder, cyHanlderLegacySIDH, service, rng)

	return r, kv, err
}

// Helper function which compares 2 partner.Manager's.
func managersEqual(expected, received partner.Manager, t *testing.T) bool {
	equal := true
	if !reflect.DeepEqual(expected.PartnerId(), received.PartnerId()) {
		t.Errorf("Did not Receive expected Manager.partnerID."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.PartnerId(), received.PartnerId())
		equal = false
	}

	if !reflect.DeepEqual(expected.ConnectionFingerprint(), received.ConnectionFingerprint()) {
		t.Errorf("Did not Receive expected Manager.Receive."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.ConnectionFingerprint(),
			received.ConnectionFingerprint())
		equal = false
	}
	if !reflect.DeepEqual(expected.MyId(), received.MyId()) {
		t.Errorf("Did not Receive expected Manager.myId."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.MyId(), received.PartnerId())
		equal = false
	}

	if !reflect.DeepEqual(expected.MyRootPrivateKey(),
		received.MyRootPrivateKey()) {
		t.Errorf("Did not Receive expected Manager.MyPrivateKey."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.MyRootPrivateKey(), received.MyRootPrivateKey())
		equal = false
	}

	if !reflect.DeepEqual(expected.SendRelationshipFingerprint(),
		received.SendRelationshipFingerprint()) {
		t.Errorf("Did not Receive expected Manager."+
			"SendRelationshipFingerprint."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.SendRelationshipFingerprint(),
			received.SendRelationshipFingerprint())
		equal = false
	}

	return equal
}

// Helper function for generating sidh keys.
func genSidhKeys(rng io.Reader, variant sidh.KeyVariant) (*sidh.PrivateKey,
	*sidh.PublicKey) {
	sidHPrivKey := util.NewSIDHPrivateKey(variant)
	sidHPubKey := util.NewSIDHPublicKey(variant)

	if err := sidHPrivKey.Generate(rng); err != nil {
		panic("failure to generate SidH A private key")
	}
	sidHPrivKey.GeneratePublicKey(sidHPubKey)

	return sidHPrivKey, sidHPubKey
}

// Implements a mock session.CypherHandler.
type mockCyHandler struct{}

func (m mockCyHandler) AddKey(k session.Cypher) {
	return
}

func (m mockCyHandler) DeleteKey(k session.Cypher) {
	return
}

// Implements a mock Services interface.
type mockServices struct{}

func (s mockServices) AddService(AddService *id.ID,
	newService message.Service,
	response message.Processor) {
}

func (s mockServices) DeleteService(clientID *id.ID,
	toDelete message.Service,
	processor message.Processor) {
}

// Implements a message.Processor interface.
type mockProcessor struct {
	name string
}

func (m *mockProcessor) Process(message format.Message,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {

}

func (m *mockProcessor) String() string {
	return m.name
}
