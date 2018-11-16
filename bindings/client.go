////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/crypto/id"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/switchboard"
	"sync"
)

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

//Message used for binding
type Message interface {
	// Returns the message's sender ID
	GetSender() []byte
	// Returns the message payload
	// Parse this with protobuf/whatever according to the type of the message
	GetPayload() []byte
	// Returns the message's recipient ID
	GetRecipient() []byte
	// Returns the message's type
	GetType() int32
}

//  Translate a bindings message to a parse message
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
func Listen(userId []byte, messageType int32, newListener Listener) string {
	typedUserId := new(id.UserID).SetBytes(userId)

	listener := &listenerProxy{proxy: newListener}

	return api.Listen(typedUserId, cmixproto.Type(messageType), listener, switchboard.Listeners)
}

// Pass the listener handle that Listen() returned to delete the listener
func StopListening(listenerHandle string) {
	api.StopListening(listenerHandle, switchboard.Listeners)
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
func InitClient(storage Storage, loc string) error {
	if storage == nil {
		return errors.New("could not init client: Storage was nil")
	}

	proxy := &storageProxy{boundStorage: storage}
	err := api.InitClient(globals.Storage(proxy), loc)

	return err
}

// Registers user and returns the User ID.  Returns null if registration fails.
// registrationCode is a one time use string.
// nick is a nickname which must be 32 characters or less.
// nodeAddr is the ip address and port of the last node in the form: 192.168.1.1:50000
// numNodes is the number of nodes in the system
// Valid codes:
// 1
// “David”
// RUHPS2MI
// 2
// “Jim”
// AXJ3XIBD
// 3
// “Ben”
// AW55QN6U
// 4
// “Rick”
// XYRAUUO6
// 5
// “Spencer”
// UAV6IWD6
// 6
// “Jake”
// XEHCZT5U
// 7
// “Mario”
// BW7NEXOZ
// 8
// “Will”
// IRZVJ55Y
// 9
// “Allan”
// YRZEM7BW
// 10
// “Jono”
// OIF3OJ5I
func Register(registrationCode string, gwAddr string, numNodes int,
	mint bool) ([]byte, error) {

	if numNodes < 1 {
		return id.ZeroID[:], errors.New("invalid number of nodes")
	}

	UID, err := api.Register(registrationCode, gwAddr, uint(numNodes), mint)

	if err != nil {
		return id.ZeroID[:], err
	}

	return UID[:], nil
}

// Logs in the user based on User ID and returns the nickname of that user.
// Returns an empty string and an error
// UID is a uint64 BigEndian serialized into a byte slice
// TODO Pass the session in a proto struct/interface in the bindings or something
func Login(UID []byte, addr string) (string, error) {
	userID := new(id.UserID).SetBytes(UID)
	session, err := api.Login(userID, addr)
	if err != nil || session == nil {
		return "", err
	} else {
		return session.GetCurrentUser().Nick, err
	}
}

//Sends a message structured via the message interface
// Automatically serializes the message type before the rest of the payload
// Returns an error if either sender or recipient are too short
func Send(m Message) error {
	sender := new(id.UserID).SetBytes(m.GetSender())
	recipient := new(id.UserID).SetBytes(m.GetRecipient())

	return api.Send(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type(m.GetType()),
			Body: m.GetPayload(),
		},
		Sender:   sender,
		Receiver: recipient,
	})
}

// Logs the user out, saving the state for the system and clearing all data
// from RAM
func Logout() error {
	return api.Logout()
}

// Turns off blocking transmission so multiple messages can be sent
// simultaneously
func DisableBlockingTransmission() {
	api.DisableBlockingTransmission()
}

// Sets the minimum amount of time, in ms, between message transmissions
// Just for testing, probably to be removed in production
func SetRateLimiting(limit int) {
	api.SetRateLimiting(uint32(limit))
}

func RegisterForUserDiscovery(emailAddress string) error {
	return api.RegisterForUserDiscovery(emailAddress)
}

type SearchResult struct {
	ResultID  []byte // Underlying type: *id.UserID
	PublicKey []byte
}

func SearchForUser(emailAddress string) (*SearchResult, error) {
	searchedUserID, key, err := api.SearchForUser(emailAddress)
	if err != nil {
		return nil, err
	} else {
		return &SearchResult{ResultID: searchedUserID.Bytes(), PublicKey: key}, nil
	}
}

// Translate a bindings listener to a switchboard listener
// Note to users of this package from other languages: Symbols that start with
// lowercase are unexported from the package and meant for internal use only.
type listenerProxy struct {
	proxy Listener
}

func (lp *listenerProxy) Hear(msg *parse.Message, isHeardElsewhere bool) {
	msgInterface := &parse.BindingsMessageProxy{Proxy: msg}
	lp.proxy.Hear(msgInterface, isHeardElsewhere)
}

// Unexported: Used to implement Lock and Unlock with the storage interface.
// Not quite sure whether this will work as intended or not. Will have to test.
type storageProxy struct {
	boundStorage Storage
	lock         sync.Mutex
}

// TODO Should these methods take the mutex? Probably
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

func (s *storageProxy) Lock() {
	s.lock.Lock()
}

func (s *storageProxy) Unlock() {
	s.lock.Unlock()
}
