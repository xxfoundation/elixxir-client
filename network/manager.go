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
	"gitlab.com/elixxir/client/network/parse"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	//	"gitlab.com/xx_network/primitives/ndf"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"time"
)

// Manager implements the NetworkManager interface inside context. It
// controls access to network resources and implements all of the communications
// functions used by the client.
type Manager struct {
	// Comms pointer to send/recv messages
	Comms *client.Comms
	// Context contains all of the keying info used to send messages
	Context *context.Context

	// runners are the Network goroutines that handle reception
	runners *stoppable.Multi

	//contains the health tracker which keeps track of if from the client's
	//perspective, the network is in good condition
	health *health.Tracker

	//contains the network instance
	instance *network.Instance

	//Partitioner
	partitioner parse.Partitioner

	//channels
	nodeRegistration chan network.NodeGateway
	roundUpdate      chan *pb.RoundInfo
	historicalLookup chan id.Round

	// Processing rounds
	Processing *ProcessingRounds

	//local pointer to user ID because it is used often
	uid *id.ID
}

// NewManager builds a new reception manager object using inputted key fields
func NewManager(ctx *context.Context) (*Manager, error) {

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
	instance, err := network.NewInstance(comms.ProtoComms, nil, nil, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create"+
			" client network manager")
	}

	opts := params.GetDefaultNetwork()

	cm := &Manager{
		Comms:            comms,
		Context:          ctx,
		runners:          stoppable.NewMulti("network.Manager"),
		health:           health.Init(ctx, 5*time.Second),
		instance:         instance,
		uid:              cryptoUser.GetUserID(),
		partitioner: parse.NewPartitioner(msgSize, ctx),
		Processing:       NewProcessingRounds(),
		roundUpdate:      make(chan *pb.RoundInfo, opts.NumWorkers),
		historicalLookup: make(chan id.Round, opts.NumWorkers),
		nodeRegistration: make(chan network.NodeGateway,
			opts.NumWorkers),
	}

	return cm, nil
}

// GetRemoteVersion contacts the permissioning server and returns the current
// supported client version.
func (m *Manager) GetRemoteVersion() (string, error) {
	permissioningHost, ok := m.Comms.GetHost(&id.Permissioning)
	if !ok {
		return "", errors.Errorf("no permissioning host with id %s",
			id.Permissioning)
	}
	registrationVersion, err := m.Comms.SendGetCurrentClientVersionMessage(
		permissioningHost)
	if err != nil {
		return "", err
	}
	return registrationVersion.Version, nil
}

// StartRunners kicks off all network reception goroutines ("threads").
func (m *Manager) StartRunners() error {
	if m.runners.IsRunning() {
		return errors.Errorf("network routines are already running")
	}

	// Start the Network Tracker
	m.runners.Add(StartTrackNetwork(m.Context, m))
	// Message reception
	m.runners.Add(StartMessageReceivers(m.Context, m))
	// Node Updates
	m.runners.Add(StartNodeKeyExchange(m.Context, m)) // Adding/Keys
	m.runners.Add(StartNodeRemover(m.Context))        // Removing
	// Round history processing
	m.runners.Add(StartProcessHistoricalRounds(m.Context, m))
	// health tracker
	m.health.Start()
	m.runners.Add(m.health)

	return nil
}

// GetRunners returns the network goroutines such that they can be named
// and stopped.
func (m *Manager) GetRunners() stoppable.Stoppable {
	return m.runners
}

// StopRunners stops all the reception goroutines
func (m *Manager) StopRunners(timeout time.Duration) error {
	err := m.runners.Close(timeout)
	m.runners = stoppable.NewMulti("network.Manager")
	return err
}

// GetHealthTracker returns the health tracker
func (m *Manager) GetHealthTracker() context.HealthTracker {
	return m.health
}

// GetInstance returns the network instance object (ndf state)
func (m *Manager) GetInstance() *network.Instance {
	return m.instance
}

// GetNodeRegistrationCh returns node registration channel for node
// events.
func (m *Manager) GetNodeRegistrationCh() chan network.NodeGateway {
	return m.nodeRegistration
}

// GetRoundUpdateCh returns the network managers round update channel
func (m *Manager) GetRoundUpdateCh() chan *pb.RoundInfo {
	return m.roundUpdate
}

// GetHistoricalLookupCh returns the historical round lookup channel
func (m *Manager) GetHistoricalLookupCh() chan id.Round {
	return m.historicalLookup
}
