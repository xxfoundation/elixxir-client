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

// TestHistoricalRounds provides a smoke test to run through most of the code
// paths for historical round lookup.
func TestHistoricalRounds(t *testing.T) {
	params := GetDefaultParams()
	params.HistoricalRoundsPeriod = 500 * time.Millisecond
	params.MaxHistoricalRounds = 3
	comms := &testRoundsComms{}
	sender := &testGWSender{sendCnt: 0}
	events := &testEventMgr{}
	hMgr := NewRetriever(params, comms, sender, events)
	stopper := hMgr.StartProcessies()

	// case 1: Send a round request and wait for timeout for
	//         processing
	hMgr.LookupHistoricalRound(42, func(info *pb.RoundInfo, success bool) {
		t.Errorf("first called when it shouldn't")
	})
	time.Sleep(501 * time.Millisecond)

	if sender.sendCnt != 1 {
		t.Errorf("did not send as expected")
	}

	// case 2: make round requests up to m.params.MaxHistoricalRounds
	for i := 0; i < 3; i++ {
		hMgr.LookupHistoricalRound(id.Round(40+i),
			func(info *pb.RoundInfo, success bool) {
				t.Errorf("i called when it shouldn't")
			})
	}

	time.Sleep(10 * time.Millisecond)

	if sender.sendCnt != 2 {
		t.Errorf("unexpected send count: %d != 2", sender.sendCnt)
	}

	err := stopper.Close()
	if err != nil {
		t.Errorf("%+v", err)
	}
	if stopper.IsRunning() {
		t.Errorf("historical rounds routine failed to close")
	}
}

// TestHistoricalRoundsProcessing exercises the
func TestProcessHistoricalRoundsResponse(t *testing.T) {
	params := GetDefaultParams()
	bad_rr := roundRequest{
		rid: id.Round(41),
		RoundResultCallback: func(info *pb.RoundInfo, success bool) {
			t.Errorf("bad called when it shouldn't")
		},
		numAttempts: params.MaxHistoricalRoundsRetries - 2,
	}
	expired_rr := roundRequest{
		rid: id.Round(42),
		RoundResultCallback: func(info *pb.RoundInfo, success bool) {
			if info == nil && !success {
				return
			}
			t.Errorf("expired called with bad params")
		},
		numAttempts: params.MaxHistoricalRoundsRetries - 1,
	}
	x := false
	callbackCalled := &x
	good_rr := roundRequest{
		rid: id.Round(43),
		RoundResultCallback: func(info *pb.RoundInfo, success bool) {
			*callbackCalled = true
		},
		numAttempts: 0,
	}
	rrs := []roundRequest{bad_rr, expired_rr, good_rr}
	rifs := make([]*pb.RoundInfo, 3)
	rifs[0] = nil
	rifs[1] = nil
	rifs[2] = &pb.RoundInfo{ID: 43}
	response := &pb.HistoricalRoundsResponse{
		Rounds: rifs,
	}
	events := &testEventMgr{}

	rids, retries := processHistoricalRoundsResponse(response, rrs,
		params.MaxHistoricalRoundsRetries, events)

	if len(rids) != 1 || rids[0] != 43 {
		t.Errorf("bad return: %v, expected [43]", rids)
	}

	// Note: 1 of the entries was expired, thats why this is not 2.
	if len(retries) != 1 {
		t.Errorf("retries not right length: %d != 1", len(retries))
	}

	time.Sleep(5 * time.Millisecond)

	if !*callbackCalled {
		t.Errorf("expected callback to be called")
	}
}

// Test structure implementations
type testRoundsComms struct{}

func (t *testRoundsComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return nil, false
}
func (t *testRoundsComms) RequestHistoricalRounds(host *connect.Host,
	message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error) {
	return nil, nil
}

type testGWSender struct {
	sendCnt int
}

func (t *testGWSender) SendToAny(sendFunc func(host *connect.Host) (interface{},
	error), stop *stoppable.Single) (interface{}, error) {
	// this is always called with at least 1 round info set
	rifs := make([]*pb.RoundInfo, 1)
	rifs[0] = nil
	m := &pb.HistoricalRoundsResponse{Rounds: rifs}
	t.sendCnt += 1
	return m, nil
}
func (t *testGWSender) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc,
	stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	return t, nil
}
func (t *testGWSender) UpdateNdf(ndf *ndf.NetworkDefinition) {
}
func (t *testGWSender) SetGatewayFilter(f gateway.Filter) {}
func (t *testGWSender) GetHostParams() connect.HostParams {
	return connect.GetDefaultHostParams()
}

type testEventMgr struct{}

func (t *testEventMgr) Report(priority int, category, evtType, details string) {
}
