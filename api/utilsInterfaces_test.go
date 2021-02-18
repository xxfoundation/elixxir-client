///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package api

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	cE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// A mock structure which should conform to the callback for getRoundResults
type mockRoundCallback struct {
	allRoundsSucceeded bool
	timedOut           bool
	rounds             map[id.Round]RoundResult
}

func NewMockRoundCB() *mockRoundCallback {
	return &mockRoundCallback{}
}

// Report simply stores the passed in values in the structure
func (mrc *mockRoundCallback) Report(allRoundsSucceeded, timedOut bool,
	rounds map[id.Round]RoundResult) {

	mrc.allRoundsSucceeded = allRoundsSucceeded
	mrc.timedOut = timedOut
	mrc.rounds = rounds
}

// Generate a mock comm which returns no historical round data
type noHistoricalRounds struct{}

func NewNoHistoricalRoundsComm() *noHistoricalRounds {
	return &noHistoricalRounds{}
}

// Returns no rounds back
func (ht *noHistoricalRounds) RequestHistoricalRounds(host *connect.Host,
	message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error) {
	return nil, nil
}
func (ht *noHistoricalRounds) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, false
}

// Generate a mock comm which returns some historical round data
type historicalRounds struct{}

func NewHistoricalRoundsComm() *historicalRounds {
	return &historicalRounds{}
}

// Return one successful and one failed mock round
const failedHistoricalRoundID = 7
const completedHistoricalRoundID = 8

func (ht *historicalRounds) RequestHistoricalRounds(host *connect.Host,
	message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error) {
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

func (ht *historicalRounds) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, true
}

// Contains a test implementation of the networkManager interface.
type testNetworkManagerGeneric struct {
	instance *network.Instance
}

func (t *testNetworkManagerGeneric) GetHealthTracker() interfaces.HealthTracker {
	return nil
}

func (t *testNetworkManagerGeneric) Follow() (stoppable.Stoppable, error) {
	return nil, nil
}

func (t *testNetworkManagerGeneric) CheckGarbledMessages() {
	return
}

func (t *testNetworkManagerGeneric) SendE2E(m message.Send, p params.E2E) (
	[]id.Round, cE2e.MessageID, error) {
	rounds := []id.Round{id.Round(0), id.Round(1), id.Round(2)}
	return rounds, cE2e.MessageID{}, nil

}

func (t *testNetworkManagerGeneric) SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error) {

	return nil, nil
}

func (t *testNetworkManagerGeneric) SendCMIX(message format.Message, rid *id.ID, p params.CMIX) (id.Round, ephemeral.Id, error) {

	return id.Round(0), ephemeral.Id{}, nil

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
