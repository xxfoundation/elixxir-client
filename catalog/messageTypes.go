////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package catalog

import "fmt"

type MessageType uint32

const MessageTypeLen = 32 / 8

const (
	/*general message types*/

	// NoType - Used as a wildcard for listeners to listen to all existing types.
	// Think of it as "No type in particular"
	NoType MessageType = 0

	// XxMessage - Type of message sent by the xx messenger.
	XxMessage = 2

	/*End to End Rekey message types*/

	// KeyExchangeTrigger - Trigger a rekey, this message is used locally in
	// client only
	KeyExchangeTrigger = 30
	// KeyExchangeConfirm - Rekey confirmation message. Sent by partner to
	//confirm completion of a rekey
	KeyExchangeConfirm = 31

	// KeyExchangeTriggerEphemeral - Trigger a rekey, this message is used
	//locally in client only. For ephemeral only e2e instances.
	KeyExchangeTriggerEphemeral = 32
	// KeyExchangeConfirmEphemeral - Rekey confirmation message. Sent by partner
	// to confirm completion of a rekey. For ephemeral only e2e instances.
	KeyExchangeConfirmEphemeral = 33

	// E2eClose message is sent when a user deletes a partner and wants to
	// inform their partner that the connection is closed.
	E2eClose MessageType = 34

	/* Group chat message types */

	// GroupCreationRequest - A group chat request message sent to all members in a group.
	GroupCreationRequest = 40

	// NewFileTransfer is transmitted first on the initialization of a file
	// transfer to inform the receiver about the incoming file.
	NewFileTransfer MessageType = 50

	// EndFileTransfer is sent once all file parts have been transmitted to
	// inform the receiver that the file transfer has ended.
	EndFileTransfer MessageType = 51

	// ConnectionAuthenticationRequest is sent by the recipient
	// of an authenticated connection request
	// (see the connect/ package)
	ConnectionAuthenticationRequest = 60

	PaymentRequest = 70
	PaymentConf    = 71
)

func (mt MessageType) String() string {
	switch mt {
	case NoType:
		return "NoType"
	case XxMessage:
		return "XxMessage"
	case KeyExchangeTrigger:
		return "KeyExchangeTrigger"
	case KeyExchangeConfirm:
		return "KeyExchangeConfirm"
	case KeyExchangeTriggerEphemeral:
		return "KeyExchangeTriggerEphemeral"
	case KeyExchangeConfirmEphemeral:
		return "KeyExchangeConfirmEphemeral"
	case E2eClose:
		return "E2eClose"
	case GroupCreationRequest:
		return "GroupCreationRequest"
	case NewFileTransfer:
		return "NewFileTransfer"
	case EndFileTransfer:
		return "EndFileTransfer"
	case ConnectionAuthenticationRequest:
		return "ConnectionAuthenticationRequest"
	case PaymentRequest:
		return "PaymentRequest"
	default:
		return fmt.Sprintf("UNKNOWN TYPE (%d)", mt)
	}
}
