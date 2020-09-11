////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/network/keyExchange"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"time"
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
	ticker := timer.NewTicker(ctx.GetTrackNetworkPeriod())
	var rounds []RoundID
	done := false
	for !done {
		shouldProcess := false
		select {
		case <-quitCh:
			done = true
		case <-ticker:
			if len(rounds) > 0 {
				shouldProcess = true
			}
		case rid := <-ctx.GetHistoricalRoundsCh():
			rounds = append(rounds, rid)
			if len(rounds) > ctx.GetSendSize() {
				shouldProcess = true
			}
		}
		if !shouldProcess {
			continue
		}

		var roundInfos []*RoundInfo
		roundInfos := processHistoricalRounds(ctx, rounds)
		rounds := make([]RoundID)
		for _, ri := range roundInfos {
			ctx.GetMessagesCh() <- ri
		}
	}
}

func processHistoricalRounds(ctx *context.Context, rids []RoundID) []*RoundInfo {
	// for loop over rids?
	network := ctx.GetNetwork()
	gw := network.GetGateway()
	ris := gw.GetHistoricalRounds(ctx.GetRoundList())
	return ris
}
