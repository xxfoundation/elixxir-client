///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package cmix

const numRounds = 10

// // Happy path
// func TestClient_GetRoundResults(t *testing.T) {
// 	// Populate a round list to request
// 	var roundList []id.Round
// 	for i := 0; i < numRounds; i++ {
// 		roundList = append(roundList, id.Round(i))
// 	}

// 	testTopology := [][]byte{id.NewIdFromUInt(1, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(2, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(3, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(4, id.Node, t).Bytes(),
// 	}

// 	// Pre-populate the results channel with successful round
// 	sendResults := make(chan ds.EventReturn, len(roundList))
// 	for i := 0; i < numRounds; i++ {
// 		sendResults <- ds.EventReturn{
// 			RoundInfo: &pb.RoundInfo{
// 				ID:       uint64(i),
// 				State:    uint32(states.COMPLETED),
// 				Topology: testTopology,
// 			},
// 			TimedOut: false,
// 		}
// 	}

// 	m, err := newTestClient(t)
// 	if err != nil {
// 		t.Fatalf("Failed in setup: %v", err)
// 	}

// 	// Construct the round call back function signature
// 	var successfulRounds, timeout bool
// 	receivedRCB := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]RoundResult) {
// 		successfulRounds = allRoundsSucceeded
// 		timeout = timedOut
// 	}

// 	// Call the round results
// 	err = m.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
// 		receivedRCB, sendResults)
// 	if err != nil {
// 		t.Errorf("Error in happy path: %v", err)
// 	}

// 	// Sleep to allow the report to come through the pipeline
// 	time.Sleep(1 * time.Second)

// 	// If any rounds timed out or any round failed, the happy path has failed
// 	if timeout || !successfulRounds {
// 		t.Errorf("Unexpected round failures in happy path. "+
// 			"Expected all rounds to succeed with no timeouts."+
// 			"\n\tTimedOut: %v"+
// 			"\n\tallRoundsSucceeded: %v", timeout, successfulRounds)
// 	}

// }

// // Checks that an two failed rounds (one timed out, one failure)
// // affects the values in the report.
// // Kept separately to ensure uncoupled failed rounds
// // affect both report booleans
// func TestClient_GetRoundResults_FailedRounds(t *testing.T) {
// 	// Populate a round list to request
// 	var roundList []id.Round
// 	for i := 0; i < numRounds; i++ {
// 		roundList = append(roundList, id.Round(i))
// 	}

// 	testTopology := [][]byte{id.NewIdFromUInt(1, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(2, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(3, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(4, id.Node, t).Bytes(),
// 	}

// 	// Pre-populate the results channel with mostly successful rounds
// 	sendResults := make(chan ds.EventReturn, len(roundList))
// 	for i := 0; i < numRounds; i++ {
// 		// Last two rounds will have a failure and a timeout respectively
// 		result := ds.EventReturn{
// 			RoundInfo: &pb.RoundInfo{
// 				ID:       uint64(i),
// 				State:    uint32(states.COMPLETED),
// 				Topology: testTopology,
// 			},
// 			TimedOut: false,
// 		}
// 		if i == numRounds-2 {
// 			result.RoundInfo.State = uint32(states.FAILED)
// 		}

// 		sendResults <- result

// 	}

// 	// Create a new copy of the test client for this test
// 	m, err := newTestClient(t)
// 	if err != nil {
// 		t.Fatalf("Failed in setup: %v", err)
// 	}

// 	// Construct the round call back function signature
// 	var successfulRounds, timeout bool
// 	receivedRCB := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]RoundResult) {
// 		successfulRounds = allRoundsSucceeded
// 		timeout = timedOut
// 	}

// 	// Call the round results
// 	err = m.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
// 		receivedRCB, sendResults)
// 	if err != nil {
// 		t.Errorf("Error in happy path: %v", err)
// 	}

// 	// Sleep to allow the report to come through the pipeline
// 	time.Sleep(2 * time.Second)

// 	// If no rounds have failed, this test has failed
// 	if successfulRounds {
// 		t.Errorf("Expected some rounds to fail. "+
// 			"\n\tTimedOut: %v"+
// 			"\n\tallRoundsSucceeded: %v", timeout, successfulRounds)
// 	}

// }

// // Use the historical rounds interface which actually sends back rounds
// func TestClient_GetRoundResults_HistoricalRounds(t *testing.T) {
// 	// Populate a round list to request
// 	var roundList []id.Round
// 	for i := 0; i < numRounds; i++ {
// 		roundList = append(roundList, id.Round(i))
// 	}

// 	testTopology := [][]byte{id.NewIdFromUInt(1, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(2, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(3, id.Node, t).Bytes(),
// 		id.NewIdFromUInt(4, id.Node, t).Bytes(),
// 	}

// 	// Pre-populate the results channel with successful rounds
// 	sendResults := make(chan ds.EventReturn, len(roundList)-2)
// 	for i := 0; i < numRounds; i++ {
// 		// Skip sending rounds intended for historical rounds comm
// 		if i == failedHistoricalRoundID ||
// 			i == completedHistoricalRoundID {
// 			continue
// 		}

// 		sendResults <- ds.EventReturn{
// 			RoundInfo: &pb.RoundInfo{
// 				ID:       uint64(i),
// 				State:    uint32(states.COMPLETED),
// 				Topology: testTopology,
// 			},
// 			TimedOut: false,
// 		}
// 	}

// 	// Create a new copy of the test client for this test
// 	m, err := newTestClient(t)
// 	if err != nil {
// 		t.Fatalf("Failed in setup: %v", err)
// 	}

// 	// Construct the round call back function signature
// 	var successfulRounds, timeout bool
// 	receivedRCB := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]RoundResult) {
// 		successfulRounds = allRoundsSucceeded
// 		timeout = timedOut
// 	}

// 	// Call the round results
// 	err = m.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
// 		receivedRCB, sendResults)
// 	if err != nil {
// 		t.Errorf("Error in happy path: %v", err)
// 	}

// 	// Sleep to allow the report to come through the pipeline
// 	time.Sleep(2 * time.Second)

// 	// If no round failed, this test has failed
// 	if successfulRounds {
// 		t.Errorf("Expected historical rounds to have round failures"+
// 			"\n\tTimedOut: %v"+
// 			"\n\tallRoundsSucceeded: %v", timeout, successfulRounds)
// 	}
// }

// // Force some timeouts by not populating the entire results channel
// func TestClient_GetRoundResults_Timeout(t *testing.T) {
// 	// Populate a round list to request
// 	var roundList []id.Round
// 	for i := 0; i < numRounds; i++ {
// 		roundList = append(roundList, id.Round(i))
// 	}

// 	// Create a broken channel which will never send,
// 	// forcing a timeout
// 	var sendResults chan ds.EventReturn
// 	sendResults = nil

// 	// Create a new copy of the test client for this test
// 	m, err := newTestClient(t)
// 	if err != nil {
// 		t.Fatalf("Failed in setup: %v", err)
// 	}

// 	// Construct the round call back function signature
// 	var successfulRounds, timeout bool
// 	receivedRCB := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]RoundResult) {
// 		successfulRounds = allRoundsSucceeded
// 		timeout = timedOut
// 	}

// 	// Call the round results
// 	err = m.getRoundResults(roundList, time.Duration(10)*time.Millisecond,
// 		receivedRCB, sendResults)
// 	if err != nil {
// 		t.Errorf("Error in happy path: %v", err)
// 	}

// 	// Sleep to allow the report to come through the pipeline
// 	time.Sleep(2 * time.Second)

// 	// If no rounds have timed out , this test has failed
// 	if !timeout {
// 		t.Errorf("Expected all rounds to timeout with no valid round reporter."+
// 			"\n\tTimedOut: %v"+
// 			"\n\tallRoundsSucceeded: %v", timeout, successfulRounds)
// 	}

// }
