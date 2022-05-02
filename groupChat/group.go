///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Group chat is used to communicate the same content with multiple clients over
// cMix. A group chat is controlled by a group leader who creates the group,
// defines all group keys, and is responsible for key rotation. To create a
// group, the group leader must have an authenticated channel with all members
// of the group.
//
// Once a group is created, neither the leader nor other members can add or
// remove users to the group. Only members can leave a group themselves.
//
// When a message is sent to the group, the sender will send an individual
// message to every member of the group.

package groupChat

import (
	"gitlab.com/elixxir/client/cmix/message"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// GroupChat is used to send and receive cMix messages to/from multiple users.
type GroupChat interface {
	// MakeGroup sends GroupChat requests to all members over an authenticated
	// channel. The leader of a GroupChat must have an authenticated channel
	// with each member of the GroupChat to add them to the GroupChat. It blocks
	// until all the GroupChat requests are sent. Returns the new group and the
	// round IDs the requests were sent on. Returns an error if at least one
	// request to a member fails to send. Also returns the status of the sent
	// requests.
	MakeGroup(membership []*id.ID, name, message []byte) (gs.Group, []id.Round,
		RequestStatus, error)

	// ResendRequest allows a GroupChat request to be sent again. It returns
	// the rounds that the requests were sent on and the status of the send.
	ResendRequest(groupID *id.ID) ([]id.Round, RequestStatus, error)

	// JoinGroup allows a user to accept a GroupChat request and stores the
	// GroupChat as active to allow receiving and sending of messages from/to
	// the GroupChat. A user can only join a GroupChat once.
	JoinGroup(g gs.Group) error

	// LeaveGroup removes a group from a list of groups the user is a part of.
	LeaveGroup(groupID *id.ID) error

	// Send sends a message to all GroupChat members using Client.SendManyCMIX.
	// The send fails if the message is too long. Returns the ID of the round
	// sent on and the timestamp of the message send.
	Send(groupID *id.ID, message []byte) (id.Round, time.Time, error)

	// GetGroups returns a list of all registered GroupChat IDs.
	GetGroups() []*id.ID

	// GetGroup returns the group with the matching ID or returns false if none
	// exist.
	GetGroup(groupID *id.ID) (gs.Group, bool)

	// NumGroups returns the number of groups the user is a part of.
	NumGroups() int

	/* ===== Services ======================================================= */

	// AddService adds a service for all group chat partners of the given tag,
	// which will call back on the given processor.
	AddService(tag string, processor message.Processor) error

	// RemoveService removes all services for the given tag.
	RemoveService(tag string) error
}

// RequestCallback is called when a GroupChat request is received.
type RequestCallback func(g gs.Group)

// ReceiveCallback is called when a GroupChat message is received.
type ReceiveCallback func(msg MessageReceive)
