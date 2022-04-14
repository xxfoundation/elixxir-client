////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity"
	cmixMsg "gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	store "gitlab.com/elixxir/client/ud/store"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
	"time"

	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
)

func newTestManager(t *testing.T) *Manager {

	keyData, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Fatalf("Could not load private key: %v", err)
	}

	key, err := rsa.LoadPrivateKeyFromPem(keyData)
	if err != nil {
		t.Fatalf("Could not load public key")
	}

	kv := versioned.NewKV(ekv.Memstore{})
	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		t.Fatalf("Failed to initialize store %v", err)
	}

	// Create our Manager object
	m := &Manager{
		services: newTestNetworkManager(t),
		e2e:      mockE2e{},
		events:   event.NewEventManager(),
		user:     mockUser{testing: t, key: key},
		store:    udStore,
		comms:    &mockComms{},
		kv:       kv,
		rng:      fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
	}

	netDef := m.services.GetInstance().GetPartialNdf().Get()
	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		t.Fatalf("failed to "+
			"unmarshal UD ID from NDF: %+v", err)
	}

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	params.SendTimeout = 20 * time.Second

	// Add a new host and return it if it does not already exist
	_, err = m.comms.AddHost(udID, netDef.UDB.Address,
		[]byte(netDef.UDB.Cert), params)
	if err != nil {
		t.Fatalf("User Discovery host " +
			"object could not be constructed.")
	}

	return m
}

func newTestNetworkManager(t *testing.T) cmix.Client {
	instanceComms := &connect.ProtoComms{
		Manager: connect.NewManagerTesting(t),
	}

	thisInstance, err := network.NewInstanceTesting(instanceComms, getNDF(),
		getNDF(), nil, nil, t)
	if err != nil {
		t.Fatalf("Failed to create new test instance: %v", err)
	}

	return &testNetworkManager{
		instance: thisInstance,
	}
}

func getGroup() *cyclic.Group {
	return cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D4941"+
			"3394C049B7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688"+
			"B55B3DD2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861"+
			"575E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC"+
			"718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FF"+
			"B1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBC"+
			"A23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD"+
			"161C7738F32BF29A841698978825B4111B4BC3E1E198455095958333D776D8B2B"+
			"EEED3A1A1A221A6E37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C"+
			"4F50D7D7803D2D4F278DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F"+
			"1390B5D3FEACAF1696015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F"+
			"96789C38E89D796138E6319BE62E35D87B1048CA28BE389B575E994DCA7554715"+
			"84A09EC723742DC35873847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
}

type mockUser struct {
	testing *testing.T
	key     *rsa.PrivateKey
}

func (m mockUser) PortableUserInfo() user.Info {

	return user.Info{
		TransmissionID:        id.NewIdFromString("test", id.User, m.testing),
		TransmissionSalt:      []byte("test"),
		TransmissionRSA:       m.key,
		ReceptionID:           id.NewIdFromString("test", id.User, m.testing),
		ReceptionSalt:         []byte("test"),
		ReceptionRSA:          m.key,
		Precanned:             false,
		RegistrationTimestamp: 0,
		E2eDhPrivateKey:       getGroup().NewInt(5),
		E2eDhPublicKey:        getGroup().NewInt(6),
	}
}

func (m mockUser) GetReceptionRegistrationValidationSignature() []byte {
	return []byte("test")
}

// testNetworkManager is a test implementation of NetworkManager interface.
type testNetworkManager struct {
	instance *network.Instance
}

func (tnm *testNetworkManager) GetInstance() *network.Instance {
	return tnm.instance
}

func (tnm *testNetworkManager) GetVerboseRounds() string {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetMaxMessageLength() int {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) Send(recipient *id.ID, fingerprint format.Fingerprint, service cmixMsg.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RemoveIdentity(id *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetIdentity(get *id.ID) (identity.TrackedID, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddFingerprint(identity *id.ID, fingerprint format.Fingerprint, mp cmixMsg.Processor) error {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteClientFingerprints(identity *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddService(clientID *id.ID, newService cmixMsg.Service, response cmixMsg.Processor) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteService(clientID *id.ID, toDelete cmixMsg.Service, processor cmixMsg.Processor) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteClientService(clientID *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) TrackServices(tracker cmixMsg.ServicesTracker) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) CheckInProgressMessages() {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) IsHealthy() bool {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) WasHealthy() bool {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddHealthCallback(f func(bool)) uint64 {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RemoveHealthCallback(u uint64) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) HasNode(nid *id.ID) bool {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) NumRegisteredNodes() int {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) TriggerNodeRegistration(nid *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback, roundList ...id.Round) error {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) LookupHistoricalRound(rid id.Round, callback rounds.RoundResultCallback) error {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendToAny(sendFunc func(host *connect.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SetGatewayFilter(f gateway.Filter) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetHostParams() connect.HostParams {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetAddressSpace() uint8 {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) UnregisterAddressSpaceNotification(tag string) {
	//TODO implement me
	panic("implement me")
}

type mockUserStore struct{}

func (m mockUserStore) PortableUserInfo() user.Info {
	//TODO implement me
	panic("implement me")
}

func (m mockUserStore) GetUsername() (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockUserStore) GetReceptionRegistrationValidationSignature() []byte {
	//TODO implement me
	panic("implement me")
}

type mockComms struct {
	udHost *connect.Host
}

func (m mockComms) SendRegisterUser(host *connect.Host, message *pb.UDBUserRegistration) (*messages.Ack, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockComms) SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockComms) SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockComms) SendRemoveFact(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockComms) SendRemoveUser(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockComms) AddHost(hid *id.ID, address string, cert []byte, params connect.HostParams) (host *connect.Host, err error) {
	h, err := connect.NewHost(hid, address, cert, params)
	if err != nil {
		return nil, err
	}

	m.udHost = h
	return h, nil
}

func (m mockComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return m.udHost, true
}

type mockE2e struct{}

func (m mockE2e) SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte, params e2e.Params) ([]id.Round, e2eCrypto.MessageID, time.Time, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) RegisterListener(senderID *id.ID, messageType catalog.MessageType, newListener receive.Listener) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) RegisterFunc(name string, senderID *id.ID, messageType catalog.MessageType, newListener receive.ListenerFunc) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) SendUnsafe(mt catalog.MessageType, recipient *id.ID, payload []byte, params e2e.Params) ([]id.Round, time.Time, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) StartProcesses() (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) RegisterChannel(name string, senderID *id.ID, messageType catalog.MessageType, newListener chan receive.Message) receive.ListenerID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) Unregister(listenerID receive.ListenerID) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) AddPartner(partnerID *id.ID, partnerPubKey, myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey, sendParams, receiveParams session.Params) (partner.Manager, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) GetPartner(partnerID *id.ID) (partner.Manager, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) DeletePartner(partnerId *id.ID) error {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) GetAllPartnerIDs() []*id.ID {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) AddService(tag string, processor cmixMsg.Processor) error {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) RemoveService(tag string) error {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) EnableUnsafeReception() {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) GetGroup() *cyclic.Group {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) GetHistoricalDHPubkey() *cyclic.Int {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) GetHistoricalDHPrivkey() *cyclic.Int {
	//TODO implement me
	panic("implement me")
}

func (m mockE2e) GetReceptionID() *id.ID {
	//TODO implement me
	panic("implement me")
}

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		UDB: ndf.UDB{
			ID:      id.DummyUser.Bytes(),
			Cert:    "",
			Address: "address",
			DhPubKey: []byte{123, 34, 86, 97, 108, 117, 101, 34, 58, 49, 44, 34,
				70, 105, 110, 103, 101, 114, 112, 114, 105, 110, 116, 34, 58,
				51, 49, 54, 49, 50, 55, 48, 53, 56, 49, 51, 52, 50, 49, 54, 54,
				57, 52, 55, 125},
		},
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B7A" +
				"8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3D" +
				"D2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E78615" +
				"75E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC" +
				"6ADC718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C" +
				"4A530E8FFB1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F2" +
				"6E5785302BEDBCA23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE" +
				"448EEF78E184C7242DD161C7738F32BF29A841698978825B4111B4BC3E1E" +
				"198455095958333D776D8B2BEEED3A1A1A221A6E37E664A64B83981C46FF" +
				"DDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F278DE8014A47323" +
				"631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696015CB79C" +
				"3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E63" +
				"19BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC3" +
				"5873847AEF49F66E43873",
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642" +
				"F0B5C48C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757" +
				"264E5A1A44FFE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F" +
				"9716BFE6117C6B5B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091E" +
				"B51743BF33050C38DE235567E1B34C3D6A5C0CEAA1A0F368213C3D19843D" +
				"0B4B09DCB9FC72D39C8DE41F1BF14D4BB4563CA28371621CAD3324B6A2D3" +
				"92145BEBFAC748805236F5CA2FE92B871CD8F9C36D3292B5509CA8CAA77A" +
				"2ADFC7BFD77DDA6F71125A7456FEA153E433256A2261C6A06ED3693797E7" +
				"995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E2480" +
				"9670716C613D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D" +
				"1AA58C4328A06C46A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A33" +
				"8661D10461C0D135472085057F3494309FFA73C611F78B32ADBB5740C361" +
				"C9F35BE90997DB2014E2EF5AA61782F52ABEB8BD6432C4DD097BC5423B28" +
				"5DAFB60DC364E8161F4A2A35ACA3A10B1C4D203CC76A470A33AFDCBDD929" +
				"59859ABD8B56E1725252D78EAC66E71BA9AE3F1DD2487199874393CD4D83" +
				"2186800654760E1E34C09E4D155179F9EC0DC4473F996BDCE6EED1CABED8" +
				"B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
	}
}
