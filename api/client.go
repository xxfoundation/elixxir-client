////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	goio "io"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/payment"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/format"
	"time"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/crypto/id"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/connect"
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
func Register(registrationCode string, gwAddr string,
	numNodes uint, mint bool) (*id.UserID, error) {

	var err error

	if numNodes < 1 {
		globals.Log.ERROR.Printf("Register: Invalid number of nodes")
		err = errors.New("could not register due to invalid number of nodes")
		return id.ZeroID, err
	}

	// Because the method returns a pointer to the user ID, don't clear the
	// user ID as the caller needs to use it
	UID, successLook := user.Users.LookupUser(registrationCode)

	if !successLook {
		globals.Log.ERROR.Printf("Register: HUID does not match")
		err = errors.New("could not register due to invalid HUID")
		return id.ZeroID, err
	}

	u, successGet := user.Users.GetUser(UID)

	if !successGet {
		globals.Log.ERROR.Printf("Register: ID lookup failed")
		err = errors.New("could not register due to ID lookup failure")
		return id.ZeroID, err
	}

	nodekeys, successKeys := user.Users.LookupKeys(u.UserID)

	if !successKeys {
		globals.Log.ERROR.Printf("Register: could not find user keys")
		err = errors.New("could not register due to missing user keys")
		return id.ZeroID, err
	}

	nk := make([]user.NodeKeys, numNodes)

	for i := uint(0); i < numNodes; i++ {
		nk[i] = *nodekeys
	}

	nus := user.NewSession(u, gwAddr, nk, cyclic.NewIntFromBytes([]byte("this is not a real public key")))

	_, err = payment.CreateWallet(nus, mint)
	if err != nil {
		return id.ZeroID, err
	}

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

// Logs in user and returns their nickname.
// returns an empty sting if login fails.
func Login(UID *id.UserID, addr string, tlsCert string) (user.Session, error) {

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
	// TODO Don't start the message receiver if it's already started.
	// Should be a pretty rare occurrence except perhaps for mobile.
	go io.Messaging.MessageReceiver(pollWaitTimeMillis)

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

// FIXME there can only be one
var listenCh chan *format.Message

func Listen(user *id.UserID, messageType cmixproto.Type,
	newListener switchboard.Listener, callbacks *switchboard.
	Switchboard) string {
	listenerId := callbacks.Register(user, messageType, newListener)
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

	io.Disconnect(io.SendAddress)
	if io.SendAddress != io.ReceiveAddress {
		io.Disconnect(io.ReceiveAddress)
	}

	errStore := user.TheSession.StoreSession()

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
	publicKeyBytes := publicKey.LeftpadBytes(256)
	return bots.Register(valueType, emailAddress, publicKeyBytes)
}

func SearchForUser(emailAddress string) (*id.UserID, []byte, error) {
	valueType := "EMAIL"
	return bots.Search(valueType, emailAddress)
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
