////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/binary"
	"strconv"
)

// MessageType is the type of message being sent to a channel.
type MessageType uint32

const (
	////////////////////////////////////////////////////////////////////////////
	// Message Contents                                                       //
	////////////////////////////////////////////////////////////////////////////

	// Text is the default type for a message. It denotes that the message only
	// contains text.
	Text MessageType = 1

	// AdminText denotes that the message only contains text and that it comes
	// from the channel admin.
	AdminText MessageType = 2

	// Reaction denotes that the message is a reaction to another message.
	Reaction MessageType = 3

	////////////////////////////////////////////////////////////////////////////
	// Message Actions                                                        //
	////////////////////////////////////////////////////////////////////////////

	// Delete denotes that the message should be deleted. It is removed from the
	// database and deleted from the user's view.
	Delete MessageType = 101

	// Pinned denotes that the message should be pinned to the channel.
	Pinned MessageType = 102

	// Mute denotes that any future messages from the user are hidden. The
	// messages are still received, but they are not visible.
	Mute MessageType = 103
)

// String returns a human-readable version of [MessageType], used for debugging
// and logging. This function adheres to the [fmt.Stringer] interface.
func (mt MessageType) String() string {
	switch mt {
	case Text:
		return "Text"
	case AdminText:
		return "AdminText"
	case Reaction:
		return "Reaction"
	case Delete:
		return "Delete"
	case Pinned:
		return "Pinned"
	case Mute:
		return "Mute"
	default:
		return "Unknown messageType " + strconv.Itoa(int(mt))
	}
}

// Bytes returns the MessageType as a 4-bit byte slice.
func (mt MessageType) Bytes() []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(mt))
	return b
}
