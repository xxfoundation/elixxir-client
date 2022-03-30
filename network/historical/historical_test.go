///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package historical

import (
	"testing"
	"time"

	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Provides a smoke test to run through most of the code paths for historical
// round lookup.
func TestHistoricalRounds(t *testing.T) {
	params := GetDefaultParams()
	params.HistoricalRoundsPeriod = 500 * time.Millisecond
	params.MaxHistoricalRounds = 3
	comms := &testRoundsComms{}
	sender := &testGWSender{sendCnt: 0}
	events := &testEventMgr{}
	hMgr := NewRetriever(params, comms, sender, events)
	stopper := hMgr.StartProcesses()

	// Case 1: Send a round request and wait for timeout for processing
	err := hMgr.LookupHistoricalRound(42, func(Round, bool) {
		t.Error("Called when it should not have been.")
	})
	if err != nil {
		t.Errorf("Failed to look up historical round: %+v", err)
	}
	time.Sleep(501 * time.Millisecond)

	if sender.sendCnt != 1 {
		t.Errorf("Did not send as expected")
	}

	// Case 2: make round requests up to m.params.MaxHistoricalRounds
	for i := id.Round(0); i < 3; i++ {
		err = hMgr.LookupHistoricalRound(40+i, func(Round, bool) {
			t.Errorf("%d called when it should not have been.", i)
		})
		if err != nil {
			t.Errorf("Failed to look up historical round (%d): %+v", i, err)
		}
	}

	time.Sleep(10 * time.Millisecond)

	if sender.sendCnt != 2 {
		t.Errorf("Unexpected send count.\nexpected: %d\nreceived: %d",
			2, sender.sendCnt)
	}

	err = stopper.Close()
	if err != nil {
		t.Error(err)
	}
	if stopper.IsRunning() {
		t.Errorf("Historical rounds routine failed to close.")
	}
}

func TestProcessHistoricalRoundsResponse(t *testing.T) {
	params := GetDefaultParams()
	badRR := roundRequest{
		rid: id.Round(41),
		RoundResultCallback: func(Round, bool) {
			t.Error("Called when it should not have been.")
		},
		numAttempts: params.MaxHistoricalRoundsRetries - 2,
	}
	expiredRR := roundRequest{
		rid: id.Round(42),
		RoundResultCallback: func(info Round, success bool) {
			if info.ID == 0 && !success {
				return
			}
			t.Errorf("Expired called with bad params.")
		},
		numAttempts: params.MaxHistoricalRoundsRetries - 1,
	}
	x := false
	callbackCalled := &x
	goodRR := roundRequest{
		rid: id.Round(43),
		RoundResultCallback: func(info Round, success bool) {
			*callbackCalled = true
		},
		numAttempts: 0,
	}
	rrs := []roundRequest{badRR, expiredRR, goodRR}
	infos := make([]*pb.RoundInfo, 3)
	infos[0] = nil
	infos[1] = nil
	infos[2] = &pb.RoundInfo{ID: 43}
	response := &pb.HistoricalRoundsResponse{Rounds: infos}
	events := &testEventMgr{}

	rids, retries := processHistoricalRoundsResponse(response, rrs,
		params.MaxHistoricalRoundsRetries, events)

	if len(rids) != 1 || rids[0] != 43 {
		t.Errorf("Bad return: %v, expected [43]", rids)
	}

	// Note: one of the entries was expired, that is why this is not 2.
	if len(retries) != 1 {
		t.Errorf("retries not right length: %d != 1", len(retries))
	}

	time.Sleep(5 * time.Millisecond)

	if !*callbackCalled {
		t.Errorf("expected callback to be called")
	}
}

// Test structure implementations.
type testRoundsComms struct{}

func (t *testRoundsComms) GetHost(*id.ID) (*connect.Host, bool) {
	return nil, false
}
func (t *testRoundsComms) RequestHistoricalRounds(*connect.Host,
	*pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error) {
	return nil, nil
}

type testGWSender struct {
	sendCnt int
}

func (t *testGWSender) SendToAny(func(host *connect.Host) (interface{}, error),
	*stoppable.Single) (interface{}, error) {
	// This is always called with at least one round info set
	infos := make([]*pb.RoundInfo, 1)
	infos[0] = nil
	m := &pb.HistoricalRoundsResponse{Rounds: infos}
	t.sendCnt += 1

	return m, nil
}

func (t *testGWSender) SendToPreferred([]*id.ID, gateway.SendToPreferredFunc,
	*stoppable.Single, time.Duration) (interface{}, error) {
	return t, nil
}

func (t *testGWSender) UpdateNdf(*ndf.NetworkDefinition) {}
func (t *testGWSender) SetGatewayFilter(gateway.Filter)  {}
func (t *testGWSender) GetHostParams() connect.HostParams {
	return connect.GetDefaultHostParams()
}

type testEventMgr struct{}

func (t *testEventMgr) Report(int, string, string, string) {}
