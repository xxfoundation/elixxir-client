////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"bufio"
	"crypto"
	"crypto/rand"
	gorsa "crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/rekey"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/switchboard"
	goio "io"
	"strings"
	"sync"
	"time"
)

type Client struct {
	storage     globals.Storage
	session     user.Session
	commManager *io.CommManager
	ndf         *ndf.NetworkDefinition
	topology    *circuit.Circuit
	opStatus    OperationProgressCallback
	rekeyChan   chan struct{}
}

var noNDFErr = errors.New("Failed to get ndf from permissioning: rpc error: code = Unknown desc = Permissioning server does not have an ndf to give to client")

//used to report the state of registration
type OperationProgressCallback func(int)

// Populates a text message and returns its wire representation
// TODO support multi-type messages or telling if a message is too long?
func FormatTextMessage(message string) []byte {
	textMessage := cmixproto.TextMessage{
		Color:   -1,
		Message: message,
		Time:    time.Now().Unix(),
	}

	wireRepresentation, _ := proto.Marshal(&textMessage)
	return wireRepresentation
}

// VerifyNDF verifies the signature of the network definition file (NDF) and
// returns the structure. Panics when the NDF string cannot be decoded and when
// the signature cannot be verified. If the NDF public key is empty, then the
// signature verification is skipped and warning is printed.
func VerifyNDF(ndfString, ndfPub string) *ndf.NetworkDefinition {
	// If there is no public key, then skip verification and print warning
	if ndfPub == "" {
		globals.Log.WARN.Printf("Running without signed network " +
			"definition file")
	} else {
		ndfReader := bufio.NewReader(strings.NewReader(ndfString))
		ndfData, err := ndfReader.ReadBytes('\n')
		ndfData = ndfData[:len(ndfData)-1]
		if err != nil {
			globals.Log.FATAL.Panicf("Could not read NDF: %v", err)
		}
		ndfSignature, err := ndfReader.ReadBytes('\n')
		if err != nil {
			globals.Log.FATAL.Panicf("Could not read NDF Sig: %v",
				err)
		}
		ndfSignature, err = base64.StdEncoding.DecodeString(
			string(ndfSignature[:len(ndfSignature)-1]))
		if err != nil {
			globals.Log.FATAL.Panicf("Could not read NDF Sig: %v",
				err)
		}
		// Load the TLS cert given to us, and from that get the RSA public key
		cert, err := tls.LoadCertificate(ndfPub)
		if err != nil {
			globals.Log.FATAL.Panicf("Could not load public key: %v", err)
		}
		pubKey := &rsa.PublicKey{PublicKey: *cert.PublicKey.(*gorsa.PublicKey)}

		// Hash NDF JSON
		rsaHash := sha256.New()
		rsaHash.Write(ndfData)

		globals.Log.INFO.Printf("%s \n::\n %s",
			ndfSignature, ndfData)

		// Verify signature
		err = rsa.Verify(
			pubKey, crypto.SHA256, rsaHash.Sum(nil), ndfSignature, nil)

		if err != nil {
			globals.Log.FATAL.Panicf("Could not verify NDF: %v", err)
		}
	}

	ndfJSON, _, err := ndf.DecodeNDF(ndfString)
	if err != nil {
		globals.Log.FATAL.Panicf("Could not decode NDF: %v", err)
	}
	return ndfJSON
}

//request calls getUpdatedNDF for a new NDF repeatedly until it gets an NDF
func requestNdf(cl *Client) error {
	// Continuously polls for a new ndf after sleeping until response if gotten
	globals.Log.INFO.Printf("Polling for a new NDF")
	newNDf, err := cl.commManager.GetUpdatedNDF(cl.ndf)

	if err != nil {
		//lets the client continue when permissioning does not provide NDFs
		if err.Error() == noNDFErr.Error() {
			globals.Log.WARN.Println("Continuing without an updated NDF")
			return nil
		}

		errMsg := fmt.Sprintf("Failed to get updated ndf: %v", err)
		globals.Log.ERROR.Printf(errMsg)
		return errors.New(errMsg)
	}

	cl.ndf = newNDf

	cl.commManager.UpdateNDF(newNDf)

	return nil
}

// Creates a new Client using the storage mechanism provided.
// If none is provided, a default storage using OS file access
// is created
// returns a new Client object, and an error if it fails
func NewClient(s globals.Storage, loc string, ndfJSON *ndf.NetworkDefinition,
	callback io.ConnectionStatusCallback) (*Client, error) {
	var store globals.Storage
	if s == nil {
		globals.Log.INFO.Printf("No storage provided," +
			" initializing Client with default storage")
		store = &globals.DefaultStorage{}
	} else {
		store = s
	}

	err := store.SetLocation(loc)

	if err != nil {
		err = errors.New("Invalid Local Storage Location: " + err.Error())
		globals.Log.ERROR.Printf(err.Error())
		return nil, err
	}

	cl := new(Client)
	cl.storage = store
	cl.commManager = io.NewCommManager(ndfJSON, callback)
	cl.ndf = ndfJSON
	//build the topology
	nodeIDs := make([]*id.Node, len(cl.ndf.Nodes))
	for i, node := range cl.ndf.Nodes {
		nodeIDs[i] = id.NewNodeFromBytes(node.ID)
	}

	//Create the cmix group and init the registry
	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString(cl.ndf.CMIX.Prime, 16),
		large.NewIntFromString(cl.ndf.CMIX.Generator, 16))
	user.InitUserRegistry(cmixGrp)

	cl.opStatus = func(int) {
		return
	}

	cl.rekeyChan = make(chan struct{}, 1)

	return cl, nil
}

// DisableTLS makes the client run with TLS disabled
// Must be called before Connect
func (cl *Client) DisableTLS() {
	globals.Log.INFO.Println("Running client without tls")
	cl.commManager.DisableTLS()
}

//GetNDF returns the clients ndf
func (cl *Client) GetNDF() *ndf.NetworkDefinition {
	return cl.ndf
}

// Checks version and connects to gateways using TLS filepaths to create
// credential information for connection establishment
func (cl *Client) Connect() error {
	//Connect to permissioning
	if cl.ndf.Registration.Address != "" {
		isConnected, err := cl.commManager.ConnectToPermissioning()
		defer cl.commManager.DisconnectFromPermissioning()

		if err != nil {
			return err
		}
		if !isConnected {
			err = errors.New("Couldn't connect to permissioning")
			return err
		}
		//Check if versioning is up to date
		err = cl.commManager.UpdateRemoteVersion()
		if err != nil {
			return err
		}

		//Request a new ndf from
		err = requestNdf(cl)
		if err != nil {
			return err

		}
	} else {
		globals.Log.WARN.Println("Registration not defined, not contacted")
	}

	//build the topology
	nodeIDs := make([]*id.Node, len(cl.ndf.Nodes))
	for i, node := range cl.ndf.Nodes {
		nodeIDs[i] = id.NewNodeFromBytes(node.ID)
	}

	cl.topology = circuit.New(nodeIDs)

	// Only check the version if we got a remote version
	// The remote version won't have been populated if we didn't connect to permissioning
	if cl.commManager.GetRegistrationVersion() != "" {
		ok, err := cl.commManager.CheckVersion()
		if err != nil {
			return err
		}
		if !ok {
			err = errors.New(fmt.Sprintf("Couldn't connect to gateways: Versions incompatible; Local version: %v; remote version: %v", globals.SEMVER,
				cl.commManager.GetRegistrationVersion()))
			return err
		}
	} else {
		globals.Log.WARN.Printf("Not checking version from " +
			"registration server, because it's not populated. Do you have " +
			"access to the registration server?")
	}
	return cl.commManager.ConnectToGateways()
}

func (cl *Client) SetOperationProgressCallback(rpc OperationProgressCallback) {
	cl.opStatus = func(i int) { go rpc(i) }
}

//registerWithNode registers a user. It serves as a helper for Register
func (cl *Client) registerWithNode(index int, salt, registrationValidationSignature []byte, UID *id.User,
	publicKeyRSA *rsa.PublicKey, privateKeyRSA *rsa.PrivateKey,
	cmixPublicKeyDH, cmixPrivateKeyDH *cyclic.Int,
	cmixGrp *cyclic.Group, nodeKey map[id.Node]user.NodeKeys, errorChan chan error) {

	gatewayID := id.NewNodeFromBytes(cl.ndf.Nodes[index].ID).NewGateway()

	// Initialise blake2b hash for transmission keys and sha256 for reception
	// keys
	transmissionHash, _ := hash.NewCMixHash()
	receptionHash := sha256.New()

	// Request nonce message from gateway
	globals.Log.INFO.Printf("Register: Requesting nonce from gateway %v/%v",
		index, len(cl.ndf.Gateways))
	nonce, dhPub, err := cl.requestNonce(salt, registrationValidationSignature, cmixPublicKeyDH,
		publicKeyRSA, privateKeyRSA, gatewayID)

	if err != nil {
		errMsg := fmt.Sprintf("Register: Failed requesting nonce from gateway: %+v", err)
		globals.Log.ERROR.Printf(errMsg)
		errorChan <- errors.New(errMsg)
	}

	// Load server DH pubkey
	serverPubDH := cmixGrp.NewIntFromBytes(dhPub)

	// Confirm received nonce
	globals.Log.INFO.Println("Register: Confirming received nonce")
	err = cl.confirmNonce(UID.Bytes(), nonce, privateKeyRSA, gatewayID)
	if err != nil {
		errMsg := fmt.Sprintf("Register: Unable to confirm nonce: %v", err)
		globals.Log.ERROR.Printf(errMsg)
		errorChan <- errors.New(errMsg)
	} else {
	}
	nodeID := *cl.topology.GetNodeAtIndex(index)
	nodeKey[nodeID] = user.NodeKeys{
		TransmissionKey: registration.GenerateBaseKey(cmixGrp,
			serverPubDH, cmixPrivateKeyDH, transmissionHash),
		ReceptionKey: registration.GenerateBaseKey(cmixGrp, serverPubDH,
			cmixPrivateKeyDH, receptionHash),
	}
}

const SaltSize = 256

// Registers user and returns the User ID.
// Returns an error if registration fails.
func (cl *Client) Register(preCan bool, registrationCode, nick, email,
	password string, privateKeyRSA *rsa.PrivateKey) (*id.User, error) {

	if !preCan && cl.commManager.GetConnectionStatus() != io.Online {
		return nil, errors.New("Cannot register when disconnected from the network")
	}

	var err error
	var u *user.User
	var UID *id.User

	cl.opStatus(globals.REG_KEYGEN)

	largeIntBits := 16

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString(cl.ndf.CMIX.Prime, largeIntBits),
		large.NewIntFromString(cl.ndf.CMIX.Generator, largeIntBits))

	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString(cl.ndf.E2E.Prime, largeIntBits),
		large.NewIntFromString(cl.ndf.E2E.Generator, largeIntBits))

	// Make CMIX keys array
	nk := make(map[id.Node]user.NodeKeys)

	// GENERATE CLIENT RSA KEYS
	if privateKeyRSA == nil {
		privateKeyRSA, err = rsa.GenerateKey(rand.Reader, rsa.DefaultRSABitLen)
		if err != nil {
			return nil, err
		}
	}

	publicKeyRSA := privateKeyRSA.GetPublic()

	cmixPrivKeyDHByte, err := csprng.GenerateInGroup(cmixGrp.GetPBytes(), 256, csprng.NewSystemRNG())

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not generate cmix DH private key: %s", err.Error()))
	}

	cmixPrivateKeyDH := cmixGrp.NewIntFromBytes(cmixPrivKeyDHByte)
	cmixPublicKeyDH := cmixGrp.ExpG(cmixPrivateKeyDH, cmixGrp.NewMaxInt())

	e2ePrivKeyDHByte, err := csprng.GenerateInGroup(cmixGrp.GetPBytes(), 256, csprng.NewSystemRNG())

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not generate e2e DH private key: %s", err.Error()))
	}

	e2ePrivateKeyDH := e2eGrp.NewIntFromBytes(e2ePrivKeyDHByte)
	e2ePublicKeyDH := e2eGrp.ExpG(e2ePrivateKeyDH, e2eGrp.NewMaxInt())

	// Initialized response from Registration Server
	regValidationSignature := make([]byte, 0)

	var salt []byte

	// Handle precanned registration
	if preCan {
		cl.opStatus(globals.REG_PRECAN)
		globals.Log.INFO.Printf("Registering precanned user...")
		u, UID, nk, err = cl.precannedRegister(registrationCode, nick, nk)
		if err != nil {
			globals.Log.ERROR.Printf("Unable to complete precanned registration: %+v", err)
			return id.ZeroID, err
		}
	} else {
		cl.opStatus(globals.REG_UID_GEN)
		globals.Log.INFO.Printf("Registering dynamic user...")

		// Generate salt for UserID
		salt = make([]byte, SaltSize)
		_, err = csprng.NewSystemRNG().Read(salt)
		if err != nil {
			globals.Log.ERROR.Printf("Register: Unable to generate salt! %s", err)
			return id.ZeroID, err
		}

		// Generate UserID by hashing salt and public key
		UID = registration.GenUserID(publicKeyRSA, salt)

		// If Registration Server is specified, contact it
		// Only if registrationCode is set
		globals.Log.INFO.Println("Register: Contacting registration server")
		if cl.ndf.Registration.Address != "" && registrationCode != "" {
			cl.opStatus(globals.REG_PERM)
			regValidationSignature, err = cl.sendRegistrationMessage(registrationCode, publicKeyRSA)
			if err != nil {
				globals.Log.ERROR.Printf("Register: Unable to send registration message: %+v", err)
				return id.ZeroID, err
			}
		}
		globals.Log.INFO.Println("Register: successfully passed Registration message")

		cl.opStatus(globals.REG_NODE)

		var wg sync.WaitGroup
		errChan := make(chan error, len(cl.ndf.Gateways))

		// Loop over all Servers
		globals.Log.INFO.Println("Register: Requesting nonces")
		for i := range cl.ndf.Gateways {
			// Multithread registration for better performance
			wg.Add(1)
			go func() {
				//Register the client over all servers
				cl.registerWithNode(i, salt, regValidationSignature, UID, publicKeyRSA, privateKeyRSA,
					cmixPublicKeyDH, cmixPrivateKeyDH, cmixGrp, nk, errChan)

				wg.Done()
			}()

			wg.Wait()
			//See if the registration returned errors at all
			var errs error
			for len(errChan) > 0 {
				err = <-errChan
				if errs != nil {
					errs = errors.Wrap(errs, err.Error())
				} else {
					errs = err
				}

			}
			//If an error every occured, return with error
			if errs != nil {
				cl.opStatus(globals.REG_FAIL)
				return id.ZeroID, errs
			}

		}

		var actualNick string
		if nick != "" {
			actualNick = nick
		} else {
			actualNick = base64.StdEncoding.EncodeToString(UID[:])
		}
		u = user.Users.NewUser(UID, actualNick)
		user.Users.UpsertUser(u)
	}

	cl.opStatus(globals.REG_SECURE_STORE)

	u.Email = email

	// Create the user session
	newSession := user.NewSession(cl.storage, u, nk, publicKeyRSA,
		privateKeyRSA, cmixPublicKeyDH, cmixPrivateKeyDH, e2ePublicKeyDH,
		e2ePrivateKeyDH, salt, cmixGrp, e2eGrp, password, regValidationSignature)
	cl.opStatus(globals.REG_SAVE)

	// Store the user session
	errStore := newSession.StoreSession()

	// FIXME If we have an error here, the session that gets created
	// doesn't get immolated. Immolation should happen in a deferred
	//  call instead.
	if errStore != nil {
		err = errors.New(fmt.Sprintf(
			"Register: could not register due to failed session save"+
				": %s", errStore.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return id.ZeroID, err
	}

	err = newSession.Immolate()
	if err != nil {
		globals.Log.ERROR.Printf("Error on immolate: %+v", err)
	}
	newSession = nil

	return UID, nil
}

// RegisterWithUDB uses the account's email to register with the UDB for
// User discovery.  Must be called after Register and Connect.
// It will fail if the user has already registered with UDB
func (cl *Client) RegisterWithUDB(timeout time.Duration) error {
	status := cl.commManager.GetConnectionStatus()
	if status == io.Connecting || status == io.Offline {
		return errors.New("ERROR: could not RegisterWithUDB - connection is either offline or connecting")
	}

	email := cl.session.GetCurrentUser().Email

	var err error

	if email != "" {
		globals.Log.INFO.Printf("Registering user as %s with UDB", email)

		valueType := "EMAIL"

		publicKeyBytes := cl.session.GetE2EDHPublicKey().Bytes()
		err = bots.Register(valueType, email, publicKeyBytes, cl.opStatus, timeout)
		if err == nil {
			globals.Log.INFO.Printf("Registered with UDB!")
		} else {
			globals.Log.WARN.Printf("Could not register with UDB: %s", err)
		}

	} else {
		globals.Log.INFO.Printf("Not registering with UDB because no " +
			"email found")
	}
	return err
}

// LoadSession loads the session object for the UID
func (cl *Client) Login(password string) (string, error) {
	session, err := user.LoadSession(cl.storage, password)

	if err != nil {
		err = errors.Wrap(err, "Login: Could not login")
		globals.Log.ERROR.Printf(err.Error())
		return "", err
	}

	if session == nil {
		return "", errors.New("Unable to load session, no error reported")
	}

	//Load Cmix keys & group
	cmixDHPrivKey := session.GetCMIXDHPrivateKey()
	cmixDHPubKey := session.GetCMIXDHPublicKey()
	cmixGrp := session.GetCmixGroup()

	//Load the rsa keys
	rsaPubKey := session.GetRSAPublicKey()
	rsaPrivKey := session.GetRSAPrivateKey()

	//Load the user ID
	UID := session.GetCurrentUser().User

	//Load the registration signature
	regSignature := session.GetRegistrationValidationSignature()

	// Make CMIX keys array
	nk := make(map[id.Node]user.NodeKeys)

	var wg sync.WaitGroup
	errChan := make(chan error, len(cl.ndf.Gateways))

	//Get the registered node keys
	registedNodes := session.GetNodes()

	salt := session.GetSalt()

	for i := range cl.ndf.Gateways {
		nodeID := *id.NewNodeFromBytes(cl.ndf.Nodes[i].ID)
		//Register with node if the node has not been registered with already
		if _, ok := registedNodes[nodeID]; !ok {
			wg.Add(1)
			go func() {
				cl.registerWithNode(i, salt, regSignature, UID, rsaPubKey, rsaPrivKey,
					cmixDHPubKey, cmixDHPrivKey, cmixGrp, nk, errChan)
				wg.Done()
			}()
		}
	}

	wg.Wait()
	//See if the registration returned errors at all
	var errs error
	for len(errChan) > 0 {
		err = <-errChan
		if errs != nil {
			errs = errors.Wrap(errs, err.Error())
		} else {
			errs = err
		}

	}
	//If an error every occured, return with error
	if errs != nil {
		cl.opStatus(globals.REG_FAIL)
		return "", errs
	}

	cl.session = session
	return cl.session.GetCurrentUser().Nick, nil
}

// Logs in user and sets session on client object
// returns the nickname or error if login fails
func (cl *Client) StartMessageReceiver() error {
	status := cl.commManager.GetConnectionStatus()
	if status == io.Connecting || status == io.Offline {
		return errors.New("ERROR: could not StartMessageReceiver - connection is either offline or connecting")
	}

	// Initialize UDB and nickname "bot" stuff here
	bots.InitBots(cl.session, cl.commManager, cl.topology, id.NewUserFromBytes(cl.ndf.UDB.ID))
	// Initialize Rekey listeners
	rekey.InitRekey(cl.session, cl.commManager, cl.topology, cl.rekeyChan)

	pollWaitTimeMillis := 1000 * time.Millisecond
	// TODO Don't start the message receiver if it's already started.
	// Should be a pretty rare occurrence except perhaps for mobile.
	go cl.commManager.MessageReceiver(cl.session, pollWaitTimeMillis, cl.rekeyChan)

	return nil
}

// TryReconnect Attemps to to reconnect with te network.  It will only cause
// an attempt if called durring a backoff timeout
func (cl *Client) TryReconnect() {
	cl.commManager.TryReconnect()
}

// Send prepares and sends a message to the cMix network
// FIXME: We need to think through the message interface part.
func (cl *Client) Send(message parse.MessageInterface) error {
	status := cl.commManager.GetConnectionStatus()
	if status == io.Connecting || status == io.Offline {
		return errors.New("Could not Send - connection is either offline or connecting")
	}

	// FIXME: There should (at least) be a version of this that takes a byte array
	recipientID := message.GetRecipient()
	cryptoType := message.GetCryptoType()
	return cl.commManager.SendMessage(cl.session, cl.topology, recipientID, cryptoType, message.Pack())
}

// DisableBlockingTransmission turns off blocking transmission, for
// use with the channel bot and dummy bot
func (cl *Client) DisableBlockingTransmission() {
	cl.commManager.DisableBlockingTransmission()
}

// SetRateLimiting sets the minimum amount of time between message
// transmissions just for testing, probably to be removed in production
func (cl *Client) SetRateLimiting(limit uint32) {
	cl.commManager.SetRateLimit(time.Duration(limit) * time.Millisecond)
}

func (cl *Client) Listen(user *id.User, messageType int32, newListener switchboard.Listener) string {
	listenerId := cl.session.GetSwitchboard().
		Register(user, messageType, newListener)
	globals.Log.INFO.Printf("Listening now: user %v, message type %v, id %v",
		user, messageType, listenerId)
	return listenerId
}

func (cl *Client) StopListening(listenerHandle string) {
	cl.session.GetSwitchboard().Unregister(listenerHandle)
}

func (cl *Client) GetSwitchboard() *switchboard.Switchboard {
	return cl.session.GetSwitchboard()
}

func (cl *Client) GetCurrentUser() *id.User {
	return cl.session.GetCurrentUser().User
}

func (cl *Client) GetKeyParams() *keyStore.KeyParams {
	return cl.session.GetKeyStore().GetKeyParams()
}

func (cl *Client) GetNetworkStatus() uint32 {
	return cl.commManager.GetConnectionStatus()
}

// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
// fixme: blocks forever is message reciever
func (cl *Client) Logout() error {
	if cl.session == nil {
		err := errors.New("Logout: Cannot Logout when you are not logged in")
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	// Stop reception runner goroutine
	close(cl.session.GetQuitChan())

	// Disconnect from the gateways
	for _, gateway := range cl.ndf.Gateways {
		cl.commManager.Comms.Disconnect(gateway.Address)
	}

	errStore := cl.session.StoreSession()

	if errStore != nil {
		err := errors.New(fmt.Sprintf("Logout: Store Failed: %s" +
			errStore.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	errImmolate := cl.session.Immolate()
	cl.session = nil

	if errImmolate != nil {
		err := errors.New(fmt.Sprintf("Logout: Immolation Failed: %s" +
			errImmolate.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	return nil
}

// Returns the local version of the client repo
func GetLocalVersion() string {
	return globals.SEMVER
}

// Returns the compatible version of client, according to permissioning
func (cl *Client) GetRemoteVersion() string {
	return cl.commManager.GetRegistrationVersion()
}

type SearchCallback interface {
	Callback(userID, pubKey []byte, err error)
}

// UDB Search API
// Pass a callback function to extract results
func (cl *Client) SearchForUser(emailAddress string,
	cb SearchCallback, timeout time.Duration) {
	status := cl.commManager.GetConnectionStatus()
	if status == io.Connecting || status == io.Offline {
		err := errors.New("Could not SearchForUser - connection is either offline or connecting")
		cb.Callback(nil, nil, err)
	}

	valueType := "EMAIL"
	go func() {
		uid, pubKey, err := bots.Search(valueType, emailAddress, cl.opStatus, timeout)
		if err == nil && uid != nil && pubKey != nil {
			cl.opStatus(globals.UDB_SEARCH_BUILD_CREDS)
			err = cl.registerUserE2E(uid, pubKey)
			if err != nil {
				cb.Callback(uid[:], pubKey, err)
				return
			}

			err = cl.session.StoreSession()
			if err != nil {
				cb.Callback(uid[:], pubKey, err)
				return
			}

			// If there is something in the channel then send it; otherwise,
			// skip over it
			select {
			case cl.rekeyChan <- struct{}{}:
			default:
			}

			cb.Callback(uid[:], pubKey, err)

		} else {
			if err == nil {
				globals.Log.INFO.Printf("UDB Search for email %s failed: user not found", emailAddress)
				err = errors.New("user not found in UDB")
				cb.Callback(nil, nil, err)
			} else {
				globals.Log.INFO.Printf("UDB Search for email %s failed: %+v", emailAddress, err)
				cb.Callback(nil, nil, err)
			}

		}
	}()
}

type NickLookupCallback interface {
	Callback(nick string, err error)
}

// Nickname lookup API
// Non-blocking, once the API call completes, the callback function
// passed as argument is called
func (cl *Client) LookupNick(user *id.User,
	cb NickLookupCallback) {
	go func() {
		status := cl.commManager.GetConnectionStatus()
		if status == io.Connecting || status == io.Offline {
			err := errors.New("Could not RegisterWithUDB - connection is either offline or connecting")
			cb.Callback("", err)
		}
		nick, err := bots.LookupNick(user)
		if err != nil {
			globals.Log.INFO.Printf("Lookup for nickname for user %s failed", user)
		}
		cb.Callback(nick, err)

	}()
}

//Message struct adherent to interface in bindings for data return from ParseMessage
type ParsedMessage struct {
	Typed   int32
	Payload []byte
}

func (p ParsedMessage) GetSender() []byte {
	return []byte{}
}

func (p ParsedMessage) GetPayload() []byte {
	return p.Payload
}

func (p ParsedMessage) GetRecipient() []byte {
	return []byte{}
}

func (p ParsedMessage) GetMessageType() int32 {
	return p.Typed
}

func (p ParsedMessage) GetTimestampNano() int64 {
	return 0
}

func (p ParsedMessage) GetTimestamp() int64 {
	return 0
}

// Parses a passed message.  Allows a message to be aprsed using the interal parser
// across the API
func ParseMessage(message []byte) (ParsedMessage, error) {
	tb, err := parse.Parse(message)

	pm := ParsedMessage{}

	if err != nil {
		return pm, err
	}

	pm.Payload = tb.Body
	pm.Typed = int32(tb.MessageType)

	return pm, nil
}

func (cl *Client) GetSessionData() ([]byte, error) {
	return cl.session.GetSessionData()
}

// Set the output of the
func SetLogOutput(w goio.Writer) {
	globals.Log.SetLogOutput(w)
}

// GetSession returns the session object for external access.  Access at your
// own risk
func (cl *Client) GetSession() user.Session {
	return cl.session
}

// CommManager returns the comm manager object for external access.  Access
// at your own risk
func (cl *Client) GetCommManager() *io.CommManager {
	return cl.commManager
}