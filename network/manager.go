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
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/ephemeral"
	"gitlab.com/elixxir/client/network/health"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/network/node"
	"gitlab.com/elixxir/client/network/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/ndf"
	"sync/atomic"

	"time"
)

// Manager implements the NetworkManager interface inside context. It
// controls access to network resources and implements all of the communications
// functions used by the client.
type manager struct {
	// parameters of the network
	param params.Network

	//Shared data with all sub managers
	internal.Internal

	//sub-managers
	round   *rounds.Manager
	message *message.Manager
	//atomic denotes if the network is running
	running *uint32
}

// NewManager builds a new reception manager object using inputted key fields
func NewManager(session *storage.Session, switchboard *switchboard.Switchboard,
	rng *fastRNG.StreamGenerator, comms *client.Comms,
	params params.Network, ndf *ndf.NetworkDefinition) (interfaces.NetworkManager, error) {

	//start network instance
	instance, err := network.NewInstance(comms.ProtoComms, ndf, nil, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create"+
			" client network manager")
	}

	running := uint32(0)

	//create manager object
	m := manager{
		param:   params,
		running: &running,
	}

	m.Internal = internal.Internal{
		Session:          session,
		Switchboard:      switchboard,
		Rng:              rng,
		Comms:            comms,
		Health:           health.Init(instance, params.NetworkHealthTimeout),
		NodeRegistration: make(chan network.NodeGateway, params.RegNodesBufferLen),
		Instance:         instance,
		Uid:              session.User().GetCryptographicIdentity().GetTransmissionID(),
	}

	//create sub managers
	m.message = message.NewManager(m.Internal, m.param.Messages, m.NodeRegistration)
	m.round = rounds.NewManager(m.Internal, m.param.Rounds, m.message.GetMessageReceptionChannel())

	return &m, nil
}

// StartRunners kicks off all network reception goroutines ("threads").
// Started Threads are:
//   - Network Follower (/network/follow.go)
//   - Historical Round Retrieval (/network/rounds/historical.go)
//	 - Message Retrieval Worker Group (/network/rounds/retrieve.go)
//	 - Message Handling Worker Group (/network/message/handle.go)
//	 - Health Tracker (/network/health)
//	 - Garbled Messages (/network/message/garbled.go)
//	 - Critical Messages (/network/message/critical.go)
//   - Ephemeral ID tracking (network/ephemeral/tracker.go)
func (m *manager) Follow() (stoppable.Stoppable, error) {
	if !atomic.CompareAndSwapUint32(m.running, 0, 1) {
		return nil, errors.Errorf("network routines are already running")
	}

	multi := stoppable.NewMulti("networkManager")

	// health tracker
	healthStop, err := m.Health.Start()
	if err != nil {
		return nil, errors.Errorf("failed to follow")
	}
	multi.Add(healthStop)

	// Node Updates
	multi.Add(node.StartRegistration(m.Instance, m.Session, m.Rng,
		m.Comms, m.NodeRegistration)) // Adding/Keys
	//TODO-remover
	//m.runners.Add(StartNodeRemover(m.Context))        // Removing

	// Start the Network Tracker
	trackNetworkStopper := stoppable.NewSingle("TrackNetwork")
	go m.followNetwork(trackNetworkStopper.Quit())
	multi.Add(trackNetworkStopper)

	// Message reception
	multi.Add(m.message.StartProcessies())

	// Round processing
	multi.Add(m.round.StartProcessors())

	// Ephemeral ID tracking
	err = checkTimestampStore(m.Session)
	if err != nil {
		return nil, errors.Errorf("Could not store timestamp " +
			"for ephemeral ID tracking: %v", err)
	}

	multi.Add(ephemeral.Track(m.Session, m.Instance, m.Comms.Id))

	//set the running status back to 0 so it can be started again
	closer := stoppable.NewCleanup(multi, func(time.Duration) error {
		if !atomic.CompareAndSwapUint32(m.running, 1, 0) {
			return errors.Errorf("network routines are already stopped")
		}
		return nil
	})

	return closer, nil
}

// Sanitation check of timestamp store. If a value has not been stored yet
// then the current time is stored
func checkTimestampStore(session *storage.Session) error {
	if _, err := session.Get(ephemeral.TimestampKey); err != nil {
		now, err := ephemeral.MarshalTimestamp(time.Now())
		if err != nil {
			return errors.Errorf("Could not marshal new timestamp for storage: %v", err)
		}
		return session.Set(ephemeral.TimestampKey, now)
	}

	return nil
}

// GetHealthTracker returns the health tracker
func (m *manager) GetHealthTracker() interfaces.HealthTracker {
	return m.Health
}

// GetInstance returns the network instance object (ndf state)
func (m *manager) GetInstance() *network.Instance {
	return m.Instance
}

// triggers a check on garbled messages to see if they can be decrypted
// this should be done when a new e2e client is added in case messages were
// received early or arrived out of order
func (m *manager) CheckGarbledMessages() {
	m.message.CheckGarbledMessages()
}
