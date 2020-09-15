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
	"encoding/binary"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/primitives/ndf"
	"io"
	"math"
	"time"
)

// ReadUint32 reads an integer from an io.Reader (which should be a CSPRNG)
func ReadUint32(rng io.Reader) uint32 {
	var rndBytes [4]byte
	i, err := rng.Read(rndBytes[:])
	if i != 4 || err != nil {
		panic(fmt.Sprintf("cannot read from rng: %+v", err))
	}
	return binary.BigEndian.Uint32(rndBytes[:])
}

// ReadRangeUint32 reduces an integer from 0, MaxUint32 to the range start, end
func ReadRangeUint32(start, end uint32, rng io.Reader) uint32 {
	size := end - start
	// note we could just do the part inside the () here, but then extra
	// can == size which means a little bit of range is wastes, either
	// choice seems negligible so we went with the "more correct"
	extra := (math.MaxUint32%size + 1) % size
	limit := math.MaxUint32 - extra
	// Loop until we read something inside the limit
	for {
		res := ReadUint32(rng)
		if res > limit {
			continue
		}
		return (res % size) + start
	}
}

// StartTrackNetwork starts a single TrackNetwork thread and returns a stoppable
func StartTrackNetwork(ctx *context.Context, net *Manager) stoppable.Stoppable {
	stopper := stoppable.NewSingle("TrackNetwork")
	go TrackNetwork(ctx, net, stopper.Quit())
	return stopper
}

// TrackNetwork polls the network to get updated on the state of nodes, the
// round status, and informs the client when messages can be retrieved.
func TrackNetwork(ctx *context.Context, network *Manager,
	quitCh <-chan struct{}) {
	ticker := time.NewTicker(ctx.GetTrackNetworkPeriod())
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case <-ticker:
			trackNetwork(ctx, network)
		}
	}
}

func trackNetwork(ctx *context.Context, network *Manager) {
	instance := ctx.Manager.GetInstance()
	comms := network.Comms
	ndf := instance.GetPartialNdf().Get()
	rng := ctx.Rng

	// Get a random gateway
	gateways := ndf.Gateways
	gwID := gateways[ReadRangeUint32(0, len(gateways), rng)].GetGatewayId()
	gwHost, ok := comms.GetHost(gwHost)
	if !ok {
		jww.ERROR.Printf("could not get host for gateway %s", gwID)
		return
	}

	// Poll for the new NDF
	pollReq := pb.GatewayPoll{
		NDFHash:       instance.GetPartialNdf().GetHash(),
		LastRound:     instance.GetLastRoundID(),
		LastMessageID: nil,
	}
	pollResp, err := comms.SendPoll(gwHost)
	if err != nil {
		jww.ERROR.Printf(err)
	}
	newNDF := pollResp.NDF
	lastRoundInfo := pollResp.RoundInfo
	roundUpdates := pollResp.Updates
	newMessageIDs := pollRespon.NewMessageIDs

	// ---- NODE EVENTS ----
	// NOTE: this updates the structure AND sends events over the node
	//       update channels
	instance.UpdatePartialNdf(newNDF)

	// ---- Round Processing -----

	// rounds, err = network.UpdateRounds(ctx, ndf)
	// if err != nil {
	// 	// ...
	// }

	// err = rounds.GetKnownRound().MaskedRange(gateway,
	// 	network.CheckRoundsFunction)
	// if err != nil {
	// 	// ...
	// }
}
