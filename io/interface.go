////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// io sends and receives messages using gRPC
package io

import (
	"time"
	"gitlab.com/elixxir/primitives/userid"
)

// Communication interface implements send/receive functionality with the server
type Communications interface {
	// SendMessage to the server
	SendMessage(recipientID *id.UserID, message []byte) error
	// MessageReceiver thread to get new messages
	MessageReceiver(delay time.Duration)
}
