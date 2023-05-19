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

	// ReplyType denotes that the message is a reaction to another message.
	ReplyType MessageType = 2

	// ReactionType denotes that the message is a reaction to another message.
	ReactionType MessageType = 3

	// SilentType denotes that the message is a silent message which should not
	// notify the user in any way.
	SilentType MessageType = 4
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
	default:
		return "Unknown messageType " + strconv.Itoa(int(mt))
	}
}
