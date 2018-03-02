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

func runfunc(wait uint64, addr string) {

	usr := globals.Session.GetCurrentUser()

	rqMsg := &pb.ClientPollMessage{UserID: usr.Id}
	for true {
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

func InitReceptionRunner(wait uint64, addr string) {
	go runfunc(wait, addr)
}
