////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
		pb "gitlab.com/privategrity/comms/mixmessages"
		"gitlab.com/privategrity/comms/mixserver"
		"os"
		"testing"
		"bytes"
		"encoding/gob"
		"gitlab.com/privategrity/client/globals"
		"strconv"
		"gitlab.com/privategrity/crypto/cyclic"
		"gitlab.com/privategrity/crypto/format"
)

const SERVER_ADDRESS = "localhost:5556"

const NICK = "Alduin"

var Session globals.SessionObj

func TestMain(m *testing.M) {
	// Verifying the registration gob requires additional setup
	// Start server for testing
	go mixserver.StartServer(SERVER_ADDRESS, TestInterface{})

	// Put some user data into a gob
	globals.InitStorage(&globals.RamStorage{}, "")

	huid, _ := strconv.ParseUint("be50nhqpqjtjj", 32, 64)

	// populate a gob in the store
	Register(huid, NICK, SERVER_ADDRESS, 1)

	// get the gob out of there again
	sessionGob := globals.LocalStorage.Load()
	var sessionBytes bytes.Buffer
	sessionBytes.Write(sessionGob)
	dec := gob.NewDecoder(&sessionBytes)
	Session = globals.SessionObj{}
	dec.Decode(&Session)

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
	regRes, err := Register(hashUID, nick, SERVER_ADDRESS, 1)
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
	_, err = Register(hashUID, nick, SERVER_ADDRESS, 0)
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
	_, err = Register(hashUID, nick, SERVER_ADDRESS, 1)
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
	_, err = Register(hashUID, nick, SERVER_ADDRESS, 1)
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
	_, err = Register(hashUID, nick, SERVER_ADDRESS, 1)
	if err == nil {
		t.Errorf("Registration worked with invalid nickname! %s",
			err.Error())
	}
	globals.LocalStorage = nil
}

/*func TestRegisterDeletedKeys(t *testing.T) {
	registrationCode := "JHJ6L9BACDVC"
	nick := "test"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()

	_, err = Register(hashUID, nick, SERVER_ADDRESS, 1)
	if err == nil {
		t.Errorf("Registration worked with invalid nickname! %s",
			err.Error())
	}
	globals.LocalStorage = nil
}*/

func TestSend(t *testing.T) {
	globals.LocalStorage = nil
	registrationCode := "be50nhqpqjtjj"
	nick := "Nickname"
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	userID, err := Register(hashUID, nick, SERVER_ADDRESS, 1)
	loginRes, err2 := Login(userID)

	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}

	// Test send with invalid sender ID
	err = Send(APIMessage{SenderID: 12, Payload: "test",
		RecipientID: userID})
	if err == nil {
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
	registrationCode := "be50nhqpqjtjj"
	nick := "Nickname"

	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()
	userID, err := Register(hashUID, nick, SERVER_ADDRESS, 1)
	loginRes, err2 := Login(userID)

	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}
	// Push a message into the FIFO
	msg, _ := format.NewMessage(10, 10, "test")
	globals.Session.PushFifo(&msg[0])

	// Test receive with message in FIFO
	receivedMsg, err := TryReceive()
	if err != nil {
		t.Errorf("Could not receive a message from a nonempty FIFO.")
	}
	if cyclic.NewIntFromBytes(receivedMsg.GetRecipient()).Uint64() != 10 {
		t.Errorf("Recipient of received message is incorrect. " +
			"Expected: 10 Actual %v", cyclic.NewIntFromBytes(receivedMsg.
				GetRecipient()).Uint64())
	}
}

func TestSetNick(t *testing.T) {
	err := SetNick(0,"Guy")
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
	// Test send when logged out
	err = Send(APIMessage{"test", 5, 5})
	if err == nil {
		t.Errorf("Message was accepted by Send when not logged in.")
	}

	// Test receive when not logged in. Should return an error
	_, err = TryReceive()
	if err == nil {
		t.Errorf("Client tried to receive a message when not logged in.")
	}
}

// Blank struct implementing ServerHandler interface for testing purposes (Passing to StartServer)
type TestInterface struct{}

func (m TestInterface) NewRound(roundId string) {}

func (m TestInterface) SetPublicKey(roundId string, pkey []byte) {}

func (m TestInterface) PrecompDecrypt(message *pb.PrecompDecryptMessage) {}

func (m TestInterface) PrecompEncrypt(message *pb.PrecompEncryptMessage) {}

func (m TestInterface) PrecompReveal(message *pb.PrecompRevealMessage) {}

func (m TestInterface) PrecompPermute(message *pb.PrecompPermuteMessage) {}

func (m TestInterface) PrecompShare(message *pb.PrecompShareMessage) {}

func (m TestInterface) PrecompShareInit(message *pb.PrecompShareInitMessage) {}

func (m TestInterface) PrecompShareCompare(message *pb.
PrecompShareCompareMessage) {}

func (m TestInterface) PrecompShareConfirm(message *pb.
PrecompShareConfirmMessage) {}

func (m TestInterface) RealtimeDecrypt(message *pb.RealtimeDecryptMessage) {}

func (m TestInterface) RealtimeEncrypt(message *pb.RealtimeEncryptMessage) {}

func (m TestInterface) RealtimePermute(message *pb.RealtimePermuteMessage) {}

func (m TestInterface) ClientPoll(message *pb.ClientPollMessage) *pb.CmixMessage {
		return &pb.CmixMessage{}
}

func (m TestInterface) RequestContactList(message *pb.ContactPoll) *pb.
ContactMessage {
		return &pb.ContactMessage{}
}

var nick = "Mario"

func (m TestInterface) SetNick(message *pb.Contact) {
	nick = message.Nick
}

func (m TestInterface) ReceiveMessageFromClient(message *pb.CmixMessage) {}

// Mock dummy storage interface for testing.
type DummyStorage struct {
	Location string
	LastSave []byte
}

func (d *DummyStorage) SetLocation(l string) (error) {
	d.Location = l
	return nil
}

func (d *DummyStorage) GetLocation() string {
	return d.Location
}

func (d *DummyStorage) Save(b []byte) (error) {
	d.LastSave = make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		d.LastSave[i] = b[i]
	}
	return nil
}

func (d *DummyStorage) Load() []byte {
	return d.LastSave
}

type DummyReceiver struct {
	LastMessage APIMessage
}

func (d *DummyReceiver) Receive(message APIMessage) {
	d.LastMessage = message
}