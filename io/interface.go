////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// Communication interface implements send/receive functionality with the server
type Communications interface {
	// SendMessage to the server

	// TODO(nen) Can we get rid of the crypto type param here?
	SendMessage(session user.Session, topology *circuit.Circuit,
		recipientID *id.User, cryptoType parse.CryptoType, message []byte) error
	// SendMessage without partitions to the server
	// This is used to send rekey messages
	SendMessageNoPartition(session user.Session, topology *circuit.Circuit,
		recipientID *id.User, cryptoType parse.CryptoType, message []byte) error
	// MessageReceiver thread to get new messages
	MessageReceiver(session user.Session, delay time.Duration)
}
