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