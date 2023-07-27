////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"encoding/binary"
	"strconv"
)

// MessageType is the type of message being sent to a channel.
type MessageType uint16

const (
	// TextType is the default type for a message. It denotes that the
	// message only contains text.
	TextType MessageType = 1

	// ReplyType denotes that the message is a reply to another message.
	ReplyType MessageType = 2

	// ReactionType denotes that the message is a reaction to another message.
	ReactionType MessageType = 3

	// SilentType denotes that the message is a silent message which should not
	// notify the user in any way.
	SilentType MessageType = 4

	// InvitationType denotes that the message is an invitation to another
	// channel.
	InvitationType MessageType = 5

	// DeleteType denotes that the message contains the ID of a message to
	// delete.
	DeleteType MessageType = 6
)

// String returns a human-readable version of [MessageType], used for debugging
// and logging. This function adheres to the [fmt.Stringer] interface.
func (mt MessageType) String() string {
	switch mt {
	case TextType:
		return "Text"
	case ReplyType:
		return "Reply"
	case ReactionType:
		return "Reaction"
	case SilentType:
		return "Silent"
	case InvitationType:
		return "Invitation"
	case DeleteType:
		return "Delete"
	default:
		return "Unknown messageType " + strconv.Itoa(int(mt))
	}
}

// Marshal returns the byte representation of the [MessageType].
func (mt MessageType) Marshal() [2]byte {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(mt))
	return b
}

// UnmarshalMessageType returns the MessageType from its byte representation.
func UnmarshalMessageType(b [2]byte) MessageType {
	return MessageType(binary.LittleEndian.Uint16(b[:]))
}

