////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"fmt"
	"gitlab.com/elixxir/client/cmix/rounds"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

// RoundLookupStatus is the enum of possible round results to pass back
type RoundLookupStatus uint

const (
	TimeOut RoundLookupStatus = iota
	Failed
	Succeeded
)

func (rr RoundLookupStatus) String() string {
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

type RoundResult struct {
	Status RoundLookupStatus
	Round  rounds.Round
}

type historicalRoundsRtn struct {
	Success bool
	Round   rounds.Round
}

// RoundEventCallback interface which reports the requested rounds.
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

// GetRoundResults adjudicates on the rounds requested. Checks if they are
// older rounds or in progress rounds.
func (c *client) GetRoundResults(timeout time.Duration,
	roundCallback RoundEventCallback, roundList ...id.Round) {

	jww.INFO.Printf("GetRoundResults(%v, %s)", roundList, timeout)

	sendResults := make(chan ds.EventReturn, len(roundList))

	c.getRoundResults(roundList, timeout, roundCallback,
		sendResults)
}

// Helper function which does all the logic for GetRoundResults
func (c *client) getRoundResults(roundList []id.Round, timeout time.Duration,
	roundCallback RoundEventCallback, sendResults chan ds.EventReturn) {

	networkInstance := c.GetInstance()

	// Generate a message to track all older rounds
	historicalRequest := make([]id.Round, 0, len(roundList))

	// Generate all tracking structures for rounds
	roundEvents := networkInstance.GetRoundEvents()
	roundsResults := make(map[id.Round]RoundResult)
	allRoundsSucceeded := true
	anyRoundTimedOut := false
	numResults := 0

	oldestRound := networkInstance.GetOldestRoundID()

	// Parse and adjudicate every round
	for _, rnd := range roundList {
		// Every round is timed out by default, until proven to have finished
		roundsResults[rnd] = RoundResult{
			Status: TimeOut,
		}
		roundInfo, err := networkInstance.GetRound(rnd)
		// If we have the round in the buffer
		if err == nil {
			// Check if the round is done (completed or failed) or in progress
			if states.Round(roundInfo.State) == states.COMPLETED {
				roundsResults[rnd] = RoundResult{
					Status: Succeeded,
					Round:  rounds.MakeRound(roundInfo),
				}
			} else if states.Round(roundInfo.State) == states.FAILED {
				roundsResults[rnd] = RoundResult{
					Status: Failed,
					Round:  rounds.MakeRound(roundInfo),
				}
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
				historicalRequest = append(historicalRequest, rnd)
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
	if len(historicalRequest) > 0 {
		for _, rnd := range historicalRequest {
			rrc := func(round rounds.Round, success bool) {
				result := ds.EventReturn{
					RoundInfo: round.Raw,
					TimedOut:  !success,
				}
				sendResults <- result
			}
			_ = c.Retriever.LookupHistoricalRound(rnd, rrc)
		}
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

			var result RoundResult
			hasResult := false

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
						hasResult = true
						// Check if the round is done (completed or failed) or in progress
						if states.Round(roundInfo.State) == states.COMPLETED {
							result = RoundResult{
								Status: Succeeded,
								Round:  rounds.MakeRound(roundInfo),
							}
						} else if states.Round(roundInfo.State) == states.FAILED {
							result = RoundResult{
								Status: Failed,
								Round:  rounds.MakeRound(roundInfo),
							}
							allRoundsSucceeded = false
						}
						continue
					}
					allRoundsSucceeded = false
					anyRoundTimedOut = true
				} else {
					hasResult = true
					// If available, denote the result
					if states.Round(roundReport.RoundInfo.State) == states.COMPLETED {
						result = RoundResult{
							Status: Succeeded,
							Round:  rounds.MakeRound(roundReport.RoundInfo),
						}
					} else {
						result = RoundResult{
							Status: Failed,
							Round:  rounds.MakeRound(roundReport.RoundInfo),
						}
						allRoundsSucceeded = false
					}
				}
			}
			if hasResult {
				roundsResults[result.Round.ID] = result
			}

		}
	}()
}
