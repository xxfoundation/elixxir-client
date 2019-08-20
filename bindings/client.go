////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"fmt"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"io"
)

type Client struct {
	client *api.Client
}

// Copy of the storage interface.
// It is identical to the interface used in Globals,
// and a results the types can be passed freely between the two
type Storage interface {
	// Give a Location for storage.  Does not need to be implemented if unused.
	SetLocation(string) error
	// Returns the Location for storage.
	// Does not need to be implemented if unused.
	GetLocation() string
	// Stores the passed byte slice
	Save([]byte) error
	// Returns the stored byte slice
	Load() []byte
}

// Message used for binding
type Message interface {
	// Returns the message's sender ID
	GetSender() []byte
	// Returns the message payload
	// Parse this with protobuf/whatever according to the type of the message
	GetPayload() []byte
	// Returns the message's recipient ID
	GetRecipient() []byte
	// Returns the message's type
	GetMessageType() int32
}

// Translate a bindings message to a parse message
// An object implementing this interface can be called back when the client
// gets a message of the type that the registerer specified at registration
// time.
type Listener interface {
	Hear(msg Message, isHeardElsewhere bool)
}

// Returns listener handle as a string.
// You can use it to delete the listener later.
// Please ensure userId has the correct length (256 bits)
// User IDs are informally big endian. If you want compatibility with the demo
// user names, set the last byte and leave all other bytes zero for userId.
// If you pass the zero user ID (256 bits of zeroes) to Listen() you will hear
// messages sent from all users.
// If you pass the zero type (just zero) to Listen() you will hear messages of
// all types.
func (cl *Client) Listen(userId []byte, messageType int32, newListener Listener) string {
	typedUserId := id.NewUserFromBytes(userId)

	listener := &listenerProxy{proxy: newListener}

	return cl.client.Listen(typedUserId, messageType, listener)
}

// Pass the listener handle that Listen() returned to delete the listener
func (cl *Client) StopListening(listenerHandle string) {
	cl.client.StopListening(listenerHandle)
}

func FormatTextMessage(message string) []byte {
	return api.FormatTextMessage(message)
}

// Initializes the client by registering a storage mechanism and a reception
// callback.
// For the mobile interface, one must be provided
// The loc can be empty, it is only necessary if the passed storage interface
// requires it to be passed via "SetLocation"
//
// Parameters: storage implements Storage.
// Implement this interface to store the user session data locally.
// You must give us something for this parameter.
//
// loc is a string. If you're using DefaultStorage for your storage,
// this would be the filename of the file that you're storing the user
// session in.
func NewClient(storage Storage, loc string, ndfStr, ndfPubKey string) (*Client, error) {
	globals.Log.INFO.Printf("Binding call: NewClient()")
	if storage == nil {
		return nil, errors.New("could not init client: Storage was nil")
	}

	ndf := api.VerifyNDF(ndfStr, ndfPubKey)

	proxy := &storageProxy{boundStorage: storage}
	cl, err := api.NewClient(globals.Storage(proxy), loc, ndf)

	return &Client{client: cl}, err
}

// DisableTLS makes the client run with TLS disabled
// Must be called before Connect
func (cl *Client) DisableTLS() {
	globals.Log.INFO.Printf("Binding call: DisableTLS()")
	cl.DisableTLS()
}

// Connects to gateways and registration server (if needed)
// using TLS filepaths to create credential information
// for connection establishment
func (cl *Client) Connect() error {
	globals.Log.INFO.Printf("Binding call: Connect()")
	return cl.client.Connect()
}

// Registers user and returns the User ID bytes.
// Returns null if registration fails and error
// If preCan set to true, registration is attempted assuming a pre canned user
// registrationCode is a one time use string
// registrationAddr is the address of the registration server
// gwAddressesList is CSV of gateway addresses
// grp is the CMIX group needed for keys generation in JSON string format
func (cl *Client) Register(preCan bool, registrationCode, nick, email, password string) ([]byte, error) {
	globals.Log.INFO.Printf("Binding call: Register()\n"+
		"   preCan: %v\n   registrationCode: %s\n   nick: %s\n   email: %s\n"+
		"   Password: ********", preCan, registrationCode, nick, email)
	fmt.Println("calling client reg")
	UID, err := cl.client.Register(preCan, registrationCode, nick, email)

	if err != nil {
		return id.ZeroID[:], err
	}

	return UID[:], nil
}

// Logs in the user based on User ID and returns the nickname of that user.
// Returns an empty string and an error
// UID is a uint64 BigEndian serialized into a byte slice
func (cl *Client) Login(UID []byte, password string) (string, error) {
	globals.Log.INFO.Printf("Binding call: Login()\n"+
		"   UID: %v\n   Password: ********", UID)
	userID := id.NewUserFromBytes(UID)
	return cl.client.Login(userID)
}

// Starts the polling of the external servers.
// Must be done after listeners are set up.
func (cl *Client) StartMessageReceiver() error {
	globals.Log.INFO.Printf("Binding call: StartMessageReceiver()")
	return cl.client.StartMessageReceiver()
}

// Sends a message structured via the message interface
// Automatically serializes the message type before the rest of the payload
// Returns an error if either sender or recipient are too short
// the encrypt bool tell the client if it should send and e2e encrypted message
// or not.  If true, and there is no keying relationship with the user specified
// in the message object, then it will return an error.  If using precanned
// users encryption must be set to false.
func (cl *Client) Send(m Message, encrypt bool) error {
	sender := id.NewUserFromBytes(m.GetSender())
	recipient := id.NewUserFromBytes(m.GetRecipient())

	var cryptoType parse.CryptoType
	if encrypt {
		cryptoType = parse.E2E
	} else {
		cryptoType = parse.Unencrypted
	}

	return cl.client.Send(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: m.GetMessageType(),
			Body:        m.GetPayload(),
		},
		InferredType: cryptoType,
		Sender:       sender,
		Receiver:     recipient,
	})
}

// Logs the user out, saving the state for the system and clearing all data
// from RAM
func (cl *Client) Logout() error {
	return cl.client.Logout()
}

// Turns off blocking transmission so multiple messages can be sent
// simultaneously
func (cl *Client) DisableBlockingTransmission() {
	cl.client.DisableBlockingTransmission()
}

// Sets the minimum amount of time, in ms, between message transmissions
// Just for testing, probably to be removed in production
func (cl *Client) SetRateLimiting(limit int) {
	cl.client.SetRateLimiting(uint32(limit))
}

type SearchCallback interface {
	Callback(userID, pubKey []byte, err error)
}

type searchCallbackProxy struct {
	proxy SearchCallback
}

func (scp *searchCallbackProxy) Callback(userID, pubKey []byte, err error) {
	scp.proxy.Callback(userID, pubKey, err)
}

func (cl *Client) SearchForUser(emailAddress string,
	cb SearchCallback) {
	proxy := &searchCallbackProxy{cb}
	cl.client.SearchForUser(emailAddress, proxy)
}

type NickLookupCallback interface {
	Callback(nick string, err error)
}

type nickCallbackProxy struct {
	proxy NickLookupCallback
}

func (ncp *nickCallbackProxy) Callback(nick string, err error) {
	ncp.proxy.Callback(nick, err)
}

// Nickname lookup API
// Non-blocking, once the API call completes, the callback function
// passed as argument is called
func (cl *Client) LookupNick(user []byte,
	cb NickLookupCallback) {
	proxy := &nickCallbackProxy{cb}
	userID := id.NewUserFromBytes(user)
	cl.client.LookupNick(userID, proxy)
}

// Parses a passed message.  Allows a message to be aprsed using the interal parser
// across the Bindings
func ParseMessage(message []byte) (Message, error) {
	return api.ParseMessage(message)
}

// Translate a bindings listener to a switchboard listener
// Note to users of this package from other languages: Symbols that start with
// lowercase are unexported from the package and meant for internal use only.
type listenerProxy struct {
	proxy Listener
}

func (lp *listenerProxy) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	msgInterface := &parse.BindingsMessageProxy{Proxy: msg.(*parse.Message)}
	lp.proxy.Hear(msgInterface, isHeardElsewhere)
}

// Translate a bindings storage to a client storage
type storageProxy struct {
	boundStorage Storage
}

func (s *storageProxy) SetLocation(location string) error {
	return s.boundStorage.SetLocation(location)
}

func (s *storageProxy) GetLocation() string {
	return s.boundStorage.GetLocation()
}

func (s *storageProxy) Save(data []byte) error {
	return s.boundStorage.Save(data)
}

func (s *storageProxy) Load() []byte {
	return s.boundStorage.Load()
}

type Writer interface{ io.Writer }

func SetLogOutput(w Writer) {
	api.SetLogOutput(w)
}

// Call this to get the session data without getting Save called from the Go side
func (cl *Client) GetSessionData() ([]byte, error) {
	return cl.client.GetSessionData()
}
