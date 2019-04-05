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
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"os"
	"testing"
	"time"
)

const (
	RegGatewayA = iota
	RegGatewayB
	RegGatewayC
	RegGatewayCount
)

const Port = 5000
const PrecannedRegCode = "UAV6IWD6"
const RegCode = "OIF3OJ6I"

var RegGatewayAddresses = [RegGatewayCount]string{}
var RegGatewayHandlers [RegGatewayCount]TestInterface

var SessionGatewayAddress string
var SessionGatewayHandler TestInterface
var Session user.SessionObj

// Setups general testing params and calls test wrapper
func TestMain(m *testing.M) {

	// Set logging params
	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	os.Exit(testMainWrapper(m))
}

// Handles initialization of mock registration server,
// gateways used for registration and gateway used for session
func testMainWrapper(m *testing.M) int {

	// Initialize registration gateway addresses and handlers
	for i := range RegGatewayAddresses {
		RegGatewayAddresses[i] = fmt.Sprintf("localhost:%d", Port + i)
		RegGatewayHandlers[i] = TestInterface{ LastReceivedMessage: pb.CmixMessage{} }
	}

	// Start registration gateways and defer their shutdown
	for _, gwAddress := range RegGatewayAddresses {
		gw := gateway.StartGateway(
			gwAddress, gateway.NewImplementation(), "", "",
		)
		defer gw()
	}

	// Set session gateway address and handler
	SessionGatewayAddress = "localhost:6000"
	SessionGatewayHandler = TestInterface{ LastReceivedMessage: pb.CmixMessage{} }

	// Set gateway address for io messaging
	io.SendAddress = SessionGatewayAddress
	io.ReceiveAddress = SessionGatewayAddress

	// Start session gateway and defer its shutdown
	gw := gateway.StartGateway(
		SessionGatewayAddress, gateway.NewImplementation(), "", "",
	)
	defer gw()

	// Wait for mock reg. server, mock reg. gateways and session gateway to start up
	time.Sleep(200 * time.Millisecond)

	return m.Run()
}

// Make sure InitClient registers storage.
func TestInitClient(t *testing.T) {
	globals.LocalStorage = nil

	err := InitClient(nil, "")
	if err != nil {
		t.Errorf("InitClient failed on valid input: %v", err)
	}

	if globals.LocalStorage == nil {
		t.Errorf("InitClient did not register storage.")
	}

	globals.LocalStorage = nil
}

// Verify that a valid precanned user can register
func TestRegister_ValidPrecannedUser(t *testing.T) {

	// Initialize client with dummy storage
	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&storage, "hello")
	if err != nil {
		t.Errorf("Failed to initialize dummy client: %s", err.Error())
	}

	// Register precanned user with all gateways
	regRes, err := Register(true, PrecannedRegCode, "", RegGatewayAddresses[:], false, getGroup())

	// Verify registration succeeds with valid precanned registration code
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}

	if *regRes == *id.ZeroID {
		t.Errorf("Invalid registration number received: %v", *regRes)
	}

	globals.LocalStorage = nil
}

// Verify that registering with an invalid number of gateways will fail
func TestRegister_InvalidNumGatewaysShouldFail(t *testing.T) {

	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")

	if err != nil {

	}

	_, err = Register(true, PrecannedRegCode, "", []string{}, false, getGroup())
	if err == nil {
		t.Errorf("Registration worked with bad numnodes! %s", err.Error())
	}

	globals.LocalStorage = nil
}

// ...
//func TestRegisterBadHUID(t *testing.T) {
//
//	// Initialize client with dummy storage
//	storage := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
//	err := InitClient(&storage, "hello")
//	if err != nil {
//		t.Errorf("Failed to initialize dummy client: %s", err.Error())
//	}
//
//	registrationCode := "OIF3OJ6I"
//
//	_, err = Register(true, registrationCode, RegistrationAddress, []string{"1", "2", "3"}, false, getGroup())
//	if err == nil {
//		t.Error("Registration worked with bad registration code!")
//	}
//	globals.LocalStorage = nil
//}

//func TestRegisterBadHUID(t *testing.T) {
//	gwShutDown := gateway.StartGateway(RegGatewayAddresses,
//		gateway.NewImplementation(), "", "")
//	time.Sleep(100 * time.Millisecond)
//	defer gwShutDown()
//
//	registrationCode := "OIF3OJ6I"
//	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
//	err := InitClient(&d, "hello")
//	p := large.NewInt(int64(107))
//	g := large.NewInt(int64(2))
//	q := large.NewInt(int64(3))
//	grp := cyclic.NewGroup(p, g, q)
//	_, err = Register(true, registrationCode, RegGatewayAddresses, []string{"1", "2", "3"}, false, grp)
//	if err == nil {
//		t.Error("Registration worked with bad registration code!")
//	}
//	globals.LocalStorage = nil
//}
//
//func TestRegisterDeletedUser(t *testing.T) {
//	gwShutDown := gateway.StartGateway(RegGatewayAddresses,
//		gateway.NewImplementation(), "", "")
//	time.Sleep(100 * time.Millisecond)
//	defer gwShutDown()
//
//	registrationCode := "UAV6IWD6"
//	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
//	err := InitClient(&d, "hello")
//	p := large.NewInt(int64(107))
//	g := large.NewInt(int64(2))
//	q := large.NewInt(int64(3))
//	grp := cyclic.NewGroup(p, g, q)
//	tempUser, _ := user.Users.GetUser(id.NewUserFromUint(5, t))
//	user.Users.DeleteUser(id.NewUserFromUint(5, t))
//	_, err = Register(true, registrationCode, RegGatewayAddresses, []string{"1", "2", "3"}, false, grp)
//	if err == nil {
//		t.Errorf("Registration worked with a deleted user: %s", err.Error())
//	}
//	user.Users.UpsertUser(tempUser)
//	globals.LocalStorage = nil
//}
//
//func SetNulKeys() {
//	// Set the transmit keys to be 1, so send/receive can work
//	// FIXME: Why doesn't crypto panic when these keys are empty?
//	keys := user.TheSession.GetKeys()
//	grp := user.TheSession.getGroup()
//	for i := range keys {
//		keys[i].TransmissionKey = grp.NewInt(1)
//		keys[i].TransmissionKey = grp.NewInt(1)
//	}
//}
//
//func TestSend(t *testing.T) {
//	gwShutDown := gateway.StartGateway(RegGatewayAddresses, &RegGatewayHandlers, "", "")
//	time.Sleep(100 * time.Millisecond)
//	defer gwShutDown()
//
//	globals.LocalStorage = nil
//	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
//	err := InitClient(&d, "hello")
//	grp := crypto.InitCrypto()
//	registrationCode := "UAV6IWD6"
//	userID, err := Register(true, registrationCode, "", []string{RegGatewayAddresses}, false, grp)
//	_, err2 := Login(userID, RegGatewayAddresses, "")
//	SetNulKeys()
//
//	if err != nil {
//		t.Errorf("Register failed: %s", err.Error())
//	}
//	if err2 != nil {
//		t.Errorf("Login failed: %s", err.Error())
//	}
//
//	// Test send with invalid sender ID
//	err = Send(APIMessage{SenderID: id.NewUserFromUint(12, t),
//		Payload:     []byte("test"),
//		RecipientID: userID})
//	if err != nil {
//		// TODO: would be nice to catch the sender but we
//		// don't have the interface/mocking for that.
//		t.Errorf("error on first message send: %v", err)
//	}
//
//	// Test send with valid inputs
//	err = Send(APIMessage{SenderID: userID, Payload: []byte("test"),
//		RecipientID: userID})
//	if err != nil {
//		t.Errorf("Error sending message: %v", err)
//	}
//}
//
//func TestLogout(t *testing.T) {
//	gwShutDown := gateway.StartGateway(RegGatewayAddresses,
//		gateway.NewImplementation(), "", "")
//	time.Sleep(100 * time.Millisecond)
//	defer gwShutDown()
//
//	err := Logout()
//	if err != nil {
//		t.Errorf("Logout failed: %v", err)
//	}
//	err = Logout()
//	if err == nil {
//		t.Errorf("Logout did not throw an error when called on a client that" +
//			" is not currently logged in.")
//	}
//}


func getGroup() *cyclic.Group {
	prime  := large.NewInt(int64(107))
	gen    := large.NewInt(int64(2))
	primeQ := large.NewInt(int64(3))

	return cyclic.NewGroup(prime, gen, primeQ)
}