///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package api

import (
	"fmt"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// Enum of possible round results to pass back
type RoundResult uint

const (
	TimeOut RoundResult = iota
	Failed
	Succeeded
)

func (rr RoundResult) String() string {
	switch rr {
	case TimeOut:
		return "TimeOut"
	case Failed:
		return "Failed"
	case Succeeded:
		return "Succeeded"
	default:
		return fmt.Sprintf("UNKNOWN RESULT: %d", rr)
	}
}

// Callback interface which reports the requested rounds.
// Designed such that the caller may decide how much detail they need.
// allRoundsSucceeded:
//   Returns false if any rounds in the round map were unsuccessful.
//   Returns true if ALL rounds were successful
// timedOut:
//    Returns true if any of the rounds timed out while being monitored
//	  Returns false if all rounds statuses were returned
// rounds contains a mapping of all previously requested rounds to
//   their respective round results
type RoundEventCallback func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]RoundResult)

// Comm interface for RequestHistoricalRounds.
// Constructed for testability with getRoundResults
type historicalRoundsComm interface {
	RequestHistoricalRounds(host *connect.Host,
		message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error)
	GetHost(hostId *id.ID) (*connect.Host, bool)
}

// Adjudicates on the rounds requested. Checks if they are
// older rounds or in progress rounds.
func (c *Client) GetRoundResults(roundList []id.Round, timeout time.Duration,
	roundCallback RoundEventCallback) error {

	jww.INFO.Printf("GetRoundResults(%v, %s)", roundList, timeout)

	sendResults := make(chan ds.EventReturn, len(roundList))

	return c.getRoundResults(roundList, timeout, roundCallback,
		sendResults, c.comms)
}

// Helper function which does all the logic for GetRoundResults
func (c *Client) getRoundResults(roundList []id.Round, timeout time.Duration,
	roundCallback RoundEventCallback, sendResults chan ds.EventReturn,
	commsInterface historicalRoundsComm) error {

	networkInstance := c.network.GetInstance()

	// Generate a message to track all older rounds
	historicalRequest := &pb.HistoricalRounds{
		Rounds: []uint64{},
	}

	// Generate all tracking structures for rounds
	roundEvents := c.GetRoundEvents()
	roundsResults := make(map[id.Round]RoundResult)
	allRoundsSucceeded := true
	anyRoundTimedOut := false
	numResults := 0

	oldestRound := networkInstance.GetOldestRoundID()

	// Parse and adjudicate every round
	for _, rnd := range roundList {
		// Every round is timed out by default, until proven to have finished
		roundsResults[rnd] = TimeOut
		roundInfo, err := networkInstance.GetRound(rnd)
		// If we have the round in the buffer
		if err == nil {
			// Check if the round is done (completed or failed) or in progress
			if states.Round(roundInfo.State) == states.COMPLETED {
				roundsResults[rnd] = Succeeded
			} else if states.Round(roundInfo.State) == states.FAILED {
				roundsResults[rnd] = Failed
				allRoundsSucceeded = false
			} else {
				// If in progress, add a channel monitoring its state
				roundEvents.AddRoundEventChan(rnd, sendResults,
					timeout-time.Millisecond, states.COMPLETED, states.FAILED)
				numResults++
			}
		} else {
			// Update the oldest round (buffer may have updated externally)
			if rnd < oldestRound {
				// If round is older that oldest round in our buffer
				// Add it to the historical round request (performed later)
				historicalRequest.Rounds = append(historicalRequest.Rounds, uint64(rnd))
				numResults++
			} else {
				// Otherwise, monitor its progress
				roundEvents.AddRoundEventChan(rnd, sendResults,
					timeout-time.Millisecond, states.COMPLETED, states.FAILED)
				numResults++
			}
		}
	}

	// Find out what happened to old (historical) rounds if any are needed
	if len(historicalRequest.Rounds) > 0 {
		go c.getHistoricalRounds(historicalRequest, sendResults, commsInterface)
	}

	// Determine the results of all rounds requested
	go func() {
		// Create the results timer
		timer := time.NewTimer(timeout)
		for {

			// If we know about all rounds, return
			if numResults == 0 {
				roundCallback(allRoundsSucceeded, anyRoundTimedOut, roundsResults)
				return
			}

			// Wait for info about rounds or the timeout to occur
			select {
			case <-timer.C:
				roundCallback(false, true, roundsResults)
				return
			case roundReport := <-sendResults:
				numResults--

				// Skip if the round is nil (unknown from historical rounds)
				// they default to timed out, so correct behavior is preserved
				roundId := roundReport.RoundInfo.GetRoundId()
				if roundReport.TimedOut {
					roundInfo, err := networkInstance.GetRound(roundId)
					// If we have the round in the buffer
					if err == nil {
						// Check if the round is done (completed or failed) or in progress
						if states.Round(roundInfo.State) == states.COMPLETED {
							roundsResults[roundId] = Succeeded
						} else if states.Round(roundInfo.State) == states.FAILED {
							roundsResults[roundId] = Failed
						}
					}
					allRoundsSucceeded = false
					anyRoundTimedOut = true
				} else {
					// If available, denote the result
					if states.Round(roundReport.RoundInfo.State) == states.COMPLETED {
						roundsResults[roundId] = Succeeded
					} else {
						roundsResults[roundId] = Failed
						allRoundsSucceeded = false
					}
				}
			}
		}
	}()

	return nil
}

// Helper function which asynchronously pings a random gateway until
// it gets information on its requested historical rounds
func (c *Client) getHistoricalRounds(msg *pb.HistoricalRounds,
	sendResults chan ds.EventReturn, comms historicalRoundsComm) {

	var resp *pb.HistoricalRoundsResponse

	//retry 5 times
	for i := 0; i < 5; i++ {
		// Find a gateway to request about the roundRequests
		result, err := c.GetNetworkInterface().GetSender().SendToAny(func(host *connect.Host) (interface{}, error) {
			return comms.RequestHistoricalRounds(host, msg)
		}, nil)

		// If an error, retry with (potentially) a different gw host.
		// If no error from received gateway request, exit loop
		// and process rounds
		if err == nil {
			resp = result.(*pb.HistoricalRoundsResponse)
			break
		} else {
			jww.ERROR.Printf("Failed to lookup historical rounds: %s", err)
		}
	}

	if resp == nil {
		return
	}

	// Service historical rounds, sending back to the caller thread
	for i, ri := range resp.Rounds {
		if ri == nil {
			// Handle unknown by historical rounds
			sendResults <- ds.EventReturn{
				RoundInfo: &pb.RoundInfo{ID: msg.Rounds[i]},
				TimedOut:  true,
			}
		} else {
			sendResults <- ds.EventReturn{
				RoundInfo: ri,
			}
		}
	}
}
