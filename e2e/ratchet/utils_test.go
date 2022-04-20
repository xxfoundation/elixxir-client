package ratchet

import (
	"io"
	"reflect"
	"strings"
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

func managersEqual(expected, received partner.Manager, t *testing.T) bool {
	equal := true
	if !reflect.DeepEqual(expected.PartnerId(), received.PartnerId()) {
		t.Errorf("Did not Receive expected Manager.partnerID."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.PartnerId(), received.PartnerId())
		equal = false
	}

	if !strings.EqualFold(expected.ConnectionFingerprint(), received.ConnectionFingerprint()) {
		t.Errorf("Did not Receive expected Manager.Receive."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.ConnectionFingerprint(), received.ConnectionFingerprint())
		equal = false
	}
	if !reflect.DeepEqual(expected.MyId(), received.MyId()) {
		t.Errorf("Did not Receive expected Manager.myId."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.MyId(), received.PartnerId())
		equal = false
	}

	if !reflect.DeepEqual(expected.MyRootPrivateKey(), received.MyRootPrivateKey()) {
		t.Errorf("Did not Receive expected Manager.MyPrivateKey."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.MyRootPrivateKey(), received.MyRootPrivateKey())
		equal = false
	}

	if !reflect.DeepEqual(expected.SendRelationshipFingerprint(), received.SendRelationshipFingerprint()) {
		t.Errorf("Did not Receive expected Manager.SendRelationshipFingerprint."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.SendRelationshipFingerprint(), received.SendRelationshipFingerprint())
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