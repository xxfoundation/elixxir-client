////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/payment"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	goio "io"
	"time"
)

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

// Initializes the client by registering a storage mechanism.
// If none is provided, the system defaults to using OS file access
// returns in error if it fails
func InitClient(s globals.Storage, loc string) error {
	storageErr := globals.InitStorage(s, loc)

	if storageErr != nil {
		storageErr = errors.New(
			"could not init client storage: " + storageErr.Error())
		return storageErr
	}

	crypto.InitCrypto()

	return nil
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func Register(registrationCode, registrationAddr string, gwAddresses []string,
	mint bool) (*id.User, error) {

	var err error

	if len(gwAddresses) < 1 {
		globals.Log.ERROR.Printf("Register: Invalid number of nodes")
		err = errors.New("could not register due to invalid number of nodes")
		return id.ZeroID, err
	}

	// Generate salt for UserID
	salt := make([]byte, 256)
	_, err = csprng.NewSystemRNG().Read(salt)
	if err != nil {
		globals.Log.ERROR.Printf("Register: Unable to generate salt! %s", err)
		return id.ZeroID, err
	}

	// Generate DSA keypair
	params := signature.NewDSAParams(rand.Reader, signature.L2048N256)
	privateKey := params.PrivateKeyGen(rand.Reader)
	publicKey := privateKey.PublicKeyGen()

	// Generate UserID by hashing salt and public key
	UID := registration.GenUserID(publicKey, salt)
	// Keep track of Server public keys provided at end of registration
	var serverPublicKeys []*signature.DSAPublicKey
	// Initialized response from Registration Server
	regHash, regR, regS := make([]byte, 0), make([]byte, 0), make([]byte, 0)

	// If Registration Server is specified, contact it
	if registrationAddr != "" {
		// Send registration code and public key to RegistrationServer
		response, err := client.SendRegistrationMessage(registrationAddr,
			&pb.RegisterUserMessage{
				RegistrationCode: registrationCode,
				Y:                publicKey.GetKey().Bytes(),
				P:                params.GetP().Bytes(),
				Q:                params.GetQ().Bytes(),
				G:                params.GetG().Bytes(),
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
		regHash, regR, regS = response.Hash, response.R, response.S
	}

	// Loop over all Servers
	for _, gwAddr := range gwAddresses {

		// Send signed public key and salt for UserID to Server
		nonceResponse, err := client.SendRequestNonceMessage(gwAddr,
			&pb.RequestNonceMessage{
				Salt: salt,
				Y:    publicKey.GetKey().Bytes(),
				P:    params.GetP().Bytes(),
				Q:    params.GetQ().Bytes(),
				G:    params.GetG().Bytes(),
				Hash: regHash,
				R:    regR,
				S:    regS,
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
		confirmResponse, err := client.SendConfirmNonceMessage(gwAddr,
			&pb.ConfirmNonceMessage{
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
					cyclic.NewIntFromBytes(confirmResponse.GetP()),
					cyclic.NewIntFromBytes(confirmResponse.GetQ()),
					cyclic.NewIntFromBytes(confirmResponse.GetG())),
				cyclic.NewIntFromBytes(confirmResponse.GetY())))

	}

	// Generate cyclic group for key generation
	grp := cyclic.NewGroup(
		params.GetP(),
		cyclic.NewInt(2),
		cyclic.NewInt(2),
		cyclic.NewRandom(cyclic.NewInt(3), cyclic.NewInt(7)),
	)

	nk := make([]user.NodeKeys, len(gwAddresses))

	// Initialise blake2b hash for transmission keys and sha256 for reception
	// keys
	transmissionHash, _ := hash.NewCMixHash()
	receptionHash := sha256.New()

	// Loop through all the server public keys
	for itr, publicKey := range serverPublicKeys {

		// Generate the base keys
		nk[itr].TransmissionKey = registration.GenerateBaseKey(
			&grp, publicKey, privateKey, transmissionHash,
		)

		nk[itr].ReceptionKey = registration.GenerateBaseKey(
			&grp, publicKey, privateKey, receptionHash,
		)
	}

	// Create the user session
	u := user.User{User: UID}
	nus := user.NewSession(&u, gwAddresses[0], nk, privateKey.PublicKeyGen(), privateKey)

	// Create the wallet
	_, err = payment.CreateWallet(nus, mint)
	if err != nil {
		return id.ZeroID, err
	}

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

	return UID, err
}

var quitReceptionRunner chan bool

// Logs in user and returns their nickname.
// returns an empty sting if login fails.
func Login(UID *id.User, addr string, tlsCert string) (user.Session, error) {

	connect.GatewayCertString = tlsCert

	session, err := user.LoadSession(UID)

	if session == nil {
		return nil, errors.New("Unable to load session: " + err.Error() +
			fmt.Sprintf("Passed parameters: %q, %s, %q", *UID, addr, tlsCert))
	}

	theWallet, err = payment.CreateWallet(session, false)
	if err != nil {
		err = fmt.Errorf("Login: Couldn't create wallet: %s", err.Error())
		globals.Log.ERROR.Printf(err.Error())
		return nil, err
	}
	theWallet.RegisterListeners()

	if addr != "" {
		session.SetGWAddress(addr)
	}

	addrToUse := session.GetGWAddress()

	// TODO: These can be separate, but we set them to the same thing
	//       until registration is completed.
	io.SendAddress = addrToUse
	io.ReceiveAddress = addrToUse

	if err != nil {
		err = errors.New(fmt.Sprintf("Login: Could not login: %s",
			err.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return nil, err
	}

	user.TheSession = session

	pollWaitTimeMillis := 1000 * time.Millisecond
	quitReceptionRunner = make(chan bool)
	// TODO Don't start the message receiver if it's already started.
	// Should be a pretty rare occurrence except perhaps for mobile.
	go io.Messaging.MessageReceiver(pollWaitTimeMillis, quitReceptionRunner)

	return session, nil
}

// Send prepares and sends a message to the cMix network
// FIXME: We need to think through the message interface part.
func Send(message parse.MessageInterface) error {
	// FIXME: There should (at least) be a version of this that takes a byte array
	recipientID := message.GetRecipient()
	err := io.Messaging.SendMessage(recipientID, message.Pack())
	return err
}

// DisableBlockingTransmission turns off blocking transmission, for
// use with the channel bot and dummy bot
func DisableBlockingTransmission() {
	io.BlockTransmissions = false
}

// SetRateLimiting sets the minimum amount of time between message
// transmissions just for testing, probably to be removed in production
func SetRateLimiting(limit uint32) {
	io.TransmitDelay = time.Duration(limit) * time.Millisecond
}

func Listen(user *id.User, outerType format.OuterType,
	messageType int32, newListener switchboard.Listener, callbacks *switchboard.
		Switchboard) string {
	listenerId := callbacks.Register(user, outerType, messageType, newListener)
	globals.Log.INFO.Printf("Listening now: user %v, message type %v, id %v",
		user, messageType, listenerId)
	return listenerId
}

func StopListening(listenerHandle string, callbacks *switchboard.Switchboard) {
	callbacks.Unregister(listenerHandle)
}

type APISender struct{}

func (s APISender) Send(messageInterface parse.MessageInterface) {
	Send(messageInterface)
}

type Sender interface {
	Send(messageInterface parse.MessageInterface)
}

// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
func Logout() error {
	if user.TheSession == nil {
		err := errors.New("Logout: Cannot Logout when you are not logged in")
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	// Stop reception runner goroutine
	quitReceptionRunner <- true

	// Disconnect from the gateway
	io.Disconnect(io.SendAddress)
	if io.SendAddress != io.ReceiveAddress {
		io.Disconnect(io.ReceiveAddress)
	}

	errStore := user.TheSession.StoreSession()
	// If a client is logging in again, the storage may need to go into a
	// different location
	// Currently, none of the storage abstractions need to do anything to
	// clean up in the long term. For example, DefaultStorage closes the
	// file every time it's written.
	globals.LocalStorage = nil

	if errStore != nil {
		err := errors.New(fmt.Sprintf("Logout: Store Failed: %s" +
			errStore.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	errImmolate := user.TheSession.Immolate()

	if errImmolate != nil {
		err := errors.New(fmt.Sprintf("Logout: Immolation Failed: %s" +
			errImmolate.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	// Reset listener structure
	switchboard.Listeners = switchboard.NewSwitchboard()

	return nil
}

func RegisterForUserDiscovery(emailAddress string) error {
	valueType := "EMAIL"
	userId, _, err := bots.Search(valueType, emailAddress)
	if userId != nil {
		globals.Log.DEBUG.Printf("Already registered %s", emailAddress)
		return nil
	}
	if err != nil {
		return err
	}

	publicKey := user.TheSession.GetPublicKey()
	publicKeyBytes := publicKey.GetKey().LeftpadBytes(256)
	return bots.Register(valueType, emailAddress, publicKeyBytes)
}

func SearchForUser(emailAddress string) (*id.User, []byte, error) {
	valueType := "EMAIL"
	return bots.Search(valueType, emailAddress)
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

func (p ParsedMessage) GetType() int32 {
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
	pm.Typed = int32(tb.InnerType)

	return pm, nil
}

// TODO Support more than one wallet per user? Maybe in v2
var theWallet *payment.Wallet

func Wallet() *payment.Wallet {
	if theWallet == nil {
		// Assume that the correct wallet is already stored in the session
		// (if necessary, minted during register)
		// So, if the wallet is nil, registration must have happened for this method to work
		var err error
		theWallet, err = payment.CreateWallet(user.TheSession, false)
		theWallet.RegisterListeners()
		if err != nil {
			globals.Log.ERROR.Println("Wallet("+
				"): Got an error creating the wallet.", err.Error())
		}
	}
	return theWallet
}

// Set the output of the
func SetLogOutput(w goio.Writer) {
	globals.Log.SetLogOutput(w)
}

func GetSessionData() ([]byte, error) {
	return user.TheSession.GetSessionData()
}
