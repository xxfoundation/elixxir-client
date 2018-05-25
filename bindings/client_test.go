////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package bindings

import (
	"bytes"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/client/api"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/node"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"os"
	"strings"
	"testing"
	"time"
)

const serverAddress = "localhost:5557"
var ServerData api.TestInterface

func TestMain(m *testing.M) {
	io.SendAddress = serverAddress
	io.ReceiveAddress = serverAddress

	ServerData = api.TestInterface{
		LastReceivedMessage: pb.CmixMessage{},
	}

	go node.StartServer(serverAddress, &ServerData)

	os.Exit(m.Run())
}

// Make sure InitClient returns an error when called incorrectly.
func TestInitClientNil(t *testing.T) {
	err := InitClient(nil, "", nil)
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, nil) input!")
	}
	globals.LocalStorage = nil

	err = InitClient(nil, "hello", nil)
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, 'hello') input!")
	}
	globals.LocalStorage = nil
}

func TestInitClient(t *testing.T) {
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	if err != nil {
		t.Errorf("InitClient returned error: %v", err)
	}
	globals.LocalStorage = nil
}

func TestGetContactListJSON(t *testing.T) {
	user, _ := globals.Users.GetUser(1)
	nk := make([]globals.NodeKeys, 1)
	globals.Session = globals.NewUserSession(user, serverAddress, "", nk)
	// This call includes validating the JSON against the schema
	result, err := GetContactListJSON()

	if err != nil {
		t.Error(err.Error())
	}

	// But, just in case,
	// let's make sure that we got the error out of validateContactList anyway
	err = validateContactListJSON(result)

	if err != nil {
		t.Error(err.Error())
	}

	// Finally, make sure that all the names we expect are in the JSON
	// Ben's name should have changed to Snicklefritz
	expected := []string{"Ben", "Rick", "Jake", "Mario",
		"Allan", "David", "Jim", "Spencer", "Will", "Jono"}

	actual := string(result)

	for _, nick := range expected {
		if !strings.Contains(actual, nick) {
			t.Errorf("Error: Expected name %v wasn't in JSON %v", nick, actual)
		}
	}
}

func TestUpdateContactList(t *testing.T) {
	user, _ := globals.Users.GetUser(1)
	nk := make([]globals.NodeKeys, 1)
	globals.Session = globals.NewUserSession(user, serverAddress, "", nk)
	err := UpdateContactList()
	if err != nil {
		t.Error(err.Error())
	}

	result, err := GetContactListJSON()

	if err != nil {
		t.Error(err.Error())
	}

	// But, just in case,
	// let's make sure that we got the error out of validateContactList anyway
	err = validateContactListJSON(result)

	if err != nil {
		t.Error(err.Error())
	}

	// Finally, make sure that all the names we expect are in the JSON
	// Ben's name should have changed to Snicklefritz
	expected := []string{"Snicklefritz", "Jonwayne", "Rick", "Jake", "Mario",
		"Allan", "David", "Jim", "Spencer", "Will", "Jono"}

	actual := string(result)

	for _, nick := range expected {
		if !strings.Contains(actual, nick) {
			t.Errorf("Error: Expected name %v wasn't in JSON %v", nick, actual)
		}
	}
}

func TestValidateContactListJSON(t *testing.T) {
	err := validateContactListJSON(([]byte)("{invalidJSON:\"hmmm\"}"))
	if err == nil {
		t.Errorf("No error from invalid JSON")
	} else {
		t.Log(err.Error())
	}

	err = validateContactListJSON(([]byte)(`{"Nick":"Jono"}`))
	if err == nil {
		t.Errorf("No error from JSON that doesn't match the schema")
	} else {
		t.Log(err.Error())
	}
}

// BytesReceiver receives the last message and puts the data it received into
// byte slices
type BytesReceiver struct {
	receptionBuffer []byte
	lastSID         []byte
	lastRID         []byte
}

// This is the method that globals.Receive calls when you set a BytesReceiver
// as the global receiver
func (br *BytesReceiver) Receive(message Message) {
	br.receptionBuffer = append(br.receptionBuffer, message.GetPayload()...)
	br.lastRID = message.GetRecipient()
	br.lastSID = message.GetSender()
}

// This test creates a struct that implements the Receiver interface, then makes
// sure that that struct can receive a message when it's set as the global
// Receiver.
func TestReceiveMessageByInterface(t *testing.T) {
	// set up the receiver
	receiver := BytesReceiver{}
	err := InitClient(&globals.RamStorage{}, "", &receiver)
	if err != nil {
		t.Error(err.Error())
	}

	// set up the message
	payload := "hello there"
	senderID := cyclic.NewIntFromUInt(50).LeftpadBytes(format.SID_LEN)
	recipientID := cyclic.NewIntFromUInt(60).LeftpadBytes(format.RID_LEN)
	msg, err := format.NewMessage(cyclic.NewIntFromBytes(senderID).Uint64(),
		cyclic.NewIntFromBytes(recipientID).Uint64(), payload)
	if err != nil {
		t.Errorf("Couldn't create messages: %v", err.Error())
	}

	// receive the message
	globals.Receive(msg[0])

	// verify that the message was correctly received
	if !bytes.Equal(receiver.receptionBuffer, []byte(payload)) {
		t.Errorf("Message payload didn't match. Got: %v, expected %v",
			string(receiver.receptionBuffer), payload)
	}
	if !bytes.Equal(senderID, receiver.lastSID) {
		t.Errorf("Sender ID didn't match. Got: %v, expected %v",
			cyclic.NewIntFromBytes(receiver.lastSID).Uint64(),
			cyclic.NewIntFromBytes(senderID).Uint64())
	}
	if !bytes.Equal(recipientID, receiver.lastRID) {
		t.Errorf("Recipient ID didn't match. Got: %v, expected %v",
			cyclic.NewIntFromBytes(receiver.lastRID).Uint64(),
			cyclic.NewIntFromBytes(recipientID).Uint64())
	}
	globals.LocalStorage = nil
}

func TestRegister(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)

	regRes, err := Register(registrationCode, serverAddress, 1)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if regRes == nil || len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	globals.LocalStorage = nil
}

func TestRegisterBadNumNodes(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)

	_, err = Register(registrationCode, serverAddress, 0)
	if err == nil {
		t.Errorf("Registration worked with bad numnodes! %s", err.Error())
	}
	globals.LocalStorage = nil
}

func TestLogin(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)

	regRes, err := Register(registrationCode, serverAddress, 1)
	loginRes, err2 := Login(regRes, serverAddress)
	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}
	//Logout() -- we can't do this because some tests run in parallel and
	// it's not thread safe
	globals.LocalStorage = nil
}

func TestLogout(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "Nickname"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)

	regRes, err := Register(registrationCode, serverAddress, 1)
	Login(regRes, serverAddress)

	err2 := Logout()
	if err2 != nil {
		t.Errorf("Logoutfailed: %s", err.Error())
	}
	globals.LocalStorage = nil
}

func TestDisableBlockingTransmission(t *testing.T) {
	if !io.BlockTransmissions {
		t.Errorf("BlockingTransmission not intilized properly")
	}
	DisableBlockingTransmission()
	if io.BlockTransmissions {
		t.Errorf("BlockingTransmission not disabled properly")
	}
}

func TestSetRateLimiting(t *testing.T) {
	if io.TransmitDelay != time.Duration(1000)*time.Millisecond {
		t.Errorf("SetRateLimiting not intilized properly")
	}
	SetRateLimiting(10)
	if io.TransmitDelay != time.Duration(10)*time.Millisecond {
		t.Errorf("SetRateLimiting not updated properly")
	}
}
