////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"gitlab.com/elixxir/client/network/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
	jww "github.com/spf13/jwalterweatherman"
)

//interface to increase east of testing of historical rounds
type historicalRoundsComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	RequestHistoricalRounds(host *connect.Host,
		message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error)
}

// ProcessHistoricalRounds analyzes round history to see if this Client
// needs to check for messages at any of the gateways which completed
// those rounds.
// Waits to request many rounds at a time or for a timeout to trigger
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
		for i, roundInfo := range response.Rounds {
			if roundInfo == nil {
				jww.ERROR.Printf("could not retreive "+
					"historical round %d", rounds[i])
				continue
			}
			m.p.Done(id.Round(rounds[i]))
			m.lookupRoundMessages <- roundInfo
		}
	}
}