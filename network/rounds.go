////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// StartProcessHistoricalRounds starts a worker for processing round
// history.
func StartProcessHistoricalRounds(ctx *context.Context,
	network *Manager) stoppable.Stoppable {
	stopper := stoppable.NewSingle("ProcessHistoricalRounds")
	go ProcessHistoricalRounds(ctx, network, stopper.Quit())
	return stopper
}

// ProcessHistoricalRounds analyzes round history to see if this Client
// needs to check for messages at any of the gateways which completed
// those rounds.
func ProcessHistoricalRounds(ctx *context.Context, network *Manager,
	quitCh <-chan struct{}) {
	opts := params.GetDefaultNetwork()
	ticker := time.NewTicker(opts.TrackNetworkPeriod)
	var rounds []id.Round
	done := false
	for !done {
		shouldProcess := false
		select {
		case <-quitCh:
			done = true
		case <-ticker.C:
			if len(rounds) > 0 {
				shouldProcess = true
			}
		case rid := <-network.GetHistoricalLookupCh():
			rounds = append(rounds, rid)
			if len(rounds) > opts.MaxHistoricalRounds {
				shouldProcess = true
			}
		}
		if !shouldProcess {
			continue
		}

		roundInfos := processHistoricalRounds(ctx, rounds)
		for i := range rounds {
			if roundInfos[i] == nil {
				jww.ERROR.Printf("could not check "+
					"historical round %d", rounds[i])
				newRounds = append(newRounds, rounds[i])
				network.Processing.Done(rounds[i])
				continue
			}
			network.GetRoundUpdateCh() <- ri
		}
	}
}

func processHistoricalRounds(ctx *context.Context,
	rids []id.Round) []*pb.RoundInfo {
	// for loop over rids?
	// network := ctx.GetNetwork()
	// gw := network.GetGateway()
	// ris := gw.GetHistoricalRounds(ctx.GetRoundList())
	// return ris
	return nil
}
