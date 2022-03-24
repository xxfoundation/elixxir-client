///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package api

import (
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	cE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
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
	sender   *gateway.Sender
}
type dummyEventMgr struct{}

func (d *dummyEventMgr) Report(p int, a, b, c string) {}
func (t *testNetworkManagerGeneric) GetEventManager() event.Manager {
	return &dummyEventMgr{}
}

/* Below methods built for interface adherence */
func (t *testNetworkManagerGeneric) GetHealthTracker() interfaces.HealthTracker {
	return nil
}
func (t *testNetworkManagerGeneric) Follow(report interfaces.ClientErrorReport) (stoppable.Stoppable, error) {
	return nil, nil
}
func (t *testNetworkManagerGeneric) CheckGarbledMessages() {
	return
}
func (t *testNetworkManagerGeneric) GetVerboseRounds() string {
	return ""
}
func (t *testNetworkManagerGeneric) SendE2E(message.Send, params.E2E, *stoppable.Single) (
	[]id.Round, cE2e.MessageID, time.Time, error) {
	rounds := []id.Round{id.Round(0), id.Round(1), id.Round(2)}
	return rounds, cE2e.MessageID{}, time.Time{}, nil
}
func (t *testNetworkManagerGeneric) SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error) {
	return nil, nil
}
func (t *testNetworkManagerGeneric) SendCMIX(message format.Message, rid *id.ID, p params.CMIX) (id.Round, ephemeral.Id, error) {
	return id.Round(0), ephemeral.Id{}, nil
}
func (t *testNetworkManagerGeneric) SendManyCMIX(messages []message.TargetedCmixMessage, p params.CMIX) (id.Round, []ephemeral.Id, error) {
	return 0, []ephemeral.Id{}, nil
}
func (t *testNetworkManagerGeneric) GetInstance() *network.Instance {
	return t.instance
}
func (t *testNetworkManagerGeneric) RegisterWithPermissioning(string) ([]byte, error) {
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

func (t *testNetworkManagerGeneric) GetSender() *gateway.Sender {
	return t.sender
}

func (t *testNetworkManagerGeneric) GetAddressSize() uint8 { return 0 }

func (t *testNetworkManagerGeneric) RegisterAddressSizeNotification(string) (chan uint8, error) {
	return nil, nil
}

func (t *testNetworkManagerGeneric) UnregisterAddressSizeNotification(string) {}
func (t *testNetworkManagerGeneric) SetPoolFilter(gateway.Filter)             {}
