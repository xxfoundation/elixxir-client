////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/primitives/id"
	"io"
)

type Client struct {
	client *api.Client
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
func NewClient(storage Storage, loc string, ndfStr, ndfPubKey string,
	csc ConnectionStatusCallback) (*Client, error) {
	globals.Log.INFO.Printf("Binding call: NewClient()")
	if storage == nil {
		return nil, errors.New("could not init client: Storage was nil")
	}

	ndf := api.VerifyNDF(ndfStr, ndfPubKey)

	proxy := &storageProxy{boundStorage: storage}

	conStatCallback := func(status uint32, TimeoutSeconds int) {
		csc.Callback(int(status), TimeoutSeconds)
	}

	cl, err := api.NewClient(globals.Storage(proxy), loc, ndf, conStatCallback)

	return &Client{client: cl}, err
}

// DisableTLS makes the client run with TLS disabled
// Must be called before Connect
func (cl *Client) DisableTLS() {
	globals.Log.INFO.Printf("Binding call: DisableTLS()")
	cl.client.DisableTLS()
}

func (cl *Client) EnableDebugLogs() {
	globals.Log.INFO.Printf("Binding call: EnableDebugLogs()")
	globals.Log.SetStdoutThreshold(jwalterweatherman.LevelDebug)
	globals.Log.SetLogThreshold(jwalterweatherman.LevelDebug)
}

// Connects to gateways and registration server (if needed)
// using TLS filepaths to create credential information
// for connection establishment
func (cl *Client) Connect() error {
	globals.Log.INFO.Printf("Binding call: Connect()")
	return cl.client.Connect()
}

// Sets a callback which receives a strings describing the current status of
// Registration or UDB Registeration
func (cl *Client) SetRegisterProgressCallback(rpcFace RegistrationProgressCallback) {
	rpc := func(i int) {
		rpcFace.Callback(i)
	}
	cl.client.SetRegisterProgressCallback(rpc)
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
	UID, err := cl.client.Register(preCan, registrationCode, nick, email,
		password, nil)

	if err != nil {
		return id.ZeroID[:], err
	}

	return UID[:], nil
}

// Register with UDB uses the account's email to register with the UDB for
// User discovery.  Must be called after Register and Connect.
// It will fail if the user has already registered with UDB
func (cl *Client) RegisterWithUDB() error {
	globals.Log.INFO.Printf("Binding call: RegisterWithUDB()\n")
	return cl.client.RegisterWithUDB()
}

// Logs in the user based on User ID and returns the nickname of that user.
// Returns an empty string and an error
// UID is a uint64 BigEndian serialized into a byte slice
func (cl *Client) Login(UID []byte, password string) (string, error) {
	globals.Log.INFO.Printf("Binding call: Login()\n"+
		"   UID: %v\n   Password: ********", UID)
	return cl.client.Login(password)
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

// Get the version string from the locally built client repository
func GetLocalVersion() string {
	return api.GetLocalVersion()
}

// Get the version string from the registration server
// You need to connect to gateways for this to be populated.
// For the client to function, the local version must be compatible with this
// version. If that's not the case, check out the git tag corresponding to the
// client release version returned here.
func (cl *Client) GetRemoteVersion() string {
	return cl.client.GetRemoteVersion()
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

func (cl *Client) SearchForUser(emailAddress string,
	cb SearchCallback) {
	proxy := &searchCallbackProxy{cb}
	cl.client.SearchForUser(emailAddress, proxy)
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

//Call to get the networking status of the client
// 0 - Offline
// 1 - Connecting
// 2 - Connected
func (cl *Client) GetNetworkStatus()int64{
	return int64(cl.GetNetworkStatus())
}
