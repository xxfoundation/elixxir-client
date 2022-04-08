///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmix

// tracker.go controls access to network resources. Interprocess communications
// and intra-client state are accessible through the context object.

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix/address"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/health"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/nodes"
	"gitlab.com/elixxir/client/cmix/pickup"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	commClient "gitlab.com/elixxir/comms/client"
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

// client implements the Client interface inside context. It controls access
// to network resources and implements all the communications functions used by
// the client.
// CRITICAL: Client must be private. It embeds submodules that export functions
// for it, but not for public consumption. By being private and returning as the
// public interface, these can be kept private.
type client struct {
	// User Identity Storage
	session storage.Session
	// Generic RNG for client
	rng *fastRNG.StreamGenerator
	// Comms pointer to send/receive messages
	comms clientCommsInterface
	// Contains the network instance
	instance *commNetwork.Instance

	// Parameters of the network
	param Params

	// Sub-managers
	gateway.Sender
	message.Handler
	nodes.Registrar
	rounds.Retriever
	pickup.Pickup
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

// NewManager builds a new reception client object using inputted key fields.
func NewManager(params Params, comms *commClient.Comms, session storage.Session,
	ndf *ndf.NetworkDefinition, rng *fastRNG.StreamGenerator,
	events event.Manager) (Client, error) {

	// Start network instance
	instance, err := commNetwork.NewInstance(
		comms.ProtoComms, ndf, nil, nil, commNetwork.None, params.FastPolling)
	if err != nil {
		return nil, errors.WithMessage(
			err, "failed to create network client")
	}

	tmpMsg := format.NewMessage(session.GetCmixGroup().GetP().ByteLen())

	tracker := uint64(0)
	earliest := uint64(0)

	// Create client object
	c := &client{
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
	c.UpdateAddressSpace(18)

	if params.VerboseRoundTracking {
		c.verboseRounds = NewRoundTracker()
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
	c.Sender, err = gateway.NewSender(
		poolParams, rng, ndf, comms, session, nodeChan)
	if err != nil {
		return nil, err
	}

	// Set up the node registrar
	c.Registrar, err = nodes.LoadRegistrar(
		session, c.Sender, c.comms, c.rng, nodeChan)
	if err != nil {
		return nil, err
	}

	// Set up the historical rounds handler
	c.Retriever = rounds.NewRetriever(
		params.Historical, comms, c.Sender, events)

	// Set up Message Handler
	c.Handler = message.NewHandler(params.Message, c.session.GetKV(), c.events,
		c.session.GetReceptionID())

	// Set up round handler
	c.Pickup = pickup.NewPickup(
		params.Pickup, c.Handler.GetMessageReceptionChannel(), c.Sender,
		c.Retriever, c.rng, c.instance, c.session)

	// Add the identity system
	c.Tracker = identity.NewOrLoadTracker(c.session, c.Space)

	// Set up the ability to register with new nodes when they appear
	c.instance.SetAddGatewayChan(nodeChan)

	// Set up the health monitor
	c.Monitor = health.Init(instance, params.NetworkHealthTimeout)

	// Set up critical message tracking (sendCmix only)
	critSender := func(msg format.Message, recipient *id.ID, params CMIXParams,
	) (id.Round, ephemeral.Id, error) {
		return sendCmixHelper(c.Sender, msg, recipient, params, c.instance,
			c.session.GetCmixGroup(), c.Registrar, c.rng, c.events,
			c.session.GetTransmissionID(), c.comms)
	}

	c.crit = newCritical(
		session.GetKV(), c.Monitor, c.instance.GetRoundEvents(), critSender)

	// Report health events
	c.Monitor.AddHealthCallback(func(isHealthy bool) {
		c.events.Report(5, "health", "IsHealthy", strconv.FormatBool(isHealthy))
	})

	return c, nil
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
func (c *client) Follow(report ClientErrorReport) (stoppable.Stoppable, error) {
	multi := stoppable.NewMulti("networkManager")

	// health tracker
	healthStop, err := c.Monitor.StartProcesses()
	if err != nil {
		return nil, errors.Errorf("failed to follow")
	}
	multi.Add(healthStop)

	// Node Updates
	multi.Add(c.Registrar.StartProcesses(c.param.ParallelNodeRegistrations)) // Adding/MixCypher
	// TODO: node remover

	// Start the Network tracker
	followNetworkStopper := stoppable.NewSingle("FollowNetwork")
	go c.followNetwork(report, followNetworkStopper)
	multi.Add(followNetworkStopper)

	// Message reception
	multi.Add(c.Handler.StartProcesses())

	// Round processing
	multi.Add(c.Pickup.StartProcessors())

	// Historical rounds processing
	multi.Add(c.Retriever.StartProcesses())

	// Start the processes for the identity handler
	multi.Add(c.Tracker.StartProcesses())

	return multi, nil
}

// GetInstance returns the network instance object (NDF state).
func (c *client) GetInstance() *commNetwork.Instance {
	return c.instance
}

// GetVerboseRounds returns verbose round information.
func (c *client) GetVerboseRounds() string {
	if c.verboseRounds == nil {
		return "Verbose Round tracking not enabled"
	}
	return c.verboseRounds.String()
}

func (c *client) SetFakeEarliestRound(rnd id.Round) {
	atomic.StoreUint64(c.earliestRound, uint64(rnd))
}

// GetMaxMessageLength returns the maximum length of a cMix message.
func (c *client) GetMaxMessageLength() int {
	return c.maxMsgLen
}
