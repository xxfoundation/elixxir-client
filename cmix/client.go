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
	"math"
	"strconv"
	"sync/atomic"
	"time"

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
	comms *commClient.Comms
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
	events event.Reporter

	// Storage of the max message length
	maxMsgLen int
}

// NewClient builds a new reception client object using inputted key fields.
func NewClient(params Params, comms *commClient.Comms, session storage.Session,
	rng *fastRNG.StreamGenerator, events event.Reporter) (Client, error) {

	tmpMsg := format.NewMessage(session.GetCmixGroup().GetP().ByteLen())

	tracker := uint64(0)
	earliest := uint64(0)

	// Create client object
	c := &client{
		param:         params,
		tracker:       &tracker,
		events:        events,
		earliestRound: &earliest,
		session:       session,
		rng:           rng,
		comms:         comms,
		maxMsgLen:     tmpMsg.ContentsSize(),
	}

	if params.VerboseRoundTracking {
		c.verboseRounds = NewRoundTracker()
	}

	// Set up Message Handler
	c.Handler = message.NewHandler(c.param.Message, c.session.GetKV(),
		c.events, c.session.GetReceptionID())

	err := c.initialize(session.GetNDF())
	return c, err
}

// initialize turns on network handlers, initializing a host pool and
// network health monitors. This should be called before
// network Follow command is called.
func (c *client) initialize(ndf *ndf.NetworkDefinition) error {
	// Start network instance
	instance, err := commNetwork.NewInstance(
		c.comms.ProtoComms, ndf, nil, nil, commNetwork.None,
		c.param.FastPolling)
	if err != nil {
		return errors.WithMessage(
			err, "failed to create network client")
	}
	c.instance = instance

	addrSize := ndf.AddressSpace[len(ndf.AddressSpace)-1].Size
	c.Space = address.NewAddressSpace(addrSize)

	/* Set up modules */
	nodeChan := make(chan commNetwork.NodeGateway, nodes.InputChanLen)

	// Set up gateway.Sender
	poolParams := gateway.DefaultPoolParams()

	// Disable KeepAlive packets
	poolParams.HostParams.KaClientOpts.Time = time.Duration(math.MaxInt64)

	// Configure the proxy error exponential moving average tracker
	poolParams.HostParams.ProxyErrorMetricParams.Cutoff = 0.30
	poolParams.HostParams.ProxyErrorMetricParams.InitialAverage =
		0.75 * poolParams.HostParams.ProxyErrorMetricParams.Cutoff

	// Enable optimized HostPool initialization
	poolParams.MaxPings = 50
	poolParams.ForceConnection = true
	sender, err := gateway.NewSender(poolParams, c.rng, ndf, c.comms,
		c.session, nodeChan)
	if err != nil {
		return err
	}
	c.Sender = sender

	// Set up the node registrar
	c.Registrar, err = nodes.LoadRegistrar(
		c.session, c.Sender, c.comms, c.rng, nodeChan)
	if err != nil {
		return err
	}

	// Set up the historical rounds handler
	c.Retriever = rounds.NewRetriever(
		c.param.Historical, c.comms, c.Sender, c.events)

	// Set up round handler
	c.Pickup = pickup.NewPickup(
		c.param.Pickup, c.Handler.GetMessageReceptionChannel(), c.Sender,
		c.Retriever, c.comms, c.rng, c.instance, c.session)

	// Add the identity system
	c.Tracker = identity.NewOrLoadTracker(c.session, c.Space)

	// Set up the ability to register with new nodes when they appear
	c.instance.SetAddGatewayChan(nodeChan)
	// Set up the health monitor
	c.Monitor = health.Init(c.instance, c.param.NetworkHealthTimeout)

	// Set up critical message tracking (sendCmix only)
	critSender := func(msg format.Message, recipient *id.ID, params CMIXParams,
	) (id.Round, ephemeral.Id, error) {
		compiler := func(round id.Round) (format.Message, error) {
			return msg, nil
		}
		rid, eid, _, sendErr := sendCmixHelper(c.Sender, compiler, recipient, params, c.instance,
			c.session.GetCmixGroup(), c.Registrar, c.rng, c.events,
			c.session.GetTransmissionID(), c.comms)
		return rid, eid, sendErr

	}

	c.crit = newCritical(c.session.GetKV(), c.Monitor,
		c.instance.GetRoundEvents(), critSender)

	// Report health events
	c.AddHealthCallback(func(isHealthy bool) {
		c.events.Report(5, "health", "IsHealthy", strconv.FormatBool(isHealthy))
	})

	return nil
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
