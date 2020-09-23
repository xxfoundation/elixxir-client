////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type historicalRoundsComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	RequestHistoricalRounds(host *connect.Host,
		message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error)
}

// ProcessHistoricalRounds analyzes round history to see if this Client
// needs to check for messages at any of the gateways which completed
// those rounds.
func (m *Manager) processHistoricalRounds(comm historicalRoundsComms, quitCh <-chan struct{}) {
	ticker := time.NewTicker(m.params.HistoricalRoundsPeriod)

	rng := m.Rng.GetStream()
	var rounds []uint64

	done := false
	for !done {
		shouldProcess := false
		select {
		case <-quitCh:
			rng.Close()
			done = true
		case <-ticker.C:
			if len(rounds) > 0 {
				shouldProcess = true
			}
		case rid := <-m.historicalRounds:
			rounds = append(rounds, uint64(rid))
			if len(rounds) > int(m.params.MaxHistoricalRounds) {
				shouldProcess = true
			}
		}
		if !shouldProcess {
			continue
		}

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
			// if the check fails to resolve, break the loop so they will be
			// checked again
			break
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
