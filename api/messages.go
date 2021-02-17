///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package api

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/interfaces/utility"
	"gitlab.com/elixxir/client/network/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Adjudicates on the rounds requested. Checks if they are older rounds or in progress rounds.
// Sends updates on the rounds with callbacks
func (c *Client)  RegisterMessageDelivery(roundList []id.Round, timeoutMS int) error {
	// Get the oldest round in the buffer
	networkInstance := c.network.GetInstance()
	oldestRound := networkInstance.GetOldestRoundID()
	timeout := time.Duration(timeoutMS)*time.Millisecond

	/*check message delivery*/
	sendResults := make(chan ds.EventReturn, len(roundList))
	roundEvents := c.GetRoundEvents()
	rndEventObjs := make([]*ds.EventCallback, len(roundList))

	// Generate a message
	msg := &pb.HistoricalRounds{
		Rounds: []uint64{},
	}

	for i, rnd := range roundList {
		if rnd < oldestRound {
			// If round is older that oldest round in our buffer
			// Add it to the historical round request (performed later)
			msg.Rounds = append(msg.Rounds, uint64(rnd))
		} else {
			// Otherwise, the round is in process OR hasn't happened yet.
			// Check if the round has happened by looking at the buffer
			ri, err := networkInstance.GetRound(rnd)
			if err != nil {
				jww.ERROR.Printf("Failed to ger round [%d] in buffer: %v", rnd, err)
				continue
			}

			// If the round is done (completed or failed) send the results
			// through the channel
			if states.Round(ri.State) == states.COMPLETED ||
				states.Round(ri.State) ==  states.FAILED {
				sendResults <- ds.EventReturn{
					RoundInfo: ri,
				}
				continue
			}

			// If it is still in progress, create a monitoring channel
			rndEventObjs[i] = roundEvents.AddRoundEventChan(rnd, sendResults,
				timeout, states.COMPLETED, states.FAILED)

		}
	}

	// Find out what happened to old (historical) rounds
	go c.getHistoricalRounds(msg, networkInstance, sendResults)

	// Determine the success of all rounds requested
	go func() {
		success, numRoundFail, numTimeout := utility.TrackResults(sendResults, len(roundList))
		if !success {
			globals.Log.WARN.Printf("RegisterMessageDelivery failed for %v/%v rounds. " +
				"%v round(s) failed, %v timeouts", numRoundFail+numTimeout, len(roundList),
				numRoundFail, numTimeout)
		}
	}()

	return nil
}

// Helper function which asynchronously pings a random gateway until
// it gets information on it's requested historical rounds
func (c *Client) getHistoricalRounds(msg *pb.HistoricalRounds,
	instance *network.Instance, sendResults chan ds.EventReturn) {

	var resp *pb.HistoricalRoundsResponse

	for {
		// Find a gateway to request about the roundRequests
		gwHost, err := gateway.Get(instance.GetPartialNdf().Get(), c.comms, c.rng.GetStream())
		if err != nil {
			globals.Log.FATAL.Panicf("Failed to track network, NDF has corrupt "+
				"data: %s", err)
		}

		// If an error, retry with (potentially) a different gw host.
		// If no error from received gateway request, exit loop
		//vand process rounds
		resp, err = c.comms.RequestHistoricalRounds(gwHost, msg)
		if err == nil {
			break
		}
	}

	// Process historical rounds, sending back to the caller thread
	for _, ri := range resp.Rounds {
		sendResults <- ds.EventReturn{
			RoundInfo: ri,
			TimedOut:  false,
		}
	}
}