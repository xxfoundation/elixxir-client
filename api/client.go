////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"crypto"
	"crypto/rand"
	gorsa "crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/rekey"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
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
	"time"
)

type Client struct {
	storage  globals.Storage
	session  user.Session
	comm     io.Communications
	ndf      *ndf.NetworkDefinition
	topology *circuit.Circuit
	tls      bool
}

var PermissioningAddrID = "registration"

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
	// Decode NDF string to a NetworkDefinition and its signature
	ndfJSON, ndfSignature, err := ndf.DecodeNDF(ndfString)
	if err != nil {
		globals.Log.FATAL.Panicf("Could not decode NDF: %v", err)
	}

	// If there is no public key, then skip verification and print warning
	if ndfPub == "" {
		globals.Log.WARN.Printf("Running without signed network " +
			"definition file")
	} else {
		// Load the TLS cert given to us, and from that get the RSA public key
		cert, err := tls.LoadCertificate(ndfPub)
		if err != nil {
			globals.Log.FATAL.Panicf("Could not load public key: %v", err)
		}
		pubKey := &rsa.PublicKey{PublicKey: *cert.PublicKey.(*gorsa.PublicKey)}

		// Hash NDF JSON
		rsaHash := sha256.New()
		rsaHash.Write(ndfJSON.Serialize())

		// Verify signature
		err = rsa.Verify(
			pubKey, crypto.SHA256, rsaHash.Sum(nil), ndfSignature, nil)

		if err != nil {
			globals.Log.FATAL.Panicf("Could not verify NDF: %v", err)
		}
	}

	return ndfJSON
}

// Creates a new Client using the storage mechanism provided.
// If none is provided, a default storage using OS file access
// is created
// returns a new Client object, and an error if it fails
func NewClient(s globals.Storage, loc string, ndfJSON *ndf.NetworkDefinition) (*Client, error) {
	globals.Log.DEBUG.Printf("NDF: %+v\n", ndfJSON)
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

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString(ndfJSON.CMIX.Prime, 16),
		large.NewIntFromString(ndfJSON.CMIX.Generator, 16),
		large.NewIntFromString(ndfJSON.CMIX.SmallPrime, 16))

	user.InitUserRegistry(cmixGrp)

	cl := new(Client)
	cl.storage = store
	cl.comm = io.NewMessenger()
	cl.ndf = ndfJSON

	//build the topology
	nodeIDs := make([]*id.Node, len(cl.ndf.Nodes))
	for i, node := range cl.ndf.Nodes {
		nodeIDs[i] = id.NewNodeFromBytes(node.ID)
	}

	cl.topology = circuit.New(nodeIDs)

	cl.tls = true

	return cl, nil
}

// DisableTLS makes the client run with TLS disabled
// Must be called before Connect
func (cl *Client) DisableTLS() {
	cl.tls = false
}

// Connects to gateways and registration server (if needed)
// using TLS filepaths to create credential information
// for connection establishment
func (cl *Client) Connect() error {
	var err error
	if len(cl.ndf.Gateways) < 1 {
		return errors.New("could not connect due to invalid number of nodes")
	}

	// connect to all gateways
	for i, gateway := range cl.ndf.Gateways {
		var gwCreds []byte
		if gateway.TlsCertificate != "" && cl.tls {
			gwCreds = []byte(gateway.TlsCertificate)
		}

		gwID := id.NewNodeFromBytes(cl.ndf.Nodes[i].ID).NewGateway()
		globals.Log.INFO.Printf("Connecting to gateway %s at %s...",
			gwID.String(), gateway.Address)
		err = (cl.comm).(*io.Messaging).Comms.ConnectToGateway(gwID, gateway.Address, gwCreds)
		if err != nil {
			return errors.New(fmt.Sprintf(
				"Failed to connect to gateway %s at %s: %+v",
				gwID.String(), gateway.Address, err))
		}
		globals.Log.INFO.Printf("Connected to gateway %s at %s successfully!",
			gwID.String(), gateway.Address)
	}

	//connect to the registration server
	if cl.ndf.Registration.Address != "" {
		var regCert []byte
		if cl.ndf.Registration.TlsCertificate != "" && cl.tls {
			regCert = []byte(cl.ndf.Registration.TlsCertificate)
		}
		addr := io.ConnAddr(PermissioningAddrID)
		globals.Log.INFO.Printf("Connecting to permissioning/registration at %s...",
			cl.ndf.Registration.Address)
		err = (cl.comm).(*io.Messaging).Comms.ConnectToRegistration(addr, cl.ndf.Registration.Address, regCert)
		if err != nil {
			return errors.New(fmt.Sprintf(
				"Failed connecting to permissioning: %+v", err))
		}
		globals.Log.INFO.Printf(
			"Connected to permissioning at %v successfully!",
			cl.ndf.Registration.Address)
	} else {
		globals.Log.WARN.Printf("Unable to find permissioning server!")
	}
	return err
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func (cl *Client) Register(preCan bool, registrationCode, nick, email string) (*id.User, error) {
	var err error
	var u *user.User
	var UID *id.User

	largeIntBits := 16

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString(cl.ndf.CMIX.Prime, largeIntBits),
		large.NewIntFromString(cl.ndf.CMIX.Generator, largeIntBits),
		large.NewIntFromString(cl.ndf.CMIX.SmallPrime, largeIntBits))

	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString(cl.ndf.E2E.Prime, largeIntBits),
		large.NewIntFromString(cl.ndf.E2E.Generator, largeIntBits),
		large.NewIntFromString(cl.ndf.E2E.SmallPrime, largeIntBits))

	// Make CMIX keys array
	nk := make(map[id.Node]user.NodeKeys)

	// GENERATE CLIENT RSA KEYS
	privateKeyRSA, err := rsa.GenerateKey(rand.Reader, rsa.DefaultRSABitLen)
	if err != nil {
		return nil, err
	}
	publicKeyRSA := privateKeyRSA.GetPublic()

	privateKeyDH := cmixGrp.RandomCoprime(cmixGrp.NewMaxInt())
	publicKeyDH := cmixGrp.ExpG(privateKeyDH, cmixGrp.NewMaxInt())

	// Handle precanned registration
	if preCan {
		globals.Log.INFO.Printf("Registering precanned user")
		u, UID, nk, err = cl.precannedRegister(registrationCode, nick, nk)
		if err != nil {
			globals.Log.ERROR.Printf("Unable to complete precanned registration: %+v", err)
			return id.ZeroID, err
		}
	} else {
		saltSize := 256
		// Generate salt for UserID
		salt := make([]byte, saltSize)
		_, err = csprng.NewSystemRNG().Read(salt)
		if err != nil {
			globals.Log.ERROR.Printf("Register: Unable to generate salt! %s", err)
			return id.ZeroID, err
		}

		// Generate UserID by hashing salt and public key
		UID = registration.GenUserID(publicKeyRSA, salt)

		// Initialized response from Registration Server
		regHash := make([]byte, 0)

		// If Registration Server is specified, contact it
		// Only if registrationCode is set
		globals.Log.INFO.Println("Register: Contacting registration server")
		if cl.ndf.Registration.Address != "" && registrationCode != "" {
			regHash, err = cl.sendRegistrationMessage(registrationCode, publicKeyRSA)
			if err != nil {
				globals.Log.ERROR.Printf("Register: Unable to send registration message: %+v", err)
				return id.ZeroID, err
			}
		}
		globals.Log.INFO.Println("Register: successfully passed Registration message")

		// Initialise blake2b hash for transmission keys and sha256 for reception
		// keys
		transmissionHash, _ := hash.NewCMixHash()
		receptionHash := sha256.New()

		// Loop over all Servers
		globals.Log.INFO.Println("Register: Requesting nonces")
		for i := range cl.ndf.Gateways {

			gwID := id.NewNodeFromBytes(cl.ndf.Nodes[i].ID).NewGateway()

			// Request nonce message from gateway
			globals.Log.INFO.Printf("Register: Requesting nonce from gateway %v/%v", i, len(cl.ndf.Gateways))
			nonce, dhPub, err := cl.requestNonce(salt, regHash, publicKeyDH, publicKeyRSA, privateKeyRSA, gwID)
			if err != nil {
				globals.Log.ERROR.Printf("Register: Failed requesting nonce from gateway: %+v", err)
				return id.ZeroID, err
			}

			// Load server DH pubkey
			serverPubDH := cmixGrp.NewIntFromBytes(dhPub)

			// Confirm received nonce
			globals.Log.INFO.Println("Register: Confirming received nonce")
			err = cl.confirmNonce(UID.Bytes(), nonce, privateKeyRSA, gwID)
			if err != nil {
				globals.Log.ERROR.Printf("Register: Unable to confirm nonce: %+v", err)
				return id.ZeroID, err
			}

			nodeID := *cl.topology.GetNodeAtIndex(i)
			nk[nodeID] = user.NodeKeys{
				TransmissionKey: registration.GenerateBaseKey(cmixGrp,
					serverPubDH, privateKeyDH, transmissionHash),
				ReceptionKey: registration.GenerateBaseKey(cmixGrp, serverPubDH,
					privateKeyDH, receptionHash),
			}

			receptionHash.Reset()
			transmissionHash.Reset()
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

	u.Email = email

	// Create the user session
	newSession := user.NewSession(cl.storage, u, nk, publicKeyRSA, privateKeyRSA, publicKeyDH, privateKeyDH, cmixGrp, e2eGrp)

	// Store the user session
	errStore := newSession.StoreSession()

	// FIXME If we have an error here, the session that gets created doesn't get immolated.
	// Immolation should happen in a deferred call instead.
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

// precannedRegister is a helper function for Register
// It handles the precanned registration case
func (cl *Client) precannedRegister(registrationCode, nick string,
	nk map[id.Node]user.NodeKeys) (*user.User, *id.User, map[id.Node]user.NodeKeys, error) {
	var successLook bool
	var UID *id.User
	var u *user.User
	var err error

	UID, successLook = user.Users.LookupUser(registrationCode)

	globals.Log.DEBUG.Printf("UID: %+v, success: %+v", UID, successLook)

	if !successLook {
		return nil, nil, nil, errors.New("precannedRegister: could not register due to invalid HUID")
	}

	var successGet bool
	u, successGet = user.Users.GetUser(UID)

	if !successGet {
		err = errors.New("precannedRegister: could not register due to ID lookup failure")
		return nil, nil, nil, err
	}

	if nick != "" {
		u.Nick = nick
	}

	nodekeys, successKeys := user.Users.LookupKeys(u.User)

	if !successKeys {
		err = errors.New("precannedRegister: could not register due to missing user keys")
		return nil, nil, nil, err
	}

	for i := 0; i < len(cl.ndf.Gateways); i++ {
		nk[*cl.topology.GetNodeAtIndex(i)] = *nodekeys
	}
	return u, UID, nk, nil
}

// sendRegistrationMessage is a helper for the Register function
// It sends a registration message and returns the registration hash
func (cl *Client) sendRegistrationMessage(registrationCode string,
	publicKeyRSA *rsa.PublicKey) ([]byte, error) {
	regHash := make([]byte, 0)
	// Send registration code and public key to RegistrationServer
	response, err := (cl.comm).(*io.Messaging).Comms.
		SendRegistrationMessage(io.ConnAddr(PermissioningAddrID),
			&pb.UserRegistration{
				RegistrationCode: registrationCode,
				ClientRSAPubKey:  string(rsa.CreatePublicKeyPem(publicKeyRSA)),
			})
	if err != nil {
		err = errors.New(fmt.Sprintf(
			"sendRegistrationMessage: Unable to contact Registration Server! %s", err))
		return nil, err
	}
	if response.Error != "" {
		err = errors.New(fmt.Sprintf("sendRegistrationMessage: error handing message: %s", response.Error))
		return nil, err
	}
	regHash = response.ClientSignedByServer.Signature
	// Disconnect from regServer here since it will not be needed
	(cl.comm).(*io.Messaging).Comms.Disconnect(cl.ndf.Registration.Address)
	return regHash, nil
}

// requestNonce is a helper for the Register function
// It sends a request nonce message containing the client's keys for signing
// Returns nonce if successful
func (cl *Client) requestNonce(salt, regHash []byte,
	publicKeyDH *cyclic.Int, publicKeyRSA *rsa.PublicKey,
	privateKeyRSA *rsa.PrivateKey, gwID *id.Gateway) ([]byte, []byte, error) {
	dhPub := publicKeyDH.Bytes()
	sha := crypto.SHA256
	opts := rsa.NewDefaultOptions()
	opts.Hash = sha
	h := sha.New()
	h.Write(dhPub)
	data := h.Sum(nil)
	fmt.Println(data)
	// Sign DH pubkey
	rng := csprng.NewSystemRNG()
	signed, err := rsa.Sign(rng, privateKeyRSA, sha, data, opts)
	if err != nil {
		return nil, nil, err
	}

	// Send signed public key and salt for UserID to Server
	nonceResponse, err := (cl.comm).(*io.Messaging).Comms.
		SendRequestNonceMessage(gwID,
			&pb.NonceRequest{
				Salt:            salt,
				ClientRSAPubKey: string(rsa.CreatePublicKeyPem(publicKeyRSA)),
				ClientSignedByServer: &pb.RSASignature{
					Signature: regHash,
				},
				ClientDHPubKey: publicKeyDH.Bytes(),
				RequestSignature: &pb.RSASignature{
					Signature: signed,
				},
			}) // TODO: modify this to return server DH
	if err != nil {
		err := errors.New(fmt.Sprintf(
			"requestNonce: Unable to request nonce! %s", err))
		return nil, nil, err
	}
	if nonceResponse.Error != "" {
		err := errors.New(fmt.Sprintf("requestNonce: nonceResponse error: %s", nonceResponse.Error))
		return nil, nil, err
	}

	// Use Client keypair to sign Server nonce
	return nonceResponse.Nonce, nonceResponse.DHPubKey, nil

}

// confirmNonce is a helper for the Register function
// It signs a nonce and sends it for confirmation
// Returns nil if successful, error otherwise
func (cl *Client) confirmNonce(UID, nonce []byte,
	privateKeyRSA *rsa.PrivateKey, gwID *id.Gateway) error {
	sha := crypto.SHA256
	opts := rsa.NewDefaultOptions()
	opts.Hash = sha
	h := sha.New()
	h.Write(nonce)
	data := h.Sum(nil)

	// Hash nonce & sign
	sig, err := rsa.Sign(rand.Reader, privateKeyRSA, sha, data, opts)
	if err != nil {
		globals.Log.ERROR.Printf(
			"Register: Unable to sign nonce! %s", err)
		return err
	}

	// Send signed nonce to Server
	// TODO: This returns a receipt that can be used to speed up registration
	msg := &pb.RequestRegistrationConfirmation{
		UserID: UID,
		NonceSignedByClient: &pb.RSASignature{
			Signature: sig,
		},
	}
	confirmResponse, err := (cl.comm).(*io.Messaging).Comms.
		SendConfirmNonceMessage(gwID, msg)
	if err != nil {
		err := errors.New(fmt.Sprintf(
			"confirmNonce: Unable to send signed nonce! %s", err))
		return err
	}
	if confirmResponse.Error != "" {
		err := errors.New(fmt.Sprintf(
			"confirmNonce: Error confirming nonce: %s", confirmResponse.Error))
		return err
	}
	return nil
}

// LoadSession loads the session object for the UID
func (cl *Client) Login(UID *id.User) (string, error) {
	session, err := user.LoadSession(cl.storage, UID)

	if err != nil {
		err = errors.New(fmt.Sprintf("Login: Could not login: %s",
			err.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return "", err
	}

	if session == nil {
		return "", errors.New("Unable to load session: " + err.Error())
	}

	cl.session = session
	return cl.session.GetCurrentUser().Nick, nil
}

// Logs in user and sets session on client object
// returns the nickname or error if login fails
func (cl *Client) StartMessageReceiver() error {
	sendGateway := id.NewNodeFromBytes(cl.ndf.Nodes[0].ID).NewGateway()
	receiveGateway := id.NewNodeFromBytes(cl.ndf.Nodes[len(cl.ndf.Nodes)-1].ID).NewGateway()
	(cl.comm).(*io.Messaging).SendGateway = sendGateway
	(cl.comm).(*io.Messaging).ReceiveGateway = receiveGateway
	globals.Log.INFO.Printf("Sending gateway: %s", sendGateway.String())
	globals.Log.INFO.Printf("Receiving gateway: %s", receiveGateway.String())

	// Initialize UDB and nickname "bot" stuff here
	bots.InitBots(cl.session, cl.comm, cl.topology)
	// Initialize Rekey listeners
	rekey.InitRekey(cl.session, cl.comm, cl.topology)

	pollWaitTimeMillis := 1000 * time.Millisecond
	// TODO Don't start the message receiver if it's already started.
	// Should be a pretty rare occurrence except perhaps for mobile.
	go cl.comm.MessageReceiver(cl.session, pollWaitTimeMillis)

	email := cl.session.GetCurrentUser().Email

	if email != "" {
		globals.Log.INFO.Printf("Registering user as %s", email)
		err := cl.registerForUserDiscovery(email)
		if err != nil {
			globals.Log.ERROR.Printf(
				"Unable to register with UDB: %s", err)
			return err
		}
		globals.Log.INFO.Printf("Registered!")
	}

	return nil
}

// Send prepares and sends a message to the cMix network
// FIXME: We need to think through the message interface part.
func (cl *Client) Send(message parse.MessageInterface) error {
	// FIXME: There should (at least) be a version of this that takes a byte array
	recipientID := message.GetRecipient()
	cryptoType := message.GetCryptoType()
	return cl.comm.SendMessage(cl.session, cl.topology, recipientID, cryptoType, message.Pack())
}

// DisableBlockingTransmission turns off blocking transmission, for
// use with the channel bot and dummy bot
func (cl *Client) DisableBlockingTransmission() {
	(cl.comm).(*io.Messaging).BlockTransmissions = false
}

// SetRateLimiting sets the minimum amount of time between message
// transmissions just for testing, probably to be removed in production
func (cl *Client) SetRateLimiting(limit uint32) {
	(cl.comm).(*io.Messaging).TransmitDelay = time.Duration(limit) * time.Millisecond
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
	cl.session.GetQuitChan() <- true

	// Disconnect from the gateways
	for _, gateway := range cl.ndf.Gateways {
		(cl.comm).(*io.Messaging).Comms.Disconnect(gateway.Address)
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

// Internal API for user discovery
func (cl *Client) registerForUserDiscovery(emailAddress string) error {
	valueType := "EMAIL"
	userId, _, err := bots.Search(valueType, emailAddress)
	if userId != nil {
		globals.Log.DEBUG.Printf("Already registered %s", emailAddress)
		return nil
	}
	if err != nil {
		return err
	}

	publicKey := cl.session.GetRSAPublicKey()
	publicKeyBytes := rsa.CreatePublicKeyPem(publicKey)
	return bots.Register(valueType, emailAddress, publicKeyBytes)
}

type SearchCallback interface {
	Callback(userID, pubKey []byte, err error)
}

// UDB Search API
// Pass a callback function to extract results
func (cl *Client) SearchForUser(emailAddress string,
	cb SearchCallback) {
	valueType := "EMAIL"
	go func() {
		uid, pubKey, err := bots.Search(valueType, emailAddress)
		if err == nil && uid != nil && pubKey != nil {
			cl.registerUserE2E(uid, pubKey)
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
		nick, err := bots.LookupNick(user)
		if err != nil {
			globals.Log.INFO.Printf("Lookup for nickname for user %s failed", user)
		}
		cb.Callback(nick, err)

	}()
}

func (cl *Client) registerUserE2E(partnerID *id.User,
	partnerPubKey []byte) {
	// Get needed variables from session
	grp := cl.session.GetCmixGroup()
	userID := cl.session.GetCurrentUser().User

	// Create user private key and partner public key
	// in the group
	privKey := cl.session.GetDHPrivateKey()
	privKeyCyclic := grp.NewIntFromLargeInt(privKey.GetLargeInt())
	partnerPubKeyCyclic := grp.NewIntFromBytes(partnerPubKey)

	// Generate baseKey
	baseKey, _ := diffieHellman.CreateDHSessionKey(
		partnerPubKeyCyclic,
		privKeyCyclic,
		grp)

	// Generate key TTL and number of keys
	params := cl.session.GetKeyStore().GetKeyParams()
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		params.MinKeys, params.MaxKeys, params.TTLParams)

	// Create Send KeyManager
	km := keyStore.NewManager(baseKey, privKeyCyclic,
		partnerPubKeyCyclic, partnerID, true,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Send Keys
	km.GenerateKeys(grp, userID, cl.session.GetKeyStore())

	// Create Receive KeyManager
	km = keyStore.NewManager(baseKey, privKeyCyclic,
		partnerPubKeyCyclic, partnerID, false,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Receive Keys
	km.GenerateKeys(grp, userID, cl.session.GetKeyStore())

	// Create RekeyKeys and add to RekeyManager
	rkm := cl.session.GetRekeyManager()

	keys := &keyStore.RekeyKeys{
		CurrPrivKey: privKeyCyclic,
		CurrPubKey:  partnerPubKeyCyclic,
	}

	rkm.AddKeys(partnerID, keys)
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
