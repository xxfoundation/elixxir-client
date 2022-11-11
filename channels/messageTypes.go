////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"strconv"
)

// MessageType is the type of message being sent to a channel.
type MessageType uint32

const (
	// Text is the default type for a message. It denotes that the message only
	// contains text.
	Text MessageType = 1

	// AdminText denotes that the message only contains text and that it comes
	// from the channel admin.
	AdminText MessageType = 2

	// Reaction denotes that the message is a reaction to another message.
	Reaction MessageType = 3
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
	default:
		return "Unknown messageType " + strconv.Itoa(int(mt))
	}
}
