////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"strconv"
)

// MessageType is the type of message being sent to a channel.
type MessageType uint32

const (
	// TextType is the default type for a message. It denotes that the
	// message only contains text.
	TextType MessageType = 1

	// ReplyType denotes that the message is a reply to another message.
	ReplyType MessageType = 2

	// ReactionType denotes that the message is a reaction to another message.
	ReactionType MessageType = 3

	// InvitationType denotes that the message is an invitation to another
	// channel.
	InvitationType MessageType = 4
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
	case InvitationType:
		return "Invitation"
	default:
		return "Unknown messageType " + strconv.Itoa(int(mt))
	}
}
