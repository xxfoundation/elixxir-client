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
	"time"
)

func runfunc(wait uint64, quit globals.ThreadTerminator) {

	usr := globals.Session.GetCurrentUser()

	rqMsg := &pb.ClientPollMessage{UserID: usr.UserID}

	q := false

	var killNotify chan<- bool

	for !q {

		select {
		case killNotify = <-quit:
			q = true
		default:
			time.Sleep(time.Duration(wait) * time.Millisecond)

			cmixMsg, _ := mixclient.SendClientPoll(globals.Session.GetNodeAddress(), rqMsg)

			if len(cmixMsg.MessagePayload) != 0 {

				msgBytes := globals.MessageBytes{
					Payload:   cyclic.NewIntFromBytes(cmixMsg.MessagePayload),
					Recipient: cyclic.NewIntFromBytes(cmixMsg.RecipientID),
				}

				msg, err := crypto.Decrypt(globals.Grp, &msgBytes)

				if err != nil {
					jww.ERROR.Printf("Decryption failed: %v", err.Error())
				} else {
					err := globals.Session.PushFifo(msg)

					if err != nil {
						jww.ERROR.Printf("Could not push message onto FIFO,"+
							" message lost: %s",
							err.Error())
					}
				}

			}
		}
	}
	close(quit)

	if killNotify != nil {
		killNotify <- true
	}

}

//Starts the reception runner which waits "wait" between checks,
// and quits via the "quit" chan
func InitReceptionRunner(wait uint64,
	quit globals.ThreadTerminator) globals.ThreadTerminator {

	if quit == nil {
		quit = globals.NewThreadTerminator()
	}

	go runfunc(wait, quit)

	return quit
}
