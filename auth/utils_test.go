////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/v4/auth/store"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	cryptoE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
/////// Mock E2E Handler //////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockE2eHandler struct {
	privKey *cyclic.Int
}

func (m mockE2eHandler) HasAuthenticatedChannel(partner *id.ID) bool {
	panic("implement me")
}

func (m mockE2eHandler) FirstPartitionSize() uint {
	panic("implement me")
}

func (m mockE2eHandler) SecondPartitionSize() uint {
	panic("implement me")
}

func (m mockE2eHandler) PartitionSize(payloadIndex uint) uint {
	panic("implement me")
}

func (m mockE2eHandler) PayloadSize() uint {
	panic("implement me")
}

func (m mockE2eHandler) GetHistoricalDHPrivkey() *cyclic.Int {
	return m.privKey
}

func (m mockE2eHandler) StartProcesses() (stoppable.Stoppable, error) {
	return nil, nil
}

func (m mockE2eHandler) SendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params e2e.Params) (cryptoE2e.SendReport, error) {
	return cryptoE2e.SendReport{}, nil
}

func (m mockE2eHandler) RegisterListener(senderID *id.ID,
	messageType catalog.MessageType,
	newListener receive.Listener) receive.ListenerID {
	return receive.ListenerID{}
}

func (m mockE2eHandler) RegisterFunc(name string, senderID *id.ID,
	messageType catalog.MessageType,
	newListener receive.ListenerFunc) receive.ListenerID {
	return receive.ListenerID{}
}

func (m mockE2eHandler) RegisterChannel(name string, senderID *id.ID,
	messageType catalog.MessageType,
	newListener chan receive.Message) receive.ListenerID {
	return receive.ListenerID{}
}

func (m mockE2eHandler) Unregister(listenerID receive.ListenerID) {
	return
}

func (m mockE2eHandler) UnregisterUserListeners(*id.ID) {}

func (m mockE2eHandler) AddPartner(partnerID *id.ID,
	partnerPubKey, myPrivKey *cyclic.Int,
	partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey,
	sendParams, receiveParams session.Params) (partner.Manager, error) {
	return nil, nil
}

func (m mockE2eHandler) GetPartner(partnerID *id.ID) (partner.Manager, error) {
	return nil, nil
}

func (m mockE2eHandler) DeletePartner(partnerId *id.ID) error {
	return nil
}

func (m mockE2eHandler) DeletePartnerNotify(partnerId *id.ID, params e2e.Params) error {
	return nil
}

func (m mockE2eHandler) GetAllPartnerIDs() []*id.ID {
	return nil
}

func (m mockE2eHandler) AddService(tag string,
	processor message.Processor) error {
	return nil
}

func (m mockE2eHandler) RemoveService(tag string) error {
	return nil
}

func (m mockE2eHandler) SendUnsafe(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params e2e.Params) ([]id.Round, time.Time, error) {
	return nil, time.Time{}, nil
}

func (m mockE2eHandler) EnableUnsafeReception() {
	return
}

func (m mockE2eHandler) GetGroup() *cyclic.Group {
	return getGroup()
}

func (m mockE2eHandler) GetHistoricalDHPubkey() *cyclic.Int {
	return nil
}

func (m mockE2eHandler) GetReceptionID() *id.ID {
	return nil
}

func (m mockE2eHandler) RegisterCallbacks(callbacks e2e.Callbacks) {
	panic("implement me")
}

func (m mockE2eHandler) AddPartnerCallbacks(partnerID *id.ID, cb e2e.Callbacks) {
	panic("implement me")
}

func (m mockE2eHandler) DeletePartnerCallbacks(partnerID *id.ID) {
	panic("implement me")
}

type mockSentRequestHandler struct{}

func (msrh *mockSentRequestHandler) Add(sr *store.SentRequest)    {}
func (msrh *mockSentRequestHandler) Delete(sr *store.SentRequest) {}

func getGroup() *cyclic.Group {
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	p := large.NewIntFromString(primeString, 16)
	g := large.NewInt(2)
	return cyclic.NewGroup(p, g)
}

// randID returns a new random ID of the specified type.
func randID(rng *rand.Rand, t id.Type) *id.ID {
	newID, _ := id.NewRandomID(rng, t)
	return newID
}

func newPayload(size int, s string) []byte {
	b := make([]byte, size)
	copy(b[:], s)
	return b
}

func newOwnership(s string) []byte {
	ownership := make([]byte, ownershipSize)
	copy(ownership[:], s)
	return ownership
}

func makeTestRound(t *testing.T) rounds.Round {
	nids := []*id.ID{
		id.NewIdFromString("one", id.User, t),
		id.NewIdFromString("two", id.User, t),
		id.NewIdFromString("three", id.User, t)}
	r := rounds.Round{
		ID:               2,
		State:            states.REALTIME,
		Topology:         connect.NewCircuit(nids),
		Timestamps:       nil,
		Errors:           nil,
		BatchSize:        0,
		AddressSpaceSize: 0,
		UpdateID:         0,
		Raw: &mixmessages.RoundInfo{
			ID:                         5,
			UpdateID:                   0,
			State:                      2,
			BatchSize:                  5,
			Topology:                   [][]byte{[]byte("test"), []byte("test")},
			Timestamps:                 []uint64{uint64(netTime.Now().UnixNano()), uint64(netTime.Now().UnixNano())},
			Errors:                     nil,
			ClientErrors:               nil,
			ResourceQueueTimeoutMillis: 0,
			Signature:                  nil,
			AddressSpaceSize:           0,
			EccSignature:               nil,
		},
	}
	return r
}
