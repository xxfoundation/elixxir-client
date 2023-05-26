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
type MessageType uint16

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

	// Silent denotes that the message is a silent message which should not
	// notify the user in any way.
	Silent MessageType = 4

	// Invitation denotes that the message is an invitation to another channel.
	Invitation MessageType = 5

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

	// AdminReplay denotes that the message contains an admin message.
	AdminReplay MessageType = 104

	////////////////////////////////////////////////////////////////////////////
	// Extensions                                                             //
	////////////////////////////////////////////////////////////////////////////

	// FileTransfer denotes that a message contains the information about a file
	// download.
	FileTransfer MessageType = 40000
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
	case Silent:
		return "Silent"
	case Invitation:
		return "Invitation"
	case Delete:
		return "Delete"
	case Pinned:
		return "Pinned"
	case Mute:
		return "Mute"
	case AdminReplay:
		return "AdminReplay"
	case FileTransfer:
		return "FileTransfer"
	default:
		return "Unknown messageType " + strconv.Itoa(int(mt))
	}
}

// Bytes returns the [MessageType] as a 2-byte byte slice.
func (mt MessageType) Bytes() []byte {
	b := mt.Marshal()
	return b[:]
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
