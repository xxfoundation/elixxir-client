package ud

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	cryptoE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
// Mock of the E2E interface within this package //////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockE2e struct {
	grp     *cyclic.Group
	events  event.Reporter
	rng     *fastRNG.StreamGenerator
	kv      *versioned.KV
	network cmix.Client
	t       testing.TB
	key     *rsa.PrivateKey
}

func (m mockE2e) GetE2E() e2e.Handler {
	return mockE2eHandler{}
}

func (m mockE2e) GetReceptionIdentity() xxdk.ReceptionIdentity {

	dhPrivKey, _ := getGroup().NewInt(5).MarshalJSON()
	grp, _ := getGroup().MarshalJSON()

	return xxdk.ReceptionIdentity{
		ID:            id.NewIdFromString("test", id.User, m.t),
		RSAPrivatePem: rsa.CreatePrivateKeyPem(m.key),
		Salt:          []byte("test"),
		DHKeyPrivate:  dhPrivKey,
		E2eGrp:        grp,
	}
}

func (m mockE2e) GetRng() *fastRNG.StreamGenerator {
	return fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
}

func (m mockE2e) GetTransmissionIdentity() xxdk.TransmissionIdentity {
	return xxdk.TransmissionIdentity{
		ID:            id.NewIdFromString("test", id.User, m.t),
		RSAPrivatePem: m.key,
		Salt:          []byte("test"),
	}
}

func (m mockE2e) GetHistoricalDHPubkey() *cyclic.Int {
	return m.grp.NewInt(6)
}

func (m mockE2e) GetReceptionID() *id.ID {
	return id.NewIdFromString("test", id.User, m.t)
}

func (m mockE2e) GetGroup() *cyclic.Group {
	return getGroup()
}

func (m mockE2e) GetEventReporter() event.Reporter {
	return mockReporter{}
}

func (m mockE2e) GetCmix() cmix.Client {
	return m.network
}

func (m mockE2e) GetStorage() storage.Session {
	//TODO implement me
	panic("implement me")
}

///////////////////////////////////////////////////////////////////////////////
// Mock of the e2e.Handler interface within this package //////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockE2eHandler struct{}

func (m mockE2eHandler) StartProcesses() (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte, params e2e.Params) ([]id.Round, cryptoE2e.MessageID, time.Time, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) RegisterListener(senderID *id.ID, messageType catalog.MessageType, newListener receive.Listener) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) RegisterFunc(name string, senderID *id.ID, messageType catalog.MessageType, newListener receive.ListenerFunc) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) RegisterChannel(name string, senderID *id.ID, messageType catalog.MessageType, newListener chan receive.Message) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) Unregister(listenerID receive.ListenerID) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) UnregisterUserListeners(userID *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) AddPartner(partnerID *id.ID, partnerPubKey, myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey, sendParams, receiveParams session.Params) (partner.Manager, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) GetPartner(partnerID *id.ID) (partner.Manager, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) DeletePartner(partnerId *id.ID) error {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) GetAllPartnerIDs() []*id.ID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) HasAuthenticatedChannel(partner *id.ID) bool {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) AddService(tag string, processor message.Processor) error {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) RemoveService(tag string) error {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) SendUnsafe(mt catalog.MessageType, recipient *id.ID, payload []byte, params e2e.Params) ([]id.Round, time.Time, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) EnableUnsafeReception() {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) GetHistoricalDHPubkey() *cyclic.Int {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) GetHistoricalDHPrivkey() *cyclic.Int {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) GetReceptionID() *id.ID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) FirstPartitionSize() uint {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) SecondPartitionSize() uint {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) PartitionSize(payloadIndex uint) uint {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) PayloadSize() uint {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) GetGroup() *cyclic.Group {
	return getGroup()
}
