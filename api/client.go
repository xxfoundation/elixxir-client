///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/elixxir/client/keyExchange"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/registration"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/region"
	"math"
	"time"
)

const followerStoppableName = "client"

type Client struct {
	//generic RNG for client
	rng *fastRNG.StreamGenerator
	// the storage session securely stores data to disk and memoizes as is
	// appropriate
	storage *storage.Session
	//the switchboard is used for inter-process signaling about received messages
	switchboard *switchboard.Switchboard
	//object used for communications
	comms *client.Comms
	// Network parameters
	parameters params.Network

	// note that the manager has a pointer to the context in many cases, but
	// this interface allows it to be mocked for easy testing without the
	// loop
	network interfaces.NetworkManager
	//object used to register and communicate with permissioning
	permissioning *registration.Registration
	//object containing auth interactions
	auth *auth.Manager

	//services system to track running threads
	followerServices *services

	clientErrorChannel chan interfaces.ClientError

	// Event reporting in event.go
	events *eventManager

	// Handles the triggering and delivery of backups
	backup *interfaces.BackupContainer
}

// NewClient creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
func NewClient(ndfJSON, storageDir string, password []byte,
	registrationCode string) error {
	jww.INFO.Printf("NewClient(dir: %s)", storageDir)
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024, csprng.NewSystemRNG)

	// Parse the NDF
	def, err := parseNDF(ndfJSON)
	if err != nil {
		return err
	}

	cmixGrp, e2eGrp := decodeGroups(def)
	start := time.Now()
	protoUser := createNewUser(rngStreamGen, cmixGrp, e2eGrp)
	jww.DEBUG.Printf("User generation took: %s", time.Now().Sub(start))

	_, err = checkVersionAndSetupStorage(def, storageDir, password, protoUser,
		cmixGrp, e2eGrp, rngStreamGen, false, registrationCode)
	if err != nil {
		return err
	}

	//TODO: close the session
	return nil
}

// NewPrecannedClient creates an insecure user with predetermined keys with nodes
// It creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
func NewPrecannedClient(precannedID uint, defJSON, storageDir string,
	password []byte) error {
	jww.INFO.Printf("NewPrecannedClient()")
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024, csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	// Parse the NDF
	def, err := parseNDF(defJSON)
	if err != nil {
		return err
	}
	cmixGrp, e2eGrp := decodeGroups(def)

	protoUser := createPrecannedUser(precannedID, rngStream, cmixGrp, e2eGrp)

	_, err = checkVersionAndSetupStorage(def, storageDir, password, protoUser,
		cmixGrp, e2eGrp, rngStreamGen, true, "")
	if err != nil {
		return err
	}
	//TODO: close the session
	return nil
}

// NewVanityClient creates a user with a receptionID that starts with the supplied prefix
// It creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
func NewVanityClient(ndfJSON, storageDir string, password []byte,
	registrationCode string, userIdPrefix string) error {
	jww.INFO.Printf("NewVanityClient()")
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024, csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	// Parse the NDF
	def, err := parseNDF(ndfJSON)
	if err != nil {
		return err
	}
	cmixGrp, e2eGrp := decodeGroups(def)

	protoUser := createNewVanityUser(rngStream, cmixGrp, e2eGrp, userIdPrefix)

	_, err = checkVersionAndSetupStorage(def, storageDir, password, protoUser,
		cmixGrp, e2eGrp, rngStreamGen, false, registrationCode)
	if err != nil {
		return err
	}

	//TODO: close the session
	return nil
}

// NewClientFromBackup constructs a new Client from an encrypted backup. The backup
// is decrypted using the backupPassphrase. On success a successful client creation,
// the function will return a JSON encoded list of the E2E partners
// contained in the backup and a json-encoded string containing parameters stored in the backup
func NewClientFromBackup(ndfJSON, storageDir string, sessionPassword,
	backupPassphrase []byte, backupFileContents []byte) ([]*id.ID, string, error) {

	backUp := &backup.Backup{}
	err := backUp.Decrypt(string(backupPassphrase), backupFileContents)
	if err != nil {
		return nil, "", errors.WithMessage(err, "Failed to unmarshal decrypted client contents.")
	}

	usr := user.NewUserFromBackup(backUp)

	// Parse the NDF
	def, err := parseNDF(ndfJSON)
	if err != nil {
		return nil, "", err
	}

	cmixGrp, e2eGrp := decodeGroups(def)

	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)

	// Create storage object.
	// Note we do not need registration
	storageSess, err := checkVersionAndSetupStorage(def, storageDir, []byte(sessionPassword), usr,
		cmixGrp, e2eGrp, rngStreamGen, false, backUp.RegistrationCode)

	// Set registration values in storage
	storageSess.User().SetReceptionRegistrationValidationSignature(backUp.ReceptionIdentity.RegistrarSignature)
	storageSess.User().SetTransmissionRegistrationValidationSignature(backUp.TransmissionIdentity.RegistrarSignature)
	storageSess.User().SetRegistrationTimestamp(backUp.RegistrationTimestamp)

	//move the registration state to indicate registered with registration on proto client
	err = storageSess.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return nil, "", err
	}

	return backUp.Contacts.Identities, backUp.JSONParams, nil
}

// OpenClient session, but don't connect to the network or log in
func OpenClient(storageDir string, password []byte, parameters params.Network) (*Client, error) {
	jww.INFO.Printf("OpenClient()")
	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024, csprng.NewSystemRNG)

	// Get current client version
	currentVersion, err := version.ParseVersion(SEMVER)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not parse version string.")
	}

	// Load Storage
	passwordStr := string(password)
	storageSess, err := storage.Load(storageDir, passwordStr, currentVersion,
		rngStreamGen)
	if err != nil {
		return nil, err
	}

	// Set up a new context
	c := &Client{
		storage:            storageSess,
		switchboard:        switchboard.New(),
		rng:                rngStreamGen,
		comms:              nil,
		network:            nil,
		followerServices:   newServices(),
		parameters:         parameters,
		clientErrorChannel: make(chan interfaces.ClientError, 1000),
		events:             newEventManager(),
		backup:             &interfaces.BackupContainer{},
	}

	return c, nil
}

// NewProtoClient_Unsafe initializes a client object from a JSON containing
// predefined cryptographic which defines a user. This is designed for some
// specific deployment procedures and is generally unsafe.
func NewProtoClient_Unsafe(ndfJSON, storageDir string, password,
	protoClientJSON []byte) error {
	jww.INFO.Printf("NewProtoClient_Unsafe")

	// Use fastRNG for RNG ops (AES fortuna based RNG using system RNG)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)

	// Parse the NDF
	def, err := parseNDF(ndfJSON)
	if err != nil {
		return err
	}

	cmixGrp, e2eGrp := decodeGroups(def)

	// Pull the proto user from the JSON
	protoUser := &user.Proto{}
	err = json.Unmarshal(protoClientJSON, protoUser)
	if err != nil {
		return err
	}

	// Initialize a user object for storage set up
	usr := user.NewUserFromProto(protoUser)

	// Set up storage
	storageSess, err := checkVersionAndSetupStorage(def, storageDir, password, usr,
		cmixGrp, e2eGrp, rngStreamGen, false, protoUser.RegCode)
	if err != nil {
		return err
	}

	// Set registration values in storage
	storageSess.User().SetReceptionRegistrationValidationSignature(protoUser.ReceptionRegValidationSig)
	storageSess.User().SetTransmissionRegistrationValidationSignature(protoUser.TransmissionRegValidationSig)
	storageSess.User().SetRegistrationTimestamp(protoUser.RegistrationTimestamp)

	//move the registration state to indicate registered with registration on proto client
	err = storageSess.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return err
	}

	return nil
}

// Login initializes a client object from existing storage.
func Login(storageDir string, password []byte, parameters params.Network) (*Client, error) {
	jww.INFO.Printf("Login()")

	//Open the client
	c, err := OpenClient(storageDir, password, parameters)
	if err != nil {
		return nil, err
	}

	u := c.storage.GetUser()
	jww.INFO.Printf("Client Logged in: \n\tTransmisstionID: %s "+
		"\n\tReceptionID: %s", u.TransmissionID, u.ReceptionID)

	// initialize comms
	err = c.initComms()
	if err != nil {
		return nil, err
	}

	//get the NDF to pass into registration and the network manager
	def := c.storage.GetNDF()

	//initialize registration
	if def.Registration.Address != "" {
		err = c.initPermissioning(def)
		if err != nil {
			return nil, err
		}
	} else {
		jww.WARN.Printf("Registration with permissioning skipped due to " +
			"blank permissioning address. Client will not be able to register " +
			"or track network.")
	}

	if def.Notification.Address != "" {
		hp := connect.GetDefaultHostParams()
		// Client will not send KeepAlive packets
		hp.KaClientOpts.Time = time.Duration(math.MaxInt64)
		hp.AuthEnabled = false
		hp.MaxRetries = 5
		_, err = c.comms.AddHost(&id.NotificationBot, def.Notification.Address,
			[]byte(def.Notification.TlsCertificate), hp)
		if err != nil {
			jww.WARN.Printf("Failed adding host for notifications: %+v", err)
		}
	}

	// Initialize network and link it to context
	c.network, err = network.NewManager(c.storage, c.switchboard, c.rng,
		c.events, c.comms, parameters, def)
	if err != nil {
		return nil, err
	}

	// initialize the auth tracker
	c.auth = auth.NewManager(c.switchboard, c.storage, c.network, c.rng,
		c.backup.TriggerBackup, parameters.ReplayRequests)

	// Add all processes to the followerServices
	err = c.registerFollower()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// LoginWithNewBaseNDF_UNSAFE initializes a client object from existing storage
// while replacing the base NDF.  This is designed for some specific deployment
// procedures and is generally unsafe.
func LoginWithNewBaseNDF_UNSAFE(storageDir string, password []byte,
	newBaseNdf string, parameters params.Network) (*Client, error) {
	jww.INFO.Printf("LoginWithNewBaseNDF_UNSAFE()")

	// Parse the NDF
	def, err := parseNDF(newBaseNdf)
	if err != nil {
		return nil, err
	}

	//Open the client
	c, err := OpenClient(storageDir, password, parameters)
	if err != nil {
		return nil, err
	}

	//initialize comms
	err = c.initComms()
	if err != nil {
		return nil, err
	}

	//store the updated base NDF
	c.storage.SetNDF(def)

	//initialize registration
	if def.Registration.Address != "" {
		err = c.initPermissioning(def)
		if err != nil {
			return nil, err
		}
	} else {
		jww.WARN.Printf("Registration with permissioning skipped due to " +
			"blank permissionign address. Client will not be able to register " +
			"or track network.")
	}

	// Initialize network and link it to context
	c.network, err = network.NewManager(c.storage, c.switchboard, c.rng,
		c.events, c.comms, parameters, def)
	if err != nil {
		return nil, err
	}

	// initialize the auth tracker
	c.auth = auth.NewManager(c.switchboard, c.storage, c.network, c.rng,
		c.backup.TriggerBackup, parameters.ReplayRequests)

	err = c.registerFollower()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// LoginWithProtoClient creates a client object with a protoclient JSON containing the
// cryptographic primitives. This is designed for some specific deployment
//// procedures and is generally unsafe.
func LoginWithProtoClient(storageDir string, password []byte, protoClientJSON []byte,
	newBaseNdf string, parameters params.Network) (*Client, error) {
	jww.INFO.Printf("LoginWithProtoClient()")

	// Parse the NDF
	def, err := parseNDF(newBaseNdf)
	if err != nil {
		return nil, err
	}

	//Open the client
	err = NewProtoClient_Unsafe(newBaseNdf, storageDir, password, protoClientJSON)
	if err != nil {
		return nil, err
	}

	//Open the client
	c, err := OpenClient(storageDir, password, parameters)
	if err != nil {
		return nil, err
	}

	//initialize comms
	err = c.initComms()
	if err != nil {
		return nil, err
	}

	//store the updated base NDF
	c.storage.SetNDF(def)

	err = c.initPermissioning(def)
	if err != nil {
		return nil, err
	}

	// Initialize network and link it to context
	c.network, err = network.NewManager(c.storage, c.switchboard, c.rng,
		c.events, c.comms, parameters, def)
	if err != nil {
		return nil, err
	}

	// initialize the auth tracker
	c.auth = auth.NewManager(c.switchboard, c.storage, c.network, c.rng,
		c.backup.TriggerBackup, parameters.ReplayRequests)

	err = c.registerFollower()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) initComms() error {
	var err error

	//get the user from session
	u := c.storage.User()
	cryptoUser := u.GetCryptographicIdentity()

	//start comms
	c.comms, err = client.NewClientComms(cryptoUser.GetTransmissionID(),
		rsa.CreatePublicKeyPem(cryptoUser.GetTransmissionRSA().GetPublic()),
		rsa.CreatePrivateKeyPem(cryptoUser.GetTransmissionRSA()),
		cryptoUser.GetTransmissionSalt())
	if err != nil {
		return errors.WithMessage(err, "failed to load client")
	}
	return nil
}

func (c *Client) initPermissioning(def *ndf.NetworkDefinition) error {
	var err error
	//initialize registration
	c.permissioning, err = registration.Init(c.comms, def)
	if err != nil {
		return errors.WithMessage(err, "failed to init "+
			"permissioning handler")
	}

	//register with registration if necessary
	if c.storage.GetRegistrationStatus() == storage.KeyGenComplete {
		jww.INFO.Printf("Client has not registered yet, attempting registration")
		err = c.registerWithPermissioning()
		if err != nil {
			jww.ERROR.Printf("Client has failed registration: %s", err)
			return errors.WithMessage(err, "failed to load client")
		}
		jww.INFO.Printf("Client successfully registered with the network")
	}
	return nil
}

// registerFollower adds the follower processes to the client's follower service list.
// This should only ever be called once
func (c *Client) registerFollower() error {
	//build the error callback
	cer := func(source, message, trace string) {
		select {
		case c.clientErrorChannel <- interfaces.ClientError{
			Source:  source,
			Message: message,
			Trace:   trace,
		}:
		default:
			jww.WARN.Printf("Failed to notify about ClientError from %s: %s", source, message)
		}
	}

	err := c.followerServices.add(c.events.eventService)
	if err != nil {
		return errors.WithMessage(err, "Couldn't start event reporting")
	}

	//register the core follower service
	err = c.followerServices.add(func() (stoppable.Stoppable, error) {
		return c.network.Follow(cer)
	})
	if err != nil {
		return errors.WithMessage(err, "Failed to start following "+
			"the network")
	}

	//register the incremental key upgrade service
	err = c.followerServices.add(c.auth.StartProcesses)
	if err != nil {
		return errors.WithMessage(err, "Failed to start following "+
			"the network")
	}

	//register the key exchange service
	keyXchange := func() (stoppable.Stoppable, error) {
		return keyExchange.Start(c.switchboard, c.storage, c.network, c.parameters.Rekey)
	}
	err = c.followerServices.add(keyXchange)

	return nil
}

// ----- Client Functions -----

// GetErrorsChannel returns a channel which passess errors from the
// long running threads controlled by StartNetworkFollower and StopNetworkFollower
func (c *Client) GetErrorsChannel() <-chan interfaces.ClientError {
	return c.clientErrorChannel
}

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
//   - Auth Callback (/auth/callback.go)
//      Handles both auth confirm and requests
func (c *Client) StartNetworkFollower(timeout time.Duration) error {
	u := c.GetUser()
	jww.INFO.Printf("StartNetworkFollower() \n\tTransmissionID: %s "+
		"\n\tReceptionID: %s", u.TransmissionID, u.ReceptionID)

	return c.followerServices.start(timeout)
}

// StopNetworkFollower stops the network follower if it is running.
// It returns errors if the Follower is in the wrong state to stop or if it
// fails to stop it.
// if the network follower is running and this fails, the client object will
// most likely be in an unrecoverable state and need to be trashed.
func (c *Client) StopNetworkFollower() error {
	jww.INFO.Printf("StopNetworkFollower()")
	return c.followerServices.stop()
}

// NetworkFollowerStatus Gets the state of the network follower. Returns:
// Stopped 	- 0
// Starting - 1000
// Running	- 2000
// Stopping	- 3000
func (c *Client) NetworkFollowerStatus() Status {
	jww.INFO.Printf("NetworkFollowerStatus()")
	return c.followerServices.status()
}

// HasRunningProcessies checks if any background threads are running
// and returns true if one or more are
func (c *Client) HasRunningProcessies() bool {
	return !c.followerServices.stoppable.IsStopped()
}

// Returns the health tracker for registration and polling
func (c *Client) GetHealth() interfaces.HealthTracker {
	jww.INFO.Printf("GetHealth()")
	return c.network.GetHealthTracker()
}

// Returns the switchboard for Registration
func (c *Client) GetSwitchboard() interfaces.Switchboard {
	jww.INFO.Printf("GetSwitchboard()")
	return c.switchboard
}

// RegisterRoundEventsCb registers a callback for round
// events.
func (c *Client) GetRoundEvents() interfaces.RoundEvents {
	jww.INFO.Printf("GetRoundEvents()")
	jww.WARN.Printf("GetRoundEvents does not handle Client Errors edge case!")
	return c.network.GetInstance().GetRoundEvents()
}

// AddService adds a service ot be controlled by the client thread control,
// these will be started and stopped with the network follower
func (c *Client) AddService(sp Service) error {
	return c.followerServices.add(sp)
}

// GetUser returns the current user Identity for this client. This
// can be serialized into a byte stream for out-of-band sharing.
func (c *Client) GetUser() user.User {
	jww.INFO.Printf("GetUser()")
	return c.storage.GetUser()
}

// GetComms returns the client comms object
func (c *Client) GetComms() *client.Comms {
	return c.comms
}

// GetRng returns the client rng object
func (c *Client) GetRng() *fastRNG.StreamGenerator {
	return c.rng
}

// GetStorage returns the client storage object
func (c *Client) GetStorage() *storage.Session {
	return c.storage
}

// GetNetworkInterface returns the client Network Interface
func (c *Client) GetNetworkInterface() interfaces.NetworkManager {
	return c.network
}

// GetBackup returns a pointer to the backup container so that the backup can be
// set and triggered.
func (c *Client) GetBackup() *interfaces.BackupContainer {
	return c.backup
}

// GetRateLimitParams retrieves the rate limiting parameters.
func (c *Client) GetRateLimitParams() (uint32, uint32, int64) {
	rateLimitParams := c.storage.GetBucketParams().Get()
	return rateLimitParams.Capacity, rateLimitParams.LeakedTokens,
		rateLimitParams.LeakDuration.Nanoseconds()
}

// GetNodeRegistrationStatus gets the current state of node registration. It
// returns the total number of nodes in the NDF and the number of those which
// are currently registers with. An error is returned if the network is not
// healthy.
func (c *Client) GetNodeRegistrationStatus() (int, int, error) {
	// Return an error if the network is not healthy
	if !c.GetHealth().IsHealthy() {
		return 0, 0, errors.New("Cannot get number of node registrations when " +
			"network is not healthy")
	}

	nodes := c.GetNetworkInterface().GetInstance().GetPartialNdf().Get().Nodes

	cmixStore := c.storage.Cmix()

	var numRegistered int
	var numStale = 0
	for i, n := range nodes {
		nid, err := id.Unmarshal(n.ID)
		if err != nil {
			return 0, 0, errors.Errorf("Failed to unmarshal node ID %v "+
				"(#%d): %s", n.ID, i, err.Error())
		}
		if n.Status == ndf.Stale {
			numStale += 1
			continue
		}
		if cmixStore.Has(nid) {
			numRegistered++
		}
	}

	// Get the number of in progress node registrations
	return numRegistered, len(nodes) - numStale, nil
}

// DeleteRequest will delete a request, agnostic of request type
// for the given partner ID. If no request exists for this
// partner ID an error will be returned.
func (c *Client) DeleteRequest(partnerId *id.ID) error {
	jww.DEBUG.Printf("Deleting request for partner ID: %s", partnerId)
	return c.GetStorage().Auth().DeleteRequest(partnerId)
}

// DeleteAllRequests clears all requests from client's auth storage.
func (c *Client) DeleteAllRequests() error {
	jww.DEBUG.Printf("Deleting all requests")
	return c.GetStorage().Auth().DeleteAllRequests()
}

// DeleteSentRequests clears sent requests from client's auth storage.
func (c *Client) DeleteSentRequests() error {
	jww.DEBUG.Printf("Deleting all sent requests")
	return c.GetStorage().Auth().DeleteSentRequests()
}

// DeleteReceiveRequests clears receive requests from client's auth storage.
func (c *Client) DeleteReceiveRequests() error {
	jww.DEBUG.Printf("Deleting all received requests")
	return c.GetStorage().Auth().DeleteReceiveRequests()
}

// DeleteContact is a function which removes a partner from Client's storage
func (c *Client) DeleteContact(partnerId *id.ID) error {
	jww.DEBUG.Printf("Deleting contact with ID %s", partnerId)
	// get the partner so that they can be removed from preimage store
	partner, err := c.storage.E2e().GetPartner(partnerId)
	if err != nil {
		return errors.WithMessagef(err, "Could not delete %s because "+
			"they could not be found", partnerId)
	}
	e2ePreimage := partner.GetE2EPreimage()
	rekeyPreimage := partner.GetSilentPreimage()
	fileTransferPreimage := partner.GetFileTransferPreimage()
	groupRequestPreimage := partner.GetGroupRequestPreimage()

	//delete the partner
	if err = c.storage.E2e().DeletePartner(partnerId); err != nil {
		return err
	}

	// Trigger backup
	c.backup.TriggerBackup("contact deleted")

	//delete the preimages
	if err = c.storage.GetEdge().Remove(edge.Preimage{
		Data:   e2ePreimage,
		Type:   preimage.E2e,
		Source: partnerId[:],
	}, c.storage.GetUser().ReceptionID); err != nil {
		jww.WARN.Printf("Failed delete the preimage for e2e "+
			"from %s on contact deletion: %+v", partnerId, err)
	}

	if err = c.storage.GetEdge().Remove(edge.Preimage{
		Data:   rekeyPreimage,
		Type:   preimage.Silent,
		Source: partnerId[:],
	}, c.storage.GetUser().ReceptionID); err != nil {
		jww.WARN.Printf("Failed delete the preimage for rekey "+
			"from %s on contact deletion: %+v", partnerId, err)
	}

	if err = c.storage.GetEdge().Remove(edge.Preimage{
		Data:   fileTransferPreimage,
		Type:   preimage.EndFT,
		Source: partnerId[:],
	}, c.storage.GetUser().ReceptionID); err != nil {
		jww.WARN.Printf("Failed delete the preimage for file transfer "+
			"from %s on contact deletion: %+v", partnerId, err)
	}

	if err = c.storage.GetEdge().Remove(edge.Preimage{
		Data:   groupRequestPreimage,
		Type:   preimage.GroupRq,
		Source: partnerId[:],
	}, c.storage.GetUser().ReceptionID); err != nil {
		jww.WARN.Printf("Failed delete the preimage for group request "+
			"from %s on contact deletion: %+v", partnerId, err)
	}

	//delete conversations
	c.storage.Conversations().Delete(partnerId)

	// call delete requests to make sure nothing is lingering.
	// this is for saftey to ensure the contact can be readded
	// in the future
	_ = c.storage.Auth().Delete(partnerId)

	return nil
}

// SetProxiedBins updates the host pool filter that filters out gateways that
// are not in one of the specified bins.
func (c *Client) SetProxiedBins(binStrings []string) error {
	// Convert each region string into a region.GeoBin and place in a map for
	// easy lookup
	bins := make(map[region.GeoBin]bool, len(binStrings))
	for i, binStr := range binStrings {
		bin, err := region.GetRegion(binStr)
		if err != nil {
			return errors.Errorf("failed to parse geographic bin #%d: %+v", i, err)
		}

		bins[bin] = true
	}

	// Create filter func
	f := func(m map[id.ID]int, netDef *ndf.NetworkDefinition) map[id.ID]int {
		prunedList := make(map[id.ID]int, len(m))
		for gwID, i := range m {
			if bins[netDef.Gateways[i].Bin] {
				prunedList[gwID] = i
			}
		}
		return prunedList
	}

	c.network.SetPoolFilter(f)

	return nil
}

// GetPreferredBins returns the geographic bin or bins that the provided two
// character country code is a part of.
func (c *Client) GetPreferredBins(countryCode string) ([]string, error) {
	// Get the bin that the country is in
	bin, exists := region.GetCountryBin(countryCode)
	if !exists {
		return nil, errors.Errorf("failed to find geographic bin for country %q",
			countryCode)
	}

	// Add bin to list of geographic bins
	bins := []string{bin.String()}

	// Add additional bins in special cases
	switch bin {
	case region.SouthAndCentralAmerica:
		bins = append(bins, region.NorthAmerica.String())
	case region.MiddleEast:
		bins = append(bins, region.EasternEurope.String(), region.CentralEurope.String(), region.WesternAsia.String())
	case region.NorthernAfrica:
		bins = append(bins, region.WesternEurope.String(), region.CentralEurope.String())
	case region.SouthernAfrica:
		bins = append(bins, region.WesternEurope.String(), region.CentralEurope.String())
	case region.EasternAsia:
		bins = append(bins, region.WesternAsia.String(), region.Oceania.String(), region.NorthAmerica.String())
	case region.WesternAsia:
		bins = append(bins, region.EasternAsia.String(), region.Russia.String(), region.MiddleEast.String())
	case region.Oceania:
		bins = append(bins, region.EasternAsia.String(), region.NorthAmerica.String())
	}

	return bins, nil
}

// ----- Utility Functions -----
// parseNDF parses the initial ndf string for the client. do not check the
// signature, it is deprecated.
func parseNDF(ndfString string) (*ndf.NetworkDefinition, error) {
	if ndfString == "" {
		return nil, errors.New("ndf file empty")
	}

	netDef, err := ndf.Unmarshal([]byte(ndfString))
	if err != nil {
		return nil, err
	}

	return netDef, nil
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

// checkVersionAndSetupStorage is common code shared by NewClient, NewPrecannedClient and NewVanityClient
// it checks client version and creates a new storage for user data
func checkVersionAndSetupStorage(def *ndf.NetworkDefinition,
	storageDir string, password []byte,
	protoUser user.User,
	cmixGrp, e2eGrp *cyclic.Group, rngStreamGen *fastRNG.StreamGenerator,
	isPrecanned bool, registrationCode string) (*storage.Session, error) {
	// Get current client version
	currentVersion, err := version.ParseVersion(SEMVER)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not parse version string.")
	}

	// Create Storage
	passwordStr := string(password)
	storageSess, err := storage.New(storageDir, passwordStr, protoUser,
		currentVersion, cmixGrp, e2eGrp, rngStreamGen, def.RateLimits)
	if err != nil {
		return nil, err
	}

	// Save NDF to be used in the future
	storageSess.SetNDF(def)

	if !isPrecanned {
		//store the registration code for later use
		storageSess.SetRegCode(registrationCode)
		//move the registration state to keys generated
		err = storageSess.ForwardRegistrationStatus(storage.KeyGenComplete)
	} else {
		//move the registration state to indicate registered with registration
		err = storageSess.ForwardRegistrationStatus(storage.PermissioningComplete)
	}

	//add the request preiamge
	storageSess.GetEdge().Add(edge.Preimage{
		Data:   preimage.GenerateRequest(protoUser.ReceptionID),
		Type:   preimage.Request,
		Source: protoUser.ReceptionID[:],
	}, protoUser.ReceptionID)

	if err != nil {
		return nil, errors.WithMessage(err, "Failed to denote state "+
			"change in session")
	}

	return storageSess, nil
}
