///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

// tracker.go controls access to network resources. Interprocess communications
// and intraclient state are accessible through the context object.

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/network/address"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/health"
	"gitlab.com/elixxir/client/network/historical"
	"gitlab.com/elixxir/client/network/identity"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/network/nodes"
	"gitlab.com/elixxir/client/network/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	commNetwork "gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"math"
	"sync/atomic"
	"time"
)

// fakeIdentityRange indicates the range generated between
// 0 (most current) and fakeIdentityRange rounds behind the earliest known
// round that will be used as the earliest round when polling with a
// fake identity.
const fakeIdentityRange = 800

// manager implements the NetworkManager interface inside context. It
// controls access to network resources and implements all the communications
// functions used by the client.
// CRITICAL: Manager must be private. It embeds sub moduals which
// export functions for it, but not for public consumption. By being private
// and returning ass the public interface, these can be kept private.
type manager struct {
	//User Identity Storage
	session storage.Session
	//generic RNG for client
	rng *fastRNG.StreamGenerator
	// comms pointer to send/recv messages
	comms *client.Comms
	//contains the network instance
	instance *commNetwork.Instance

	// parameters of the network
	param Params

	//sub-managers
	gateway.Sender
	message.Handler
	nodes.Registrar
	historical.Retriever
	rounds.Pickup
	address.Space
	identity.Tracker
	health.Monitor

	// Earliest tracked round
	earliestRound *uint64

	//number of polls done in a period of time
	tracker       *uint64
	latencySum    uint64
	numLatencies  uint64
	verboseRounds *RoundTracker

	// Event reporting api
	events event.Manager

	//storage of the max message length
	maxMsgLen int
}

// NewManager builds a new reception manager object using inputted key fields
func NewManager(params Params, comms *client.Comms, session storage.Session,
	ndf *ndf.NetworkDefinition, rng *fastRNG.StreamGenerator, events event.Manager,
) (Manager, error) {

	//start network instance
	instance, err := commNetwork.NewInstance(comms.ProtoComms, ndf, nil, nil, commNetwork.None, params.FastPolling)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create"+
			" client network manager")
	}

	tmpMsg := format.NewMessage(session.GetCmixGroup().GetP().ByteLen())

	tracker := uint64(0)
	earliest := uint64(0)
	// create manager object
	m := &manager{
		param:         params,
		tracker:       &tracker,
		Space:         address.NewAddressSpace(),
		events:        events,
		earliestRound: &earliest,
		session:       session,
		rng:           rng,
		comms:         comms,
		instance:      instance,
		maxMsgLen:     tmpMsg.ContentsSize(),
	}
	m.UpdateAddressSpace(18)

	if params.VerboseRoundTracking {
		m.verboseRounds = NewRoundTracker()
	}

	/* set up modules */

	nodechan := make(chan commNetwork.NodeGateway, nodes.InputChanLen)

	// Set up gateway.Sender
	poolParams := gateway.DefaultPoolParams()
	// Client will not send KeepAlive packets
	poolParams.HostParams.KaClientOpts.Time = time.Duration(math.MaxInt64)
	// Enable optimized HostPool initialization
	poolParams.MaxPings = 50
	poolParams.ForceConnection = true
	m.Sender, err = gateway.NewSender(poolParams, rng,
		ndf, comms, session, nodechan)
	if err != nil {
		return nil, err
	}

	//setup the node registrar
	m.Registrar, err = nodes.LoadRegistrar(session, m.Sender, m.comms, m.rng, nodechan)
	if err != nil {
		return nil, err
	}

	//setup the historical rounds handler
	m.Retriever = historical.NewRetriever(params.Historical, comms, m.Sender, events)

	//Set up Message Handler
	m.Handler = message.NewHandler(params.Message, m.session.GetKV(), m.events)

	//set up round handler
	m.Pickup = rounds.NewPickup(params.Rounds, m.Handler.GetMessageReceptionChannel(),
		m.Sender, m.Retriever, m.rng, m.instance, m.session)

	//add the identity system
	m.Tracker = identity.NewOrLoadTracker(m.session, m.Space)

	// Set upthe ability to register with new nodes when they appear
	m.instance.SetAddGatewayChan(nodechan)

	m.Monitor = health.Init(instance, params.NetworkHealthTimeout)

	// Report health events
	m.Monitor.AddHealthCallback(func(isHealthy bool) {
		m.events.Report(5, "health", "IsHealthy",
			fmt.Sprintf("%v", isHealthy))
	})

	return m, nil
}

// Follow StartRunners kicks off all network reception goroutines ("threads").
// Started Threads are:
//   - Network Follower (/network/follow.go)
//   - Historical Round Retrieval (/network/rounds/historical.go)
//	 - Message Retrieval Worker Group (/network/rounds/retrieve.go)
//	 - Message Handling Worker Group (/network/message/handle.go)
//	 - health tracker (/network/health)
//	 - Garbled Messages (/network/message/inProgress.go)
//	 - Critical Messages (/network/message/critical.go)
//   - Ephemeral ID tracking (network/address/tracker.go)
func (m *manager) Follow(report ClientErrorReport) (stoppable.Stoppable, error) {
	multi := stoppable.NewMulti("networkManager")

	// health tracker
	healthStop, err := m.Monitor.StartProcessies()
	if err != nil {
		return nil, errors.Errorf("failed to follow")
	}
	multi.Add(healthStop)

	// Node Updates
	multi.Add(m.Registrar.StartProcesses(m.param.ParallelNodeRegistrations)) // Adding/MixCypher
	//TODO-node remover

	// Start the Network tracker
	followNetworkStopper := stoppable.NewSingle("FollowNetwork")
	go m.followNetwork(report, followNetworkStopper)
	multi.Add(followNetworkStopper)

	// Message reception
	multi.Add(m.Handler.StartProcesses())

	// Round processing
	multi.Add(m.Pickup.StartProcessors())

	// Historical rounds processing
	multi.Add(m.Retriever.StartProcessies())

	//start the processies for the identity handler
	multi.Add(m.Tracker.StartProcessies())

	return multi, nil
}

// GetInstance returns the network instance object (ndf state)
func (m *manager) GetInstance() *commNetwork.Instance {
	return m.instance
}

// GetVerboseRounds returns verbose round information
func (m *manager) GetVerboseRounds() string {
	if m.verboseRounds == nil {
		return "Verbose Round tracking not enabled"
	}
	return m.verboseRounds.String()
}

func (m *manager) SetFakeEarliestRound(rnd id.Round) {
	atomic.StoreUint64(m.earliestRound, uint64(rnd))
}

// GetMaxMessageLength returns the maximum length of a cmix message
func (m *manager) GetMaxMessageLength() int {
	return m.maxMsgLen
}
