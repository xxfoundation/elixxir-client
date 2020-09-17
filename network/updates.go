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
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	//"gitlab.com/elixxir/comms/network"
	//"gitlab.com/xx_network/primitives/ndf"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
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
	opts := params.GetDefaultNetwork()
	ticker := time.NewTicker(opts.TrackNetworkPeriod)
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case <-ticker.C:
			trackNetwork(ctx, network)
		}
	}
}

func trackNetwork(ctx *context.Context, network *Manager) {
	instance := ctx.Manager.GetInstance()
	comms := network.Comms
	ndf := instance.GetPartialNdf().Get()
	rng := ctx.Rng.GetStream()
	defer rng.Close()
	sess := ctx.Session

	// Get a random gateway
	gateways := ndf.Gateways
	gwIdx := ReadRangeUint32(0, uint32(len(gateways)), rng)
	gwID, err := gateways[gwIdx].GetGatewayId()
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}
	gwHost, ok := comms.GetHost(gwID)
	if !ok {
		jww.ERROR.Printf("could not get host for gateway %s", gwID)
		return
	}

	// Poll for the new NDF
	pollReq := pb.GatewayPoll{
		Partial: &pb.NDFHash{
			Hash: instance.GetPartialNdf().GetHash(),
		},
		LastUpdate:    uint64(instance.GetLastRoundID()),
		LastMessageID: "",
	}
	pollResp, err := comms.SendPoll(gwHost, &pollReq)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}
	newNDF := pollResp.PartialNDF
	lastRoundInfo := pollResp.LastRound
	roundUpdates := pollResp.Updates
	// This is likely unused in favor of new API
	//newMessageIDs := pollResp.NewMessageIDs

	// ---- NODE EVENTS ----
	// NOTE: this updates the structure AND sends events over the node
	//       update channels
	instance.UpdatePartialNdf(newNDF)

	// ---- Round Processing -----
	checkedRounds := sess.GetCheckedRounds()
	roundChecker := getRoundChecker(ctx, network, roundUpdates)
	checkedRounds.RangeUnchecked(id.Round(lastRoundInfo.ID), roundChecker)

	// FIXME: Seems odd/like a race condition to do this here, but this is
	// spec. Fix this to either eliminate race condition or not make it
	// weird. This is tied to if a round is processing. It appears that
	// if it is processing OR already checked is the state we care about,
	// because we really want to know if we should look it up and process,
	// and that could be done via storage inside range Unchecked?
	for _, ri := range roundUpdates {
		checkedRounds.Check(id.Round(ri.ID))
	}
}

// getRoundChecker passes a context and the round infos received by the
// gateway to the funky round checker api to update round state.
// The returned function passes round event objects over the context
// to the rest of the message handlers for getting messages.
func getRoundChecker(ctx *context.Context, network *Manager,
	roundInfos []*pb.RoundInfo) func(roundID id.Round) bool {
	return func(roundID id.Round) bool {
		//sess := ctx.Session
		processing := network.Processing

		// Set round to processing, if we can
		// FIXME: this appears to be a race condition -- either fix
		// or make it not look like one.
		if processing.IsProcessing(roundID) {
			return false
		}
		processing.Add(roundID)
		// FIXME: Spec has us SETTING processing, but not REMOVING it
		// until the get messages thread completes the lookup, this
		// is smell that needs refining. It seems as if there should be
		// a state that lives with the round info as soon as we know
		// about it that gets updated at different parts...not clear
		// needs to be thought through.
		//defer processing.Remove(roundID)

		// TODO: Bloom filter lookup -- return true when we don't have
		// Go get the round from the round infos, if it exists

		// For now, if we have the round in th round updates,
		// process it, otherwise call historical rounds code
		// to go find it if possible.
		for _, ri := range roundInfos {
			rID := id.Round(ri.ID)
			if rID == roundID {
				// Send to get message processor
				network.GetRoundUpdateCh() <- ri
				return false
			}
		}

		// If we didn't find it, send to historical rounds processor
		network.GetHistoricalLookupCh() <- roundID

		return false
	}
}
