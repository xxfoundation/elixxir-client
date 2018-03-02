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
)

func Login(userId int, serverAddress string) bool {
	isValidUser := globals.Session.Login(uint64(userId), serverAddress)
	if isValidUser {
		pollWaitTimeMillis := uint64(1000)
		io.InitReceptionRunner(pollWaitTimeMillis, globals.Session.GetNodeAddress())
	}
	return isValidUser
}

func Send(recipientID int, message string) {
	// NewMessage takes the ID of the sender, not the recipient
	sender := globals.Session.GetCurrentUser()
	// TODO: don't lose data with this type cast
	newMessages := globals.NewMessage(sender.Id, uint64(recipientID), message)

	// Prepare the new messages to be sent
	for _, newMessage := range (newMessages) {
		newMessageBytes := crypto.Encrypt(globals.Grp, newMessage)
		// Send the message
		io.TransmitMessage(globals.Session.GetNodeAddress(), newMessageBytes)
	}
}

func TryReceive() string {
	message := globals.Session.PopFifo()
	if message != nil {
		return message.GetPayloadString()
	} else {
		return ""
	}
}

func GetNick(userId int) string {
	user, ok := globals.Users.GetUser(uint64(userId))
	if ok && user != nil {
		return user.Nick
	} else {
		return ""
	}
}

// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
func Logout(userId int, serverAddress string) bool {
	io.Disconnect(serverAddress)
	return true
}
