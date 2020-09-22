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
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/network/health"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/network/node"
	"gitlab.com/elixxir/client/network/permissioning"
	"gitlab.com/elixxir/client/network/rounds"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"

	"time"
)

// Manager implements the NetworkManager interface inside context. It
// controls access to network resources and implements all of the communications
// functions used by the client.
type Manager struct {
	m manager
}

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
func NewManager(ctx *context.Context, params params.Network, ndf *ndf.NetworkDefinition) (*Manager, error) {

	//get the user from storage
	user := ctx.Session.User()
	cryptoUser := user.GetCryptographicIdentity()

	//start comms
	comms, err := client.NewClientComms(cryptoUser.GetUserID(),
		rsa.CreatePublicKeyPem(cryptoUser.GetRSA().GetPublic()),
		rsa.CreatePrivateKeyPem(cryptoUser.GetRSA()),
		cryptoUser.GetSalt())
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create"+
			" client network manager")
	}

	//start network instance
	// TODO: Need to parse/retrieve the ntework string and load it
	// from the context storage session!
	instance, err := network.NewInstance(comms.ProtoComms, ndf, nil, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create"+
			" client network manager")
	}

	//create manager object
	m := manager{
		param:    params,
		runners:  stoppable.NewMulti("network.Manager"),
	}

	m.Internal = internal.Internal{
		Comms:            comms,
		Health:           health.Init(ctx, 5*time.Second),
		NodeRegistration: make(chan network.NodeGateway, params.RegNodesBufferLen),
		Instance:         instance,
	}

	m.Internal.Context = ctx

	//create sub managers
	m.message = message.NewManager(m.Internal, m.param.Messages, m.NodeRegistration)
	m.round = rounds.NewManager(m.Internal, m.param.Rounds, m.message.GetMessageReceptionChannel())

	return &Manager{m: m}, nil
}

// GetRemoteVersion contacts the permissioning server and returns the current
// supported client version.
func (m *Manager) GetRemoteVersion() (string, error) {
	permissioningHost, ok := m.m.Comms.GetHost(&id.Permissioning)
	if !ok {
		return "", errors.Errorf("no permissioning host with id %s",
			id.Permissioning)
	}
	registrationVersion, err := m.m.Comms.SendGetCurrentClientVersionMessage(
		permissioningHost)
	if err != nil {
		return "", err
	}
	return registrationVersion.Version, nil
}

func (m *Manager) StartRunners() error {
	return m.m.startRunners()
}

// StartRunners kicks off all network reception goroutines ("threads").
func (m *manager) startRunners() error {
	if m.runners.IsRunning() {
		return errors.Errorf("network routines are already running")
	}

	// health tracker
	m.Health.Start()
	m.runners.Add(m.Health)

	// Node Updates
	m.runners.Add(node.StartRegistration(m.Context, m.Comms, m.NodeRegistration)) // Adding/Keys
	//TODO-remover
	//m.runners.Add(StartNodeRemover(m.Context))        // Removing

	// Start the Network Tracker
	trackNetworkStopper := stoppable.NewSingle("TrackNetwork")
	go m.trackNetwork(trackNetworkStopper.Quit())
	m.runners.Add(trackNetworkStopper)

	// Message reception
	m.runners.Add(m.message.StartMessageReceptionWorkerPool())

	// Round processing
	m.runners.Add(m.round.StartProcessors())

	return nil
}

func (m *Manager) RegisterWithPermissioning(registrationCode string) ([]byte, error) {
	pubKey := m.m.Session.User().GetCryptographicIdentity().GetRSA().GetPublic()
	return permissioning.Register(m.m.Comms, pubKey, registrationCode)
}

// GetRunners returns the network goroutines such that they can be named
// and stopped.
func (m *Manager) GetRunners() stoppable.Stoppable {
	return m.m.runners
}

// StopRunners stops all the reception goroutines
func (m *Manager) GetStoppable() stoppable.Stoppable {
	return m.m.runners
}

// GetHealthTracker returns the health tracker
func (m *Manager) GetHealthTracker() context.HealthTracker {
	return m.m.Health
}

// GetInstance returns the network instance object (ndf state)
func (m *Manager) GetInstance() *network.Instance {
	return m.m.Instance
}

// GetNodeRegistrationCh returns node registration channel for node
// events.
func (m *Manager) GetNodeRegistrationCh() chan network.NodeGateway {
	return m.m.NodeRegistration
}
