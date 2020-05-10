////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"bufio"
	"crypto"
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
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/switchboard"
	goio "io"
	"strings"
	"testing"
	"time"
)

type Client struct {
	storage             globals.Storage
	session             user.Session
	receptionManager    *io.ReceptionManager
	ndf                 *ndf.NetworkDefinition
	topology            *connect.Circuit
	opStatus            OperationProgressCallback
	rekeyChan           chan struct{}
	registrationVersion string

	// Pointer to a send function, which allows testing to override the default
	// using NewTestClient
	sendFunc sender
}

// Type that defines what the default and any testing send functions should look like
type sender func(message parse.MessageInterface, rm *io.ReceptionManager, session user.Session, topology *connect.Circuit, host *connect.Host) error

//used to report the state of registration
type OperationProgressCallback func(int)

// Creates a new client with the default send function
func NewClient(s globals.Storage, locA, locB string, ndfJSON *ndf.NetworkDefinition) (*Client, error) {
	return newClient(s, locA, locB, ndfJSON, send)
}

// Creates a new test client with an overridden send function
func NewTestClient(s globals.Storage, locA, locB string, ndfJSON *ndf.NetworkDefinition, i interface{}, sendFunc sender) (*Client, error) {
	switch i.(type) {
	case *testing.T:
		break
	case *testing.M:
		break
	case *testing.B:
		break
	default:
		globals.Log.FATAL.Panicf("GenerateId is restricted to testing only. Got %T", i)
	}
	return newClient(s, locA, locB, ndfJSON, sendFunc)
}

// Creates a new Client using the storage mechanism provided.
// If none is provided, a default storage using OS file access
// is created
// returns a new Client object, and an error if it fails
func newClient(s globals.Storage, locA, locB string, ndfJSON *ndf.NetworkDefinition, sendFunc sender) (*Client, error) {
	var store globals.Storage
	if s == nil {
		globals.Log.INFO.Printf("No storage provided," +
			" initializing Client with default storage")
		store = &globals.DefaultStorage{}
	} else {
		store = s
	}

	err := store.SetLocation(locA, locB)

	if err != nil {
		err = errors.New("Invalid Local Storage Location: " + err.Error())
		globals.Log.ERROR.Printf(err.Error())
		return nil, err
	}

	cl := new(Client)
	cl.storage = store
	cl.ndf = ndfJSON
	cl.sendFunc = sendFunc

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

// LoadSession loads the session object for the UID
func (cl *Client) Login(password string) (*id.ID, error) {

	var session user.Session
	var err error
	done := make(chan struct{})

	// run session loading in a separate goroutine so if it panics it can
	// be caught and an error can be returned
	go func() {
		defer func() {
			if r := recover(); r != nil {
				globals.Log.ERROR.Println("Session file loading crashed")
				err = sessionFileError
				done <- struct{}{}
			}
		}()

		session, err = user.LoadSession(cl.storage, password)
		done <- struct{}{}
	}()

	//wait for session file loading to complete
	<-done

	if err != nil {
		return nil, errors.Wrap(err, "Login: Could not login")
	}

	if session == nil {
		return nil, errors.New("Unable to load session, no error reported")
	}
	if session.GetRegState() < user.KeyGenComplete {
		return nil, errors.New("Cannot log a user in which has not " +
			"completed registration ")
	}

	cl.session = session
	newRm, err := io.NewReceptionManager(cl.rekeyChan, cl.session.GetCurrentUser().User,
		rsa.CreatePrivateKeyPem(cl.session.GetRSAPrivateKey()),
		rsa.CreatePublicKeyPem(cl.session.GetRSAPublicKey()),
		cl.session.GetSalt())
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create new reception manager")
	}
	newRm.Comms.Manager = cl.receptionManager.Comms.Manager
	cl.receptionManager = newRm
	return cl.session.GetCurrentUser().User, nil
}

// Logout closes the connection to the server and the messageReceiver and clears out the client values,
// so we can effectively shut everything down.  at this time it does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory. Recommended time out is 500ms.
func (cl *Client) Logout(timeoutDuration time.Duration) error {
	if cl.session == nil {
		err := errors.New("Logout: Cannot Logout when you are not logged in")
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	// Here using a select statement and the fact that making cl.sess.GetQuitChan is blocking, we can detect when
	// killing the reception manager is taking too long and we use the time out to stop the attempt and return an error.
	timer := time.NewTimer(timeoutDuration)
	select {
	case cl.session.GetQuitChan() <- struct{}{}:
		cl.receptionManager.Comms.DisconnectAll()
	case <-timer.C:
		return errors.Errorf("Message receiver shut down timed out after %s ms", timeoutDuration)
	}

	// Store the user session files before logging out
	errStore := cl.session.StoreSession()
	if errStore != nil {
		err := errors.New(fmt.Sprintf("Logout: Store Failed: %s" +
			errStore.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	// Clear all keys from ram
	errImmolate := cl.session.Immolate()
	cl.session = nil
	if errImmolate != nil {
		err := errors.New(fmt.Sprintf("Logout: Immolation Failed: %s" +
			errImmolate.Error()))
		globals.Log.ERROR.Printf(err.Error())
		return err
	}

	// Here we clear away all state in the client struct that should not be persistent
	cl.session = nil
	cl.receptionManager = nil
	cl.topology = nil
	cl.registrationVersion = ""

	return nil
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

func (cl *Client) GetRegistrationVersion() string { // on client
	return cl.registrationVersion
}

//GetNDF returns the clients ndf
func (cl *Client) GetNDF() *ndf.NetworkDefinition {
	return cl.ndf
}

func (cl *Client) SetOperationProgressCallback(rpc OperationProgressCallback) {
	cl.opStatus = func(i int) { go rpc(i) }
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

var sessionFileError = errors.New("Session file cannot be loaded and " +
	"is possibly corrupt. Please contact support@xxmessenger.io")

func (cl *Client) InitListeners() error {
	transmitGateway, err := id.Unmarshal(cl.ndf.Gateways[0].ID)
	if err != nil {
		return err
	}
	transmissionHost, ok := cl.receptionManager.Comms.GetHost(transmitGateway)
	if !ok {
		return errors.New("Failed to retrieve host for transmission")
	}

	// Initialize UDB and nickname "bot" stuff here
	udbID, err := id.Unmarshal(cl.ndf.UDB.ID)
	if err != nil {
		return err
	}
	bots.InitBots(cl.session, cl.receptionManager, cl.topology, udbID, transmissionHost)
	// Initialize Rekey listeners
	rekey.InitRekey(cl.session, cl.receptionManager, cl.topology, transmissionHost, cl.rekeyChan)
	return nil
}

// Logs in user and sets session on client object
// returns the nickname or error if login fails
func (cl *Client) StartMessageReceiver(callback func(error)) error {
	pollWaitTimeMillis := 500 * time.Millisecond
	// TODO Don't start the message receiver if it's already started.
	// Should be a pretty rare occurrence except perhaps for mobile.
	receptionGateway, err := id.Unmarshal(cl.ndf.Gateways[len(cl.ndf.Gateways)-1].ID)
	if err != nil {
		return err
	}
	receptionHost, ok := cl.receptionManager.Comms.GetHost(receptionGateway)
	if !ok {
		return errors.New("Failed to retrieve host for transmission")
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				globals.Log.ERROR.Println("Message Receiver Panicked: ", r)
				time.Sleep(1 * time.Second)
				go func() {
					callback(errors.New(fmt.Sprintln("Message Receiver Panicked", r)))
				}()
			}
		}()
		cl.receptionManager.MessageReceiver(cl.session, pollWaitTimeMillis, receptionHost, callback)
	}()

	return nil
}

// Default send function, can be overridden for testing
func (cl *Client) Send(message parse.MessageInterface) error {
	transmitGateway, err := id.Unmarshal(cl.ndf.Gateways[0].ID)
	if err != nil {
		return err
	}
	transmitGateway.SetType(id.Gateway)
	host, ok := cl.receptionManager.Comms.GetHost(transmitGateway)
	if !ok {
		return errors.New("Failed to retrieve host for transmission")
	}

	return cl.sendFunc(message, cl.receptionManager, cl.session, cl.topology, host)
}

// Send prepares and sends a message to the cMix network
func send(message parse.MessageInterface, rm *io.ReceptionManager, session user.Session, topology *connect.Circuit, host *connect.Host) error {
	recipientID := message.GetRecipient()
	cryptoType := message.GetCryptoType()
	return rm.SendMessage(session, topology, recipientID, cryptoType, message.Pack(), host)
}

// DisableBlockingTransmission turns off blocking transmission, for
// use with the channel bot and dummy bot
func (cl *Client) DisableBlockingTransmission() {
	cl.receptionManager.DisableBlockingTransmission()
}

// SetRateLimiting sets the minimum amount of time between message
// transmissions just for testing, probably to be removed in production
func (cl *Client) SetRateLimiting(limit uint32) {
	cl.receptionManager.SetRateLimit(time.Duration(limit) * time.Millisecond)
}

func (cl *Client) Listen(user *id.ID, messageType int32, newListener switchboard.Listener) string {
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

func (cl *Client) GetCurrentUser() *id.ID {
	return cl.session.GetCurrentUser().User
}

func (cl *Client) GetKeyParams() *keyStore.KeyParams {
	return cl.session.GetKeyStore().GetKeyParams()
}

// Returns the local version of the client repo
func GetLocalVersion() string {
	return globals.SEMVER
}

type SearchCallback interface {
	Callback(userID, pubKey []byte, err error)
}

// UDB Search API
// Pass a callback function to extract results
func (cl *Client) SearchForUser(emailAddress string,
	cb SearchCallback, timeout time.Duration) {
	//see if the user has been searched before, if it has, return it
	uid, pk := cl.session.GetContactByValue(emailAddress)

	if uid != nil {
		cb.Callback(uid.Bytes(), pk, nil)
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
			//store the user so future lookups can find it
			cl.session.StoreContactByValue(emailAddress, uid, pubKey)

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

func (cl *Client) DeleteUser(u *id.ID) (string, error) {

	//delete from session
	v, err1 := cl.session.DeleteContact(u)

	//delete from keystore
	err2 := cl.session.GetKeyStore().DeleteContactKeys(u)

	if err1 == nil && err2 == nil {
		return v, nil
	}

	if err1 != nil && err2 == nil {
		return "", errors.Wrap(err1, "Failed to remove from value store")
	}

	if err1 == nil && err2 != nil {
		return v, errors.Wrap(err2, "Failed to remove from key store")
	}

	if err1 != nil && err2 != nil {
		return "", errors.Wrap(fmt.Errorf("%s\n%s", err1, err2),
			"Failed to remove from key store and value store")
	}

	return v, nil

}

// Nickname lookup API
// Non-blocking, once the API call completes, the callback function
// passed as argument is called
func (cl *Client) LookupNick(user *id.ID,
	cb NickLookupCallback) {
	go func() {
		nick, err := bots.LookupNick(user)
		if err != nil {
			globals.Log.INFO.Printf("Lookup for nickname for user %+v failed", user)
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

// GetSession returns the session object for external access.  Access at yourx
// own risk
func (cl *Client) GetSession() user.Session {
	return cl.session
}

// ReceptionManager returns the comm manager object for external access.  Access
// at your own risk
func (cl *Client) GetCommManager() *io.ReceptionManager {
	return cl.receptionManager
}

// LoadSessionText: load the encrypted session as a string
func (cl *Client) LoadEncryptedSession() (string, error) {
	encryptedSession, err := cl.GetSession().LoadEncryptedSession(cl.storage)
	if err != nil {
		return "", err
	}
	//Encode session to bas64 for useability
	encodedSession := base64.StdEncoding.EncodeToString(encryptedSession)

	return encodedSession, nil
}

//WriteToSession: Writes an arbitrary string to the session file
// Takes in a string that is base64 encoded (meant to be output of LoadEncryptedSession)
func (cl *Client) WriteToSessionFile(replacement string, store globals.Storage) error {
	//This call must not occur prior to a newClient call, thus check that client has been initialized
	if cl.ndf == nil || cl.topology == nil {
		errMsg := errors.Errorf("Cannot write to session if client hasn't been created yet")
		return errMsg
	}
	//Decode the base64 encoded replacement string (assumed to be encoded form LoadEncryptedSession)
	decodedSession, err := base64.StdEncoding.DecodeString(replacement)
	if err != nil {
		errMsg := errors.Errorf("Failed to decode replacment string: %+v", err)
		return errMsg
	}
	//Write the new session data to both locations
	err = user.WriteToSession(decodedSession, store)
	if err != nil {
		errMsg := errors.Errorf("Failed to store session: %+v", err)
		return errMsg
	}

	return nil
}
