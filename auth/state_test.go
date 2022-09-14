////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/e2e"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/storage"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

///////////////////////////////////////////////////////////////////////////////
/////// Mock Event Manager ////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockEventManager struct{}

func (mem *mockEventManager) Report(priority int, category, evtType, details string) {}

type mockNetManager struct{}

func (mnm *mockNetManager) IsHealthy() bool {
	return true
}
func (mnm *mockNetManager) GetMaxMessageLength() int {
	return 5000
}
func (mnm *mockNetManager) AddService(clientID *id.ID, newService message.Service,
	response message.Processor) {
}
func (mnm *mockNetManager) DeleteService(clientID *id.ID, toDelete message.Service,
	processor message.Processor) {
}
func (mnm *mockNetManager) GetIdentity(get *id.ID) (identity.TrackedID, error) {
	return identity.TrackedID{}, nil
}
func (mnm *mockNetManager) AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
	mp message.Processor) error {
	return nil
}
func (mnm *mockNetManager) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {}
func (mnm *mockNetManager) Send(recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	return id.Round(5), ephemeral.Id{}, nil
}

type mockE2E struct {
	group      *cyclic.Group
	reception  *id.ID
	privateKey *cyclic.Int
}

func (me2e *mockE2E) GetHistoricalDHPubkey() *cyclic.Int {
	return me2e.privateKey
}
func (me2e *mockE2E) GetHistoricalDHPrivkey() *cyclic.Int {
	return me2e.group.NewInt(4)
}
func (me2e *mockE2E) GetGroup() *cyclic.Group {
	return me2e.group
}
func (me2e *mockE2E) AddPartner(partnerID *id.ID,
	partnerPubKey, myPrivKey *cyclic.Int,
	partnerSIDHPubKey *sidh.PublicKey,
	mySIDHPrivKey *sidh.PrivateKey, sendParams,
	receiveParams session.Params) (partner.Manager, error) {
	return nil, nil
}
func (me2e *mockE2E) GetPartner(partnerID *id.ID) (partner.Manager, error) {
	return nil, nil
}
func (me2e *mockE2E) DeletePartner(partnerId *id.ID) error {
	return nil
}
func (me2e *mockE2E) DeletePartnerNotify(partnerId *id.ID, params e2e.Params) error {
	return nil
}
func (me2e *mockE2E) GetReceptionID() *id.ID {
	return me2e.reception
}

type mockCallbacks struct {
	req chan bool
	con chan bool
	res chan bool
}

func (mc *mockCallbacks) Request(requestor contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	mc.req <- true
}
func (mc *mockCallbacks) Confirm(requestor contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	mc.con <- true
}
func (mc *mockCallbacks) Reset(requestor contact.Contact, receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	mc.res <- true
}

func TestManager_ReplayRequests(t *testing.T) {
	sess := storage.InitTestingSession(t)

	s, err := store.NewOrLoadStore(sess.GetKV(), sess.GetCmixGroup(), &mockSentRequestHandler{})
	if err != nil {
		t.Errorf("Failed to create store: %+v", err)
	}

	ch := make(chan bool)

	// Construct barebones manager
	m := state{
		callbacks: &mockCallbacks{
			con: ch,
		},
		net: &mockNetManager{},
		e2e: &mockE2E{
			group:     sess.GetCmixGroup(),
			reception: id.NewIdFromString("zezima", id.User, t),
		},
		rng:   fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		store: s,
		event: &mockEventManager{},
		params: Params{
			ReplayRequests: true,
		},
	}

	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	c := contact.Contact{ID: partnerID, DhPubKey: sess.GetCmixGroup().NewInt(5), OwnershipProof: []byte("proof")}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)

	r := makeTestRound(t)
	if err := m.store.AddReceived(c, sidhPubKey, r); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	_, err = m.Confirm(c)
	if err != nil {
		t.Errorf("Failed to confirm: %+v", err)
	}

	_, err = m.ReplayConfirm(partnerID)
	if err != nil {
		t.Errorf("Failed to replay confirm: %+v", err)
	}

	timeout := time.NewTimer(1 * time.Second)
	numChannelReceived := 0
loop:
	for {
		select {
		case <-ch:
			numChannelReceived++
		case <-timeout.C:
			break loop
		}
	}

	if numChannelReceived > 0 {
		t.Errorf("Unexpected number of callbacks called"+
			"\nExpected: 1"+
			"\nReceived: %d", numChannelReceived)
	}
}

func makeTestStore(t *testing.T) (*store.Store, *versioned.KV, []*cyclic.Int) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(0))
	privKeys := make([]*cyclic.Int, 10)
	for i := range privKeys {
		privKeys[i] = grp.NewInt(rand.Int63n(170) + 1)
	}

	store, err := store.NewOrLoadStore(kv, grp, &mockSentRequestHandler{})
	if err != nil {
		t.Fatalf("Failed to create new Store: %+v", err)
	}

	return store, kv, privKeys
}

func genSidhAKeys(rng io.Reader) (*sidh.PrivateKey, *sidh.PublicKey) {
	sidHPrivKeyA := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	sidHPubKeyA := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)

	if err := sidHPrivKeyA.Generate(rng); err != nil {
		panic("failure to generate SidH A private key")
	}
	sidHPrivKeyA.GeneratePublicKey(sidHPubKeyA)

	return sidHPrivKeyA, sidHPubKeyA
}
