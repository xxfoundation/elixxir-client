////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"gitlab.com/xx_network/primitives/id"
)

// Message is a message received from the cMix network in the clear
// or that has been decrypted using established E2E keys.
type Message interface {
	// Returns the message's sender ID, if available
	GetSender() id.ID
	GetSenderBytes() []byte

	// Returns the message payload/contents
	// Parse this with protobuf/whatever according to the message type
	GetPayload() []byte

	// Returns the message's recipient ID
	// This is usually your userID but could be an ephemeral/group ID
	GetRecipient() id.ID
	GetRecipientBytes() []byte

	// Returns the message's type
	GetMessageType() int32

	// Returns the message's timestamp in seconds since unix epoc
	GetTimestamp() int64
	// Returns the message's timestamp in ns since unix epoc
	GetTimestampNano() int64
}

// RoundEvent contains event information for a given round.
// TODO: This is a half-baked interface and will be filled out later.
type RoundEvent interface {
	// GetID returns the round ID for this round.
	GetID() int
	// GetStatus returns the status of this round.
	GetStatus() int
}
