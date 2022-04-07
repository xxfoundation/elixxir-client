///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package cmix

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/nodes"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	commClient "gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/mixmessages"
	commsNetwork "gitlab.com/elixxir/comms/network"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"testing"
	"time"
)

// mockManagerComms
type mockManagerComms struct {
	mockFollowNetworkComms
	mockSendCmixComms
	mockRegisterNodeComms
}

// mockFollowNetworkComms
type mockFollowNetworkComms struct{}

func (mfnc *mockFollowNetworkComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, true
}
func (mfnc *mockFollowNetworkComms) SendPoll(host *connect.Host, message *mixmessages.GatewayPoll) (
	*mixmessages.GatewayPollResponse, error) {
	return &mixmessages.GatewayPollResponse{}, nil
}

// mockSendCmixComms
type mockSendCmixComms struct{}

func (mscc *mockSendCmixComms) SendPutMessage(host *connect.Host, message *mixmessages.GatewaySlot,
	timeout time.Duration) (*mixmessages.GatewaySlotResponse, error) {
	return &mixmessages.GatewaySlotResponse{
		Accepted: true,
		RoundID:  5,
	}, nil
}

func (mscc *mockSendCmixComms) SendPutManyMessages(host *connect.Host, messages *mixmessages.GatewaySlots,
	timeout time.Duration) (*mixmessages.GatewaySlotResponse, error) {
	return &mixmessages.GatewaySlotResponse{
		Accepted: true,
		RoundID:  5,
	}, nil
}

// mockRegisterNodeComms
type mockRegisterNodeComms struct{}

func (mrnc *mockRegisterNodeComms) SendRequestClientKeyMessage(host *connect.Host,
	message *mixmessages.SignedClientKeyRequest) (*mixmessages.SignedKeyResponse, error) {
	return &mixmessages.SignedKeyResponse{}, nil
}

// mockMixCypher
type mockMixCypher struct{}

func (mmc *mockMixCypher) Encrypt(msg format.Message, salt []byte, roundID id.Round) (
	format.Message, [][]byte) {
	return format.Message{}, nil
}
func (mmc *mockMixCypher) MakeClientGatewayAuthMAC(salt, digest []byte) []byte {
	return nil
}

// mockEventManager
type mockEventManager struct{}

func (mem *mockEventManager) Report(priority int, category, evtType, details string) {}

// mockNodesRegistrar
type mockNodesRegistrar struct{}

func (mnr *mockNodesRegistrar) StartProcesses(numParallel uint) stoppable.Stoppable {
	return stoppable.NewSingle("mockNodesRegistrar")
}
func (mnr *mockNodesRegistrar) HasNode(nid *id.ID) bool {
	return true
}
func (mnr *mockNodesRegistrar) RemoveNode(nid *id.ID) {
	return
}
func (mnr *mockNodesRegistrar) GetNodeKeys(topology *connect.Circuit) (nodes.MixCypher, error) {
	return &mockMixCypher{}, nil
}
func (mnr *mockNodesRegistrar) NumRegisteredNodes() int {
	return 1
}
func (mnr *mockNodesRegistrar) GetInputChannel() chan<- commsNetwork.NodeGateway {
	return nil
}
func (mnr *mockNodesRegistrar) TriggerNodeRegistration(nid *id.ID) {
	return
}

// mockGatewaySender
type mockGatewaySender struct{}

func (mgw *mockGatewaySender) SendToAny(sendFunc func(host *connect.Host) (interface{}, error),
	stop *stoppable.Single) (interface{}, error) {
	return nil, nil
}
func (mgw *mockGatewaySender) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc,
	stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	hp := connect.GetDefaultHostParams()
	hp.MaxSendRetries = 5
	hp.MaxRetries = 5
	h, err := connect.NewHost(targets[0], "0.0.0.0", []byte(pub), hp)
	if err != nil {
		return nil, errors.WithMessage(err, "[mockGatewaySender] Failed to create host during sendtopreferred")
	}
	return sendFunc(h, targets[0], time.Second)
	//ret := &mixmessages.GatewaySlotResponse{
	//	Accepted: true,
	//	RoundID:  5,
	//}
	//return ret, nil
}
func (mgw *mockGatewaySender) UpdateNdf(ndf *ndf.NetworkDefinition) {
	return
}
func (mgw *mockGatewaySender) SetGatewayFilter(f gateway.Filter) {}
func (mgw *mockGatewaySender) GetHostParams() connect.HostParams {
	return connect.GetDefaultHostParams()
}

// mockMonitor
type mockMonitor struct{}

func (mm *mockMonitor) AddHealthCallback(f func(bool)) uint64 {
	return 0
}
func (mm *mockMonitor) RemoveHealthCallback(uint64) {
	return
}
func (mm *mockMonitor) IsHealthy() bool {
	return true
}
func (mm *mockMonitor) WasHealthy() bool {
	return true
}
func (mm *mockMonitor) StartProcesses() (stoppable.Stoppable, error) {
	return stoppable.NewSingle("t"), nil
}

// mockRoundEventRegistrar
type mockRoundEventRegistrar struct {
	statusReturn bool
}

func (mrr *mockRoundEventRegistrar) AddRoundEventChan(rid id.Round, eventChan chan ds.EventReturn,
	timeout time.Duration, validStates ...states.Round) *ds.EventCallback {
	eventChan <- ds.EventReturn{
		RoundInfo: &mixmessages.RoundInfo{
			ID:                         2,
			UpdateID:                   0,
			State:                      0,
			BatchSize:                  0,
			Topology:                   nil,
			Timestamps:                 nil,
			Errors:                     nil,
			ClientErrors:               nil,
			ResourceQueueTimeoutMillis: 0,
			Signature:                  nil,
			AddressSpaceSize:           0,
			EccSignature:               nil,
		},
		TimedOut: mrr.statusReturn,
	}
	return &ds.EventCallback{}
}

// mockCriticalSender
func mockCriticalSender(msg format.Message, recipient *id.ID,
	params CMIXParams) (id.Round, ephemeral.Id, error) {
	return id.Round(1), ephemeral.Id{}, nil
}

// mockFailCriticalSender
func mockFailCriticalSender(msg format.Message, recipient *id.ID,
	params CMIXParams) (id.Round, ephemeral.Id, error) {
	return id.Round(1), ephemeral.Id{}, errors.New("Test error")
}

func newTestManager(t *testing.T) (*client, error) {
	kv := versioned.NewKV(ekv.Memstore{})
	myID := id.NewIdFromString("zezima", id.User, t)
	comms, err := commClient.NewClientComms(myID, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	inst, err := commsNetwork.NewInstanceTesting(comms.ProtoComms, getNDF(), getNDF(), getGroup(), getGroup(), t)
	if err != nil {
		return nil, err
	}
	pk, err := rsa.GenerateKey(csprng.NewSystemRNG(), 2048)
	if err != nil {
		return nil, err
	}
	pubKey := pk.GetPublic()

	now := time.Now()
	timestamps := []uint64{
		uint64(now.Add(-30 * time.Second).UnixNano()), //PENDING
		uint64(now.Add(-25 * time.Second).UnixNano()), //PRECOMPUTING
		uint64(now.Add(-5 * time.Second).UnixNano()),  //STANDBY
		uint64(now.Add(5 * time.Second).UnixNano()),   //QUEUED
		0} //REALTIME

	nid1 := id.NewIdFromString("nid1", id.Node, t)
	nid2 := id.NewIdFromString("nid2", id.Node, t)
	nid3 := id.NewIdFromString("nid3", id.Node, t)
	ri := &mixmessages.RoundInfo{
		ID:                         3,
		UpdateID:                   0,
		State:                      uint32(states.QUEUED),
		BatchSize:                  0,
		Topology:                   [][]byte{nid1.Marshal(), nid2.Marshal(), nid3.Marshal()},
		Timestamps:                 timestamps,
		Errors:                     nil,
		ClientErrors:               nil,
		ResourceQueueTimeoutMillis: 0,
		Signature:                  nil,
		AddressSpaceSize:           4,
	}

	err = signature.SignRsa(ri, pk)
	if err != nil {
		return nil, err
	}
	rnd := ds.NewRound(ri, pubKey, nil)
	inst.GetWaitingRounds().Insert([]*ds.Round{rnd}, nil)

	m := &client{
		session:   storage.InitTestingSession(t),
		rng:       fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		instance:  inst,
		comms:     &mockManagerComms{},
		param:     GetDefaultParams(),
		Sender:    &mockGatewaySender{},
		Registrar: &mockNodesRegistrar{},
		Monitor:   &mockMonitor{},
		crit:      newCritical(kv, &mockMonitor{}, &mockRoundEventRegistrar{}, mockCriticalSender),
		events:    &mockEventManager{},
	}
	return m, nil
}

// Constructs a mock ndf
func getNDF() *ndf.NetworkDefinition {
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

var pub = "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n"

// Round IDs to return on mock historicalRounds comm
const failedHistoricalRoundID = 7
const completedHistoricalRoundID = 8
