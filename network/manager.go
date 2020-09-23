////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

// manager.go controls access to network resources. Interprocess communications
// and intraclient state are accessible through the context object.

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/keyExchange"
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

	// runners are the Network goroutines that handle reception
	runners *stoppable.Multi

	//sub-managers
	round   *rounds.Manager
	message *message.Manager
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

	//create manager object
	m := manager{
		param:   params,
		runners: stoppable.NewMulti("network.Manager"),
	}

	m.Internal = internal.Internal{
		Session:          session,
		Switchboard:      switchboard,
		Rng:              rng,
		Comms:            comms,
		Health:           health.Init(instance, 5*time.Second),
		NodeRegistration: make(chan network.NodeGateway, params.RegNodesBufferLen),
		Instance:         instance,
		Uid:              session.User().GetCryptographicIdentity().GetUserID(),
	}

	//create sub managers
	m.message = message.NewManager(m.Internal, m.param.Messages, m.NodeRegistration)
	m.round = rounds.NewManager(m.Internal, m.param.Rounds, m.message.GetMessageReceptionChannel())

	return &m, nil
}

// StartRunners kicks off all network reception goroutines ("threads").
func (m *manager) StartRunners() error {
	if m.runners.IsRunning() {
		return errors.Errorf("network routines are already running")
	}

	// health tracker
	m.Health.Start()
	m.runners.Add(m.Health)

	// Node Updates
	m.runners.Add(node.StartRegistration(m.Instance, m.Session, m.Rng,
		m.Comms, m.NodeRegistration)) // Adding/Keys
	//TODO-remover
	//m.runners.Add(StartNodeRemover(m.Context))        // Removing

	// Start the Network Tracker
	trackNetworkStopper := stoppable.NewSingle("TrackNetwork")
	go m.trackNetwork(trackNetworkStopper.Quit())
	m.runners.Add(trackNetworkStopper)

	// Message reception
	m.runners.Add(m.message.StartProcessies())

	// Round processing
	m.runners.Add(m.round.StartProcessors())

	// Key exchange
	m.runners.Add(keyExchange.Start(m.Switchboard, m.Session, m,
		m.message.GetTriggerGarbledCheckChannel()))

	return nil
}

// StopRunners stops all the reception goroutines
func (m *manager) GetStoppable() stoppable.Stoppable {
	return m.runners
}

// GetHealthTracker returns the health tracker
func (m *manager) GetHealthTracker() interfaces.HealthTracker {
	return m.Health
}

// GetInstance returns the network instance object (ndf state)
func (m *manager) GetInstance() *network.Instance {
	return m.Instance
}

