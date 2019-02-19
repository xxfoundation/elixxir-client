////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// io sends and receives messages using gRPC
package io

import (
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// Communication interface implements send/receive functionality with the server
type Communications interface {
	// SendMessage to the server
	SendMessage(recipientID *id.User, message []byte) error
	// MessageReceiver thread to get new messages
	MessageReceiver(delay time.Duration)
}
