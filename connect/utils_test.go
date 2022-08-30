package connect

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	cryptoE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Mock Partner Interface                                                     //
////////////////////////////////////////////////////////////////////////////////

// Tests that mockPartner adheres to the partner.Manager interface.
var _ partner.Manager = (*mockPartner)(nil)

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

func (m *mockPartner) PartnerId() *id.ID                           { return m.partnerId }
func (m *mockPartner) MyId() *id.ID                                { return m.myID }
func (m *mockPartner) MyRootPrivateKey() *cyclic.Int               { return m.myDhPrivKey }
func (m *mockPartner) PartnerRootPublicKey() *cyclic.Int           { return m.partnerDhPubKey }
func (m *mockPartner) SendRelationshipFingerprint() []byte         { return nil }
func (m *mockPartner) ReceiveRelationshipFingerprint() []byte      { return nil }
func (m *mockPartner) ConnectionFingerprint() partner.ConnectionFp { return partner.ConnectionFp{} }
func (m *mockPartner) Contact() contact.Contact {
	return contact.Contact{
		ID:       m.partnerId,
		DhPubKey: m.partnerDhPubKey,
	}
}
func (m *mockPartner) PopSendCypher() (session.Cypher, error)  { return nil, nil }
func (m *mockPartner) PopRekeyCypher() (session.Cypher, error) { return nil, nil }
func (m *mockPartner) NewReceiveSession(*cyclic.Int, *sidh.PublicKey, session.Params, *session.Session) (*session.Session, bool) {
	return nil, false
}
func (m *mockPartner) NewSendSession(*cyclic.Int, *sidh.PrivateKey, session.Params, *session.Session) *session.Session {
	return nil
}
func (m *mockPartner) GetSendSession(session.SessionID) *session.Session    { return nil }
func (m *mockPartner) GetReceiveSession(session.SessionID) *session.Session { return nil }
func (m *mockPartner) Confirm(session.SessionID) error                      { return nil }
func (m *mockPartner) TriggerNegotiations() []*session.Session              { return nil }
func (m *mockPartner) MakeService(string) message.Service                   { return message.Service{} }
func (m *mockPartner) Delete() error                                        { return nil }

////////////////////////////////////////////////////////////////////////////////
// Mock Connection Interface                                                  //
////////////////////////////////////////////////////////////////////////////////

// Tests that mockConnection adheres to the Connection interface.
var _ Connection = (*mockConnection)(nil)

type mockConnection struct {
	partner     *mockPartner
	payloadChan chan []byte
	listener    server
	lastUse     time.Time
	closed      bool
}

func newMockConnection(partnerId, myId *id.ID, myDhPrivKey,
	partnerDhPubKey *cyclic.Int) *mockConnection {

	return &mockConnection{
		partner:     newMockPartner(partnerId, myId, myDhPrivKey, partnerDhPubKey),
		payloadChan: make(chan []byte, 1),
	}
}

func (m *mockConnection) FirstPartitionSize() uint  { return 0 }
func (m *mockConnection) SecondPartitionSize() uint { return 0 }
func (m *mockConnection) PartitionSize(uint) uint   { return 0 }
func (m *mockConnection) PayloadSize() uint         { return 0 }

func (m *mockConnection) Close() error {
	m.closed = true
	return nil
}

func (m *mockConnection) GetPartner() partner.Manager { return m.partner }

func (m *mockConnection) SendE2E(
	mt catalog.MessageType, payload []byte, _ e2e.Params) (
	[]id.Round, cryptoE2e.MessageID, time.Time, error) {
	m.payloadChan <- payload
	m.listener.Hear(receive.Message{
		MessageType: mt,
		Payload:     payload,
		Sender:      m.partner.myID,
		RecipientID: m.partner.partnerId,
	})
	return nil, cryptoE2e.MessageID{}, time.Time{}, nil
}

func (m *mockConnection) RegisterListener(
	catalog.MessageType, receive.Listener) (receive.ListenerID, error) {
	return receive.ListenerID{}, nil
}
func (m *mockConnection) Unregister(receive.ListenerID) {}
func (m *mockConnection) LastUse() time.Time            { return m.lastUse }

////////////////////////////////////////////////////////////////////////////////
// Mock cMix                                                                  //
////////////////////////////////////////////////////////////////////////////////

// Tests that mockCmix adheres to the cmix.Client interface.
var _ cmix.Client = (*mockCmix)(nil)

type mockCmix struct {
	instance *network.Instance
}

func newMockCmix() *mockCmix {
	return &mockCmix{}
}

func (m *mockCmix) Follow(cmix.ClientErrorReport) (stoppable.Stoppable, error) { return nil, nil }

func (m *mockCmix) GetMaxMessageLength() int { return 4096 }

func (m *mockCmix) Send(*id.ID, format.Fingerprint, message.Service, []byte,
	[]byte, cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	return 0, ephemeral.Id{}, nil
}

func (m *mockCmix) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	return 0, ephemeral.Id{}, nil
}

func (m *mockCmix) SendMany([]cmix.TargetedCmixMessage, cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
	return 0, []ephemeral.Id{}, nil
}
func (m *mockCmix) AddIdentity(*id.ID, time.Time, bool) {}
func (m *mockCmix) RemoveIdentity(*id.ID)               {}

func (m *mockCmix) GetIdentity(*id.ID) (identity.TrackedID, error) {
	return identity.TrackedID{Creation: netTime.Now().Add(-time.Minute)}, nil
}

func (m *mockCmix) AddFingerprint(*id.ID, format.Fingerprint, message.Processor) error { return nil }
func (m *mockCmix) DeleteFingerprint(*id.ID, format.Fingerprint)                       {}
func (m *mockCmix) DeleteClientFingerprints(*id.ID)                                    {}
func (m *mockCmix) AddService(*id.ID, message.Service, message.Processor)              {}
func (m *mockCmix) DeleteService(*id.ID, message.Service, message.Processor)           {}
func (m *mockCmix) DeleteClientService(*id.ID)                                         {}
func (m *mockCmix) TrackServices(message.ServicesTracker)                              {}
func (m *mockCmix) CheckInProgressMessages()                                           {}
func (m *mockCmix) IsHealthy() bool                                                    { return true }
func (m *mockCmix) WasHealthy() bool                                                   { return true }
func (m *mockCmix) AddHealthCallback(func(bool)) uint64                                { return 0 }
func (m *mockCmix) RemoveHealthCallback(uint64)                                        {}
func (m *mockCmix) HasNode(*id.ID) bool                                                { return true }
func (m *mockCmix) NumRegisteredNodes() int                                            { return 24 }
func (m *mockCmix) TriggerNodeRegistration(*id.ID)                                     {}

func (m *mockCmix) GetRoundResults(_ time.Duration, roundCallback cmix.RoundEventCallback, _ ...id.Round) error {
	roundCallback(true, false, nil)
	return nil
}

func (m *mockCmix) LookupHistoricalRound(id.Round, rounds.RoundResultCallback) error { return nil }
func (m *mockCmix) SendToAny(func(host *connect.Host) (interface{}, error), *stoppable.Single) (interface{}, error) {
	return nil, nil
}
func (m *mockCmix) SendToPreferred([]*id.ID, gateway.SendToPreferredFunc, *stoppable.Single, time.Duration) (interface{}, error) {
	return nil, nil
}
func (m *mockCmix) SetGatewayFilter(gateway.Filter)                             {}
func (m *mockCmix) GetHostParams() connect.HostParams                           { return connect.GetDefaultHostParams() }
func (m *mockCmix) GetAddressSpace() uint8                                      { return 32 }
func (m *mockCmix) RegisterAddressSpaceNotification(string) (chan uint8, error) { return nil, nil }
func (m *mockCmix) UnregisterAddressSpaceNotification(string)                   {}
func (m *mockCmix) GetInstance() *network.Instance                              { return m.instance }
func (m *mockCmix) GetVerboseRounds() string                                    { return "" }

////////////////////////////////////////////////////////////////////////////////
// Misc set-up utils                                                          //
////////////////////////////////////////////////////////////////////////////////

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
