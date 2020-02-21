////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"crypto/rand"
	"errors"
	"github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/primitives/id"
	"io"
	"math/big"
	"strings"
	"time"
)

type Client struct {
	client         *api.Client
	statusCallback ConnectionStatusCallback
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
func NewClient(storage Storage, locA, locB string, ndfStr, ndfPubKey string) (*Client, error) {
	globals.Log.INFO.Printf("Binding call: NewClient()")
	if storage == nil {
		return nil, errors.New("could not init client: Storage was nil")
	}

	ndf := api.VerifyNDF(ndfStr, ndfPubKey)

	proxy := &storageProxy{boundStorage: storage}

	cl, err := api.NewClient(globals.Storage(proxy), locA, locB, ndf)

	return &Client{client: cl}, err
}

func NewClient_deprecated(storage Storage, locA, locB string, ndfStr, ndfPubKey string,
	csc ConnectionStatusCallback) (*Client, error) {

	return &Client{client: nil, statusCallback: csc}, nil
}

func (cl *Client) EnableDebugLogs() {
	globals.Log.INFO.Printf("Binding call: EnableDebugLogs()")
	globals.Log.SetStdoutThreshold(jwalterweatherman.LevelDebug)
	globals.Log.SetLogThreshold(jwalterweatherman.LevelDebug)
}

// Connects to gateways and registration server (if needed)
// using tls filepaths to create credential information
// for connection establishment
func (cl *Client) InitNetwork() error {
	globals.Log.INFO.Printf("Binding call: InitNetwork()")
	return cl.client.InitNetwork()
}

// Sets a callback which receives a strings describing the current status of
// Registration or UDB Registration, or UDB Search
func (cl *Client) SetOperationProgressCallback(rpcFace OperationProgressCallback) {
	rpc := func(i int) {
		rpcFace.Callback(i)
	}
	cl.client.SetOperationProgressCallback(rpc)
}

// Generate Keys generates the user identity for the network and stores it
func (cl *Client) GenerateKeys(password string) error {

	globals.Log.INFO.Printf("Binding call: GenerateKeys()\n" +
		"  Password: ********")
	return cl.client.GenerateKeys(nil, password)
}

// Registers user and returns the User ID bytes.
// Returns null if registration fails and error
// If preCan set to true, registration is attempted assuming a pre canned user
// registrationCode is a one time use string
// registrationAddr is the address of the registration server
// gwAddressesList is CSV of gateway addresses
// grp is the CMIX group needed for keys generation in JSON string format
func (cl *Client) RegisterWithPermissioning(preCan bool, registrationCode string) ([]byte, error) {

	globals.Log.INFO.Printf("Binding call: RegisterWithPermissioning()\n"+
		"   preCan: %v\n   registrationCode: %s\n  "+
		"   Password: ********", preCan, registrationCode)
	UID, err := cl.client.RegisterWithPermissioning(preCan, registrationCode)

	if err != nil {
		return id.ZeroID[:], err
	}

	return UID[:], nil
}

// Registers user with all nodes it has not been registered with.
// Returns error if registration fails
func (cl *Client) RegisterWithNodes() error {
	globals.Log.INFO.Printf("Binding call: RegisterWithNodes()")
	err := cl.client.RegisterWithNodes()
	return err
}

// Register with UDB uses the account's email to register with the UDB for
// User discovery.  Must be called after Register and InitNetwork.
// It will fail if the user has already registered with UDB
func (cl *Client) RegisterWithUDB(username string, timeoutMS int) error {
	globals.Log.INFO.Printf("Binding call: RegisterWithUDB()\n")
	return cl.client.RegisterWithUDB(username, time.Duration(timeoutMS)*time.Millisecond)
}

// Logs in the user based on User ID and returns the nickname of that user.
// Returns an empty string and an error
// UID is a uint64 BigEndian serialized into a byte slice
func (cl *Client) Login(UID []byte, password string) ([]byte, error) {
	globals.Log.INFO.Printf("Binding call: Login()\n"+
		"   UID: %v\n   Password: ********", UID)

	uid, err := cl.client.Login(password)

	if uid == nil {
		return make([]byte, 0), err
	}

	return (*uid)[:], err
}

func (cl *Client) GetUsername() string {
	globals.Log.INFO.Printf("Binding call: GetUsername()")

	return cl.client.GetSession().GetCurrentUser().Username
}

func (cl *Client) GetUserID() []byte {
	globals.Log.INFO.Printf("Binding call: GetUserID()")

	return cl.client.GetSession().GetCurrentUser().User[:]
}

type MessageReceiverCallback interface {
	Callback(err error)
}

// Starts the polling of the external servers.
// Must be done after listeners are set up.
func (cl *Client) StartMessageReceiver(mrc MessageReceiverCallback) error {
	globals.Log.INFO.Printf("Binding call: StartMessageReceiver()")
	return cl.client.StartMessageReceiver(mrc.Callback)
}

func (cl *Client) StartMessageReceiver_deprecated() error {
	receiverCallback := func(err error) {
		cl.backoff(0)
	}

	err := cl.client.StartMessageReceiver(receiverCallback)
	if err != nil {
		return err
	}
	return nil
}

func (cl *Client) backoff(backoffCount int) {
	receiverCallback := func(err error) {
		cl.backoff(backoffCount + 1)
	}

	// Compute backoff time
	var delay time.Duration
	var block = false
	if backoffCount > 15 {
		delay = time.Hour
		block = true
	}
	wait := 2 ^ backoffCount
	if wait > 180 {
		wait = 180
	}
	jitter, _ := rand.Int(csprng.NewSystemRNG(), big.NewInt(1000))
	delay = time.Second*time.Duration(wait) + time.Millisecond*time.Duration(jitter.Int64())

	cl.statusCallback.Callback(0, int(delay.Seconds()))

	// Start timer, or stop if max attempts reached
	timer := time.NewTimer(delay)
	if block {
		timer.Stop()
	}

	select {
	case <-timer.C:
		backoffCount = 0
	}

	// attempt to start the message receiver
	cl.statusCallback.Callback(1, 0)
	err := cl.client.StartMessageReceiver(receiverCallback)
	if err != nil {
		cl.statusCallback.Callback(0, 0)
	}
	cl.statusCallback.Callback(2, 0)
}

// Overwrites the username in registration. Only succeeds if the client
// has registered with permissioning but not UDB
func (cl *Client) ChangeUsername(un string) error {
	globals.Log.INFO.Printf("Binding call: ChangeUsername()\n"+
		"   username: %s", un)
	return cl.client.GetSession().ChangeUsername(un)
}

// gets the curent registration status.  they cane be:
//  0 - NotStarted
//	1 - PermissioningComplete
//	2 - UDBComplete
func (cl *Client) GetRegState() int64 {
	globals.Log.INFO.Printf("Binding call: GetRegState()")
	return int64(cl.client.GetSession().GetRegState())
}

// Registers user with all nodes it has not been registered with.
// Returns error if registration fails
func (cl *Client) StorageIsEmpty() bool {
	globals.Log.INFO.Printf("Binding call: StorageIsEmpty()")
	return cl.client.GetSession().StorageIsEmpty()
}

// Sends a message structured via the message interface
// Automatically serializes the message type before the rest of the payload
// Returns an error if either sender or recipient are too short
// the encrypt bool tell the client if it should send and e2e encrypted message
// or not.  If true, and there is no keying relationship with the user specified
// in the message object, then it will return an error.  If using precanned
// users encryption must be set to false.
func (cl *Client) Send(m Message, encrypt bool) (int64, error) {
	globals.Log.INFO.Printf("Binding call: Send()\n"+
		"Sender: %v\n"+
		"Payload: %v\n"+
		"Recipient: %v\n"+
		"MessageTye: %v", m.GetSender(), m.GetPayload(),
		m.GetRecipient(), m.GetMessageType())

	sender := id.NewUserFromBytes(m.GetSender())
	recipient := id.NewUserFromBytes(m.GetRecipient())

	var cryptoType parse.CryptoType
	if encrypt {
		cryptoType = parse.E2E
	} else {
		cryptoType = parse.Unencrypted
	}

	return time.Now().UnixNano(), cl.client.Send(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: m.GetMessageType(),
			Body:        m.GetPayload(),
		},
		InferredType: cryptoType,
		Sender:       sender,
		Receiver:     recipient,
	})
}

//a version of the send function which does not return a timestamp for use
//on iOS
func (cl *Client) SendNoTimestamp(m Message, encrypt bool) error {
	_, err := cl.Send(m, encrypt)
	return err
}

// Logs the user out, saving the state for the system and clearing all data
// from RAM
func (cl *Client) Logout() error {
	globals.Log.INFO.Printf("Binding call: Logout()\n")
	return cl.client.Logout(500 * time.Millisecond)
}

// Get the version string from the locally built client repository
func GetLocalVersion() string {
	globals.Log.INFO.Printf("Binding call: GetLocalVersion()\n")
	return api.GetLocalVersion()
}

// Get the version string from the registration server
// You need to connect to gateways for this to be populated.
// For the client to function, the local version must be compatible with this
// version. If that's not the case, check out the git tag corresponding to the
// client release version returned here.
func (cl *Client) GetRemoteVersion() string {
	globals.Log.INFO.Printf("Binding call: GetRemoteVersion()\n")
	return cl.GetRemoteVersion()
}

// Turns off blocking transmission so multiple messages can be sent
// simultaneously
func (cl *Client) DisableBlockingTransmission() {
	globals.Log.INFO.Printf("Binding call: DisableBlockingTransmission()\n")
	cl.client.DisableBlockingTransmission()
}

// Sets the minimum amount of time, in ms, between message transmissions
// Just for testing, probably to be removed in production
func (cl *Client) SetRateLimiting(limit int) {
	globals.Log.INFO.Printf("Binding call: SetRateLimiting()\n"+
		"   limit: %v", limit)
	cl.client.SetRateLimiting(uint32(limit))
}

// SearchForUser searches for the user with the passed username.
// returns state on the search callback.  A timeout in ms is required.
// A recommended timeout is 2 minutes or 120000
func (cl *Client) SearchForUser(username string,
	cb SearchCallback, timeoutMS int) {

	globals.Log.INFO.Printf("Binding call: SearchForUser()\n"+
		"   username: %v\n"+
		"   timeout: %v\n", username, timeoutMS)

	proxy := &searchCallbackProxy{cb}
	cl.client.SearchForUser(username, proxy, time.Duration(timeoutMS)*time.Millisecond)
}

// DeleteContact deletes the contact at the given userID.  returns the emails
// of that contact if possible
func (cl *Client) DeleteContact(uid []byte) (string, error) {
	globals.Log.INFO.Printf("Binding call: DeleteContact()\n"+
		"   uid: %v\n", uid)
	u := id.NewUserFromBytes(uid)

	return cl.client.DeleteUser(u)
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

// Parses a passed message.  Allows a message to be parsed using the internal parser
// across the Bindings
func ParseMessage(message []byte) (Message, error) {
	return api.ParseMessage(message)
}

func (s *storageProxy) SetLocation(locationA, locationB string) error {
	return s.boundStorage.SetLocation(locationA, locationB)
}

func (s *storageProxy) GetLocation() (string, string) {
	locsStr := s.boundStorage.GetLocation()
	locs := strings.Split(locsStr, ",")

	if len(locs) == 2 {
		return locs[0], locs[1]
	} else {
		return locsStr, locsStr + "-2"
	}
}

func (s *storageProxy) SaveA(data []byte) error {
	return s.boundStorage.SaveA(data)
}

func (s *storageProxy) LoadA() []byte {
	return s.boundStorage.LoadA()
}

func (s *storageProxy) SaveB(data []byte) error {
	return s.boundStorage.SaveB(data)
}

func (s *storageProxy) LoadB() []byte {
	return s.boundStorage.LoadB()
}

func (s *storageProxy) IsEmpty() bool {
	return s.boundStorage.IsEmpty()
}

type Writer interface{ io.Writer }

func SetLogOutput(w Writer) {
	api.SetLogOutput(w)
}

// Call this to get the session data without getting Save called from the Go side
func (cl *Client) GetSessionData() ([]byte, error) {
	return cl.client.GetSessionData()
}

//LoadEncryptedSession: Spits out the encrypted session file in text
func (cl *Client) LoadEncryptedSession() (string, error) {
	globals.Log.INFO.Printf("Binding call: LoadEncryptedSession()")
	return cl.client.LoadEncryptedSession()
}

//WriteToSession: Writes to file the replacement string
func (cl *Client) WriteToSession(replacement string, storage globals.Storage) error {
	globals.Log.INFO.Printf("Binding call: WriteToSession")
	return cl.client.WriteToSessionFile(replacement, storage)
}

func (cl *Client) InitListeners() error {
	globals.Log.INFO.Printf("Binding call: InitListeners")
	return cl.client.InitListeners()
}

// RegisterForNotifications sends a message to notification bot indicating it
// is registering for notifications
func (cl *Client) RegisterForNotifications(notificationToken []byte) error {
	return cl.client.RegisterForNotifications(notificationToken)
}

// UnregisterForNotifications sends a message to notification bot indicating it
// no longer wants to be registered for notifications
func (cl *Client) UnregisterForNotifications() error {
	return cl.client.UnregisterForNotifications()
}
