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
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/ephemeral"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/health"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/network/nodes"
	"gitlab.com/elixxir/client/network/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	commNetwork "gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
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

// Manager implements the NetworkManager interface inside context. It
// controls access to network resources and implements all the communications
// functions used by the client.
type manager struct {
	// parameters of the network
	param params.Network
	// handles message sending
	sender *gateway.Sender

	//Shared data with all sub managers
	internal.Internal

	//sub-managers
	round   *rounds.Manager
	message *message.Manager

	// Earliest tracked round
	earliestRound *uint64

	//number of polls done in a period of time
	tracker       *uint64
	latencySum    uint64
	numLatencies  uint64
	verboseRounds *RoundTracker

	// Address space size
	addrSpace *ephemeral.AddressSpace

	// Event reporting api
	events interfaces.EventManager
}

// NewManager builds a new reception manager object using inputted key fields
func NewManager(session *storage.Session, switchboard *switchboard.Switchboard,
	rng *fastRNG.StreamGenerator, events interfaces.EventManager,
	comms *client.Comms, params params.Network,
	ndf *ndf.NetworkDefinition) (interfaces.NetworkManager, error) {

	//start network instance
	instance, err := commNetwork.NewInstance(comms.ProtoComms, ndf, nil, nil, commNetwork.None, params.FastPolling)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create"+
			" client network manager")
	}

	// Note: These are not loaded/stored in E2E Store, but the
	// E2E Session Params are a part of the network parameters, so we
	// set them here when they are needed on startup
	session.E2e().SetE2ESessionParams(params.E2EParams)

	tracker := uint64(0)
	earliest := uint64(0)
	// create manager object
	m := manager{
		param:         params,
		tracker:       &tracker,
		addrSpace:     ephemeral.NewAddressSpace(),
		events:        events,
		earliestRound: &earliest,
	}
	m.addrSpace.Update(18)

	if params.VerboseRoundTracking {
		m.verboseRounds = NewRoundTracker()
	}

	m.Internal = internal.Internal{
		Session:          session,
		Switchboard:      switchboard,
		Rng:              rng,
		Comms:            comms,
		Health:           health.Init(instance, params.NetworkHealthTimeout),
		NodeRegistration: make(chan commNetwork.NodeGateway, params.RegNodesBufferLen),
		Instance:         instance,
		TransmissionID:   session.User().GetCryptographicIdentity().GetTransmissionID(),
		ReceptionID:      session.User().GetCryptographicIdentity().GetReceptionID(),
		Events:           events,
	}

	// Set up nodes registration chan for network instance
	m.Instance.SetAddGatewayChan(m.NodeRegistration)

	// Set up gateway.Sender
	poolParams := gateway.DefaultPoolParams()
	// Client will not send KeepAlive packets
	poolParams.HostParams.KaClientOpts.Time = time.Duration(math.MaxInt64)
	// Enable optimized HostPool initialization
	poolParams.MaxPings = 50
	poolParams.ForceConnection = true
	m.sender, err = gateway.NewSender(poolParams, rng,
		ndf, comms, session, m.NodeRegistration)
	if err != nil {
		return nil, err
	}

	// Report health events
	m.Internal.Health.AddFunc(func(isHealthy bool) {
		m.Internal.Events.Report(5, "Health", "IsHealthy",
			fmt.Sprintf("%v", isHealthy))
	})

	//create sub managers
	m.message = message.NewManager(m.Internal, m.param, m.NodeRegistration, m.sender)
	m.round = rounds.NewManager(m.Internal, m.param.Rounds, m.message.GetMessageReceptionChannel(), m.sender)

	return &m, nil
}

// Follow StartRunners kicks off all network reception goroutines ("threads").
// Started Threads are:
//   - Network Follower (/network/follow.go)
//   - Historical Round Retrieval (/network/rounds/historical.go)
//	 - Message Retrieval Worker Group (/network/rounds/retrieve.go)
//	 - Message Handling Worker Group (/network/message/handle.go)
//	 - Health Tracker (/network/health)
//	 - Garbled Messages (/network/message/garbled.go)
//	 - Critical Messages (/network/message/critical.go)
//   - Ephemeral ID tracking (network/ephemeral/tracker.go)
func (m *manager) Follow(report interfaces.ClientErrorReport) (stoppable.Stoppable, error) {
	multi := stoppable.NewMulti("networkManager")

	// health tracker
	healthStop, err := m.Health.Start()
	if err != nil {
		return nil, errors.Errorf("failed to follow")
	}
	multi.Add(healthStop)

	// Node Updates
	multi.Add(nodes.StartRegistration(m.GetSender(), m.Session, m.Rng,
		m.Comms, m.NodeRegistration, m.param.ParallelNodeRegistrations)) // Adding/MixCypher
	//TODO-remover
	//m.runners.Add(StartNodeRemover(m.Context))        // Removing

	// Start the Network Tracker
	trackNetworkStopper := stoppable.NewSingle("TrackNetwork")
	go m.followNetwork(report, trackNetworkStopper)
	multi.Add(trackNetworkStopper)

	// Message reception
	multi.Add(m.message.StartProcessies())

	// Round processing
	multi.Add(m.round.StartProcessors())

	multi.Add(ephemeral.Track(m.Session, m.addrSpace, m.ReceptionID))

	return multi, nil
}

// GetEventManager returns the health tracker
func (m *manager) GetEventManager() interfaces.EventManager {
	return m.events
}

// GetHealthTracker returns the health tracker
func (m *manager) GetHealthTracker() interfaces.HealthTracker {
	return m.Health
}

// GetInstance returns the network instance object (ndf state)
func (m *manager) GetInstance() *commNetwork.Instance {
	return m.Instance
}

// GetSender returns the gateway.Sender object
func (m *manager) GetSender() *gateway.Sender {
	return m.sender
}

// CheckGarbledMessages triggers a check on garbled messages to see if they can be decrypted
// this should be done when a new e2e client is added in case messages were
// received early or arrived out of order
func (m *manager) CheckGarbledMessages() {
	m.message.CheckGarbledMessages()
}

// InProgressRegistrations returns an approximation of the number of in progress
// nodes registrations.
func (m *manager) InProgressRegistrations() int {
	return len(m.Internal.NodeRegistration)
}

// GetAddressSize returns the current address space size. It blocks until an
// address space size is set.
func (m *manager) GetAddressSize() uint8 {
	return m.addrSpace.Get()
}

// RegisterAddressSizeNotification returns a channel that will trigger for every
// address space size update. The provided tag is the unique ID for the channel.
// Returns an error if the tag is already used.
func (m *manager) RegisterAddressSizeNotification(tag string) (chan uint8, error) {
	return m.addrSpace.RegisterNotification(tag)
}

// UnregisterAddressSizeNotification stops broadcasting address space size
// updates on the channel with the specified tag.
func (m *manager) UnregisterAddressSizeNotification(tag string) {
	m.addrSpace.UnregisterNotification(tag)
}

// SetPoolFilter sets the filter used to filter gateway IDs.
func (m *manager) SetPoolFilter(f gateway.Filter) {
	m.sender.SetFilter(f)
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

// GetFakeEarliestRound generates a random earliest round for a fake identity.
func (m *manager) GetFakeEarliestRound() id.Round {
	b, err := csprng.Generate(8, rand.Reader)
	if err != nil {
		jww.FATAL.Panicf("Could not get random number: %v", err)
	}

	rangeVal := binary.LittleEndian.Uint64(b) % 800

	earliestKnown := atomic.LoadUint64(m.earliestRound)

	return id.Round(earliestKnown - rangeVal)
}
