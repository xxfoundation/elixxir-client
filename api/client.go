////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/xx_network/primitives/ndf"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

type Client struct {
	storage     *storage.Session
	ctx         *context.Context
	switchboard *switchboard.Switchboard
	network     *network.Network
}

// NewClient creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
func NewClient(network, storageDir string, password []byte) (Client, error) {
	if clientStorageExists(storageDir) {
		return errors.New("client already exists at %s",
			storageDir)
	}

	// Parse the NDF
	ndf, err := parseNDF(network)
	if err != nil {
		return nil, err
	}

	// Create Storage

	// Create network, context, switchboard

	// Generate Keys

	// Register with network

	client = Client{
		storage:     nil,
		ctx:         nil,
		switchboard: nil,
		network:     nil,
	}
	return client, nil
}

// LoadClient initalizes a client object from existing storage.
func LoadClient(storageDir string, password []byte) (Client, error) {
	if !clientStorageExists(storageDir) {
		return errors.New("client does not exist at %s",
			storageDir)
	}

	// Load Storage

	// Load and create network, context, switchboard

	client = Client{
		storage:     nil,
		ctx:         nil,
		switchboard: nil,
		network:     nil,
	}
	return client, nil
}

// ----- Client Functions -----

// RegisterListener registers a listener callback function that is called
// every time a new message matches the specified parameters.
func (c Client) RegisterListener(uid id.ID, msgType int, username string,
	listenerCb func(msg Message)) {
	jww.INFO.Printf("RegisterListener(%s, %d, %s, %v)", uid, msgType,
		username, listenerCb)
}

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (c Client) SendE2E(payload []byte, recipient id.ID, msgType int) (
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
func (c Client) SendUnsafe(payload []byte, recipient id.ID, msgType int) ([]int,
	error) {
	jww.INFO.Printf("SendUnsafe(%s, %s, %d)", payload, recipient,
		msgType)
	return nil, nil
}

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (c Client) SendCMIX(payload []byte, recipient id.ID) (int, error) {
	jww.INFO.Printf("SendCMIX(%s, %s)", payload, recipient,
		msgType)
	return 0, nil
}

// RegisterForNotifications allows a client to register for push
// notifications.
// Note that clients are not required to register for push notifications
// especially as these rely on third parties (i.e., Firebase *cough*
// *cough* google's palantir *cough*) that may represent a security
// risk to the user.
func (c Client) RegisterForNotifications(token []byte) error {
	jww.INFO.Printf("RegisterForNotifications(%s)", token)
	return nil
}

// UnregisterForNotifications turns of notifications for this client
func (c Client) UnregisterForNotifications() error {
	jww.INFO.Printf("UnregisterForNotifications()")
	return nil
}

// Returns true if the cryptographic identity has been registered with
// the CMIX user discovery agent.
// Note that clients do not need to perform this step if they use
// out of band methods to exchange cryptographic identities
// (e.g., QR codes), but failing to be registered precludes usage
// of the user discovery mechanism (this may be preferred by user).
func (c Client) IsRegistered() bool {
	jww.INFO.Printf("IsRegistered(%s, %s, %d)", payload, recipient,
		msgType)
	return false
}

// RegisterIdentity registers an arbitrary username with the user
// discovery protocol. Returns an error when it cannot connect or
// the username is already registered.
func (c Client) RegisterIdentity(username string) error {
	jww.INFO.Printf("RegisterIdentity(%s)", username)
	return nil
}

// RegisterEmail makes the users email searchable after confirmation.
// It returns a registration confirmation token to be used with
// ConfirmRegistration or an error on failure.
func (c Client) RegisterEmail(email string) ([]byte, error) {
	jww.INFO.Printf("RegisterEmail(%s)", email)
	return nil, nil
}

// RegisterPhone makes the users phone searchable after confirmation.
// It returns a registration confirmation token to be used with
// ConfirmRegistration or an error on failure.
func (c Client) RegisterPhone(phone string) ([]byte, error) {
	jww.INFO.Printf("RegisterPhone(%s)", phone)
	return nil, nil
}

// ConfirmRegistration sends the user discovery agent a confirmation
// token (from Register Email/Phone) and code (string sent via Email
// or SMS to confirm ownership) to confirm ownership.
func (c Client) ConfirmRegistration(token, code []byte) error {
	jww.INFO.Printf("ConfirmRegistration(%s, %s)", token, code)
	return nil
}

// GetUser returns the current user Identity for this client. This
// can be serialized into a byte stream for out-of-band sharing.
func (c Client) GetUser() (Contact, error) {
	jww.INFO.Printf("GetUser()")
	return Contact{}, nil
}

// MakeContact creates a contact from a byte stream (i.e., unmarshal's a
// Contact object), allowing out-of-band import of identities.
func (c Client) MakeContact(contactBytes []byte) (Contact, error) {
	jww.INFO.Printf("MakeContact(%s)", contactBytes)
	return Contact{}, nil
}

// GetContact returns a Contact object for the given user id, or
// an error
func (c Client) GetContact(uid []byte) (Contact, error) {
	jww.INFO.Printf("GetContact(%s)", uid)
	return Contact{}, nil
}

// Search accepts a "separator" separated list of search elements with
// an associated list of searchTypes. It returns a ContactList which
// allows you to iterate over the found contact objects.
func (c Client) Search(data, separator string, searchTypes []byte) []Contact {
	jww.INFO.Printf("Search(%s, %s, %s)", data, separator, searchTypes)
	return nil
}

// SearchWithHandler is a non-blocking search that also registers
// a callback interface for user disovery events.
func (c Client) SearchWithCallback(data, separator string, searchTypes []byte,
	cb func(results []Contact)) {
	resultCh := make(chan []Contact, 1)
	go func(out chan []Contact, data, separator string, srchTypes []byte) {
		out <- c.Search(data, separator, srchTypes)
		out.Close()
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
func (c Client) CreateAuthenticatedChannel(recipient Contact,
	payload []byte) error {
	jww.INFO.Printf("CreateAuthenticatedChannel(%s, %s)",
		recipient, payload)
	return nil
}

// RegisterAuthConfirmationCb registers a callback for channel
// authentication confirmation events.
func (c Client) RegisterAuthConfirmationCb(cb func(contact Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthConfirmationCb(...)")
}

// RegisterAuthRequestCb registers a callback for channel
// authentication request events.
func (c Client) RegisterAuthRequestCb(cb func(contact Contact,
	payload []byte)) {
	jww.INFO.Printf("RegisterAuthRequestCb(...)")
}

// StartNetworkRunner kicks off the longrunning network client threads
// and returns an object for checking state and stopping those threads.
// Call this when returning from sleep and close when going back to
// sleep.
func (c Client) StartNetworkRunner() NetworkRunner {
	jww.INFO.Printf("StartNetworkRunner()")
}

// RegisterRoundEventsCb registers a callback for round
// events.
func (c Client) RegisterRoundEventsCb(cb func(re RoundEvent)) {
	jww.INFO.Printf("RegisterRoundEventsCb(...)")
}

// ----- Utility Functions -----

// clientStorageExists returns true if an EKV (storage.Session) exists in the
// given location or not.
func clientStorageExists(storageDir string) bool {
	// Check if diretory exists.

	// If directory exists, check if either .ekv.1 or .ekv.2 files exist in
	// the directory.

	return false
}

// parseNDF parses the initial ndf string for the client. This includes a
// network public key that is also used to verify integrity of the ndf.
func parseNDF(ndfString string) (*ndf.NetworkDefinition, error) {
	if ndfString == "" {
		return nil, errors.New("ndf file empty")
	}

	ndfReader := bufio.NewReader(strings.NewReader(ndfString))

	// ndfData is the json string defining the ndf
	ndfData, err := ndfReader.ReadBytes('\n')
	ndfData = ndfData[:len(ndfData)-1]
	if err != nil {
		return nil, err
	}

	// ndfSignature is the second line of the file, used to verify
	// integrity.
	ndfSignature, err := ndfReader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	ndfSignature, err = base64.StdEncoding.DecodeString(
		string(ndfSignature[:len(ndfSignature)-1]))
	if err != nil {
		return nil, err
	}

	ndf, _, err := ndf.DecodeNDF(ndfString)
	if err != nil {
		return nil, err
	}

	// Load the TLS cert given to us, and from that get the RSA public key
	cert, err := tls.LoadCertificate(ndf.NdfPub)
	if err != nil {
		return nil, err
	}
	pubKey := &rsa.PublicKey{PublicKey: *cert.PublicKey.(*gorsa.PublicKey)}

	// Hash NDF JSON
	rsaHash := sha256.New()
	rsaHash.Write(ndfData)

	// Verify signature
	err = rsa.Verify(
		pubKey, crypto.SHA256, rsaHash.Sum(nil), ndfSignature, nil)
	if err != nil {
		return nil, err
	}

	return ndf, nil
}
