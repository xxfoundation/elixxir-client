////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/crypto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/forward"
	"gitlab.com/privategrity/crypto/format"
	"math"
	"bytes"
)

// Initializes the client by registering a storage mechanism.
// If none is provided, the system defaults to using OS file access
// returns in error if it fails
func InitClient(s globals.Storage, loc string) error {

	var err error

	storeState := globals.InitStorage(s, loc)

	if !storeState {
		err = errors.New("could not init client")
	}

	globals.InitCrypto()

	return err
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func Register(HUID uint64, nick string, nodeAddr string,
	numNodes uint) (uint64, error) {

	var err error

	if numNodes < 1 {
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

	user, successGet := globals.Users.GetUser(UID)

	if !successGet {
		jww.ERROR.Printf("Register: UserID lookup failed")
		err = errors.New("could not register due to UserID lookup failure")
		return 0, err
	}

	if len(nick) > 36 || len(nick) < 1 {
		jww.ERROR.Printf("Register: Nickname too long")
		err = errors.New("could not register due to invalid nickname")
		return 0, err
	}

	user.Nick = nick
	io.SetNick(nodeAddr, user)

	nodekeys, successKeys := globals.Users.LookupKeys(user.UserID)
	nodekeys.PublicKey = cyclic.NewInt(0)

	if !successKeys {
		jww.ERROR.Printf("Register: could not find user keys")
		err = errors.New("could not register due to missing user keys")
		return 0, err
	}

	nk := make([]globals.NodeKeys, numNodes)

	for i := uint(0); i < numNodes; i++ {
		nk[i] = *nodekeys
	}

	nus := globals.NewUserSession(user, nodeAddr, nk)

	errStore := nus.StoreSession()

	if errStore != nil {
		err = errors.New(fmt.Sprintf(
			"Register: could not register due to failed session save"+
				": %s", errStore.Error()))
		jww.ERROR.Printf(err.Error())
		return 0, err
	}

	nus.Immolate()
	nus = nil

	return UID, err
}

// Logs in user and returns their nickname.
// returns an empty sting if login fails.
func Login(UID uint64) (string, error) {

	pollTerm := globals.NewThreadTerminator()

	err := globals.LoadSession(UID, pollTerm)

	if err != nil {
		err = errors.New(fmt.Sprintf("Login: Could not login: %s",
			err.Error()))
		jww.ERROR.Printf(err.Error())
		return "", err
	}

	pollWaitTimeMillis := uint64(1000)
	io.InitReceptionRunner(pollWaitTimeMillis, pollTerm)

	return globals.Session.GetCurrentUser().Nick, nil
}

func SetReceiver(receiver globals.Receiver) {
	globals.CurrentReceiver = receiver
}

func Send(message format.MessageInterface) error {

	if globals.Session == nil {
		err := errors.New("Send: Could not send when not logged in")
		jww.ERROR.Printf(err.Error())
		return err
	}

	// TODO: this could be a lot cleaner if we stored IDs as byte slices
	if bytes.Equal(message.GetSender(), cyclic.NewIntFromUInt(globals.Session.
		GetCurrentUser().UserID).LeftpadBytes(format.SID_LEN)) {
		err := errors.New("Send: Cannot send a message from someone other" +
			" than yourself")
		jww.ERROR.Printf(err.Error())
		return err
	}

	sender := globals.Session.GetCurrentUser()
	newMessages, _ := format.NewMessage(sender.UserID,
		cyclic.NewIntFromBytes(message.GetRecipient()).Uint64(),
		message.GetPayload())

	// Prepare the new messages to be sent
	for _, newMessage := range newMessages {
		newMessageBytes := crypto.Encrypt(globals.Grp, &newMessage)
		// Send the message
		err := io.TransmitMessage(globals.Session.GetNodeAddress(),
			newMessageBytes)
		// If we get an error, return it
		if err != nil {
			return err
		}
	}

	return nil
}

// Checks if there is a received message on the internal fifo.
// returns nil if there isn't.
func TryReceive() (format.MessageInterface, error) {

	var err error

	var m format.Message

	if globals.Session == nil {
		jww.ERROR.Printf("TryReceive: Could not receive when not logged in")
		err = errors.New("cannot receive when not logged in")
	} else {
		var message *format.Message
		message, err = globals.Session.PopFifo()

		if err == nil {
			messages, _ := format.NewMessage(message.GetSenderIDUint(),
				message.GetRecipientIDUint(), message.GetPayload())
			m = messages[0]
		}
	}

	return m, err
}

// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
func Logout() error {
	if globals.Session == nil {
		err := errors.New("Logout: Cannot Logout when you are not logged in" +
			" than yourself")
		jww.ERROR.Printf(err.Error())
		return err
	}

	io.Disconnect(globals.Session.GetNodeAddress())

	errStore := globals.Session.StoreSession()

	if errStore != nil {
		err := errors.New(fmt.Sprintf("Logout: Store Failed: %s" +
			errStore.Error()))
		jww.ERROR.Printf(err.Error())
		return err
	}

	errImmolate := globals.Session.Immolate()

	if errImmolate != nil {
		err := errors.New(fmt.Sprintf("Logout: Immolation Failed: %s" +
			errImmolate.Error()))
		jww.ERROR.Printf(err.Error())
		return err
	}

	return nil
}

func SetNick(UID uint64, nick string) error {
	u, success := globals.Users.GetUser(UID)

	if success {
		io.SetNick(globals.Session.GetNodeAddress(), u)
	} else {
		jww.ERROR.Printf("Tried to set nick for user %v, "+
			"but that user wasn't in the registry", u.UserID)
		return errors.New("That user wasn't in the user registry")
	}

	return nil
}

func UpdateContactList() error {
	return io.UpdateUserRegistry(globals.Session.GetNodeAddress())
}

func GetContactList() ([]uint64, []string) {
	return globals.Users.GetContactList()
}

func clearUint64(u *uint64) {
	*u = math.MaxUint64
	*u = 0
}

func DisableRatchet() {
	forward.SetRatchetStatus(false)
}
