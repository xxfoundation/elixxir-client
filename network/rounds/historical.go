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

// Long running thread which process historical rounds
// Can be killed by sending a signal to the quit channel
// takes a comms interface to aid in testing
func (m *Manager) processHistoricalRounds(comm historicalRoundsComms, quitCh <-chan struct{}) {

	timerCh := make(<-chan time.Time)

	rng := m.Rng.GetStream()
	var rounds []uint64

	done := false
	for !done {
		shouldProcess := false
		// wait for a quit or new round to check
		select {
		case <-quitCh:
			rng.Close()
			// return all rounds in the queue to the input channel so they can
			// be checked in the future. If the queue is full, disable them as
			// processing so they are picked up from the beginning
			for _, rid := range rounds {
				select {
				case m.historicalRounds <- id.Round(rid):
				default:
					m.p.NotProcessing(id.Round(rid))
				}
			}
			done = true
		// if the timer elapses process rounds to ensure the delay isn't too long
		case <-timerCh:
			if len(rounds) > 0 {
				shouldProcess = true
			}
		// get new round to lookup and force a lookup if
		case rid := <-m.historicalRounds:
			rounds = append(rounds, uint64(rid))
			if len(rounds) > int(m.params.MaxHistoricalRounds) {
				shouldProcess = true
			} else if len(rounds) == 1 {
				//if this is the first round, start the timeout
				timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
			}
		}
		if !shouldProcess {
			continue
		}

		//find a gateway to request about the rounds
		gwHost, err := gateway.Get(m.Instance.GetPartialNdf().Get(), comm, rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to track network, NDF has corrupt "+
				"data: %s", err)
		}

		//send the historical rounds request
		hr := &pb.HistoricalRounds{
			Rounds: rounds,
		}

		response, err := comm.RequestHistoricalRounds(gwHost, hr)
		if err != nil {
			jww.ERROR.Printf("Failed to request historical rounds "+
				"data: %s", response)
			// if the check fails to resolve, break the loop and so they will be
			// checked again
			timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
			continue
		}

		// process the returned historical rounds.
		for i, roundInfo := range response.Rounds {
			// The interface has missing returns returned as nil, such rounds
			// need be be removes as processing so the network follower will
			// pick them up in the future.
			if roundInfo == nil {
				jww.ERROR.Printf("could not retreive "+
					"historical round %d", rounds[i])
				m.p.Fail(id.Round(rounds[i]))
				continue
			}
			// Successfully retrieved rounds are sent to the Message
			// Retrieval Workers
			m.lookupRoundMessages <- roundInfo
		}
	}
}
