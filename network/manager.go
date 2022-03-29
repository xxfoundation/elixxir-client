///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

// tracker.go controls access to network resources. Interprocess communications
// and intra-client state are accessible through the context object.

import (
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
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"math"
	"strconv"
	"sync/atomic"
	"time"
)

// fakeIdentityRange indicates the range generated between 0 (most current) and
// fakeIdentityRange rounds behind the earliest known round that will be used as
// the earliest round when polling with a fake identity.
const fakeIdentityRange = 800

// manager implements the Manager interface inside context. It controls access
// to network resources and implements all the communications functions used by
// the client.
// CRITICAL: Manager must be private. It embeds submodules that export functions
// for it, but not for public consumption. By being private and returning as the
// public interface, these can be kept private.
type manager struct {
	// User Identity Storage
	session storage.Session
	// Generic RNG for client
	rng *fastRNG.StreamGenerator
	// Comms pointer to send/receive messages
	comms *client.Comms
	// Contains the network instance
	instance *commNetwork.Instance

	// Parameters of the network
	param Params

	// Sub-managers
	gateway.Sender
	message.Handler
	nodes.Registrar
	historical.Retriever
	rounds.Pickup
	address.Space
	identity.Tracker
	health.Monitor
	crit *critical

	// Earliest tracked round
	earliestRound *uint64

	// Number of polls done in a period of time
	tracker       *uint64
	latencySum    uint64
	numLatencies  uint64
	verboseRounds *RoundTracker

	// Event reporting API
	events event.Manager

	// Storage of the max message length
	maxMsgLen int
}

// NewManager builds a new reception manager object using inputted key fields.
func NewManager(params Params, comms *client.Comms, session storage.Session,
	ndf *ndf.NetworkDefinition, rng *fastRNG.StreamGenerator,
	events event.Manager) (Manager, error) {

	// Start network instance
	instance, err := commNetwork.NewInstance(
		comms.ProtoComms, ndf, nil, nil, commNetwork.None, params.FastPolling)
	if err != nil {
		return nil, errors.WithMessage(
			err, "failed to create client network manager")
	}

	tmpMsg := format.NewMessage(session.GetCmixGroup().GetP().ByteLen())

	tracker := uint64(0)
	earliest := uint64(0)

	// Create manager object
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

	/* Set up modules */
	nodeChan := make(chan commNetwork.NodeGateway, nodes.InputChanLen)

	// Set up gateway.Sender
	poolParams := gateway.DefaultPoolParams()

	// Client will not send KeepAlive packets
	poolParams.HostParams.KaClientOpts.Time = time.Duration(math.MaxInt64)

	// Enable optimized HostPool initialization
	poolParams.MaxPings = 50
	poolParams.ForceConnection = true
	m.Sender, err = gateway.NewSender(
		poolParams, rng, ndf, comms, session, nodeChan)
	if err != nil {
		return nil, err
	}

	// Set up the node registrar
	m.Registrar, err = nodes.LoadRegistrar(
		session, m.Sender, m.comms, m.rng, nodeChan)
	if err != nil {
		return nil, err
	}

	// Set up the historical rounds handler
	m.Retriever = historical.NewRetriever(
		params.Historical, comms, m.Sender, events)

	// Set up Message Handler
	m.Handler = message.NewHandler(params.Message, m.session.GetKV(), m.events,
		m.session.GetReceptionID())

	// Set up round handler
	m.Pickup = rounds.NewPickup(params.Rounds, m.Handler.GetMessageReceptionChannel(),
		m.Sender, m.Retriever, m.rng, m.instance, m.session)

	// Add the identity system
	m.Tracker = identity.NewOrLoadTracker(m.session, m.Space)

	// Set up the ability to register with new nodes when they appear
	m.instance.SetAddGatewayChan(nodeChan)

	// Set up the health monitor
	m.Monitor = health.Init(instance, params.NetworkHealthTimeout)

	// Set up critical message tracking (sendCmix only)
	critSender := func(msg format.Message, recipient *id.ID, params CMIXParams,
	) (id.Round, ephemeral.Id, error) {
		return sendCmixHelper(m.Sender, msg, recipient, params, m.instance,
			m.session.GetCmixGroup(), m.Registrar, m.rng, m.events,
			m.session.GetTransmissionID(), m.comms)
	}

	m.crit = newCritical(
		session.GetKV(), m.Monitor, m.instance.GetRoundEvents(), critSender)

	// Report health events
	m.Monitor.AddHealthCallback(func(isHealthy bool) {
		m.events.Report(5, "health", "IsHealthy", strconv.FormatBool(isHealthy))
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
	// TODO-node remover

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

	// Start the processes for the identity handler
	multi.Add(m.Tracker.StartProcessies())

	return multi, nil
}

// GetInstance returns the network instance object (NDF state).
func (m *manager) GetInstance() *commNetwork.Instance {
	return m.instance
}

// GetVerboseRounds returns verbose round information.
func (m *manager) GetVerboseRounds() string {
	if m.verboseRounds == nil {
		return "Verbose Round tracking not enabled"
	}
	return m.verboseRounds.String()
}

func (m *manager) SetFakeEarliestRound(rnd id.Round) {
	atomic.StoreUint64(m.earliestRound, uint64(rnd))
}

// GetMaxMessageLength returns the maximum length of a cMix message.
func (m *manager) GetMaxMessageLength() int {
	return m.maxMsgLen
}
