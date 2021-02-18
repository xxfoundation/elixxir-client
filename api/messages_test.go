///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package api

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"testing"
	"time"
)

const numRounds = 10

var testClient *Client

func TestMain(m *testing.M) {
	var err error
	testClient, err = newTestingClient(m)
	t := testing.T{}
	if err != nil {
		t.Errorf("Failed in setup: %v", err)
	}

	os.Exit(m.Run())
}

// Happy path
func TestClient_GetRoundResults(t *testing.T) {
	// Populate a round list to request
	var roundList []id.Round
	for i := 0; i < numRounds; i++ {
		roundList = append(roundList, id.Round(i))
	}

	// Pre-populate the results channel with successful rounds
	sendResults := make(chan ds.EventReturn, len(roundList))
	for i := 0; i < numRounds; i++ {
		sendResults <- ds.EventReturn{
			RoundInfo: &pb.RoundInfo{
				ID:    uint64(i),
				State: uint32(states.COMPLETED),
			},
			TimedOut: false,
		}
	}

	// Create a new copy of the test client for this test
	client := &Client{}
	*client = *testClient

	// Call the round results
	receivedRCB := NewMockRoundCB()
	err := client.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
		receivedRCB, sendResults, NewNoHistoricalRoundsComm())
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	// Sleep to allow the report to come through the pipeline
	time.Sleep(1 * time.Second)

	// If any rounds timed out or any round failed, the happy path has failed
	if receivedRCB.timedOut || !receivedRCB.allRoundsSucceeded {
		t.Errorf("Unexpected round failures in happy path. "+
			"Expected all rounds to succeed with no timeouts."+
			"\n\tTimedOut: %v"+
			"\n\tallRoundsSucceeded: %v", receivedRCB.timedOut, receivedRCB.allRoundsSucceeded)
	}

}

// Checks that an two failed rounds (one timed out, one failure)
// affects the values in the report.
// Kept separately to ensure uncoupled failed rounds
// affect both report booleans
func TestClient_GetRoundResults_FailedRounds(t *testing.T) {
	// Populate a round list to request
	var roundList []id.Round
	for i := 0; i < numRounds; i++ {
		roundList = append(roundList, id.Round(i))
	}

	// Pre-populate the results channel with mostly successful rounds
	sendResults := make(chan ds.EventReturn, len(roundList))
	for i := 0; i < numRounds; i++ {
		// Last two rounds will have a failure and a timeout respectively
		result := ds.EventReturn{
			RoundInfo: &pb.RoundInfo{
				ID:    uint64(i),
				State: uint32(states.COMPLETED),
			},
			TimedOut: false,
		}
		if i == numRounds-2 {
			result.RoundInfo.State = uint32(states.FAILED)
			sendResults <- result
		} else if i == numRounds-1 {
			result.TimedOut = true
			sendResults <- result
		} else {
			sendResults <- result
		}

	}

	// Create a new copy of the test client for this test
	client := &Client{}
	*client = *testClient

	// Call the round results
	receivedRCB := NewMockRoundCB()
	err := client.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
		receivedRCB, sendResults, NewNoHistoricalRoundsComm())
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	// Sleep to allow the report to come through the pipeline
	time.Sleep(2 * time.Second)

	// If no rounds have timed out or no round failed, this test has failed
	if !receivedRCB.timedOut || receivedRCB.allRoundsSucceeded {
		t.Errorf("Expected some rounds to fail and others to timeout. "+
			"\n\tTimedOut: %v"+
			"\n\tallRoundsSucceeded: %v", receivedRCB.timedOut, receivedRCB.allRoundsSucceeded)
	}

}

// Force some timeouts by not populating the entire results channel
func TestClient_GetRoundResults_Timeout(t *testing.T) {
	// Populate a round list to request
	var roundList []id.Round
	for i := 0; i < numRounds; i++ {
		roundList = append(roundList, id.Round(i))
	}

	// Generate a results which never sends (empty chan)
	sendResults := make(chan ds.EventReturn)

	// Create a new copy of the test client for this test
	client := &Client{}
	*client = *testClient

	// Call the round results
	receivedRCB := NewMockRoundCB()
	err := client.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
		receivedRCB, sendResults, NewNoHistoricalRoundsComm())
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
	// Sleep to allow the report to come through the pipeline
	time.Sleep(2*time.Second)

	// If no rounds have timed out , this test has failed
	if !receivedRCB.timedOut  {
		t.Errorf("Unexpected round failures in happy path. "+
			"Expected all rounds to succeed with no timeouts."+
			"\n\tTimedOut: %v", receivedRCB.timedOut)
	}

}

// Use the historical rounds interface which actually sends back rounds
func TestClient_GetRoundResults_HistoricalRounds(t *testing.T)  {
	// Populate a round list to request
	var roundList []id.Round
	for i := 0; i < numRounds; i++ {
		roundList = append(roundList, id.Round(i))
	}

	// Pre-populate the results channel with successful rounds
	sendResults := make(chan ds.EventReturn, len(roundList)-2)
	for i := 0; i < numRounds; i++ {
		// Skip sending rounds intended for historical rounds comm
		if i == failedHistoricalRoundID ||
			i == completedHistoricalRoundID {continue}

		sendResults <- ds.EventReturn{
			RoundInfo: &pb.RoundInfo{
				ID:    uint64(i),
				State: uint32(states.COMPLETED),
			},
			TimedOut: false,
		}
	}


	// Create a new copy of the test client for this test
	client := &Client{}
	*client = *testClient


	// Overpopulate the round buffer, ensuring a circle back of the ring buffer
	for i := 1; i <= ds.RoundInfoBufLen + completedHistoricalRoundID + 1 ; i++ {
		ri := &pb.RoundInfo{ID: uint64(i)}
		signRoundInfo(ri)
		client.network.GetInstance().RoundUpdate(ri)

	}

	// Call the round results
	receivedRCB := NewMockRoundCB()
	err := client.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
		receivedRCB, sendResults, NewHistoricalRoundsComm())
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
	// Sleep to allow the report to come through the pipeline
	time.Sleep(2*time.Second)

	// If no round failed, this test has failed
	if  receivedRCB.allRoundsSucceeded {
		t.Errorf("Unexpected round failures in happy path. "+
			"Expected all rounds to succeed with no timeouts."+
			"\n\tTimedOut: %v"+
			"\n\tallRoundsSucceeded: %v", receivedRCB.timedOut, receivedRCB.allRoundsSucceeded)
	}
}