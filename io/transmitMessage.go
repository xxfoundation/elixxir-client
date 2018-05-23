////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/comms/client"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/format"
	"time"
)

type transmitFunc func(addr string, messageBytes *pb.CmixMessage) error

// Send a cMix message to the server
func TransmitMessage(addr string, messageBytes *format.MessageSerial) error {
	oldTransmit := func(addr string, cmixmsg *pb.CmixMessage) error {
		_, err := client.SendMessageToServer(addr, cmixmsg)
		return err
	}
	err := Transmit(oldTransmit, addr, messageBytes)
	return err
}

// Send a cMix message to the gateway
func TransmitMessageGW(addr string, messageBytes *format.MessageSerial) error {
	err := Transmit(client.SendPutMessage, addr, messageBytes)
	return err
}

func Transmit(tFunc transmitFunc, addr string,
	messageBytes *format.MessageSerial) error {
	if globals.BlockingTransmission {
		globals.TransmissionMutex.Lock()
	}

	cmixmsg := &pb.CmixMessage{
		SenderID:       globals.Session.GetCurrentUser().UserID,
		MessagePayload: messageBytes.Payload.Bytes(),
		RecipientID:    messageBytes.Recipient.Bytes(),
	}

	err := tFunc(addr, cmixmsg)

	if globals.BlockingTransmission {
		time.Sleep(globals.TransmitDelay)
		globals.TransmissionMutex.Unlock()
	}

	return err
}
