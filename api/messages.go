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
	"gitlab.com/elixxir/client/network/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type RoundEventCallback interface {
	Report(succeeded, timedout bool, rounds map[id.Round]RoundResult)
}

type RoundResult uint
const(
	TimeOut RoundResult = iota
	Failed
	Succeeded
)


func (c *Client)  GetRoundResults(roundList []id.Round,
	roundCallback RoundEventCallback, timeout time.Duration) error {
}

// Adjudicates on the rounds requested. Checks if they are older rounds or in progress rounds.
// Sends updates on the rounds with callbacks
func (c *Client)  getRoundResults(roundList []id.Round,
	roundCallback RoundEventCallback,  timeout time.Duration, sendResults chan ds.EventReturn, ) error {
	// Get the oldest round in the buffer
	networkInstance := c.network.GetInstance()

	/*check message delivery*/
	roundEvents := c.GetRoundEvents()
	numResults := 0

	// Generate a message
	historicalRequest := &pb.HistoricalRounds{
		Rounds: []uint64{},
	}


	rounds := make(map[id.Round]RoundResult)
	succeeded := true

	for _, rnd := range roundList {
		rounds[rnd]=TimeOut
		roundInfo, err := networkInstance.GetRound(rnd)

		if err==nil{
			if states.Round(roundInfo.State) == states.COMPLETED {
				rounds[rnd] = Succeeded
			}else if states.Round(roundInfo.State) ==  states.FAILED {
				rounds[rnd] = Failed
				succeeded = false
			}else{
				roundEvents.AddRoundEventChan(rnd, sendResults,
					timeout-time.Millisecond, states.COMPLETED, states.FAILED)
				numResults++
			}
		}else {
			jww.DEBUG.Printf("Failed to ger round [%d] in buffer: %v", rnd, err)
			// Update oldest round (buffer may have updated externally)
			oldestRound := networkInstance.GetOldestRoundID()
			if rnd < oldestRound {
				// If round is older that oldest round in our buffer
				// Add it to the historical round request (performed later)
				historicalRequest.Rounds = append(historicalRequest.Rounds, uint64(rnd))
				numResults++
			}else{
				roundEvents.AddRoundEventChan(rnd, sendResults,
					timeout-time.Millisecond, states.COMPLETED, states.FAILED)
				numResults++
			}
		}
	}

	//request historical rounds if any are needed
	if len(historicalRequest.Rounds)>0{
		// Find out what happened to old (historical) rounds
		go c.getHistoricalRounds(historicalRequest, networkInstance, sendResults)
	}

	// Determine the success of all rounds requested
	go func() {
		//create the results timer
		timer := time.NewTimer(timeout)
		for {
			//if we know about all rounds, return
			if numResults==0{
				roundCallback.Report(succeeded, true, rounds)
				return
			}

			//wait for info about rounds or the timeout to occur
			select {
			case <- timer.C:
				roundCallback.Report(false, true, rounds)
				return
			case roundReport := <-sendResults:
				numResults--
				// skip if the round is nil (unknown from historical rounds)
				// they default to timed out, so correct behavior is preserved
				if roundReport.RoundInfo==nil || roundReport.TimedOut{
					succeeded = false
				}else{
					//if available, denote the result
					if states.Round(roundReport.RoundInfo.State) == states.COMPLETED{
						rounds[id.Round(roundReport.RoundInfo.ID)] = Succeeded
					} else{
						rounds[id.Round(roundReport.RoundInfo.ID)] = Failed
						succeeded = false
					}
				}
			}
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
		sendResults<-  ds.EventReturn{
			ri,
			false,
		}
	}
}