package ratchet

import (
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
	"io"
	"reflect"
	"strings"
	"testing"
)

func makeTestRatchet() (*Ratchet, *versioned.KV, error) {
	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
	privKey := grp.NewInt(57)
	kv := versioned.NewKV(make(ekv.Memstore))
	rng := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)
	err := New(kv, &id.ID{}, privKey, grp)
	if err != nil {
		return nil, nil, errors.Errorf("NewStore() produced an error: %v", err)
	}

	cyHanlder := mockCyHandler{}
	service := mockServices{}

	if err != nil {
		panic("NewStore() produced an error: " + err.Error())
	}

	r, err := Load(kv, &id.ID{}, grp, cyHanlder, service, rng)

	return r, kv, err
}

func managersEqual(expected, received *partner.Manager, t *testing.T) bool {
	equal := true
	if !reflect.DeepEqual(expected.GetPartnerID(), received.GetPartnerID()) {
		t.Errorf("Did not Receive expected Manager.partnerID."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.GetPartnerID(), received.GetPartnerID())
		equal = false
	}

	if !strings.EqualFold(expected.GetRelationshipFingerprint(), received.GetRelationshipFingerprint()) {
		t.Errorf("Did not Receive expected Manager.Receive."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.GetRelationshipFingerprint(), received.GetRelationshipFingerprint())
		equal = false
	}
	if !reflect.DeepEqual(expected.GetMyID(), received.GetMyID()) {
		t.Errorf("Did not Receive expected Manager.myId."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.GetMyID(), received.GetPartnerID())
		equal = false
	}

	if !reflect.DeepEqual(expected.GetMyOriginPrivateKey(), received.GetMyOriginPrivateKey()) {
		t.Errorf("Did not Receive expected Manager.MyPrivateKey."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.GetMyOriginPrivateKey(), received.GetMyOriginPrivateKey())
		equal = false
	}

	if !reflect.DeepEqual(expected.GetSendRelationshipFingerprint(), received.GetSendRelationshipFingerprint()) {
		t.Errorf("Did not Receive expected Manager.SendRelationshipFingerprint."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.GetSendRelationshipFingerprint(), received.GetSendRelationshipFingerprint())
		equal = false
	}

	return equal
}

func genSidhKeys(rng io.Reader, variant sidh.KeyVariant) (*sidh.PrivateKey, *sidh.PublicKey) {
	sidHPrivKey := util.NewSIDHPrivateKey(variant)
	sidHPubKey := util.NewSIDHPublicKey(variant)

	if err := sidHPrivKey.Generate(rng); err != nil {
		panic("failure to generate SidH A private key")
	}
	sidHPrivKey.GeneratePublicKey(sidHPubKey)

	return sidHPrivKey, sidHPubKey
}

type mockCyHandler struct{}

func (m mockCyHandler) AddKey(k *session.Cypher) {
	return
}

func (m mockCyHandler) DeleteKey(k *session.Cypher) {
	return
}

type mockServices struct{}

func (s mockServices) AddService(AddService *id.ID, newService message.Service,
	response message.Processor) {
}

func (s mockServices) DeleteService(clientID *id.ID, toDelete message.Service,
	processor message.Processor) {
}
