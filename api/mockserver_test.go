////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/comms/gateway"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"math/rand"
	"os"
	"testing"
	"time"
)

var gwAddress = "localhost:8080"

var Session globals.SessionObj
var GatewayData TestInterface

func TestMain(m *testing.M) {
	rand.Seed(time.Now().Unix())
	gwAddress = fmt.Sprintf("localhost:%d", (rand.Intn(1000) + 5001))
	io.SendAddress = gwAddress
	io.ReceiveAddress = gwAddress
	GatewayData = TestInterface{
		LastReceivedMessage: pb.CmixMessage{},
	}
	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)

	os.Exit(m.Run())
}

// Make sure InitClient registers storage.
func TestInitClient(t *testing.T) {
	globals.LocalStorage = nil
	err := InitClient(nil, "", nil)
	if err != nil {
		t.Errorf("InitClient failed on valid input: %v", err)
	}
	if globals.LocalStorage == nil {
		t.Errorf("InitClient did not register storage.")
	}
	globals.LocalStorage = nil
}

func TestRegister(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "JHJ6L9BACDVC"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	regRes, err := Register(hashUID, gwAddress, 1)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if regRes == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	globals.LocalStorage = nil
}

func TestRegisterBadNumNodes(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "JHJ6L9BACDVC"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	_, err = Register(hashUID, gwAddress, 0)
	if err == nil {
		t.Errorf("Registration worked with bad numnodes! %s", err.Error())
	}
	globals.LocalStorage = nil
}

func TestRegisterBadHUID(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "JHJ6L9BACDV"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	_, err = Register(hashUID, gwAddress, 1)
	if err == nil {
		t.Errorf("Registration worked with bad registration code! %s",
			err.Error())
	}
	globals.LocalStorage = nil
}

func TestRegisterDeletedUser(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	registrationCode := "JHJ6L9BACDVC"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	tempUser, _ := globals.Users.GetUser(10)
	globals.Users.DeleteUser(10)
	_, err = Register(hashUID, gwAddress, 1)
	if err == nil {
		t.Errorf("Registration worked with a deleted user: %s",
			err.Error())
	}
	globals.Users.UpsertUser(tempUser)
	globals.LocalStorage = nil
}

func SetNulKeys() {
	// Set the transmit keys to be 1, so send/receive can work
	// FIXME: Why doesn't crypto panic when these keys are empty?
	keys := globals.Session.GetKeys()
	for i := range keys {
		keys[i].TransmissionKeys.Base = cyclic.NewInt(1)
		keys[i].TransmissionKeys.Recursive = cyclic.NewInt(1)
	}
	DisableRatchet()
}

func TestSend(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, &GatewayData)
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	globals.LocalStorage = nil
	registrationCode := "be50nhqpqjtjj"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	userID, err := Register(hashUID, gwAddress, 1)
	loginRes, err2 := Login(userID, gwAddress)
	SetNulKeys()

	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}

	// Test send with invalid sender ID
	err = Send(APIMessage{SenderID: 12, Payload: "test",
		RecipientID: userID})
	if err != nil {
		// TODO: would be nice to catch the sender but we
		// don't have the interface/mocking for that.
		t.Errorf("error on first message send: %v", err)
	}

	// Test send with valid inputs
	err = Send(APIMessage{SenderID: userID, Payload: "test",
		RecipientID: userID})
	if err != nil {
		t.Errorf("Error sending message: %v", err)
	}
}

func TestReceive(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	globals.LocalStorage = nil

	// Initialize client and log in
	registrationCode := "be50nhqpqjtjj"

	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	userID, err := Register(hashUID, gwAddress, 1)
	loginRes, err2 := Login(userID, gwAddress)
	SetNulKeys()

	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}
	if globals.Session == nil {
		t.Errorf("Could not load session!")
	}

	msg, _ := format.NewMessage(10, 10, "test")
	Send(&msg[0])
	time.Sleep(500 * time.Millisecond)

	receivedMsg, err := TryReceive()
	if err != nil || receivedMsg == nil {
		t.Errorf("Could not receive a message.")
	}
	if cyclic.NewIntFromBytes(receivedMsg.GetRecipient()).Uint64() != 0 {
		t.Errorf("Recipient of received message is incorrect. "+
			"Expected: 0 Actual %v", cyclic.NewIntFromBytes(receivedMsg.
			GetRecipient()).Uint64())
	}
}

func TestLogout(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()

	err := Logout()
	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}
	err = Logout()
	if err == nil {
		t.Errorf("Logout did not throw an error when called on a client that" +
			" is not currently logged in.")
	}
}
