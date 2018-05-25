////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// io sends and receives messages using gRPC
package io

import (
	"gitlab.com/privategrity/crypto/format"
	"time"
)

// Communication interface implements send/receive functionality with the server
type Communications interface {
	// SendMessage to the server
	SendMessage(recipientID uint64, message string) error
	// Listen for messages from a given sender
	Listen(senderID uint64) chan *format.Message
	// StopListening to a given listener (closes and deletes)
	StopListening(listenerCh chan *format.Message)
	// MessageReceiver thread to get new messages
	MessageReceiver(delay time.Duration)
}
