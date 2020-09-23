////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

// follow.go tracks the network for:
//   1. The status of the network and its accessibility
//   2. New/Active/Complete rounds and their contact gateways
//   3. Node addition and removal
// This information is tracked by polling a gateway for the network definition
// file (NDF). Once it detects an event it sends it off to the proper channel
// for a worker to update the client state (add/remove a node, check for
// messages at a gateway, etc). See:
//   - /node/register.go for add/remove node events
//   - /rounds/historical.go for old round retrieval
//   - /rounds/retrieve.go for message retrieval
//   - /message/reception.go decryption, partitioning, and signaling of messages
//   - /health/tracker.go - tracks the state of the network through the network
//		instance

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

type followNetworkComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	SendPoll(host *connect.Host, message *pb.GatewayPoll) (*pb.GatewayPollResponse, error)
}

// followNetwork polls the network to get updated on the state of nodes, the
// round status, and informs the client when messages can be retrieved.
func (m *manager) followNetwork(quitCh <-chan struct{}) {
	ticker := time.NewTicker(m.param.TrackNetworkPeriod)
	rng := m.Rng.GetStream()

	for {
		select {
		case <-quitCh:
			rng.Close()
			break
		case <-ticker.C:
			m.follow(rng, m.Comms)
		}
	}
}

func (m *manager) follow(rng csprng.Source, comms followNetworkComms) {

	//randomly select a gateway to poll
	//TODO: make this more intelligent
	gwHost, err := gateway.Get(m.Instance.GetPartialNdf().Get(), comms, rng)
	if err != nil {
		jww.FATAL.Panicf("Failed to follow network, NDF has corrupt "+
			"data: %s", err)
	}

	// Poll network updates
	pollReq := pb.GatewayPoll{
		Partial: &pb.NDFHash{
			Hash: m.Instance.GetPartialNdf().GetHash(),
		},
		LastUpdate: uint64(m.Instance.GetLastUpdateID()),
	}
	pollResp, err := comms.SendPoll(gwHost, &pollReq)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}

	// ---- Process Update Data ----
	lastTrackedRound := id.Round(pollResp.LastTrackedRound)
	gwRoundsState := &knownRounds.KnownRounds{}
	err = gwRoundsState.Unmarshal(pollResp.KnownRounds)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return
	}

	// ---- Node Events ----
	// NOTE: this updates the structure, AND sends events over the node
	//       update channels about new and removed nodes
	if pollResp.PartialNDF != nil {
		err = m.Instance.UpdatePartialNdf(pollResp.PartialNDF)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			return
		}
	}

	// NOTE: this updates rounds and updates the tracking of the health of the
	// network
	if pollResp.Updates != nil {
		err = m.Instance.RoundUpdates(pollResp.Updates)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			return
		}
	}

	// ---- Round Processing -----
	//build the round checker
	roundChecker := func(rid id.Round) bool {
		return m.round.Checker(rid)
	}

	//check rounds
	checkedRounds := m.Session.GetCheckedRounds()
	checkedRounds.Forward(lastTrackedRound)
	checkedRounds.RangeUncheckedMasked(gwRoundsState, roundChecker,
		int(m.param.MaxCheckedRounds))
}
