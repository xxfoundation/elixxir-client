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
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	sessionImport "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
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

	// Send sends a message to all GroupChat members using Cmix.SendManyCMIX.
	// The send fails if the message is too long. Returns the ID of the round
	// sent on and the timestamp of the message send.
	Send(groupID *id.ID, tag string, message []byte) (
		id.Round, time.Time, group.MessageID, error)

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
	AddService(tag string, p Processor) error

	// RemoveService removes all services for the given tag.
	RemoveService(tag string) error
}

// RequestCallback is called when a GroupChat request is received.
type RequestCallback func(g gs.Group)

// ReceiveCallback is called when a GroupChat message is received.
type ReceiveCallback func(msg MessageReceive)

////////////////////////////////////////////////////////////////////////////////////
// Sub-interfaces from other packages //////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////

// groupE2e is a sub-interface mocking the xxdk.E2e object.
// This contains methods specific for this package.
type groupE2e interface {
	GetCmix() cmix.Client
	GetE2E() e2e.Handler
	GetReceptionIdentity() xxdk.ReceptionIdentity
	GetRng() *fastRNG.StreamGenerator
	GetStorage() storage.Session
}

// groupCmix is a subset of the cmix.Client interface containing only the
// methods needed by GroupChat
type groupCmix interface {
	SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (
		id.Round, []ephemeral.Id, error)
	AddService(
		clientID *id.ID, newService message.Service, response message.Processor)
	DeleteService(
		clientID *id.ID, toDelete message.Service, processor message.Processor)
	GetMaxMessageLength() int
}

// groupE2eHandler is a subset of the e2e.Handler interface containing only the methods
// needed by GroupChat
type groupE2eHandler interface {
	SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte,
		params e2e.Params) (e2e.SendReport, error)
	RegisterListener(senderID *id.ID, messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID
	AddService(tag string, processor message.Processor) error
	AddPartner(partnerID *id.ID, partnerPubKey, myPrivKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey, mySIDHPrivKey *sidh.PrivateKey,
		sendParams, receiveParams sessionImport.Params) (partner.Manager, error)
	GetPartner(partnerID *id.ID) (partner.Manager, error)
	GetHistoricalDHPubkey() *cyclic.Int
	GetHistoricalDHPrivkey() *cyclic.Int
}
