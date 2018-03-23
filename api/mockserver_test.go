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
