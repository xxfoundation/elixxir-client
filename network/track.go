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
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/primitives/knownRounds"
	"gitlab.com/xx_network/comms/connect"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type trackNetworkComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	SendPoll(host *connect.Host, message *pb.GatewayPoll) (*pb.GatewayPollResponse, error)
}

// TrackNetwork polls the network to get updated on the state of nodes, the
// round status, and informs the client when messages can be retrieved.
func (m *Manager) trackNetwork(quitCh <-chan struct{}) {
	ticker := time.NewTicker(m.param.TrackNetworkPeriod)
	rng := m.context.Rng.GetStream()

	for {
		select {
		case <-quitCh:
			rng.Close()
			break
		case <-ticker.C:
			m.track(rng, m.comms)
		}
	}
}

func (m *Manager) track(rng csprng.Source, comms trackNetworkComms) {

	gwHost, err := gateway.Get(m.instance.GetPartialNdf().Get(), comms, rng)
	if err != nil {
		jww.FATAL.Panicf("Failed to track network, NDF has corrupt "+
			"data: %s", err)
	}

	// Poll for the new NDF
	pollReq := pb.GatewayPoll{
		Partial: &pb.NDFHash{
			Hash: m.instance.GetPartialNdf().GetHash(),
		},
		LastUpdate: uint64(m.instance.GetLastUpdateID()),
	}
	pollResp, err := comms.SendPoll(gwHost, &pollReq)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}

	//handle updates
	newNDF := pollResp.PartialNDF
	lastTrackedRound := id.Round(pollResp.LastTrackedRound)
	roundUpdates := pollResp.Updates
	gwRoundsState := &knownRounds.KnownRounds{}
	err = gwRoundsState.Unmarshal(pollResp.KnownRounds)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}

	// ---- NODE EVENTS ----
	// NOTE: this updates the structure AND sends events over the node
	//       update channels
	err = m.instance.UpdatePartialNdf(newNDF)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}
	err = m.instance.RoundUpdates(roundUpdates)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}

	// ---- Round Processing -----
	//build the round checker
	roundChecker := func(rid id.Round) bool {
		return m.round.Checker(rid, m.instance)
	}

	//check rounds
	checkedRounds := m.context.Session.GetCheckedRounds()
	checkedRounds.Forward(lastTrackedRound)
	checkedRounds.RangeUncheckedMasked(gwRoundsState, roundChecker,
		int(m.param.MaxCheckCheckedRounds))
}

