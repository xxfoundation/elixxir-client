////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	goio "io"
	"time"
)

type Client struct {
	storage globals.Storage
	sess user.Session
	comm io.Communications
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
		globals.Log.WARN.Printf("No storage provided," +
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
	return cl, nil
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func (cl *Client) Register(registrationCode string, gwAddr string,
	numNodes uint, mint bool, grp *cyclic.Group) (*id.User, error) {

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

	nodekeys, successKeys := user.Users.LookupKeys(u.User)

	if !successKeys {
		globals.Log.ERROR.Printf("Register: could not find user keys")
		err = errors.New("could not register due to missing user keys")
		return id.ZeroID, err
	}

	nk := make([]user.NodeKeys, numNodes)

	for i := uint(0); i < numNodes; i++ {
		nk[i] = *nodekeys
	}

	nus := user.NewSession(cl.storage, u, gwAddr, nk,
		grp.NewIntFromBytes([]byte("this is not a real public key")), grp)

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

// Logs in user and sets session on client object
// returns an error if login fails
func (cl *Client) Login(UID *id.User, addr string, tlsCert string) (string, error) {

	connect.GatewayCertString = tlsCert

	session, err := user.LoadSession(cl.storage, UID)

	if session == nil {
		return "", errors.New("Unable to load session: " + err.Error() +
			fmt.Sprintf("Passed parameters: %q, %s, %q", *UID, addr, tlsCert))
	}

	if addr != "" {
		session.SetGWAddress(addr)
	}

	addrToUse := session.GetGWAddress()

	// TODO: These can be separate, but we set them to the same thing
	//       until registration is completed.
	(cl.comm).(*io.Messaging).SendAddress = addrToUse
	(cl.comm).(*io.Messaging).ReceiveAddress = addrToUse

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

	// Initialize UDB stuff here
	bots.InitUDB(cl.sess, cl.comm, cl.sess.GetSwitchboard())

	return session.GetCurrentUser().Nick, nil
}

// Send prepares and sends a message to the cMix network
// FIXME: We need to think through the message interface part.
func (cl *Client) Send(message parse.MessageInterface) error {
	// FIXME: There should (at least) be a version of this that takes a byte array
	recipientID := message.GetRecipient()
	err := cl.comm.SendMessage(cl.sess, recipientID, message.Pack())
	return err
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

	// Disconnect from the gateway
	io.Disconnect(
		(cl.comm).(*io.Messaging).SendAddress)
	if (cl.comm).(*io.Messaging).SendAddress !=
		(cl.comm).(*io.Messaging).ReceiveAddress {
		io.Disconnect(
			(cl.comm).(*io.Messaging).ReceiveAddress)
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

func (cl *Client) RegisterForUserDiscovery(emailAddress string) error {
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
	publicKeyBytes := publicKey.LeftpadBytes(256)
	return bots.Register(valueType, emailAddress, publicKeyBytes)
}

func (cl *Client) SearchForUser(emailAddress string) (*id.User, []byte, error) {
	valueType := "EMAIL"
	return bots.Search(valueType, emailAddress)
}

func (cl *Client) registerUserE2E(partnerID *id.User,
	ownPrivKey *cyclic.Int,
	partnerPubKey *cyclic.Int) {
	// Get needed variables from session
	grp := cl.sess.GetGroup()
	userID := cl.sess.GetCurrentUser().User

	// Generate baseKey
	baseKey, _ := diffieHellman.CreateDHSessionKey(
		partnerPubKey,
		ownPrivKey,
		grp)

	// Generate key TTL and number of keys
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		keyStore.MinKeys, keyStore.MaxKeys,
		e2e.TTLParams{keyStore.TTLScalar,
			keyStore.Threshold})

	// Create KeyManager
	km := keyStore.NewManager(baseKey, partnerID,
		numKeys, keysTTL, keyStore.NumReKeys)

	// Generate Keys
	km.GenerateKeys(grp, userID, cl.sess.GetKeyStore())

	// Add Key Manager to session
	cl.sess.AddKeyManager(km)
}

//Message struct adherent to interface in bindings for data return from ParseMessage
type ParsedMessage struct{
	Typed int32
	Payload []byte
}

func (p ParsedMessage) GetSender()[]byte{
	return []byte{}
}

func (p ParsedMessage) GetPayload()[]byte{
	return p.Payload
}

func (p ParsedMessage) GetRecipient()[]byte{
	return []byte{}
}

func (p ParsedMessage) GetMessageType()int32{
	return p.Typed
}

// Parses a passed message.  Allows a message to be aprsed using the interal parser
// across the API
func ParseMessage(message []byte)(ParsedMessage,error){
	tb, err := parse.Parse(message)

	pm := ParsedMessage{}

	if err!=nil{
		return pm,err
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
