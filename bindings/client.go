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
	GetSender() *id.UserID
	// Returns the message payload
	GetPayload() string
	// Returns the message's recipient ID
	GetRecipient() *id.UserID
}

// An object implementing this interface can be called back when the client
// gets a message
type Receiver interface {
	Receive(message Message)
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
func Send(m Message) error {
	return api.Send(m)
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

func SearchForUser(emailAddress string) (map[uint64][]byte, error) {
	return api.SearchForUser(emailAddress)
}
