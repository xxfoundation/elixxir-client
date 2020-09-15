////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	//	"time"
)

// StartProcessHistoricalRounds starts a worker for processing round
// history.
func StartProcessHistoricalRounds(ctx *context.Context) stoppable.Stoppable {
	stopper := stoppable.NewSingle("ProcessHistoricalRounds")
	go ProcessHistoricalRounds(ctx, stopper.Quit())
	return stopper
}

// ProcessHistoricalRounds analyzes round history to see if this Client
// needs to check for messages at any of the gateways which completed
// those rounds.
func ProcessHistoricalRounds(ctx *context.Context, quitCh <-chan struct{}) {
	// ticker := time.NewTicker(ctx.GetTrackNetworkPeriod())
	// var rounds []RoundID
	done := false
	for !done {
		//shouldProcess := false
		select {
		case <-quitCh:
			done = true
			// case <-ticker:
			// 	if len(rounds) > 0 {
			// 		shouldProcess = true
			// 	}
			// case rid := <-ctx.GetHistoricalRoundsCh():
			// 	rounds = append(rounds, rid)
			// 	if len(rounds) > ctx.GetSendSize() {
			// 		shouldProcess = true
			// 	}
			// }
			// if !shouldProcess {
			// 	continue
			// }

			// var roundInfos []*RoundInfo
			// roundInfos = processHistoricalRounds(ctx, rounds)
			// rounds := make([]RoundID)
			// for _, ri := range roundInfos {
			// 	ctx.GetMessagesCh() <- ri
			// }
		}
	}
}

func processHistoricalRounds(ctx *context.Context,
	rids []uint64) []*pb.RoundInfo {
	// for loop over rids?
	// network := ctx.GetNetwork()
	// gw := network.GetGateway()
	// ris := gw.GetHistoricalRounds(ctx.GetRoundList())
	// return ris
	return nil
}
