////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

// updates.go tracks the network for:
//   1. Node addition and removal
//   2. New/Active/Complete rounds and their contact gateways
// This information is tracked by polling a gateway for the network definition
// file (NDF). Once it detects an event it sends it off to the proper channel
// for a worker to update the client state (add/remove a node, check for
// messages at a gateway, etc). See:
//   - nodes.go for add/remove node events
//   - rounds.go for round event handling & processing
//   - receive.go for message handling

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
)

// GetUpdates polls the network for updates.
func (m *Manager) GetUpdates() (*network.Instance, error) {
	return nil, nil
}

// StartTrackNetwork starts a single TrackNetwork thread and returns a stoppable
func StartTrackNetwork(ctx *context.Context) stoppable.Stoppable {
	stopper := stoppable.NewSingle("TrackNetwork")
	go TrackNetwork(ctx, stopper.Quit())
	return stopper
}

// TrackNetwork polls the network to get updated on the state of nodes, the
// round status, and informs the client when messages can be retrieved.
func TrackNetwork(ctx *context.Context, quitCh <-chan struct{}) {
	ticker := timer.NewTicker(ctx.GetTrackNetworkPeriod())
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case <-ticker:
			trackNetwork(ctx)
		}
	}
}

func trackNetwork(ctx *context.Context) {
	gateway, err := ctx.Session.GetNodeKeys().GetGatewayForSending()
	if err != nil {
		//...
	}

	network := ctx.GetNetwork()
	ndf, err := network.PollNDF(ctx, gateway)
	if err != nil {
		// ....
	}

	newNodes, removedNodes := network.UpdateNDF(ndf)
	for _, n := range newNodes {
		network.addNodeCh <- n
	}
	for _, n := range removedNodes {
		network.removeNodeCh <- n
	}

	rounds, err = network.UpdateRounds(ctx, ndf)
	if err != nil {
		// ...
	}

	err = rounds.GetKnownRound().MaskedRange(gateway,
		network.CheckRoundsFunction)
	if err != nil {
		// ...
	}
}
