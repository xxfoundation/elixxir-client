package connect

import (
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	cryptoE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Mock Partner Interface                                                           //
////////////////////////////////////////////////////////////////////////////////

type mockPartner struct {
	partnerId       *id.ID
	myID            *id.ID
	myDhPrivKey     *cyclic.Int
	partnerDhPubKey *cyclic.Int
}

func newMockPartner(partnerId, myId *id.ID,
	myDhPrivKey, partnerDhPubKey *cyclic.Int) *mockPartner {
	return &mockPartner{
		partnerId:       partnerId,
		myID:            myId,
		myDhPrivKey:     myDhPrivKey,
		partnerDhPubKey: partnerDhPubKey,
	}
}

func (m mockPartner) PartnerId() *id.ID {
	return m.partnerId
}

func (m mockPartner) MyId() *id.ID {
	return m.myID
}

func (m mockPartner) MyRootPrivateKey() *cyclic.Int {
	return m.myDhPrivKey
}

func (m mockPartner) PartnerRootPublicKey() *cyclic.Int {
	return m.partnerDhPubKey
}

func (m mockPartner) SendRelationshipFingerprint() []byte {
	return nil
}

func (m mockPartner) ReceiveRelationshipFingerprint() []byte {
	return nil
}

func (m mockPartner) ConnectionFingerprint() partner.ConnectionFp {
	return partner.ConnectionFp{}
}

func (m mockPartner) Contact() contact.Contact {
	return contact.Contact{
		ID:       m.partnerId,
		DhPubKey: m.partnerDhPubKey,
	}
}

func (m mockPartner) PopSendCypher() (*session.Cypher, error) {
	return nil, nil
}

func (m mockPartner) PopRekeyCypher() (*session.Cypher, error) {
	return nil, nil
}

func (m mockPartner) NewReceiveSession(partnerPubKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey, e2eParams session.Params, source *session.Session) (*session.Session, bool) {
	return nil, false
}

func (m mockPartner) NewSendSession(myDHPrivKey *cyclic.Int, mySIDHPrivateKey *sidh.PrivateKey, e2eParams session.Params, source *session.Session) *session.Session {
	return nil
}

func (m mockPartner) GetSendSession(sid session.SessionID) *session.Session {
	return nil
}

func (m mockPartner) GetReceiveSession(sid session.SessionID) *session.Session {
	return nil
}

func (m mockPartner) Confirm(sid session.SessionID) error {
	return nil
}

func (m mockPartner) TriggerNegotiations() []*session.Session {
	return nil
}

func (m mockPartner) MakeService(tag string) message.Service {
	return message.Service{}
}

func (m mockPartner) Delete() error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Mock Connection Interface                                                           //
////////////////////////////////////////////////////////////////////////////////

type mockConnection struct {
	partner     *mockPartner
	payloadChan chan []byte
	listener    server
}

func newMockConnection(partnerId, myId *id.ID,
	myDhPrivKey, partnerDhPubKey *cyclic.Int) *mockConnection {

	return &mockConnection{
		partner: newMockPartner(partnerId, myId,
			myDhPrivKey, partnerDhPubKey),
		payloadChan: make(chan []byte, 1),
	}
}

func (m mockConnection) Close() error {
	return nil
}

func (m mockConnection) GetPartner() partner.Manager {
	return m.partner
}

func (m mockConnection) SendE2E(mt catalog.MessageType, payload []byte, params e2e.Params) ([]id.Round, cryptoE2e.MessageID, time.Time, error) {
	m.payloadChan <- payload
	m.listener.Hear(receive.Message{
		MessageType: mt,
		Payload:     payload,
		Sender:      m.partner.myID,
		RecipientID: m.partner.partnerId,
	})
	return nil, cryptoE2e.MessageID{}, time.Time{}, nil
}

func (m mockConnection) RegisterListener(messageType catalog.MessageType, newListener receive.Listener) receive.ListenerID {
	return receive.ListenerID{}
}

func (m mockConnection) Unregister(listenerID receive.ListenerID) {

}

////////////////////////////////////////////////////////////////////////////////
// Mock cMix Client                                                           //
////////////////////////////////////////////////////////////////////////////////

type mockCmixHandler struct {
	processorMap map[id.ID]map[string][]message.Processor
	sync.RWMutex
}

func newMockCmixHandler() *mockCmixHandler {
	return &mockCmixHandler{
		processorMap: make(map[id.ID]map[string][]message.Processor),
	}
}

type mockCmix struct {
	numPrimeBytes int
	health        bool
	handler       *mockCmixHandler
	instance      *network.Instance
}

func newMockCmix(handler *mockCmixHandler, face interface{}) (*mockCmix, error) {

	instanceComms := &connect.ProtoComms{
		Manager: connect.NewManagerTesting(face),
	}

	thisInstance, err := network.NewInstanceTesting(instanceComms, getNDF(),
		getNDF(), nil, nil, face)
	if err != nil {
		return nil, err
	}

	return &mockCmix{
		numPrimeBytes: 4096,
		health:        true,
		handler:       handler,
		instance:      thisInstance,
	}, nil
}

func (m mockCmix) Connect(ndf *ndf.NetworkDefinition) error {
	return nil
}

func (m *mockCmix) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
	return nil, nil
}

func (m *mockCmix) GetMaxMessageLength() int {
	return format.NewMessage(m.numPrimeBytes).ContentsSize()
}

func (m *mockCmix) Send(recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service, payload, mac []byte,
	cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	msg := format.NewMessage(m.numPrimeBytes)
	msg.SetContents(payload)
	msg.SetMac(mac)
	msg.SetKeyFP(fingerprint)

	mockRound := rounds.Round{
		State:      states.COMPLETED,
		Timestamps: map[states.Round]time.Time{states.COMPLETED: netTime.Now().Add(5 * time.Second)},
		Raw:        &pb.RoundInfo{},
	}

	m.handler.RLock()
	defer m.handler.RUnlock()
	fmt.Printf("len procs: %v\n", len(m.handler.processorMap[*recipient][service.Tag]))
	fmt.Printf("recipient: %s, tag: %s\n", recipient, service.Tag)

	if service.Tag == catalog.E2e {
		// todo need to activate the authenticated listener for the server here...
		return 0, ephemeral.Id{}, nil
	}

	for _, p := range m.handler.processorMap[*recipient][service.Tag] {
		fmt.Printf("running process with tag: %s\n", service.Tag)
		p.Process(msg, receptionID.EphemeralIdentity{}, mockRound)
	}

	return 0, ephemeral.Id{}, nil
}

func (m *mockCmix) SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
	return 0, []ephemeral.Id{}, nil
}

func (m *mockCmix) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
	m.handler.Lock()
	defer m.handler.Unlock()

	if _, exists := m.handler.processorMap[*id]; exists {
		return
	}
	fmt.Printf("AddIdentity recipient: %s\n", id)

	m.handler.processorMap[*id] = make(map[string][]message.Processor)
}

func (m *mockCmix) RemoveIdentity(id *id.ID) {
	m.handler.Lock()
	defer m.handler.Unlock()
	fmt.Printf("RemoveIdentity recipient: %s\n", id)

	delete(m.handler.processorMap, *id)
}

func (m *mockCmix) GetIdentity(get *id.ID) (identity.TrackedID, error) {
	m.handler.RLock()
	defer m.handler.RUnlock()

	return identity.TrackedID{
		Creation: netTime.Now().Add(-time.Minute),
	}, nil
}

func (m *mockCmix) AddFingerprint(identity *id.ID, fp format.Fingerprint, mp message.Processor) error {
	if _, exists := m.handler.processorMap[*identity][fp.String()]; !exists {

		if underMap := m.handler.processorMap[*identity]; underMap == nil {
			underMap = make(map[string][]message.Processor)
			underMap[fp.String()] = []message.Processor{mp}
			m.handler.processorMap[*identity] = underMap
			return nil
		}

		m.handler.processorMap[*identity][fp.String()] =
			[]message.Processor{mp}

		return nil
	}

	m.handler.processorMap[*identity][fp.String()] =
		append(m.handler.processorMap[*identity][fp.String()], mp)
	return nil
}

func (m *mockCmix) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {
	return
}

func (m *mockCmix) DeleteClientFingerprints(identity *id.ID) {
	return
}

func (m *mockCmix) AddService(clientID *id.ID, newService message.Service, response message.Processor) {
	fmt.Printf("AddService recipient: %s, tag: %s, proc: %v\n", clientID, newService.Tag, response)

	if _, exists := m.handler.processorMap[*clientID][newService.Tag]; !exists {

		if underMap := m.handler.processorMap[*clientID]; underMap == nil {
			underMap = make(map[string][]message.Processor)
			underMap[newService.Tag] = []message.Processor{response}
			m.handler.processorMap[*clientID] = underMap
			return
		}

		m.handler.processorMap[*clientID][newService.Tag] =
			[]message.Processor{response}

		return
	}

	m.handler.processorMap[*clientID][newService.Tag] =
		append(m.handler.processorMap[*clientID][newService.Tag], response)
	return
}

func (m *mockCmix) DeleteService(clientID *id.ID, toDelete message.Service, processor message.Processor) {
	return
}

func (m *mockCmix) DeleteClientService(clientID *id.ID) {
	m.handler.Lock()
	defer m.handler.Unlock()

	for tag := range m.handler.processorMap[*clientID] {
		delete(m.handler.processorMap[*clientID], tag)
	}
}

func (m *mockCmix) TrackServices(tracker message.ServicesTracker) {
	return
}

func (m *mockCmix) CheckInProgressMessages() {
	return
}

func (m *mockCmix) IsHealthy() bool {
	return true
}

func (m *mockCmix) WasHealthy() bool {
	return true
}

func (m *mockCmix) AddHealthCallback(f func(bool)) uint64 {
	return 0
}

func (m *mockCmix) RemoveHealthCallback(u uint64) {
	return
}

func (m *mockCmix) HasNode(nid *id.ID) bool {
	return true
}

func (m *mockCmix) NumRegisteredNodes() int {
	return 24
}

func (m *mockCmix) TriggerNodeRegistration(nid *id.ID) {
	return
}

func (m *mockCmix) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback, roundList ...id.Round) error {
	roundCallback(true, false, nil)
	return nil
}

func (m *mockCmix) LookupHistoricalRound(rid id.Round, callback rounds.RoundResultCallback) error {
	return nil
}

func (m *mockCmix) SendToAny(sendFunc func(host *connect.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error) {
	return nil, nil
}

func (m *mockCmix) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	return nil, nil
}

func (m *mockCmix) SetGatewayFilter(f gateway.Filter) {
	return
}

func (m *mockCmix) GetHostParams() connect.HostParams {
	return connect.GetDefaultHostParams()
}

func (m *mockCmix) GetAddressSpace() uint8 {
	return 32
}

func (m *mockCmix) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
	return nil, nil
}

func (m *mockCmix) UnregisterAddressSpaceNotification(tag string) {
	return
}

func (m *mockCmix) GetInstance() *network.Instance {
	return m.instance
}

func (m *mockCmix) GetVerboseRounds() string {
	return ""
}

////////////////////////////////////////////////////////////////////////////////
// Misc set-up utils                                                          //
////////////////////////////////////////////////////////////////////////////////

var testCert = `-----BEGIN CERTIFICATE-----
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
EnretBzQkeKeBwoB2u6NTiOmUjk=
-----END CERTIFICATE-----
`

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		UDB: ndf.UDB{
			ID:       id.DummyUser.Bytes(),
			Cert:     testCert,
			Address:  "address",
			DhPubKey: []byte{123, 34, 86, 97, 108, 117, 101, 34, 58, 53, 48, 49, 53, 53, 53, 52, 54, 53, 49, 48, 54, 49, 56, 57, 53, 54, 51, 48, 54, 52, 49, 51, 53, 49, 57, 56, 55, 57, 52, 57, 50, 48, 56, 49, 52, 57, 52, 50, 57, 51, 57, 53, 49, 50, 51, 54, 52, 56, 49, 57, 55, 48, 50, 50, 49, 48, 55, 55, 50, 52, 52, 48, 49, 54, 57, 52, 55, 52, 57, 53, 53, 56, 55, 54, 50, 57, 53, 57, 53, 48, 54, 55, 57, 55, 48, 53, 48, 48, 54, 54, 56, 49, 57, 50, 56, 48, 52, 48, 53, 51, 50, 48, 57, 55, 54, 56, 56, 53, 57, 54, 57, 56, 57, 49, 48, 54, 56, 54, 50, 52, 50, 52, 50, 56, 49, 48, 51, 51, 51, 54, 55, 53, 55, 54, 52, 51, 54, 55, 54, 56, 53, 56, 48, 55, 56, 49, 52, 55, 49, 52, 53, 49, 52, 52, 52, 52, 53, 51, 57, 57, 51, 57, 57, 53, 50, 52, 52, 53, 51, 56, 48, 49, 48, 54, 54, 55, 48, 52, 50, 49, 55, 54, 57, 53, 57, 57, 57, 51, 52, 48, 54, 54, 54, 49, 50, 48, 54, 56, 57, 51, 54, 57, 48, 52, 55, 55, 54, 50, 49, 49, 56, 56, 53, 51, 50, 57, 57, 50, 54, 53, 48, 52, 57, 51, 54, 55, 54, 48, 57, 56, 56, 49, 55, 52, 52, 57, 53, 57, 54, 53, 50, 55, 53, 52, 52, 52, 49, 57, 55, 49, 54, 50, 52, 52, 56, 50, 55, 55, 50, 49, 48, 53, 56, 56, 57, 54, 51, 53, 54, 54, 53, 53, 53, 53, 49, 56, 50, 53, 49, 49, 50, 57, 50, 48, 49, 56, 48, 48, 54, 49, 56, 57, 48, 55, 48, 51, 53, 51, 51, 56, 57, 52, 49, 50, 57, 49, 55, 50, 56, 55, 57, 57, 52, 55, 53, 51, 49, 55, 55, 48, 53, 55, 55, 49, 50, 51, 57, 49, 51, 55, 54, 48, 50, 49, 55, 50, 54, 54, 52, 56, 52, 48, 48, 54, 48, 52, 48, 53, 56, 56, 53, 54, 52, 56, 56, 49, 52, 52, 51, 57, 56, 51, 51, 57, 54, 55, 48, 49, 53, 55, 52, 53, 50, 56, 51, 49, 51, 48, 53, 52, 49, 49, 49, 49, 49, 56, 51, 53, 52, 52, 52, 52, 48, 53, 54, 57, 48, 54, 52, 56, 57, 52, 54, 53, 50, 56, 51, 53, 50, 48, 48, 50, 48, 48, 49, 50, 51, 51, 48, 48, 53, 48, 49, 50, 52, 56, 57, 48, 49, 51, 54, 55, 52, 57, 55, 50, 49, 48, 55, 53, 54, 49, 50, 52, 52, 57, 55, 48, 50, 56, 55, 55, 51, 51, 50, 53, 50, 48, 57, 52, 56, 57, 49, 49, 56, 49, 54, 57, 50, 55, 50, 51, 57, 51, 57, 54, 50, 56, 48, 54, 54, 49, 57, 55, 48, 50, 48, 57, 49, 51, 54, 50, 49, 50, 53, 50, 54, 50, 53, 53, 55, 57, 54, 51, 56, 49, 57, 48, 51, 49, 54, 54, 53, 51, 56, 56, 49, 48, 56, 48, 51, 57, 53, 49, 53, 53, 55, 49, 53, 57, 48, 57, 57, 55, 49, 56, 53, 55, 54, 48, 50, 54, 48, 49, 55, 57, 52, 55, 53, 51, 57, 49, 51, 53, 52, 49, 48, 50, 49, 55, 52, 51, 57, 48, 50, 56, 48, 50, 51, 53, 51, 54, 56, 49, 56, 50, 49, 55, 50, 57, 52, 51, 49, 56, 48, 56, 56, 50, 51, 53, 52, 56, 55, 49, 52, 55, 53, 50, 56, 48, 57, 55, 49, 53, 48, 48, 51, 50, 48, 57, 50, 50, 53, 50, 56, 51, 57, 55, 57, 49, 57, 50, 53, 56, 51, 55, 48, 51, 57, 54, 48, 50, 55, 54, 48, 54, 57, 55, 52, 53, 54, 52, 51, 56, 52, 53, 54, 48, 51, 57, 55, 55, 55, 49, 53, 57, 57, 49, 57, 52, 57, 56, 56, 54, 56, 50, 49, 49, 54, 56, 55, 56, 55, 51, 51, 57, 52, 53, 49, 52, 52, 55, 57, 53, 57, 49, 57, 52, 48, 51, 53, 49, 49, 49, 51, 48, 53, 54, 54, 50, 49, 56, 57, 52, 55, 50, 49, 54, 53, 57, 53, 50, 57, 50, 48, 51, 51, 52, 48, 56, 55, 54, 50, 49, 49, 49, 56, 53, 54, 57, 51, 57, 50, 53, 48, 53, 56, 56, 55, 56, 53, 54, 55, 51, 56, 55, 50, 53, 57, 56, 52, 54, 53, 49, 51, 50, 54, 51, 50, 48, 56, 56, 57, 52, 53, 57, 53, 56, 57, 57, 54, 52, 55, 55, 50, 57, 51, 51, 52, 55, 51, 48, 52, 56, 56, 50, 51, 50, 52, 53, 48, 51, 50, 56, 56, 50, 49, 55, 51, 51, 53, 54, 55, 51, 50, 51, 52, 56, 53, 52, 55, 48, 51, 56, 50, 51, 49, 53, 55, 52, 53, 53, 48, 55, 55, 56, 55, 48, 50, 51, 52, 50, 53, 52, 51, 48, 57, 56, 56, 54, 56, 54, 49, 57, 54, 48, 55, 55, 52, 57, 55, 56, 51, 48, 51, 57, 49, 55, 52, 49, 51, 49, 54, 57, 54, 50, 49, 52, 50, 55, 57, 55, 56, 56, 51, 49, 55, 51, 50, 54, 56, 49, 56, 53, 57, 48, 49, 49, 53, 48, 52, 53, 51, 51, 56, 52, 57, 57, 55, 54, 51, 55, 55, 48, 55, 49, 52, 50, 49, 54, 48, 49, 54, 52, 49, 57, 53, 56, 49, 54, 50, 55, 49, 52, 49, 52, 56, 49, 51, 52, 50, 53, 56, 55, 53, 57, 55, 52, 49, 57, 49, 55, 51, 55, 49, 51, 57, 54, 51, 49, 51, 49, 56, 53, 50, 49, 53, 52, 49, 51, 44, 34, 70, 105, 110, 103, 101, 114, 112, 114, 105, 110, 116, 34, 58, 49, 54, 56, 48, 49, 53, 52, 49, 53, 49, 49, 50, 51, 51, 48, 57, 56, 51, 54, 51, 125},
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

func getPrivKey() []byte {
	return []byte(`-----BEGIN PRIVATE KEY-----
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
OaRA5ZbdE7g7vxKRV36jT3wvD7W+
-----END PRIVATE KEY-----`)
}
