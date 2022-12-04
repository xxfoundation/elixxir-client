////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"time"

	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
)

// Client the direct message client implements a Listener and Sender interface.
type Client interface {
	Sender
	Listener
}

// Sender implemntors allow the API user to send to a given partner over
// cMix.
type Sender interface {
	// SendText is used to send a formatted message to another user.
	SendText(partnerPubKey *ed25519.PublicKey, partnerToken []byte,
		msg string, params cmix.CMIXParams) (
		MessageID, rounds.Round, ephemeral.Id, error)

	// SendReply is used to send a formatted direct message reply.
	//
	// If the message ID that the reply is sent to does not exist,
	// then the other side will post the message as a normal
	// message and not as a reply.
	SendReply(partnerPubKey *ed25519.PublicKey, partnerToken []byte,
		msg string, replyTo MessageID,
		params cmix.CMIXParams) (MessageID, rounds.Round,
		ephemeral.Id, error)

	// SendReaction is used to send a reaction to a direct
	// message. The reaction must be a single emoji with no other
	// characters, and will be rejected otherwise.
	//
	// Clients will drop the reaction if they do not recognize the reactTo
	// message.
	SendReaction(partnerPubKey *ed25519.PublicKey, partnerToken []byte,
		reaction string, reactTo MessageID,
		params cmix.CMIXParams) (MessageID, rounds.Round,
		ephemeral.Id, error)

	// Send is used to send a raw message. In general, it
	// should be wrapped in a function that defines the wire protocol.
	//
	// If the final message, before being sent over the wire, is
	// too long, this will return an error. Due to the underlying
	// encoding using compression, it is not possible to define
	// the largest payload that can be sent, but it will always be
	// possible to send a payload of 802 bytes at minimum.
	Send(partnerPubKey *ed25519.PublicKey, partnerToken []byte,
		messageType MessageType, plaintext []byte,
		params cmix.CMIXParams) (MessageID,
		rounds.Round, ephemeral.Id, error)
}

// Listener allows API users to register a Receiver to receive DMs.
type Listener interface {
	// Register registers a listener for direct messages.
	Register(receiver Receiver, checkSent messageReceiveFunc) error

	// TODO: These unimplemented at this time.
	// BlockDMs disables DMs from a specific user. Received messages
	// will be dropped during event processing.
	// BlockDMs(partnerPubKey *ed25519.PublicKey, dmToken []byte) error
	// UnblockDMs enables DMs from a specific user.
	// UnblockDMs(conversationID *id.ID) error
}

// DMReceiverBuilder initialises the event model using the given path.
type ReceiverBuilder func(path string) (Receiver, error)

// Receiver is all of the reception functions an API user must implement.
// This is similar to the event model system in channels.
type Receiver interface {
	// Receive is called whenever a raw direct message is
	// received. It may be called multiple times on the same
	// message. It is incumbent on the user of the API to filter
	// such called by message ID.
	//
	// Receive includes the message Type so that the implementor
	// can determine what to do with the message.
	//
	// The API needs to return a UUID of the message that can be
	// referenced at a later time.
	//
	// messageID, timestamp, and round are all nillable and may be
	// updated based upon the UUID at a later date. A time of
	// time.Time{} will be passed for a nilled timestamp.
	//
	// Nickname may be empty, in which case the UI is expected to
	// display the codename.
	//
	// Message type is included in the call; it will always be
	// Text (1) for this call, but it may be required in
	// downstream databases.
	Receive(messageID MessageID,
		nickname string, text []byte, pubKey ed25519.PublicKey,
		dmToken []byte,
		codeset uint8, timestamp time.Time,
		round rounds.Round, mType MessageType, status Status) uint64

	// Receive is called whenever a direct message is
	// received. It may be called multiple times on the same
	// message. It is incumbent on the user of the API to filter
	// such called by message ID.
	//
	// The API needs to return a UUID of the message that can be
	// referenced at a later time.
	//
	// messageID, timestamp, and round are all nillable and may be
	// updated based upon the UUID at a later date. A time of
	// time.Time{} will be passed for a nilled timestamp.
	//
	// Nickname may be empty, in which case the UI is expected to
	// display the codename.
	//
	// Message type is included in the call; it will always be
	// Text (1) for this call, but it may be required in
	// downstream databases.
	ReceiveText(messageID MessageID,
		nickname, text string, pubKey ed25519.PublicKey, dmToken []byte,
		codeset uint8, timestamp time.Time,
		round rounds.Round, status Status) uint64

	// ReceiveReply is called whenever a direct message is
	// received that is a reply. It may be called multiple times
	// on the same message. It is incumbent on the user of the API
	// to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply, in theory,
	// can arrive before the initial message. As a result, it may
	// be important to buffer replies.
	//
	// The API needs to return a UUID of the message that can be
	// referenced at a later time.
	//
	// messageID, timestamp, and round are all nillable and may be
	// updated based upon the UUID at a later date. A time of
	// time.Time{} will be passed for a nilled timestamp.
	//
	// Nickname may be empty, in which case the UI is expected to
	// display the codename.
	//
	// Message type is included in the call; it will always be
	// Text (1) for this call, but it may be required in
	// downstream databases.
	ReceiveReply(messageID MessageID,
		reactionTo MessageID, nickname, text string,
		pubKey ed25519.PublicKey, dmToken []byte, codeset uint8,
		timestamp time.Time, round rounds.Round,
		status Status) uint64

	// ReceiveReaction is called whenever a reaction to a direct
	// message is received. It may be called multiple times on the
	// same reaction. It is incumbent on the user of the API to
	// filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply, in theory,
	// can arrive before the initial message. As a result, it may
	// be important to buffer replies.
	//
	// The API needs to return a UUID of the message that can be
	// referenced at a later time.
	//
	// messageID, timestamp, and round are all nillable and may be
	// updated based upon the UUID at a later date. A time of
	// time.Time{} will be passed for a nilled timestamp.
	//
	// Nickname may be empty, in which case the UI is expected to
	// display the codename.
	//
	// Message type is included in the call; it will always be
	// Text (1) for this call, but it may be required in
	// downstream databases.
	ReceiveReaction(messageID MessageID,
		reactionTo MessageID, nickname, reaction string,
		pubKey ed25519.PublicKey, dmToken []byte, codeset uint8,
		timestamp time.Time, round rounds.Round,
		status Status) uint64

	// UpdateSentStatus is called whenever the sent status of a message has
	// changed.
	//
	// messageID, timestamp, and round are all nillable and may be updated based
	// upon the UUID at a later date. A time of time.Time{} will be passed for a
	// nilled timestamp. If a nil value is passed, make no update.
	UpdateSentStatus(uuid uint64, messageID MessageID,
		timestamp time.Time, round rounds.Round, status Status)
}

// cmixClient are the required cmix functions we need for direct messages
type cMixClient interface {
	GetMaxMessageLength() int
	SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
		cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	AddIdentityWithHistory(
		id *id.ID, validUntil, beginning time.Time, persistent bool)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	RemoveIdentity(id *id.ID)
	GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback,
		roundList ...id.Round)
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
}

// nickNameManager interface is an object that handles the mapping of nicknames
// to cMix reception IDs.
type nickNameManager interface {
	// GetNick gets a nickname associated with this DM partner (reception)
	// ID.
	GetNickname(id *id.ID) (string, bool)
}
