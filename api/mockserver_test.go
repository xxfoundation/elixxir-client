////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"os"
	"testing"
	"time"
)

const serverAddress = "localhost:5556"

const Nick = "Alduin"

var Session globals.SessionObj
var ServerData TestInterface

func TestMain(m *testing.M) {
	io.SendAddress = serverAddress
	io.ReceiveAddress = serverAddress
	ServerData = TestInterface{
		LastReceivedMessage: pb.CmixMessage{},
	}
	jww.SetLogThreshold(jww.LevelTrace)
	jww.SetStdoutThreshold(jww.LevelTrace)
	// Start server for all tests in this package
	go node.StartServer(serverAddress, &ServerData)

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
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	regRes, err := Register(hashUID, nick, serverAddress, "", 1)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if regRes == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	globals.LocalStorage = nil
}

func TestRegisterBadNumNodes(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	_, err = Register(hashUID, nick, serverAddress, "", 0)
	if err == nil {
		t.Errorf("Registration worked with bad numnodes! %s", err.Error())
	}
	globals.LocalStorage = nil
}

func TestRegisterBadHUID(t *testing.T) {
	registrationCode := "JHJ6L9BACDV"
	nick := "Nickname"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	_, err = Register(hashUID, nick, serverAddress, "", 1)
	if err == nil {
		t.Errorf("Registration worked with bad registration code! %s",
			err.Error())
	}
	globals.LocalStorage = nil
}

func TestRegisterDeletedUser(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	tempUser, _ := globals.Users.GetUser(10)
	globals.Users.DeleteUser(10)
	_, err = Register(hashUID, nick, serverAddress, "", 1)
	if err == nil {
		t.Errorf("Registration worked with a deleted user: %s",
			err.Error())
	}
	globals.Users.UpsertUser(tempUser)
	globals.LocalStorage = nil
}

func TestRegisterInvalidNick(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := ""
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	_, err = Register(hashUID, nick, serverAddress, "", 1)
	if err == nil {
		t.Errorf("Registration worked with invalid nickname! %s",
			err.Error())
	}
	globals.LocalStorage = nil
}

func TestUpdateUserRegistry(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	regRes, err := Register(hashUID, nick, serverAddress, "", 1)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if regRes == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	testContact := pb.Contact{UserID: uint64(15), Nick: "Guy"}
	testContactList := []*pb.Contact{&testContact}
	io.CheckContacts(&pb.ContactMessage{
		Contacts: testContactList,
	})
	userIDs, nicks := globals.Users.GetContactList()
	err = io.UpdateUserRegistry(serverAddress)
	if err != nil {
		t.Errorf("UpdateUserRegistry failed")
	}
	pass := false
	for i, id := range userIDs {
		if id == uint64(15) {
			if nicks[i] == "Guy" {
				pass = true
			}
		}
	}
	if !pass {
		t.Errorf("UpdateUserRegistry failed to update the user registry when" +
			" the helper function was passed a pb.ContactMessage")
	}
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
	globals.LocalStorage = nil
	nick := "Nickname"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := globals.UserHash(5)
	userID, err := Register(hashUID, nick, serverAddress, "", 1)
	loginRes, err2 := Login(userID, serverAddress)
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
	// 500ms for the other thread to catch it
	time.Sleep(100 * time.Millisecond)
	if err == nil && ServerData.LastReceivedMessage.SenderID != userID {
		t.Errorf("Invalid message was accepted by Send. " +
			"Sender ID must match current user")
	}

	// Test send with valid inputs
	err = Send(APIMessage{SenderID: userID, Payload: "test",
		RecipientID: userID})
	if err != nil {
		t.Errorf("Error sending message: %v", err)
	}
}

func TestReceive(t *testing.T) {
	globals.LocalStorage = nil

	// Initialize client and log in
	nick := "Nickname"

	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := globals.UserHash(5)
	userID, err := Register(hashUID, nick, serverAddress, "", 1)
	loginRes, err2 := Login(userID, serverAddress)
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

	msg, _ := format.NewMessage(userID, userID, "test")
	Send(&msg[0])
	time.Sleep(100 * time.Millisecond)

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

func TestSetNick(t *testing.T) {
	err := SetNick(0, "Guy")
	if err == nil {
		t.Errorf("SetNick did not error out on an invalid UID")
	}
	err = SetNick(5, "Guy")
	if err != nil {
		t.Errorf("SetNick failed: %v", err)
	}

}

func TestLogout(t *testing.T) {
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
