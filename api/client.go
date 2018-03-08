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
	"errors"
	"fmt"
	"gitlab.com/privategrity/crypto/cyclic"
)

//Structure used to return a message
type APIMessage struct{
	Sender 	uint64
	Payload string
	Recipient uint64
}

// Initializes the client by registering a storage mechanism.
// If none is provided, the system defaults to using OS file access
// returns in error if it fails
func InitClient(s globals.Storage, loc string)(error){

	var err error

	storeState := globals.InitStorage(s, loc)

	if !storeState{
		err = errors.New("could not init client")
	}

	globals.InitCrypto()

	return err
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func Register(HUID uint64, nick string, nodeAddr string,
	numNodes uint)(uint64, error){

	var err error

	if numNodes<1{
		jww.ERROR.Printf("Register: Invalid number of nodes")
		err = errors.New("could not register due to invalid number of nodes")
		return 0, err
	}

	UID, successLook := globals.Users.LookupUser(HUID)
	defer clearUint64(&UID)

	if !successLook {
		jww.ERROR.Printf("Register: HUID does not match")
		err = errors.New("could not register due to invalid HUID")
		return 0, err
	}

	user, successGet  := globals.Users.GetUser(UID)

	if !successGet {
		jww.ERROR.Printf("Register: UID lookup failed")
		err = errors.New("could not register due to UID lookup failure")
		return 0, err
	}

	if len(nick) > 36 || len(nick)<1{
		jww.ERROR.Printf("Register: Nickname too long")
		err = errors.New("could not register due to invalid nickname")
		return 0, err
	}

	user.Nick = nick

	nodekeys, successKeys := globals.Users.LookupKeys(user.UID)
	nodekeys.PublicKey = cyclic.NewInt(0)

	fmt.Println(nodekeys)

	if !successKeys {
		jww.ERROR.Printf("Register: could not find user keys")
		err = errors.New("could not register due to missing user keys")
		return 0, err
	}

	nk := make([]globals.NodeKeys,numNodes)

	for i:=uint(0);i<numNodes;i++{
		nk[i] = *nodekeys
	}

	nus := globals.NewUserSession(user, nodeAddr, nk)

	successStore := nus.StoreSession()

	if !successStore{
		jww.ERROR.Printf("Register: unable to save session")
		err = errors.New("could not register due to failed session save")
		return 0, err
	}

	nus.Immolate()
	nus = nil

	//TODO: Register nickname with contacts server

	return UID, err
}

// Logs in user and returns their nickname.
// returns an empty sting if login fails.
func Login(UID uint64) (string, error) {

	pollTerm := globals.NewThreadTerminator()

	success := globals.LoadSession(UID, pollTerm)

	if !success {
		jww.ERROR.Printf("Login: Could not login")
		return "", errors.New("could not login")
	}

	pollWaitTimeMillis := uint64(1000)
	io.InitReceptionRunner(pollWaitTimeMillis, pollTerm)

	return globals.Session.GetCurrentUser().Nick, nil
}

func Send(message APIMessage) (error){

	if globals.Session == nil {
		jww.ERROR.Printf("Send: Could not send when not logged in")
		return errors.New("cannot send message when not logged in")
	}

	if message.Sender != globals.Session.GetCurrentUser().UID {
		jww.ERROR.Printf("Send: Cannot send a message from someone other" +
			" than yourself")
		return errors.New("cannot send message from a different user")
	}

	sender := globals.Session.GetCurrentUser()
	newMessages := globals.NewMessage(sender.UID, message.Recipient, message.Payload)

	// Prepare the new messages to be sent
	for _, newMessage := range newMessages {
		newMessageBytes := crypto.Encrypt(globals.Grp, newMessage)
		// Send the message
		io.TransmitMessage(globals.Session.GetNodeAddress(), newMessageBytes)
	}

	return nil
}

// Checks if there is a received message on the internal fifo.
// returns nil if there isn't.
func TryReceive() (APIMessage, error) {

	var err error

	var m APIMessage

	if globals.Session == nil {
		jww.ERROR.Printf("TryReceive: Could not receive when not logged in")
		err = errors.New("cannot receive when not logged in")
	}else{
		message := globals.Session.PopFifo()
		if message != nil {
			m.Payload = message.GetPayloadString()
			m.Sender = message.GetSenderID().Uint64()
			m.Recipient = message.GetRecipientID().Uint64()
		}
	}

	return m, err
}


// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
func Logout() error {
	if globals.Session == nil {
		jww.ERROR.Printf("Logout: Cannot Logout when you are not logged in")
		return errors.New("cannot logout when you are not logged in")
	}

	io.Disconnect(globals.Session.GetNodeAddress())

	successStore := globals.Session.StoreSession()

	if !successStore {
		jww.ERROR.Printf("Logout: Store Failed")
		return errors.New("cannot logout because state could not be saved")
	}

	successImmolate := globals.Session.Immolate()

	if !successImmolate{
		jww.ERROR.Printf("Logout: Immolation Failed")
		return errors.New("cannot logout because ram could not be cleared")
	}

	return nil
}

func clearUint64(u *uint64){
	*u = math.MaxUint64
	*u = 0
}