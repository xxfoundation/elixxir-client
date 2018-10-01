////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/id"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/cmixproto"
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
	GetPayload() string
	// Returns the message's recipient ID
	GetRecipient() []byte
	// Returns the message's type
	GetType() int32
}

//  Translate a bindings message to a bindings message
type messageProxy struct {
	proxy Message
}

func (m *messageProxy) GetRecipient() *id.UserID {
	userId, err := new(id.UserID).SetBytes(m.proxy.GetRecipient())
	if err != nil {
		globals.Log.ERROR.Printf(
			"messageProxy GetRecipient: Error converting byte array to"+
				" recipient: %v", err.Error())
	}
	return userId
}

func (m *messageProxy) GetSender() *id.UserID {
	userId, err := new(id.UserID).SetBytes(m.proxy.GetSender())
	if err != nil {
		globals.Log.ERROR.Printf(
			"messageProxy GetSender: Error converting byte array to"+
				" sender: %v", err.Error())
	}
	return userId
}

func (m *messageProxy) GetPayload() string {
	return m.proxy.GetPayload()
}

func (m *messageProxy) GetType() cmixproto.Type {
	return cmixproto.Type(m.proxy.GetType())
}

// An object implementing this interface can be called back when the client
// gets a message of the type that the registerer specified at registration
// time.
type Listener interface {
	Hear(msg Message, isHeardElsewhere bool)
}

// Translate a bindings listener to a switchboard listener
type listenerProxy struct {
	proxy Listener
}

// Translate a parse message to a bindings message
type parseMessageProxy struct {
	proxy *parse.Message
}

func (p *parseMessageProxy) GetSender() []byte {
	return p.proxy.GetSender().Bytes()
}

func (p *parseMessageProxy) GetRecipient() []byte {
	return p.proxy.GetRecipient().Bytes()
}

// TODO Should we actually pass this over the boundary as a byte slice?
// It's essentially a binary blob.
func (p *parseMessageProxy) GetPayload() string {
	return p.proxy.GetPayload()
}

func (p *parseMessageProxy) GetType() int32 {
	return int32(p.proxy.GetType())
}

func (lp *listenerProxy) Hear(msg *parse.Message, isHeardElsewhere bool) {
	msgInterface := &parseMessageProxy{proxy: msg}
	lp.proxy.Hear(msgInterface, isHeardElsewhere)
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
	typedUserId, err := new(id.UserID).SetBytes(userId)
	if err != nil {
		globals.Log.ERROR.Printf("bindings."+
			"Listen user ID creation error: %v", err.Error())
	}

	listener := &listenerProxy{proxy: newListener}

	return api.Listen(typedUserId, cmixproto.Type(messageType), listener)
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

	err := api.InitClient(storage.(globals.Storage), loc)

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
	userID, err := new(id.UserID).SetBytes(UID)
	if err != nil {
		return "", err
	}
	session, err := api.Login(userID, addr)
	return session.GetCurrentUser().Nick, err
}

//Sends a message structured via the message interface
// FIXME Auto serialize type before sending somewhere
func Send(m Message) error {
	return api.Send(&messageProxy{proxy: m})
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

// FIXME This method doesn't get bound because of the exotic type it uses.
// Map types can't go over the boundary.
// The correct way to do over the boundary is to define
// a struct with a user ID and public key in it and return a pointer to that.
// Search() in bots only returns one user ID anyway. Returning a map would only
// be useful if a search could return more than one user.
func SearchForUser(emailAddress string) (map[uint64][]byte, error) {
	return api.SearchForUser(emailAddress)
}
