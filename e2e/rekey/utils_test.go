////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"gitlab.com/elixxir/crypto/e2e"
	"math/rand"
	"testing"
	"time"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	session2 "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
	util "gitlab.com/elixxir/client/storage/utility"
	network2 "gitlab.com/elixxir/comms/network"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/region"
)

func GeneratePartnerID(aliceKey, bobKey *cyclic.Int,
	group *cyclic.Group, alicePrivKey *sidh.PrivateKey,
	bobPubKey *sidh.PublicKey) session2.SessionID {
	baseKey := session2.GenerateE2ESessionBaseKey(aliceKey, bobKey, group,
		alicePrivKey, bobPubKey)

	h, _ := hash.NewCMixHash()
	h.Write(baseKey.Bytes())
	sid := session2.SessionID{}

	copy(sid[:], h.Sum(nil))

	return sid
}

func genSidhKeys() (*sidh.PrivateKey, *sidh.PublicKey, *sidh.PrivateKey, *sidh.PublicKey) {
	aliceVariant := sidh.KeyVariantSidhA
	prng1 := rand.New(rand.NewSource(int64(1)))
	aliceSIDHPrivKey := util.NewSIDHPrivateKey(aliceVariant)
	aliceSIDHPubKey := util.NewSIDHPublicKey(aliceVariant)
	aliceSIDHPrivKey.Generate(prng1)
	aliceSIDHPrivKey.GeneratePublicKey(aliceSIDHPubKey)

	bobVariant := sidh.KeyVariant(sidh.KeyVariantSidhB)
	prng2 := rand.New(rand.NewSource(int64(2)))
	bobSIDHPrivKey := util.NewSIDHPrivateKey(bobVariant)
	bobSIDHPubKey := util.NewSIDHPublicKey(bobVariant)
	bobSIDHPrivKey.Generate(prng2)
	bobSIDHPrivKey.GeneratePublicKey(bobSIDHPubKey)

	return aliceSIDHPrivKey, bobSIDHPubKey, aliceSIDHPrivKey, bobSIDHPubKey
}

func testSendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, cmixParams cmix.CMIXParams) (
	e2e.SendReport, error) {
	rounds := []id.Round{id.Round(0), id.Round(1), id.Round(2)}
	alicePartner, err := r.GetPartner(aliceID)
	if err != nil {
		print(err)
	}
	bobPartner, err := r.GetPartner(bobID)
	if err != nil {
		print(err)
	}

	alicePrivKey := alicePartner.MyRootPrivateKey()
	bobPubKey := bobPartner.MyRootPrivateKey()
	grp := getGroup()

	aliceSIDHPrivKey, bobSIDHPubKey, _, _ := genSidhKeys()

	sessionID := GeneratePartnerID(alicePrivKey, bobPubKey, grp,
		aliceSIDHPrivKey, bobSIDHPubKey)

	rekeyConfirm, _ := proto.Marshal(&RekeyConfirm{
		SessionID: sessionID.Marshal(),
	})
	messagePayload := make([]byte, 0)
	messagePayload = append(payload, rekeyConfirm...)

	confirmMessage := receive.Message{
		Payload:     messagePayload,
		MessageType: catalog.KeyExchangeConfirm,
		Sender:      aliceID,
		Timestamp:   netTime.Now(),
		Encrypted:   true,
	}

	bobSwitchboard.Speak(confirmMessage)

	return e2e.SendReport{
		RoundList: rounds,
	}, nil
}

var pub = "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n"

func getNDF(t *testing.T) *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B" +
				"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE" +
				"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F" +
				"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041" +
				"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45" +
				"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209" +
				"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29" +
				"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E" +
				"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2" +
				"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696" +
				"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E" +
				"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873" +
				"847AEF49F66E43873",
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
				"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
				"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
				"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
				"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
				"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
				"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
				"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
				"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
				"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
				"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
				"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
				"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
				"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
				"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
		Registration: ndf.Registration{
			EllipticPubKey: "/WRtT+mDZGC3FXQbvuQgfqOonAjJ47IKE0zhaGTQQ70=",
		},
		Gateways: []ndf.Gateway{
			{
				ID:             id.NewIdFromString("GW1", id.Gateway, t).Bytes(),
				Address:        "0.0.0.0:11420",
				TlsCertificate: pub,
				Bin:            region.NorthernAfrica,
			},
		}, Nodes: []ndf.Node{
			{
				ID:             id.NewIdFromString("NODE1", id.Gateway, t).Bytes(),
				Address:        "0.0.0.0:11420",
				TlsCertificate: pub,
			},
		},
	}
}

func getGroup() *cyclic.Group {
	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))

	return e2eGrp

}

type mockCommsInstance struct {
	*ds.RoundEvents
}

func (mci *mockCommsInstance) GetRoundEvents() *ds.RoundEvents {
	return mci.RoundEvents
}

type mockCyHandler struct{}

func (m mockCyHandler) AddKey(session2.Cypher)    {}
func (m mockCyHandler) DeleteKey(session2.Cypher) {}

type mockServiceHandler struct {
}

func (m mockServiceHandler) AddService(AddService *id.ID, newService message.Service,
	response message.Processor) {
	return
}
func (m mockServiceHandler) DeleteService(clientID *id.ID, toDelete message.Service,
	processor message.Processor) {
	return
}

type mockNetManager struct{}

func (m *mockNetManager) GetIdentity(get *id.ID) (identity.TrackedID, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockNetManager) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
	return nil, nil
}

func (m *mockNetManager) GetMaxMessageLength() int {
	return 0
}

func (m *mockNetManager) Send(recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	return rounds.Round{}, ephemeral.Id{}, nil
}

func (m *mockNetManager) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	return rounds.Round{}, ephemeral.Id{}, nil
}

func (m *mockNetManager) SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (
	rounds.Round, []ephemeral.Id, error) {
	return rounds.Round{}, nil, nil
}

func (m *mockNetManager) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {}

func (m *mockNetManager) RemoveIdentity(id *id.ID) {}

func (m *mockNetManager) AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
	mp message.Processor) error {
	return nil
}

func (m *mockNetManager) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {}

func (m *mockNetManager) DeleteClientFingerprints(identity *id.ID) {}

func (m *mockNetManager) AddService(clientID *id.ID, newService message.Service,
	response message.Processor) {
}

func (m *mockNetManager) DeleteService(clientID *id.ID, toDelete message.Service,
	processor message.Processor) {
}

func (m *mockNetManager) DeleteClientService(clientID *id.ID) {}

func (m *mockNetManager) TrackServices(tracker message.ServicesTracker) {}

func (m *mockNetManager) CheckInProgressMessages() {}

func (m *mockNetManager) IsHealthy() bool {
	return true
}

func (m *mockNetManager) WasHealthy() bool {
	return true
}

func (m *mockNetManager) AddHealthCallback(f func(bool)) uint64 {
	return 0
}

func (m *mockNetManager) RemoveHealthCallback(uint64) {}

func (m *mockNetManager) HasNode(nid *id.ID) bool {
	return true
}

func (m *mockNetManager) NumRegisteredNodes() int {
	return 0
}

func (m *mockNetManager) TriggerNodeRegistration(nid *id.ID) {}

func (m *mockNetManager) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback,
	roundList ...id.Round) error {
	return nil
}

func (m *mockNetManager) LookupHistoricalRound(
	rid id.Round, callback rounds.RoundResultCallback) error {
	return nil
}

func (m *mockNetManager) SendToAny(sendFunc func(host *connect.Host) (interface{}, error),
	stop *stoppable.Single) (interface{}, error) {
	return nil, nil
}

func (m *mockNetManager) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc,
	stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	return nil, nil
}

func (m *mockNetManager) SetGatewayFilter(f gateway.Filter) {}

func (m *mockNetManager) GetHostParams() connect.HostParams {
	return connect.GetDefaultHostParams()
}

func (m *mockNetManager) GetAddressSpace() uint8 {
	return 0
}

func (m *mockNetManager) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
	return make(chan uint8), nil
}

func (m *mockNetManager) UnregisterAddressSpaceNotification(tag string) {}

func (m *mockNetManager) GetInstance() *network2.Instance {
	return nil
}

func (m *mockNetManager) GetVerboseRounds() string {
	return ""
}
