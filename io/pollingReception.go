////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/client/crypto"
	"gitlab.com/privategrity/client/globals"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixclient"
	"time"
	"gitlab.com/privategrity/crypto/cyclic"
)

func runfunc(wait uint64, addr string, quit chan chan bool) {

	usr := globals.Session.GetCurrentUser()

	rqMsg := &pb.ClientPollMessage{UserID: usr.UID}

	q := false

	var killNotify chan<- bool

	for !q {

		select{
			case killNotify = <-quit:
				q = true
			default:
				time.Sleep(time.Duration(wait) * time.Millisecond)

				cmixMsg, _ := mixclient.SendClientPoll(addr, rqMsg)

				if len(cmixMsg.MessagePayload) != 0 {

					msgBytes := globals.MessageBytes{
						Payload:      cyclic.NewIntFromBytes(cmixMsg.MessagePayload),
						PayloadMIC:   cyclic.NewInt(0),
						Recipient:    cyclic.NewIntFromBytes(cmixMsg.RecipientID),
						RecipientMIC: cyclic.NewInt(0),
					}

					msg := crypto.Decrypt(globals.Grp, &msgBytes)

					globals.Session.PushFifo(msg)
				}
		}

	}

	close(quit)

	killNotify <- true

}

func InitReceptionRunner(wait uint64, addr string)(chan chan bool) {

	quit := make (chan chan bool)

	go runfunc(wait, addr, quit)

	return quit
}
