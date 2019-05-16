////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"crypto/rand"
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
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"google.golang.org/grpc/credentials"
	goio "io"
	"time"
)

type Client struct {
	storage       globals.Storage
	sess          user.Session
	comm          io.Communications
	gwAddresses   []io.ConnAddr
	regAddress    io.ConnAddr
}

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

// Creates a new Client using the storage mechanism provided.
// If none is provided, a default storage using OS file access
// is created
// returns a new Client object, and an error if it fails
func NewClient(s globals.Storage, loc string) (*Client, error) {
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
	cl.comm = io.NewMessenger()
	cl.gwAddresses = make([]io.ConnAddr, 0)
	return cl, nil
}

// Connects to gateways and registration server (if needed)
// using TLS filepaths to create credential information
// for connection establishment
func (cl *Client) Connect(gwAddresses []string, gwCertPath,
    regAddr, regCertPath string) error {
	if len(gwAddresses) < 1 {
		globals.Log.ERROR.Printf("Connect: Invalid number of nodes")
		return errors.New("could not connect due to invalid number of nodes")
	}

	var gwCreds credentials.TransportCredentials = nil
	if gwCertPath != "" {
		gwCreds = connect.NewCredentialsFromFile(gwCertPath, "")
	}

	for _, gw := range gwAddresses {
		info := &connect.ConnectionInfo{
			Address: gw,
			Creds:   gwCreds,
		}
		addr := io.ConnAddr(gw)
		(cl.comm).(*io.Messaging).Comms.ConnectToGateway(addr, info)
		cl.gwAddresses = append(cl.gwAddresses, addr)
	}

	if regAddr != "" {
		var regCreds credentials.TransportCredentials = nil
		if regCertPath != "" {
			regCreds = connect.NewCredentialsFromFile(regCertPath, "")
		}
		info := &connect.ConnectionInfo{
			Address: regAddr,
			Creds:   regCreds,
		}
		addr := io.ConnAddr(regAddr)
		(cl.comm).(*io.Messaging).Comms.ConnectToRegistration(addr, info)
		cl.regAddress = addr
	}
	return nil
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func (cl *Client) Register(preCan bool, registrationCode, nick string,
	mint bool, grp *cyclic.Group) (*id.User, error) {
	var err error
	var u *user.User
	var UID *id.User
	// Make CMIX keys array
	nk := make([]user.NodeKeys, len(cl.gwAddresses))

	// Generate DSA keypair even for precanned users as it will probably
	// be needed for the new UDB flow
	params := signature.GetDefaultDSAParams()
	privateKey := params.PrivateKeyGen(rand.Reader)
	publicKey := privateKey.PublicKeyGen()

	// Handle precanned registration
	if preCan {
		var successLook bool
		globals.Log.DEBUG.Printf("Registering precanned user")
		UID, successLook = user.Users.LookupUser(registrationCode)

		if !successLook {
			globals.Log.ERROR.Printf("Register: HUID does not match")
			err = errors.New("could not register due to invalid HUID")
			return id.ZeroID, err
		}

		var successGet bool
		u, successGet = user.Users.GetUser(UID)

		if !successGet {
			globals.Log.ERROR.Printf("Register: ID lookup failed")
			err = errors.New("could not register due to ID lookup failure")
			return id.ZeroID, err
		}

		if nick != "" {
			u.Nick = nick;
		}

		nodekeys, successKeys := user.Users.LookupKeys(u.User)

		if !successKeys {
			globals.Log.ERROR.Printf("Register: could not find user keys")
			err = errors.New("could not register due to missing user keys")
			return id.ZeroID, err
		}

		for i := 0; i < len(cl.gwAddresses); i++ {
			nk[i] = *nodekeys
		}
	} else {
		// Generate salt for UserID
		salt := make([]byte, 256)
		_, err = csprng.NewSystemRNG().Read(salt)
		if err != nil {
			globals.Log.ERROR.Printf("Register: Unable to generate salt! %s", err)
			return id.ZeroID, err
		}

		// Generate UserID by hashing salt and public key
		UID = registration.GenUserID(publicKey, salt)
		// Keep track of Server public keys provided at end of registration
		var serverPublicKeys []*signature.DSAPublicKey
		// Initialized response from Registration Server
		regHash, regR, regS := make([]byte, 0), make([]byte, 0), make([]byte, 0)

		// If Registration Server is specified, contact it
		// Only if registrationCode is set
		if cl.regAddress != "" && registrationCode != "" {
			// Send registration code and public key to RegistrationServer
			response, err := (cl.comm).(*io.Messaging).Comms.
				SendRegistrationMessage(cl.regAddress,
				&pb.UserRegistration{
					RegistrationCode: registrationCode,
					Client: &pb.DSAPublicKey{
						Y: publicKey.GetKey().Bytes(),
						P: params.GetP().Bytes(),
						Q: params.GetQ().Bytes(),
						G: params.GetG().Bytes(),
					},
				})
			if err != nil {
				globals.Log.ERROR.Printf(
					"Register: Unable to contact Registration Server! %s", err)
				return id.ZeroID, err
			}
			if response.Error != "" {
				globals.Log.ERROR.Printf("Register: %s", response.Error)
				return id.ZeroID, errors.New(response.Error)
			}
			regHash = response.ClientSignedByServer.Hash
			regR = response.ClientSignedByServer.R
			regS = response.ClientSignedByServer.S
			// Disconnect from regServer here since it will not be needed
			(cl.comm).(*io.Messaging).Comms.Disconnect(cl.regAddress.String())
			cl.regAddress = ""
		}

		// Loop over all Servers
		for _, gwAddr := range cl.gwAddresses {

			// Send signed public key and salt for UserID to Server
			nonceResponse, err := (cl.comm).(*io.Messaging).Comms.
				SendRequestNonceMessage(gwAddr,
				&pb.NonceRequest{
					Salt: salt,
					Client: &pb.DSAPublicKey{
						Y: publicKey.GetKey().Bytes(),
						P: params.GetP().Bytes(),
						Q: params.GetQ().Bytes(),
						G: params.GetG().Bytes(),
					},
					ClientSignedByServer: &pb.DSASignature{
						Hash: regHash,
						R:    regR,
						S:    regS,
					},
				})
			if err != nil {
				globals.Log.ERROR.Printf(
					"Register: Unable to request nonce! %s",
					err)
				return id.ZeroID, err
			}
			if nonceResponse.Error != "" {
				globals.Log.ERROR.Printf("Register: %s", nonceResponse.Error)
				return id.ZeroID, errors.New(nonceResponse.Error)
			}

			// Use Client keypair to sign Server nonce
			nonce := nonceResponse.Nonce
			sig, err := privateKey.Sign(nonce, rand.Reader)
			if err != nil {
				globals.Log.ERROR.Printf(
					"Register: Unable to sign nonce! %s", err)
				return id.ZeroID, err
			}

			// Send signed nonce to Server
			// TODO: This returns a receipt that can be used to speed up registration
			confirmResponse, err := (cl.comm).(*io.Messaging).Comms.
				SendConfirmNonceMessage(gwAddr,
				&pb.DSASignature{
					Hash: nonce,
					R:    sig.R.Bytes(),
					S:    sig.S.Bytes(),
				})
			if err != nil {
				globals.Log.ERROR.Printf(
					"Register: Unable to send signed nonce! %s", err)
				return id.ZeroID, err
			}
			if confirmResponse.Error != "" {
				globals.Log.ERROR.Printf(
					"Register: %s", confirmResponse.Error)
				return id.ZeroID, errors.New(confirmResponse.Error)
			}

			// Append Server public key
			serverPublicKeys = append(serverPublicKeys,
				signature.ReconstructPublicKey(signature.
					CustomDSAParams(
						large.NewIntFromBytes(confirmResponse.Server.GetP()),
						large.NewIntFromBytes(confirmResponse.Server.GetQ()),
						large.NewIntFromBytes(confirmResponse.Server.GetG())),
					large.NewIntFromBytes(confirmResponse.Server.GetY())))

		}

		// Initialise blake2b hash for transmission keys and sha256 for reception
		// keys
		transmissionHash, _ := hash.NewCMixHash()
		receptionHash := sha256.New()

		// Loop through all the server public keys
		for itr, publicKey := range serverPublicKeys {

			// Generate the base keys
			nk[itr].TransmissionKey = registration.GenerateBaseKey(
				grp, publicKey, privateKey, transmissionHash,
			)

			transmissionHash.Reset()

			nk[itr].ReceptionKey = registration.GenerateBaseKey(
				grp, publicKey, privateKey, receptionHash,
			)

			receptionHash.Reset()
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

	// Create the user session
	nus := user.NewSession(cl.storage, u, nk, publicKey, privateKey, grp)

	// Store the user session
	errStore := nus.StoreSession()

	// FIXME If we have an error here, the session that gets created doesn't get immolated.
	// Immolation should happen in a deferred call instead.
	if errStore != nil {
		err = errors.New(fmt.Sprintf(
			"Register: could not register due to failed session save"+
				": %s", errStore.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return id.ZeroID, err
	}

	nus.Immolate()
	nus = nil

	return UID, nil
}

// Logs in user and sets session on client object
// returns the nickname or error if login fails
func (cl *Client) Login(UID *id.User, email string) (string, error) {
	session, err := user.LoadSession(cl.storage, UID)

	if session == nil {
		return "", errors.New("Unable to load session: " + err.Error())
	}

	(cl.comm).(*io.Messaging).SendAddress = cl.gwAddresses[0]
	(cl.comm).(*io.Messaging).ReceiveAddress =
		cl.gwAddresses[len(cl.gwAddresses)-1]

	if err != nil {
		err = errors.New(fmt.Sprintf("Login: Could not login: %s",
			err.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return "", err
	}

	cl.sess = session

	pollWaitTimeMillis := 1000 * time.Millisecond
	// TODO Don't start the message receiver if it's already started.
	// Should be a pretty rare occurrence except perhaps for mobile.
	go cl.comm.MessageReceiver(session, pollWaitTimeMillis)

	// Initialize UDB and nickname "bot" stuff here
	bots.InitBots(cl.sess, cl.comm)
	// Initialize Rekey listeners
	rekey.InitRekey(cl.sess, cl.comm)

	if email != "" {
		err = cl.registerForUserDiscovery(email)
		if err != nil {
			globals.Log.ERROR.Printf(
				"Unable to register with UDB: %s", err)
			return "", err
		}
	}

	return session.GetCurrentUser().Nick, nil
}

// Send prepares and sends a message to the cMix network
// FIXME: We need to think through the message interface part.
func (cl *Client) Send(message parse.MessageInterface) error {
	// FIXME: There should (at least) be a version of this that takes a byte array
	recipientID := message.GetRecipient()
	cryptoType := message.GetCryptoType()
	return cl.comm.SendMessage(cl.sess, recipientID, cryptoType, message.Pack())
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

func (cl *Client) Listen(user *id.User, outerType format.CryptoType,
	messageType int32, newListener switchboard.Listener) string {
	listenerId := cl.sess.GetSwitchboard().
		Register(user, outerType, messageType, newListener)
	globals.Log.INFO.Printf("Listening now: user %v, message type %v, id %v",
		user, messageType, listenerId)
	return listenerId
}

func (cl *Client) StopListening(listenerHandle string) {
	cl.sess.GetSwitchboard().Unregister(listenerHandle)
}

func (cl *Client) GetSwitchboard() *switchboard.Switchboard {
	return cl.sess.GetSwitchboard()
}

func (cl *Client) GetCurrentUser() *id.User {
	return cl.sess.GetCurrentUser().User
}

func (cl *Client) GetKeyParams() *keyStore.KeyParams {
	return cl.sess.GetKeyStore().GetKeyParams()
}

// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
func (cl *Client) Logout() error {
	if cl.sess == nil {
		err := errors.New("Logout: Cannot Logout when you are not logged in")
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	// Stop reception runner goroutine
	cl.sess.GetQuitChan() <- true

	// Disconnect from the gateways
	for _, gw := range cl.gwAddresses {
		(cl.comm).(*io.Messaging).Comms.Disconnect(gw.String())
	}

	errStore := cl.sess.StoreSession()

	if errStore != nil {
		err := errors.New(fmt.Sprintf("Logout: Store Failed: %s" +
			errStore.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	errImmolate := cl.sess.Immolate()
	cl.sess = nil

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

	publicKey := cl.sess.GetPublicKey()
	publicKeyBytes := publicKey.GetKey().LeftpadBytes(256)
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
		if err == nil {
			cl.registerUserE2E(uid, pubKey)
		} else {
			globals.Log.INFO.Printf("UDB Search for email %s failed", emailAddress)
		}
		cb.Callback(uid[:], pubKey, err)
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
		cb.Callback(nick, err)
	}()
}

func (cl *Client) registerUserE2E(partnerID *id.User,
	partnerPubKey []byte) {
	// Get needed variables from session
	grp := cl.sess.GetGroup()
	userID := cl.sess.GetCurrentUser().User

	// Create user private key and partner public key
	// in the group
	privKey := cl.sess.GetPrivateKey()
	privKeyCyclic := grp.NewIntFromLargeInt(privKey.GetKey())
	partnerPubKeyCyclic := grp.NewIntFromBytes(partnerPubKey)

	// Generate baseKey
	baseKey, _ := diffieHellman.CreateDHSessionKey(
		partnerPubKeyCyclic,
		privKeyCyclic,
		grp)

	// Generate key TTL and number of keys
	params := cl.sess.GetKeyStore().GetKeyParams()
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		params.MinKeys, params.MaxKeys, params.TTLParams)

	// Create Send KeyManager
	km := keyStore.NewManager(baseKey, privKeyCyclic,
		partnerPubKeyCyclic, partnerID, true,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Send Keys
	km.GenerateKeys(grp, userID, cl.sess.GetKeyStore())

	// Create Receive KeyManager
	km = keyStore.NewManager(baseKey, privKeyCyclic,
		partnerPubKeyCyclic, partnerID, false,
		numKeys, keysTTL, params.NumRekeys)

	// Generate Receive Keys
	km.GenerateKeys(grp, userID, cl.sess.GetKeyStore())

	// Create RekeyKeys and add to RekeyManager
	rkm := cl.sess.GetRekeyManager()

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
	return cl.sess.GetSessionData()
}

// Set the output of the
func SetLogOutput(w goio.Writer) {
	globals.Log.SetLogOutput(w)
}
