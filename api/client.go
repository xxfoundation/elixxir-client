////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/elixxir/client/keyExchange"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/permissioning"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

type Client struct {
	//generic RNG for client
	rng *fastRNG.StreamGenerator
	// the storage session securely stores data to disk and memoizes as is
	// appropriate
	storage     *storage.Session
	//the switchboard is used for inter-process signaling about received messages
	switchboard *switchboard.Switchboard
	//object used for communications
	comms *client.Comms

	// note that the manager has a pointer to the context in many cases, but
	// this interface allows it to be mocked for easy testing without the
	// loop
	network interfaces.NetworkManager
	//object used to register and communicate with permissioning
	permissioning *permissioning.Permissioning

	//contains stopables for all running threads
	runner *stoppable.Multi
	status *statusTracker
}

// NewClient creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
func NewClient(ndfJSON, storageDir string, password []byte, registrationCode string) (*Client, error) {
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	// Parse the NDF
	def, err := parseNDF(ndfJSON)
	if err != nil {
		return nil, err
	}
	cmixGrp, e2eGrp := decodeGroups(def)

	protoUser := createNewUser(rngStream, cmixGrp, e2eGrp)

	// Create Storage
	passwordStr := string(password)
	storageSess, err := storage.New(storageDir, passwordStr, protoUser,
		cmixGrp, e2eGrp, rngStreamGen)
	if err != nil {
		return nil, err
	}

	// Save NDF to be used in the future
	storageSess.SetBaseNDF(def)

	//store the registration code for later use
	storageSess.SetRegCode(registrationCode)

	//move the registration state to keys generated
	err = storageSess.ForwardRegistrationStatus(storage.KeyGenComplete)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to denote state "+
			"change in session")
	}

	//execute the rest of the loading as normal
	return loadClient(storageSess, rngStreamGen)
}

// NewPrecannedClient creates an insecure user with predetermined keys with nodes
// It creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
func NewPrecannedClient(precannedID uint, defJSON, storageDir string, password []byte) (*Client, error) {
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	// Parse the NDF
	def, err := parseNDF(defJSON)
	if err != nil {
		return nil, err
	}
	cmixGrp, e2eGrp := decodeGroups(def)

	protoUser := createPrecannedUser(precannedID, rngStream, cmixGrp, e2eGrp)

	// Create Storage
	passwordStr := string(password)
	storageSess, err := storage.New(storageDir, passwordStr, protoUser,
		cmixGrp, e2eGrp, rngStreamGen)
	if err != nil {
		return nil, err
	}

	// Save NDF to be used in the future
	storageSess.SetBaseNDF(def)

	//move the registration state to keys generated
	err = storageSess.ForwardRegistrationStatus(storage.KeyGenComplete)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to denote state "+
			"change in session")
	}

	//execute the rest of the loading as normal
	return loadClient(storageSess, rngStreamGen)
}


// LoadClient initalizes a client object from existing storage.
func LoadClient(storageDir string, password []byte) (*Client, error) {
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)

	// Load Storage
	passwordStr := string(password)
	storageSess, err := storage.Load(storageDir, passwordStr, rngStreamGen)
	if err != nil {
		return nil, err
	}

	//execute the rest of the loading as normal
	return loadClient(storageSess, rngStreamGen)
}

// LoadClient initalizes a client object from existing storage.
func loadClient(session *storage.Session, rngStreamGen *fastRNG.StreamGenerator) (c *Client, err error) {

	// Set up a new context
	c = &Client{
		storage:     session,
		switchboard: switchboard.New(),
		rng:         rngStreamGen,
		comms:       nil,
		network:     nil,
		runner:      stoppable.NewMulti("client"),
		status:      newStatusTracker(),
	}

	//get the user from session
	user := c.storage.User()
	cryptoUser := user.GetCryptographicIdentity()

	//start comms
	c.comms, err = client.NewClientComms(cryptoUser.GetUserID(),
		rsa.CreatePublicKeyPem(cryptoUser.GetRSA().GetPublic()),
		rsa.CreatePrivateKeyPem(cryptoUser.GetRSA()),
		cryptoUser.GetSalt())
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load client")
	}

	//get the NDF to pass into permissioning and the network manager
	def := session.GetBaseNDF()

	//initialize permissioning
	c.permissioning, err = permissioning.Init(c.comms, def)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to init "+
			"permissioning handler")
	}

	// check the client version is up to date to the network
	err = c.checkVersion()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load client")
	}

	//register with permissioning if necessary
	if c.storage.GetRegistrationStatus() == storage.KeyGenComplete {
		jww.INFO.Printf("Client has not registered yet, attempting registration")
		err = c.registerWithPermissioning()
		if err != nil {
			jww.ERROR.Printf("Client has failed registration: %s", err)
			return nil, errors.WithMessage(err, "failed to load client")
		}
		jww.INFO.Printf("Client sucsecfully registered with the network")
	}

	// Initialize network and link it to context
	c.network, err = network.NewManager(c.storage, c.switchboard, c.rng, c.comms,
		params.GetDefaultNetwork(), def)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// ----- Client Functions -----
// StartNetworkFollower kicks off the tracking of the network. It starts
// long running network client threads and returns an object for checking
// state and stopping those threads.
// Call this when returning from sleep and close when going back to
// sleep.
// These threads may become a significant drain on battery when offline, ensure
// they are stopped if there is no internet access
// Threads Started:
//   - Network Follower (/network/follow.go)
//   	tracks the network events and hands them off to workers for handling
//   - Historical Round Retrieval (/network/rounds/historical.go)
//		Retrieves data about rounds which are too old to be stored by the client
//	 - Message Retrieval Worker Group (/network/rounds/retrieve.go)
//		Requests all messages in a given round from the gateway of the last node
//	 - Message Handling Worker Group (/network/message/handle.go)
//		Decrypts and partitions messages when signals via the Switchboard
//	 - Health Tracker (/network/health)
//		Via the network instance tracks the state of the network
//	 - Garbled Messages (/network/message/garbled.go)
//		Can be signaled to check all recent messages which could be be decoded
//		Uses a message store on disk for persistence
//	 - Critical Messages (/network/message/critical.go)
//		Ensures all protocol layer mandatory messages are sent
//		Uses a message store on disk for persistence
//	 - KeyExchange Trigger (/keyExchange/trigger.go)
//		Responds to sent rekeys and executes them
//   - KeyExchange Confirm (/keyExchange/confirm.go)
//		Responds to confirmations of successful rekey operations
func (c *Client) StartNetworkFollower() error {
	jww.INFO.Printf("StartNetworkFollower()")

	err := c.status.toStarting()
	if err != nil {
		return errors.WithMessage(err, "Failed to Start the Network Follower")
	}

	stopFollow, err := c.network.Follow()
	if err != nil {
		return errors.WithMessage(err, "Failed to start following "+
			"the network")
	}
	c.runner.Add(stopFollow)
	// Key exchange
	c.runner.Add(keyExchange.Start(c.switchboard, c.storage, c.network))

	err = c.status.toRunning()
	if err != nil {
		return errors.WithMessage(err, "Failed to Start the Network Follower")
	}

	return nil
}

// StopNetworkFollower stops the network follower if it is running.
// It returns errors if the Follower is in the wrong status to stop or if it
// fails to stop it.
// if the network follower is running and this fails, the client object will
// most likely be in an unrecoverable state and need to be trashed.
func (c *Client) StopNetworkFollower(timeout time.Duration) error {
	err := c.status.toStopping()
	if err != nil {
		return errors.WithMessage(err, "Failed to Stop the Network Follower")
	}
	err = c.runner.Close(timeout)
	if err != nil {
		return errors.WithMessage(err, "Failed to Stop the Network Follower")
	}
	c.runner = stoppable.NewMulti("client")
	err = c.status.toStopped()
	if err != nil {
		return errors.WithMessage(err, "Failed to Stop the Network Follower")
	}
	return nil
}

// Gets the state of the network follower. Returns:
// Stopped 	- 0
// Starting - 1000
// Running	- 2000
// Stopping	- 3000
func (c *Client) NetworkFollowerStatus() Status {
	jww.INFO.Printf("NetworkFollowerStatus()")
	return c.status.get()
}

// Returns the health tracker for registration and polling
func (c *Client) GetHealth() interfaces.HealthTracker {
	jww.INFO.Printf("GetHealth()")
	return c.network.GetHealthTracker()
}

// RegisterRoundEventsCb registers a callback for round
// events.
func (c *Client) GetRoundEvents() interfaces.RoundEvents {
	jww.INFO.Printf("GetRoundEvents()")
	return c.network.GetInstance().GetRoundEvents()
}


// Returns the switchboard for Registration
func (c *Client) GetSwitchboard() interfaces.Switchboard {
	jww.INFO.Printf("GetSwitchboard()")
	return c.switchboard
}

// GetUser returns the current user Identity for this client. This
// can be serialized into a byte stream for out-of-band sharing.
func (c *Client) GetUser() user.User {
	jww.INFO.Printf("GetUser()")
	return c.storage.GetUser()
}

// ----- Utility Functions -----
// parseNDF parses the initial ndf string for the client. do not check the
// signature, it is deprecated.
func parseNDF(ndfString string) (*ndf.NetworkDefinition, error) {
	if ndfString == "" {
		return nil, errors.New("ndf file empty")
	}

	ndf, _, err := ndf.DecodeNDF(ndfString)
	if err != nil {
		return nil, err
	}

	return ndf, nil
}

// decodeGroups returns the e2e and cmix groups from the ndf
func decodeGroups(ndf *ndf.NetworkDefinition) (cmixGrp, e2eGrp *cyclic.Group) {
	largeIntBits := 16

	//Generate the cmix group
	cmixGrp = cyclic.NewGroup(
		large.NewIntFromString(ndf.CMIX.Prime, largeIntBits),
		large.NewIntFromString(ndf.CMIX.Generator, largeIntBits))
	//Generate the e2e group
	e2eGrp = cyclic.NewGroup(
		large.NewIntFromString(ndf.E2E.Prime, largeIntBits),
		large.NewIntFromString(ndf.E2E.Generator, largeIntBits))

	return cmixGrp, e2eGrp
}
