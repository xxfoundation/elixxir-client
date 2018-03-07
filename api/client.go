////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"gitlab.com/privategrity/client/crypto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	jww "github.com/spf13/jwalterweatherman"
	"math"
)

//Structure used to return a message
type APIMessage struct{
	Sender 	uint64
	Payload string
	Recipient uint64
}

// Initializes the client by registering a storage mechanism.
// If none is provided, the system defaults to using OS file access
func InitClient(s globals.Storage, loc string)(bool){

	storeState := globals.InitStorage(s, loc)

	if !storeState{
		jww.ERROR.Printf("could not init client")
	}

	return storeState
}

//Registers user and returns the User ID.  Returns 0 if registration fails.
func Register(HUID uint64, nick string, nodeAddr string,
	numNodes int)(uint64){

	if numNodes<1{
		jww.ERROR.Printf("Register: Invalid number of nodes")
		return 0
	}

	UID, successLook := globals.Users.LookupUser(HUID)
	defer clearUint64(&UID)

	if !successLook {
		jww.ERROR.Printf("Register: HUID does not match")
		return 0
	}

	user, successGet  := globals.Users.GetUser(UID)

	if !successGet {
		jww.ERROR.Printf("Register: UID lookup failed")
		return 0
	}

	if len(nick) > 36 {
		jww.ERROR.Printf("Register: Nickname too long")
		return 0
	}

	user.Nick = nick

	nodekeys, successKeys := globals.Users.LookupKeys(user.UID)

	if !successKeys {
		jww.ERROR.Printf("Register: could not find user keys")
		return 0
	}

	nk := make([]globals.NodeKeys,numNodes)

	for i:=0;i<numNodes;i++{
		nk[i] = *nodekeys
	}

	nus := globals.NewUserSession(user, nodeAddr, nk)

	successStore := nus.StoreSession()

	if !successStore{
		jww.ERROR.Printf("Register: unable to save session")
		return 0
	}



	nus.Immolate()
	nus = nil


	//TODO: Register nickname

	return UID
}

func Login(UID uint64) bool {
	success := globals.LoadSession(UID)

	if !success {
		jww.ERROR.Printf("Login: Could not login")
		return false
	}
	return true
}

func Send(message APIMessage) (bool){

	if globals.Session == nil {
		jww.ERROR.Printf("Send: Could not send when not logged in")
		return false
	}

	if message.Sender != globals.Session.GetCurrentUser().UID {
		jww.ERROR.Printf("Send: Cannot send a message from someone other" +
			" than yourself")
		return false
	}

	sender := globals.Session.GetCurrentUser()
	newMessages := globals.NewMessage(sender.UID, message.Recipient, message.Payload)

	// Prepare the new messages to be sent
	for _, newMessage := range newMessages {
		newMessageBytes := crypto.Encrypt(globals.Grp, newMessage)
		// Send the message
		io.TransmitMessage(globals.Session.GetNodeAddress(), newMessageBytes)
	}

	return true
}

// Checks if there is a received message on the internal fifo.
// returns nil if there isn't.
func TryReceive() APIMessage {

	var m APIMessage

	if globals.Session == nil {
		jww.ERROR.Printf("TryReceive: Could not receive when not logged in")
	}else{
		message := globals.Session.PopFifo()
		if message != nil {
			m.Payload = message.GetPayloadString()
			m.Sender = message.GetSenderID().Uint64()
			m.Recipient = message.GetRecipientID().Uint64()
		}
	}

	return m
}


// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
func Logout() bool {
	if globals.Session == nil {
		jww.ERROR.Printf("Logout: Cannot Logout when you are not logged in")
		return false
	}

	io.Disconnect(globals.Session.GetNodeAddress())

	successStore := globals.Session.StoreSession()

	if !successStore {
		jww.ERROR.Printf("Logout: Store Failed")
		return false
	}

	successImmolate := globals.Session.Immolate()

	if !successImmolate{
		jww.ERROR.Printf("Logout: Immolation Failed")
		return false
	}

	return true
}

func clearUint64(u *uint64){
	*u = math.MaxUint64
	*u = 0
}