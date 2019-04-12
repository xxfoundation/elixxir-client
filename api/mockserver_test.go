////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"os"
	"testing"
)

const NumGWs = 3
const RegPort = 5000
const RegGWsStartPort = 10000
const SessionGWPort = 15000

var RegAddress = fmtAddress(RegPort)
var RegGWAddresses [NumGWs]string
var SessionGWAddress = fmtAddress(SessionGWPort)

const ValidRegCode = "UAV6IWD6"
const InvalidRegCode = "INVALID_REG_CODE"

var RegGWHandlers = [NumGWs]*TestInterface{
	{LastReceivedMessage: pb.CmixMessage{}},
	{LastReceivedMessage: pb.CmixMessage{}},
	{LastReceivedMessage: pb.CmixMessage{}},
}

var RegHandler = MockRegistration{}

var SessionGWHandler = TestInterface{LastReceivedMessage: pb.CmixMessage{}}
var Session user.SessionObj

// Setups general testing params and calls test wrapper
func TestMain(m *testing.M) {

	// Set logging params
	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	os.Exit(testMainWrapper(m))
}

// Verify that a valid precanned user can register
func TestRegister_ValidPrecannedRegCodeReturnsZeroID(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Register precanned user with all gateways
	regRes, err := client.Register(true, ValidRegCode,
		"", RegGWAddresses[:], false, getGroup())

	// Verify registration succeeds with valid precanned registration code
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %v", *regRes)
	}
}

// Verify that a valid precanned user can register
func TestRegister_ValidRegParams___(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Register precanned user with all gateways
	regRes, err := client.Register(false, ValidRegCode,
		RegAddress, RegGWAddresses[:], false, getGroup())

	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %v", *regRes)
	}
}

// Verify that registering with an invalid number of gateways will fail
func TestRegister_InvalidNumGatewaysReturnsError(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Register with no gateways
	_, err = client.Register(true, ValidRegCode,
		"", []string{}, false, getGroup())
	if err == nil {
		t.Errorf("Registration worked with invalid number of gateways!")
	}
}

// Verify that registering with an invalid registration code will fail
func TestRegister_InvalidPrecannedRegCodeReturnsError(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Register with invalid reg code
	_, err = client.Register(true, InvalidRegCode,
		RegAddress, RegGWAddresses[:], false, getGroup())

	if err == nil {
		t.Error("Registration worked with invalid registration code!")
	}
}

func TestRegister_DeletedUserReturnsErr(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// ...
	tempUser, _ := user.Users.GetUser(id.NewUserFromUint(5, t))
	user.Users.DeleteUser(id.NewUserFromUint(5, t))

	// Register
	_, err = client.Register(true, ValidRegCode,
		RegAddress, RegGWAddresses[:], false, getGroup())

	if err == nil {
		t.Errorf("Registration worked with a deleted user: %s", err.Error())
	}

	// ...
	user.Users.UpsertUser(tempUser)
}

func TestSend(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Register with a valid registration code
	userID, err := client.Register(true, ValidRegCode,
		RegAddress, RegGWAddresses[:], false, getGroup())

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}

	// Login to gateway
	_, err = client.Login(userID, SessionGWAddress, "")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	// Test send with invalid sender ID
	err = client.Send(
		APIMessage{
			SenderID:    id.NewUserFromUint(12, t),
			Payload:     []byte("test"),
			RecipientID: userID,
		},
	)

	if err != nil {
		// TODO: would be nice to catch the sender but we
		// don't have the interface/mocking for that.
		t.Errorf("error on first message send: %v", err)
	}

	// Test send with valid inputs
	err = client.Send(APIMessage{SenderID: userID, Payload: []byte("test"),
		RecipientID: client.GetCurrentUser()})

	if err != nil {
		t.Errorf("Error sending message: %v", err)
	}

	err = client.Logout()

	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}
}

func TestLogout(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Logout before logging in should return an error
	err = client.Logout()

	if err == nil {
		t.Errorf("Logout did not throw an error when called on a client that" +
			" is not currently logged in.")
	}

	// Register with a valid registration code
	userID, err := client.Register(true, ValidRegCode,
		RegAddress, RegGWAddresses[:], false, getGroup())

	if err != nil {
		t.Errorf("Register failed: %s", err.Error())
	}

	// Login to gateway
	_, err = client.Login(userID, SessionGWAddress, "")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	err = client.Logout()

	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}

	// Logout after logout has been called should return an error
	err = client.Logout()

	if err == nil {
		t.Errorf("Logout did not throw an error when called on a client that" +
			" is not currently logged in.")
	}
}

// Handles initialization of mock registration server,
// gateways used for registration and gateway used for session
func testMainWrapper(m *testing.M) int {

	// Start mock gateways used by registration and defer their shutdown (may not be needed)
	for i, handler := range RegGWHandlers {
		RegGWAddresses[i] = fmtAddress(RegGWsStartPort+i)
		gw := gateway.StartGateway(
			RegGWAddresses[i], handler, "", "",
		)
		defer gw()
	}

	// Start mock registration server and defer its shutdown
	defer registration.StartRegistrationServer(RegAddress, &RegHandler, "", "")()

	// Start session gateway and defer its shutdown
	defer gateway.StartGateway(
		SessionGWAddress, &SessionGWHandler, "", "",
	)()

	return m.Run()
}

func getGroup() *cyclic.Group {
	return crypto.InitCrypto()
}

func fmtAddress(port int) string { return fmt.Sprintf("localhost:%d", port)}
