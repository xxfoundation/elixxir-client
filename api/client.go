///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"encoding/json"
	"math"
	"time"

	"gitlab.com/xx_network/primitives/netTime"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/registration"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/comms/client"
	cryptoBackup "gitlab.com/elixxir/crypto/backup"
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
)

const followerStoppableName = "client"

type Client struct {
	//generic RNG for client
	rng *fastRNG.StreamGenerator
	// the storage session securely stores data to disk and memoizes as is
	// appropriate
	storage storage.Session

	//Low level comms object
	comms *client.Comms

	//facilitates sane communications with cMix
	network cmix.Client

	//object used to register and communicate with permissioning
	permissioning *registration.Registration

	//services system to track running threads
	followerServices   *services
	clientErrorChannel chan interfaces.ClientError

	// Event reporting in event.go
	events *event.Manager
}

// NewClient creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
func NewClient(ndfJSON, storageDir string, password []byte,
	registrationCode string) error {
	jww.INFO.Printf("NewClient(dir: %s)", storageDir)
	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024,
		csprng.NewSystemRNG)

	def, err := parseNDF(ndfJSON)
	if err != nil {
		return err
	}

	cmixGrp, e2eGrp := decodeGroups(def)
	start := netTime.Now()
	protoUser := createNewUser(rngStreamGen)
	jww.DEBUG.Printf("PortableUserInfo generation took: %s",
		netTime.Now().Sub(start))

	_, err = checkVersionAndSetupStorage(def, storageDir, password,
		protoUser, cmixGrp, e2eGrp, registrationCode)
	if err != nil {
		return err
	}

	return nil
}

// NewVanityClient creates a user with a receptionID that starts with
// the supplied prefix It creates client storage, generates keys,
// connects, and registers with the network. Note that this does not
// register a username/identity, but merely creates a new
// cryptographic identity for adding such information at a later date.
func NewVanityClient(ndfJSON, storageDir string, password []byte,
	registrationCode string, userIdPrefix string) error {
	jww.INFO.Printf("NewVanityClient()")

	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024,
		csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()

	def, err := parseNDF(ndfJSON)
	if err != nil {
		return err
	}
	cmixGrp, e2eGrp := decodeGroups(def)

	protoUser := createNewVanityUser(rngStream, cmixGrp, e2eGrp,
		userIdPrefix)

	_, err = checkVersionAndSetupStorage(def, storageDir, password,
		protoUser, cmixGrp, e2eGrp, registrationCode)
	if err != nil {
		return err
	}

	return nil
}

// NewClientFromBackup constructs a new Client from an encrypted
// backup. The backup is decrypted using the backupPassphrase. On
// success a successful client creation, the function will return a
// JSON encoded list of the E2E partners contained in the backup and a
// json-encoded string containing parameters stored in the backup
func NewClientFromBackup(ndfJSON, storageDir string, sessionPassword,
	backupPassphrase []byte, backupFileContents []byte) ([]*id.ID,
	string, error) {

	backUp := &cryptoBackup.Backup{}
	err := backUp.Decrypt(string(backupPassphrase), backupFileContents)
	if err != nil {
		return nil, "", errors.WithMessage(err,
			"Failed to unmarshal decrypted client contents.")
	}

	usr := user.NewUserFromBackup(backUp)

	def, err := parseNDF(ndfJSON)
	if err != nil {
		return nil, "", err
	}

	cmixGrp, e2eGrp := decodeGroups(def)

	// Note we do not need registration here
	storageSess, err := checkVersionAndSetupStorage(def, storageDir,
		[]byte(sessionPassword), usr, cmixGrp, e2eGrp,
		backUp.RegistrationCode)

	storageSess.SetReceptionRegistrationValidationSignature(
		backUp.ReceptionIdentity.RegistrarSignature)
	storageSess.SetTransmissionRegistrationValidationSignature(
		backUp.TransmissionIdentity.RegistrarSignature)
	storageSess.SetRegistrationTimestamp(backUp.RegistrationTimestamp)

	//move the registration state to indicate registered with
	// registration on proto client
	err = storageSess.ForwardRegistrationStatus(
		storage.PermissioningComplete)
	if err != nil {
		return nil, "", err
	}

	return backUp.Contacts.Identities, backUp.JSONParams, nil
}

// OpenClient session, but don't connect to the network or log in
func OpenClient(storageDir string, password []byte,
	parameters Params) (*Client, error) {
	jww.INFO.Printf("OpenClient()")

	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024,
		csprng.NewSystemRNG)

	currentVersion, err := version.ParseVersion(SEMVER)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Could not parse version string.")
	}

	passwordStr := string(password)
	storageSess, err := storage.Load(storageDir, passwordStr,
		currentVersion)
	if err != nil {
		return nil, err
	}

	c := &Client{
		storage:            storageSess,
		rng:                rngStreamGen,
		comms:              nil,
		network:            nil,
		followerServices:   newServices(),
		clientErrorChannel: make(chan interfaces.ClientError, 1000),
		events:             event.NewEventManager(),
	}

	err = c.initComms()
	if err != nil {
		return nil, err
	}

	c.network, err = cmix.NewClient(parameters.CMix, c.comms, c.storage,
		c.rng, c.events)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewProtoClient_Unsafe initializes a client object from a JSON containing
// predefined cryptographic which defines a user. This is designed for some
// specific deployment procedures and is generally unsafe.
func NewProtoClient_Unsafe(ndfJSON, storageDir string, password,
	protoClientJSON []byte) error {
	jww.INFO.Printf("NewProtoClient_Unsafe")

	def, err := parseNDF(ndfJSON)
	if err != nil {
		return err
	}

	cmixGrp, e2eGrp := decodeGroups(def)

	protoUser := &user.Proto{}
	err = json.Unmarshal(protoClientJSON, protoUser)
	if err != nil {
		return err
	}

	usr := user.NewUserFromProto(protoUser)

	storageSess, err := checkVersionAndSetupStorage(def, storageDir,
		password, usr, cmixGrp, e2eGrp, protoUser.RegCode)
	if err != nil {
		return err
	}

	storageSess.SetReceptionRegistrationValidationSignature(
		protoUser.ReceptionRegValidationSig)
	storageSess.SetTransmissionRegistrationValidationSignature(
		protoUser.TransmissionRegValidationSig)
	storageSess.SetRegistrationTimestamp(protoUser.RegistrationTimestamp)

	// move the registration state to indicate registered with
	// registration on proto client
	err = storageSess.ForwardRegistrationStatus(
		storage.PermissioningComplete)
	if err != nil {
		return err
	}

	return nil
}

// Login initializes a client object from existing storage.
func Login(storageDir string, password []byte,
	authCallbacks auth.Callbacks, parameters Params) (*Client, error) {
	jww.INFO.Printf("Login()")

	c, err := OpenClient(storageDir, password, parameters)
	if err != nil {
		return nil, err
	}

	jww.INFO.Printf("Client Logged in: \n\tTransmissionID: %s "+
		"\n\tReceptionID: %s", c.storage.GetTransmissionID(), c.storage.GetReceptionID())

	def := c.storage.GetNDF()

	//initialize registration
	if def.Registration.Address != "" {
		err = c.initPermissioning(def)
		if err != nil {
			return nil, err
		}
	} else {
		jww.WARN.Printf("Registration with permissioning skipped due " +
			"to blank permissioning address. Client will not be " +
			"able to register or track network.")
	}

	if def.Notification.Address != "" {
		hp := connect.GetDefaultHostParams()
		// Client will not send KeepAlive packets
		hp.KaClientOpts.Time = time.Duration(math.MaxInt64)
		hp.AuthEnabled = false
		hp.MaxRetries = 5
		_, err = c.comms.AddHost(&id.NotificationBot,
			def.Notification.Address,
			[]byte(def.Notification.TlsCertificate), hp)
		if err != nil {
			jww.WARN.Printf("Failed adding host for "+
				"notifications: %+v", err)
		}
	}

	err = c.network.Connect(def)
	if err != nil {
		return nil, err
	}

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
	newBaseNdf string, authCallbacks auth.Callbacks,
	params Params) (*Client, error) {
	jww.INFO.Printf("LoginWithNewBaseNDF_UNSAFE()")

	def, err := parseNDF(newBaseNdf)
	if err != nil {
		return nil, err
	}

	c, err := OpenClient(storageDir, password, params)
	if err != nil {
		return nil, err
	}

	//store the updated base NDF
	c.storage.SetNDF(def)

	if def.Registration.Address != "" {
		err = c.initPermissioning(def)
		if err != nil {
			return nil, err
		}
	} else {
		jww.WARN.Printf("Registration with permissioning skipped due " +
			"to blank permissionign address. Client will not be " +
			"able to register or track network.")
	}

	err = c.network.Connect(def)
	if err != nil {
		return nil, err
	}

	err = c.registerFollower()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// LoginWithProtoClient creates a client object with a protoclient
// JSON containing the cryptographic primitives. This is designed for
// some specific deployment procedures and is generally unsafe.
func LoginWithProtoClient(storageDir string, password []byte,
	protoClientJSON []byte, newBaseNdf string,
	params Params) (*Client, error) {
	jww.INFO.Printf("LoginWithProtoClient()")

	def, err := parseNDF(newBaseNdf)
	if err != nil {
		return nil, err
	}

	err = NewProtoClient_Unsafe(newBaseNdf, storageDir, password,
		protoClientJSON)
	if err != nil {
		return nil, err
	}

	c, err := OpenClient(storageDir, password, params)
	if err != nil {
		return nil, err
	}

	c.storage.SetNDF(def)

	err = c.initPermissioning(def)
	if err != nil {
		return nil, err
	}

	err = c.network.Connect(def)
	if err != nil {
		return nil, err
	}

	// FIXME: The callbacks need to be set, so I suppose we would need to
	//        either set them via a special type or add them
	//        to the login call?
	if err != nil {
		return nil, err
	}
	err = c.registerFollower()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) initComms() error {
	var err error

	//get the user from session
	privKey := c.storage.GetTransmissionRSA()
	pubPEM := rsa.CreatePublicKeyPem(privKey.GetPublic())
	privPEM := rsa.CreatePrivateKeyPem(privKey)

	//start comms
	c.comms, err = client.NewClientComms(c.storage.GetTransmissionID(),
		pubPEM, privPEM, c.storage.GetTransmissionSalt())
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
		jww.INFO.Printf("Client has not registered yet, " +
			"attempting registration")
		err = c.registerWithPermissioning()
		if err != nil {
			jww.ERROR.Printf("Client has failed registration: %s",
				err)
			return errors.WithMessage(err, "failed to load client")
		}
		jww.INFO.Printf("Client successfully registered " +
			"with the network")
	}
	return nil
}

// registerFollower adds the follower processes to the client's
// follower service list.
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
			jww.WARN.Printf("Failed to notify about ClientError "+
				"from %s: %s", source, message)
		}
	}

	err := c.followerServices.add(c.events.EventService)
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

	return nil
}

// ----- Client Functions -----

// GetErrorsChannel returns a channel which passess errors from the
// long running threads controlled by StartNetworkFollower and
// StopNetworkFollower
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
// 		Retrieves data about rounds which are too old to be
// 		stored by the client
//	 - Message Retrieval Worker Group (/network/rounds/retrieve.go)
//		Requests all messages in a given round from the
//		gateway of the last nodes
//	 - Message Handling Worker Group (/network/message/handle.go)
//		Decrypts and partitions messages when signals via the
//		Switchboard
//	 - health Tracker (/network/health)
//		Via the network instance tracks the state of the network
//	 - Garbled Messages (/network/message/garbled.go)
//		Can be signaled to check all recent messages which
//		could be be decoded Uses a message store on disk for
//		persistence
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
	jww.INFO.Printf("StartNetworkFollower() \n\tTransmissionID: %s "+
		"\n\tReceptionID: %s", c.storage.GetTransmissionID(), c.storage.GetReceptionID())

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

// RegisterRoundEventsCb registers a callback for round
// events.
func (c *Client) GetRoundEvents() interfaces.RoundEvents {
	jww.INFO.Printf("GetRoundEvents()")
	jww.WARN.Printf("GetRoundEvents does not handle Client Errors " +
		"edge case!")
	return c.network.GetInstance().GetRoundEvents()
}

// AddService adds a service ot be controlled by the client thread control,
// these will be started and stopped with the network follower
func (c *Client) AddService(sp Service) error {
	return c.followerServices.add(sp)
}

// GetUser returns the current user Identity for this client. This
// can be serialized into a byte stream for out-of-band sharing.
func (c *Client) GetUser() user.Info {
	jww.INFO.Printf("GetUser()")
	cMixUser := c.storage.PortableUserInfo()
	return cMixUser
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
func (c *Client) GetStorage() storage.Session {
	return c.storage
}

// GetCmix returns the client Network Interface
func (c *Client) GetCmix() cmix.Client {
	return c.network
}

// GetEventReporter returns the event reporter
func (c *Client) GetEventReporter() event.Reporter {
	return c.events
}

// GetNodeRegistrationStatus gets the current state of nodes registration. It
// returns the total number of nodes in the NDF and the number of those which
// are currently registers with. An error is returned if the network is not
// healthy.
func (c *Client) GetNodeRegistrationStatus() (int, int, error) {
	// Return an error if the network is not healthy
	if !c.GetCmix().IsHealthy() {
		return 0, 0, errors.New("Cannot get number of nodes " +
			"registrations when network is not healthy")
	}

	nodes := c.network.GetInstance().GetPartialNdf().Get().Nodes

	var numRegistered int
	var numStale = 0
	for i, n := range nodes {
		nid, err := id.Unmarshal(n.ID)
		if err != nil {
			return 0, 0, errors.Errorf("Failed to unmarshal nodes "+
				"ID %v (#%d): %s", n.ID, i, err.Error())
		}
		if n.Status == ndf.Stale {
			numStale += 1
			continue
		}
		if c.network.HasNode(nid) {
			numRegistered++
		}
	}

	// get the number of in progress nodes registrations
	return numRegistered, len(nodes) - numStale, nil
}

// GetPreferredBins returns the geographic bin or bins that the provided two
// character country code is a part of.
func (c *Client) GetPreferredBins(countryCode string) ([]string, error) {
	// get the bin that the country is in
	bin, exists := region.GetCountryBin(countryCode)
	if !exists {
		return nil, errors.Errorf("failed to find geographic bin "+
			"for country %q", countryCode)
	}

	// Add bin to list of geographic bins
	bins := []string{bin.String()}

	// Add additional bins in special cases
	switch bin {
	case region.SouthAndCentralAmerica:
		bins = append(bins, region.NorthAmerica.String())
	case region.MiddleEast:
		bins = append(bins, region.EasternEurope.String(),
			region.CentralEurope.String(),
			region.WesternAsia.String())
	case region.NorthernAfrica:
		bins = append(bins, region.WesternEurope.String(),
			region.CentralEurope.String())
	case region.SouthernAfrica:
		bins = append(bins, region.WesternEurope.String(),
			region.CentralEurope.String())
	case region.EasternAsia:
		bins = append(bins, region.WesternAsia.String(),
			region.Oceania.String(), region.NorthAmerica.String())
	case region.WesternAsia:
		bins = append(bins, region.EasternAsia.String(),
			region.Russia.String(), region.MiddleEast.String())
	case region.Oceania:
		bins = append(bins, region.EasternAsia.String(),
			region.NorthAmerica.String())
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

// checkVersionAndSetupStorage is common code shared by NewClient,
// NewPrecannedClient and NewVanityClient it checks client version and
// creates a new storage for user data
func checkVersionAndSetupStorage(def *ndf.NetworkDefinition,
	storageDir string, password []byte, protoUser user.Info,
	cmixGrp, e2eGrp *cyclic.Group, registrationCode string) (
	storage.Session, error) {
	// get current client version
	currentVersion, err := version.ParseVersion(SEMVER)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Could not parse version string.")
	}

	// Create Storage
	passwordStr := string(password)
	storageSess, err := storage.New(storageDir, passwordStr, protoUser,
		currentVersion, cmixGrp, e2eGrp)
	if err != nil {
		return nil, err
	}

	// Save NDF to be used in the future
	storageSess.SetNDF(def)

	//store the registration code for later use
	storageSess.SetRegCode(registrationCode)
	//move the registration state to keys generated
	err = storageSess.ForwardRegistrationStatus(storage.KeyGenComplete)

	if err != nil {
		return nil, errors.WithMessage(err, "Failed to denote state "+
			"change in session")
	}

	// create new E2E
	err = e2e.Init(storageSess.GetKV(), protoUser.ReceptionID,
		protoUser.E2eDhPrivateKey, e2eGrp, rekey.GetDefaultParams())
	if err != nil {
		return nil, err
	}

	return storageSess, nil
}
