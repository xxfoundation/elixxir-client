///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package xxdk

import (
	"time"

	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// Mock comm struct which returns no historical round data
type noHistoricalRounds struct{}

// Constructor for noHistoricalRounds
func NewNoHistoricalRoundsComm() *noHistoricalRounds {
	return &noHistoricalRounds{}
}

// Returns no rounds back
func (ht *noHistoricalRounds) RequestHistoricalRounds(host *connect.Host,
	message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error) {
	return nil, nil
}

// Built for interface adherence
func (ht *noHistoricalRounds) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, false
}

// Generate a mock comm which returns some historical round data
type historicalRounds struct{}

// Constructor for historicalRounds comm interface
func NewHistoricalRoundsComm() *historicalRounds {
	return &historicalRounds{}
}

// Round IDs to return on mock historicalRounds comm
const failedHistoricalRoundID = 7
const completedHistoricalRoundID = 8

//  Mock comms endpoint which returns historical rounds
func (ht *historicalRounds) RequestHistoricalRounds(host *connect.Host,
	message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error) {
	// Return one successful and one failed mock round
	failedRound := &pb.RoundInfo{
		ID:    failedHistoricalRoundID,
		State: uint32(states.FAILED),
	}

	completedRound := &pb.RoundInfo{
		ID:    completedHistoricalRoundID,
		State: uint32(states.COMPLETED),
	}

	return &pb.HistoricalRoundsResponse{
		Rounds: []*pb.RoundInfo{failedRound, completedRound},
	}, nil
}

// Build for interface adherence
func (ht *historicalRounds) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, true
}

// Contains a test implementation of the networkManager interface.
type testNetworkManagerGeneric struct {
	instance *network.Instance
	sender   gateway.Sender
}
type dummyEventMgr struct{}

func (d *dummyEventMgr) Report(p int, a, b, c string) {}
func (d *dummyEventMgr) EventService() (stoppable.Stoppable, error) {
	return nil, nil
}

/* Below methods built for interface adherence */
func (t *testNetworkManagerGeneric) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
	return nil, nil
}
func (t *testNetworkManagerGeneric) GetMaxMessageLength() int { return 0 }

func (t *testNetworkManagerGeneric) CheckInProgressMessages() {
	return
}
func (t *testNetworkManagerGeneric) GetVerboseRounds() string {
	return ""
}
func (t *testNetworkManagerGeneric) AddFingerprint(identity *id.ID, fingerprint format.Fingerprint, mp message.Processor) error {
	return nil
}

func (t *testNetworkManagerGeneric) Send(*id.ID, format.Fingerprint,
	message.Service, []byte, []byte, cmix.CMIXParams) (id.Round,
	ephemeral.Id, error) {
	return id.Round(0), ephemeral.Id{}, nil
}

func (t *testNetworkManagerGeneric) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {

	return 0, ephemeral.Id{}, nil
}

func (t *testNetworkManagerGeneric) SendMany(messages []cmix.TargetedCmixMessage,
	p cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
	return 0, []ephemeral.Id{}, nil
}
func (t *testNetworkManagerGeneric) GetInstance() *network.Instance {
	return t.instance
}
func (t *testNetworkManagerGeneric) RegisterWithPermissioning(string) (
	[]byte, error) {
	return nil, nil
}
func (t *testNetworkManagerGeneric) GetRemoteVersion() (string, error) {
	return "test", nil
}
func (t *testNetworkManagerGeneric) GetStoppable() stoppable.Stoppable {
	return &stoppable.Multi{}
}

func (t *testNetworkManagerGeneric) InProgressRegistrations() int {
	return 0
}

func (t *testNetworkManagerGeneric) GetSender() gateway.Sender {
	return t.sender
}

func (t *testNetworkManagerGeneric) GetAddressSize() uint8 { return 0 }

func (t *testNetworkManagerGeneric) RegisterAddressSizeNotification(string) (chan uint8, error) {
	return nil, nil
}

func (t *testNetworkManagerGeneric) UnregisterAddressSizeNotification(string) {}
func (t *testNetworkManagerGeneric) SetPoolFilter(gateway.Filter)             {}
func (t *testNetworkManagerGeneric) AddHealthCallback(f func(bool)) uint64 {
	return 0
}
func (t *testNetworkManagerGeneric) AddIdentity(id *id.ID,
	validUntil time.Time, persistent bool) {
}
func (t *testNetworkManagerGeneric) RemoveIdentity(id *id.ID) {}
func (t *testNetworkManagerGeneric) AddService(clientID *id.ID,
	newService message.Service, response message.Processor) {
}
func (t *testNetworkManagerGeneric) DeleteService(clientID *id.ID,
	toDelete message.Service, processor message.Processor) {
}
func (t *testNetworkManagerGeneric) DeleteClientService(clientID *id.ID) {
}
func (t *testNetworkManagerGeneric) DeleteFingerprint(identity *id.ID,
	fingerprint format.Fingerprint) {
}
func (t *testNetworkManagerGeneric) DeleteClientFingerprints(identity *id.ID) {
}
func (t *testNetworkManagerGeneric) GetAddressSpace() uint8 { return 0 }
func (t *testNetworkManagerGeneric) GetHostParams() connect.HostParams {
	return connect.GetDefaultHostParams()
}
func (t *testNetworkManagerGeneric) GetIdentity(get *id.ID) (
	identity.TrackedID, error) {
	return identity.TrackedID{}, nil
}
func (t *testNetworkManagerGeneric) GetRoundResults(timeout time.Duration,
	roundCallback cmix.RoundEventCallback, roundList ...id.Round) error {
	return nil
}
func (t *testNetworkManagerGeneric) HasNode(nid *id.ID) bool { return false }
func (t *testNetworkManagerGeneric) IsHealthy() bool         { return true }
func (t *testNetworkManagerGeneric) WasHealthy() bool        { return true }
func (t *testNetworkManagerGeneric) LookupHistoricalRound(rid id.Round,
	callback rounds.RoundResultCallback) error {
	return nil
}
func (t *testNetworkManagerGeneric) NumRegisteredNodes() int { return 0 }
func (t *testNetworkManagerGeneric) RegisterAddressSpaceNotification(
	tag string) (chan uint8, error) {
	return nil, nil
}
func (t *testNetworkManagerGeneric) RemoveHealthCallback(uint64) {}
func (t *testNetworkManagerGeneric) SendToAny(
	sendFunc func(host *connect.Host) (interface{}, error),
	stop *stoppable.Single) (interface{}, error) {
	return nil, nil
}
func (t *testNetworkManagerGeneric) SendToPreferred(targets []*id.ID,
	sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single,
	timeout time.Duration) (interface{}, error) {
	return nil, nil
}
func (t *testNetworkManagerGeneric) SetGatewayFilter(f gateway.Filter) {}
func (t *testNetworkManagerGeneric) TrackServices(
	tracker message.ServicesTracker) {
}
func (t *testNetworkManagerGeneric) TriggerNodeRegistration(nid *id.ID) {}
func (t *testNetworkManagerGeneric) UnregisterAddressSpaceNotification(
	tag string) {
}
