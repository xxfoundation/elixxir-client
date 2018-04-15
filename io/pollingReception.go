////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/crypto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/comms/mixclient"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"sync/atomic"
	"time"
)

// This function continually polls for new messages for this client on the
// server.
func PollForMessages(wait time.Duration, quit globals.ThreadTerminator) {
	usr := globals.Session.GetCurrentUser()
	rqMsg := &pb.ClientPollMessage{UserID: usr.UserID}

	for len(quit) == 0 {
		time.Sleep(wait)
		cmixMsg, err := mixclient.SendClientPoll(globals.Session.GetNodeAddress(),
			rqMsg)

		// Skip process if we don't have content
		if err != nil {
			jww.WARN.Printf("SendClientPoll error during Polling: %v", err.Error())
			continue
		}
		if cmixMsg == nil || len(cmixMsg.MessagePayload) == 0 {
			continue
		}

		// Receive and decrypt a message
		msgBytes := format.MessageSerial{
			Payload:   cyclic.NewIntFromBytes(cmixMsg.MessagePayload),
			Recipient: cyclic.NewIntFromBytes(cmixMsg.RecipientID),
		}
		msg, err := crypto.Decrypt(globals.Grp, &msgBytes)
		if err != nil {
			jww.ERROR.Printf("Decryption failed: %v", err.Error())
			continue
		}

		err = globals.Receive(*msg)
		if err != nil {
			jww.ERROR.Printf(
				"Couldn't receive message using receiver: %s",
				err.Error())
		}
		atomic.AddUint64(&globals.ReceptionCounter, uint64(1))
	}

	// Signal to the thread terminator that I have finished.
	killNotify := <-quit
	close(quit)
	if killNotify != nil {
		killNotify <- true
	}
}

// Wrapper for pushfifo function, which may be deprecated soon.
func ReceiveFifo(message format.MessageInterface) {
	Msg := message.(format.Message)
	err := globals.Session.PushFifo(&Msg)
	if err != nil {
		jww.WARN.Printf("Error when calling PushFifo: %v", err.Error())
	}
}

//Starts the reception runner which waits "wait" between checks,
// and quits via the "quit" chan
func InitReceptionRunner(wait time.Duration,
	quit globals.ThreadTerminator) globals.ThreadTerminator {

	if quit == nil {
		quit = globals.NewThreadTerminator()
	}

	if !globals.UsingReceiver() {
		globals.SetReceiver(ReceiveFifo)
	}

	go PollForMessages(wait, quit)

	return quit
}
