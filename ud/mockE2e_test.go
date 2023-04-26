////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/event"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	cryptoE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
// Mock of the udE2e interface within this package //////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockE2e struct {
	grp       *cyclic.Group
	events    event.Reporter
	rng       *fastRNG.StreamGenerator
	kv        versioned.KV
	network   cmix.Client
	mockStore mockStorage
	t         testing.TB
	key       rsa.PrivateKey
}

func (m mockE2e) GetBackupContainer() *xxdk.Container {
	return &xxdk.Container{}
}

func (m mockE2e) GetE2E() e2e.Handler {
	return mockE2eHandler{}
}

func (m mockE2e) GetReceptionIdentity() xxdk.ReceptionIdentity {

	dhPrivKey, _ := getGroup().NewInt(5).MarshalJSON()
	grp, _ := getGroup().MarshalJSON()

	return xxdk.ReceptionIdentity{
		ID:            id.NewIdFromString("test", id.User, m.t),
		RSAPrivatePem: m.key.MarshalPem(),
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
		ID:         id.NewIdFromString("test", id.User, m.t),
		RSAPrivate: m.key,
		Salt:       []byte("test"),
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
	return m.mockStore
}

///////////////////////////////////////////////////////////////////////////////
// Mock of the e2e.Handler interface within this package //////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockE2eHandler struct{}

func (m mockE2eHandler) RegisterCallbacks(callbacks e2e.Callbacks) {
	// TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) StartProcesses() (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte, params e2e.Params) (cryptoE2e.SendReport, error) {
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
	// TODO implement me
	panic("implement me")
}

func (tnm mockE2eHandler) DeletePartnerNotify(partnerId *id.ID, params e2e.Params) error {
	// TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) AddPartnerCallbacks(partnerID *id.ID, cb e2e.Callbacks) {
	// TODO implement me
	panic("implement me")
}

func (m mockE2eHandler) DeletePartnerCallbacks(partnerID *id.ID) {
	// TODO implement me
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
