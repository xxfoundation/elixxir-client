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
	"gitlab.com/elixxir/client/keyExchange"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/permissioning"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
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
	storageSess, err := storage.New(storageDir, passwordStr,
		protoUser.UID, protoUser.Salt, protoUser.RSAKey, protoUser.IsPrecanned,
		protoUser.CMixKey, protoUser.E2EKey, cmixGrp, e2eGrp, rngStreamGen)
	if err != nil {
		return nil, err
	}

	// Save NDF to be used in the future
	storageSess.SetBaseNDF(def)

	//store the registration code for later use
	storageSess.SetRegCode(registrationCode)

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
	storageSess, err := storage.New(storageDir, passwordStr,
		protoUser.UID, protoUser.Salt, protoUser.RSAKey, protoUser.IsPrecanned,
		protoUser.CMixKey, protoUser.E2EKey, cmixGrp, e2eGrp, rngStreamGen)
	if err != nil {
		return nil, err
	}

	// Save NDF to be used in the future
	storageSess.SetBaseNDF(def)

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

	// check the client version is up to date to the network
	err = c.checkVersion()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load client")
	}

	//register with permissioning if necessary
	if c.storage.GetRegistrationStatus() == storage.KeyGenComplete {
		err = c.registerWithPermissioning()
		if err != nil {
			return nil, errors.WithMessage(err, "failed to load client")
		}
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
// Threads Started:
//   - Network Follower (/network/follow.go)
//   	tracks the network events and hands them off to workers for handling
//   - Historical Round Retrieval (/network/rounds/historical.go)
//		Retrieves data about rounds which are too old to be stored by the client
//	 - Message Retrieval Worker Group (/network/rounds/retreive.go)
//		Requests all messages in a given round from the gateway of the last node
//	 - Message Handling Worker Group (/network/message/reception.go)
//		Decrypts and partitions messages when signals via the Switchboard
//	 - Health Tracker (/network/health)
func (c *Client) StartNetworkFollower() (stoppable.Stoppable, error) {
	jww.INFO.Printf("StartNetworkFollower()")
	multi := stoppable.NewMulti("client")

	stopFollow, err := c.network.Follow()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to start following "+
			"the network")
	}
	multi.Add(stopFollow)
	// Key exchange
	multi.Add(keyExchange.Start(c.switchboard, c.storage, c.network))
	return multi, nil
}



// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (c *Client) SendE2E(payload []byte, recipient id.ID, msgType int) (
	[]int, error) {
	jww.INFO.Printf("SendE2E(%s, %s, %d)", payload, recipient,
		msgType)
	return nil, nil
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (c *Client) SendUnsafe(payload []byte, recipient id.ID, msgType int) ([]int,
	error) {
	jww.INFO.Printf("SendUnsafe(%s, %s, %d)", payload, recipient,
		msgType)
	return nil, nil
}

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (c *Client) SendCMIX(payload []byte, recipient id.ID) (int, error) {
	jww.INFO.Printf("SendCMIX(%s, %s)", payload, recipient)
	return 0, nil
}

// RegisterListener registers a listener callback function that is called
// every time a new message matches the specified parameters.
func (c *Client) RegisterListenerCb(uid id.ID, msgType int, username string,
	listenerCb func(msg Message)) {
	jww.INFO.Printf("RegisterListener(%s, %d, %s, func())", uid, msgType,
		username)
}

// RegisterForNotifications allows a client to register for push
// notifications.
// Note that clients are not required to register for push notifications
// especially as these rely on third parties (i.e., Firebase *cough*
// *cough* google's palantir *cough*) that may represent a security
// risk to the user.
func (c *Client) RegisterForNotifications(token []byte) error {
	jww.INFO.Printf("RegisterForNotifications(%s)", token)
	// // Pull the host from the manage
	// notificationBotHost, ok := cl.receptionManager.Comms.GetHost(&id.NotificationBot)
	// if !ok {
	// 	return errors.New("Failed to retrieve host for notification bot")
	// }

	// // Send the register message
	// _, err := cl.receptionManager.Comms.RegisterForNotifications(notificationBotHost,
	// 	&mixmessages.NotificationToken{
	// 		Token: notificationToken,
	// 	})
	// if err != nil {
	// 	err := errors.Errorf(
	// 		"RegisterForNotifications: Unable to register for notifications! %s", err)
	// 	return err
	// }

	return nil
}

// UnregisterForNotifications turns of notifications for this client
func (c *Client) UnregisterForNotifications() error {
	jww.INFO.Printf("UnregisterForNotifications()")
	// // Pull the host from the manage
	// notificationBotHost, ok := cl.receptionManager.Comms.GetHost(&id.NotificationBot)
	// if !ok {
	// 	return errors.New("Failed to retrieve host for notification bot")
	// }

	// // Send the unregister message
	// _, err := cl.receptionManager.Comms.UnregisterForNotifications(notificationBotHost)
	// if err != nil {
	// 	err := errors.Errorf(
	// 		"RegisterForNotifications: Unable to register for notifications! %s", err)
	// 	return err
	// }

	return nil
}

// Returns true if the cryptographic identity has been registered with
// the CMIX user discovery agent.
// Note that clients do not need to perform this step if they use
// out of band methods to exchange cryptographic identities
// (e.g., QR codes), but failing to be registered precludes usage
// of the user discovery mechanism (this may be preferred by user).
func (c *Client) IsRegistered() bool {
	jww.INFO.Printf("IsRegistered()")
	return false
}

// RegisterIdentity registers an arbitrary username with the user
// discovery protocol. Returns an error when it cannot connect or
// the username is already registered.
func (c *Client) RegisterIdentity(username string) error {
	jww.INFO.Printf("RegisterIdentity(%s)", username)
	return nil
}

// RegisterEmail makes the users email searchable after confirmation.
// It returns a registration confirmation token to be used with
// ConfirmRegistration or an error on failure.
func (c *Client) RegisterEmail(email string) ([]byte, error) {
	jww.INFO.Printf("RegisterEmail(%s)", email)
	return nil, nil
}

// RegisterPhone makes the users phone searchable after confirmation.
// It returns a registration confirmation token to be used with
// ConfirmRegistration or an error on failure.
func (c *Client) RegisterPhone(phone string) ([]byte, error) {
	jww.INFO.Printf("RegisterPhone(%s)", phone)
	return nil, nil
}

// ConfirmRegistration sends the user discovery agent a confirmation
// token (from register Email/Phone) and code (string sent via Email
// or SMS to confirm ownership) to confirm ownership.
func (c *Client) ConfirmRegistration(token, code []byte) error {
	jww.INFO.Printf("ConfirmRegistration(%s, %s)", token, code)
	return nil
}

// GetUser returns the current user Identity for this client. This
// can be serialized into a byte stream for out-of-band sharing.
func (c *Client) GetUser() (Contact, error) {
	jww.INFO.Printf("GetUser()")
	return Contact{}, nil
}

// MakeContact creates a contact from a byte stream (i.e., unmarshal's a
// Contact object), allowing out-of-band import of identities.
func (c *Client) MakeContact(contactBytes []byte) (Contact, error) {
	jww.INFO.Printf("MakeContact(%s)", contactBytes)
	return Contact{}, nil
}

// GetContact returns a Contact object for the given user id, or
// an error
func (c *Client) GetContact(uid []byte) (Contact, error) {
	jww.INFO.Printf("GetContact(%s)", uid)
	return Contact{}, nil
}

// Search accepts a "separator" separated list of search elements with
// an associated list of searchTypes. It returns a ContactList which
// allows you to iterate over the found contact objects.
func (c *Client) Search(data, separator string, searchTypes []byte) []Contact {
	jww.INFO.Printf("Search(%s, %s, %s)", data, separator, searchTypes)
	return nil
}

// SearchWithHandler is a non-blocking search that also registers
// a callback interface for user disovery events.
func (c *Client) SearchWithCallback(data, separator string, searchTypes []byte,
	cb func(results []Contact)) {
	resultCh := make(chan []Contact, 1)
	go func(out chan []Contact, data, separator string, srchTypes []byte) {
		out <- c.Search(data, separator, srchTypes)
		close(out)
	}(resultCh, data, separator, searchTypes)

	go func(in chan []Contact, cb func(results []Contact)) {
		select {
		case contacts := <-in:
			cb(contacts)
			//TODO: Timer
		}
	}(resultCh, cb)
}

// CreateAuthenticatedChannel creates a 1-way authenticated channel
// so this user can send messages to the desired recipient Contact.
// To receive confirmation from the remote user, clients must
// register a listener to do that.
func (c *Client) CreateAuthenticatedChannel(recipient Contact,
	payload []byte) error {
	jww.INFO.Printf("CreateAuthenticatedChannel(%v, %v)",
		recipient, payload)
	return nil
}

// RegisterAuthConfirmationCb registers a callback for channel
// authentication confirmation events.
func (c *Client) RegisterAuthConfirmationCb(cb func(contact Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthConfirmationCb(...)")
}

// RegisterAuthRequestCb registers a callback for channel
// authentication request events.
func (c *Client) RegisterAuthRequestCb(cb func(contact Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthRequestCb(...)")
}

// RegisterRoundEventsCb registers a callback for round
// events.
func (c *Client) RegisterRoundEventsCb(
	cb func(re *pb.RoundInfo, timedOut bool)) {
	jww.INFO.Printf("RegisterRoundEventsCb(...)")
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
