///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/storage/reception"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Historical Rounds looks up the round history via random gateways.
// It batches these quests but never waits longer than
// params.HistoricalRoundsPeriod to do a lookup.
// Historical rounds receives input from:
//   - Network Follower (/network/follow.go)
// Historical Rounds sends the output to:
//	 - Message Retrieval Workers (/network/round/retrieve.go)

//interface to increase east of testing of historical rounds
type historicalRoundsComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	RequestHistoricalRounds(host *connect.Host,
		message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error)
}

//structure which contains a historical round lookup
type historicalRoundRequest struct {
	rid      id.Round
	identity reception.IdentityUse
	numAttempts uint
}

// Long running thread which process historical rounds
// Can be killed by sending a signal to the quit channel
// takes a comms interface to aid in testing
func (m *Manager) processHistoricalRounds(comm historicalRoundsComms, quitCh <-chan struct{}) {

	timerCh := make(<-chan time.Time)

	rng := m.Rng.GetStream()
	var roundRequests []historicalRoundRequest

	done := false
	for !done {
		shouldProcess := false
		// wait for a quit or new round to check
		select {
		case <-quitCh:
			rng.Close()
			// return all roundRequests in the queue to the input channel so they can
			// be checked in the future. If the queue is full, disable them as
			// processing so they are picked up from the beginning
			for _, r := range roundRequests {
				select {
				case m.historicalRounds <- r:
				default:
				}
			}
			done = true
		// if the timer elapses process roundRequests to ensure the delay isn't too long
		case <-timerCh:
			if len(roundRequests) > 0 {
				shouldProcess = true
			}
		// get new round to lookup and force a lookup if
		case r := <-m.historicalRounds:
			jww.DEBUG.Printf("Recieved and quing round %d for "+
				"historical rounds lookup", r.rid)
			roundRequests = append(roundRequests, r)
			if len(roundRequests) > int(m.params.MaxHistoricalRounds) {
				shouldProcess = true
			} else if len(roundRequests) != 0 {
				//if this is the first round, start the timeout
				timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
			}
		}
		if !shouldProcess {
			continue
		}

		//find a gateway to request about the roundRequests
		gwHost, err := gateway.Get(m.Instance.GetPartialNdf().Get(), comm, rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to track network, NDF has corrupt "+
				"data: %s", err)
		}

		rounds := make([]uint64, len(roundRequests))
		for i, rr := range roundRequests {
			rounds[i] = uint64(rr.rid)
		}

		//send the historical roundRequests request
		hr := &pb.HistoricalRounds{
			Rounds: rounds,
		}

		jww.DEBUG.Printf("Requesting Historical rounds %v from "+
			"gateway %s", rounds, gwHost.GetId())

		response, err := comm.RequestHistoricalRounds(gwHost, hr)
		if err != nil {
			jww.ERROR.Printf("Failed to request historical roundRequests "+
				"data for rounds %v: %s", rounds, response)
			// if the check fails to resolve, break the loop and so they will be
			// checked again
			timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
			continue
		}

		// process the returned historical roundRequests.
		for i, roundInfo := range response.Rounds {
			// The interface has missing returns returned as nil, such roundRequests
			// need be be removes as processing so the network follower will
			// pick them up in the future.
			if roundInfo == nil {
				roundRequests[i].numAttempts++
				if roundRequests[i].numAttempts==m.params.MaxHistoricalRoundsRetries{
					jww.ERROR.Printf("Failed to retreive historical " +
						"round %d on last attempt, will not try again",
						roundRequests[i].rid)
				}else{
					select {
					case m.historicalRounds <-roundRequests[i]:
						jww.WARN.Printf("Failed to retreive historical " +
							"round %d, will try up to %d more times",
							roundRequests[i].rid, m.params.MaxHistoricalRoundsRetries-roundRequests[i].numAttempts)
					default:
						jww.WARN.Printf("Failed to retreive historical " +
							"round %d, failed to try again, round will not be " +
							"retreived", roundRequests[i].rid)
					}
				}
				continue
			}
			// Successfully retrieved roundRequests are sent to the Message
			// Retrieval Workers
			rl := roundLookup{
				roundInfo: roundInfo,
				identity:  roundRequests[i].identity,
			}
			m.lookupRoundMessages <- rl
		}

		//clear the buffer now that all have been checked
		roundRequests = make([]historicalRoundRequest, 0)
	}
}
