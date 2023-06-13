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
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/fastRNG"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/nike"
)

// Client the direct message client implements a Listener and Sender interface.
type Client interface {
	Sender

	// GetPublicKey returns the public key of this client.
	GetPublicKey() nike.PublicKey

	// GetToken returns the DM token of this client.
	GetToken() uint32

	// GetIdentity returns the public identity associated with this client.
	GetIdentity() codename.Identity

	// ExportPrivateIdentity encrypts and exports the private identity to a
	// portable string.
	ExportPrivateIdentity(password string) ([]byte, error)

	// BlockPartner prevents receiving messages and notifications from the
	// partner.
	BlockPartner(partnerPubKey ed25519.PublicKey)

	// UnblockPartner unblocks a blocked sender to allow DM messages.
	UnblockPartner(partnerPubKey ed25519.PublicKey)

	// IsBlocked indicates if the given partner is blocked.
	IsBlocked(partnerPubKey ed25519.PublicKey) bool

	// GetBlockedPartners returns all partners who are blocked by this user.
	GetBlockedPartners() []ed25519.PublicKey

	// GetNotificationLevel returns the notification level for the given channel.
	GetNotificationLevel(
		partnerPubKey ed25519.PublicKey) (NotificationLevel, error)

	// SetMobileNotificationsLevel sets the notification level for the given DM
	// conversation partner.
	SetMobileNotificationsLevel(
		partnerPubKey ed25519.PublicKey, level NotificationLevel) error

	NickNameManager
}

// Sender implementers allow the API user to send to a given partner over
// cMix.
type Sender interface {
	// SendText is used to send a formatted message to another user.
	SendText(partnerPubKey ed25519.PublicKey, partnerToken uint32,
		msg string, params cmix.CMIXParams) (
		cryptoMessage.ID, rounds.Round, ephemeral.Id, error)

	// SendReply is used to send a formatted direct message reply.
	//
	// If the message ID that the reply is sent to does not exist,
	// then the other side will post the message as a normal
	// message and not as a reply.
	SendReply(partnerPubKey ed25519.PublicKey, partnerToken uint32,
		msg string, replyTo cryptoMessage.ID,
		params cmix.CMIXParams) (cryptoMessage.ID, rounds.Round,
		ephemeral.Id, error)

	// SendReaction is used to send a reaction to a direct
	// message. The reaction must be a single emoji with no other
	// characters, and will be rejected otherwise.
	//
	// Clients will drop the reaction if they do not recognize the reactTo
	// message.
	SendReaction(partnerPubKey ed25519.PublicKey, partnerToken uint32,
		reaction string, reactTo cryptoMessage.ID,
		params cmix.CMIXParams) (cryptoMessage.ID, rounds.Round,
		ephemeral.Id, error)

	// SendSilent is used to send to a channel a message with no notifications.
	// Its primary purpose is to communicate new nicknames without calling
	// SendMessage.
	//
	// It takes no payload intentionally as the message should be very
	// lightweight.
	SendSilent(partnerPubKey ed25519.PublicKey, partnerToken uint32,
		params cmix.CMIXParams) (
		cryptoMessage.ID, rounds.Round, ephemeral.Id, error)

	// Send is used to send a raw message. In general, it
	// should be wrapped in a function that defines the wire protocol.
	//
	// If the final message, before being sent over the wire, is
	// too long, this will return an error. Due to the underlying
	// encoding using compression, it is not possible to define
	// the largest payload that can be sent, but it will always be
	// possible to send a payload of 802 bytes at a minimum.
	Send(partnerPubKey ed25519.PublicKey, partnerToken uint32,
		messageType MessageType, plaintext []byte,
		params cmix.CMIXParams) (cryptoMessage.ID,
		rounds.Round, ephemeral.Id, error)
}

// ReceiverBuilder initialises the event model using the given path.
type ReceiverBuilder func(path string) (EventModel, error)

// EventModel is all of the reception functions an API user must implement.
// This is similar to the event model system in channels.
type EventModel interface {
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
	Receive(messageID cryptoMessage.ID,
		nickname string, text []byte,
		partnerPubKey, senderPubKey ed25519.PublicKey,
		partnerToken uint32,
		codeset uint8, timestamp time.Time,
		round rounds.Round, mType MessageType, status Status) uint64

	// ReceiveText is called whenever a direct message is
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
	ReceiveText(messageID cryptoMessage.ID,
		nickname, text string,
		partnerPubKey, senderPubKey ed25519.PublicKey,
		partnerToken uint32,
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
	ReceiveReply(messageID cryptoMessage.ID,
		reactionTo cryptoMessage.ID, nickname, text string,
		partnerPubKey, senderPubKey ed25519.PublicKey,
		partnerToken uint32, codeset uint8,
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
	ReceiveReaction(messageID cryptoMessage.ID,
		reactionTo cryptoMessage.ID, nickname, reaction string,
		partnerPubKey, senderPubKey ed25519.PublicKey,
		partnerToken uint32, codeset uint8,
		timestamp time.Time, round rounds.Round,
		status Status) uint64

	// UpdateSentStatus is called whenever the sent status of a message has
	// changed.
	//
	// messageID, timestamp, and round are all nillable and may be
	// updated based upon the UUID at a later date. A time of
	// time.Time{} will be passed for a nilled timestamp. If a nil
	// value is passed, make no update.
	UpdateSentStatus(uuid uint64, messageID cryptoMessage.ID,
		timestamp time.Time, round rounds.Round, status Status)

	// GetConversation returns any conversations held by the
	// model (receiver)
	GetConversation(senderPubKey ed25519.PublicKey) *ModelConversation
	// GetConversations returns any conversations held by the
	// model (receiver)
	GetConversations() []ModelConversation
}

// cmixClient are the required cmix functions we need for direct messages
type cMixClient interface {
	GetMaxMessageLength() int
	SendManyWithAssembler(recipients []*id.ID, assembler cmix.ManyMessageAssembler,
		params cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error)
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool,
		fallthroughProcessor message.Processor)
	AddIdentityWithHistory(
		id *id.ID, validUntil, beginning time.Time, persistent bool,
		fallthroughProcessor message.Processor)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteClientService(clientID *id.ID)
	RemoveIdentity(id *id.ID)
	GetRoundResults(timeout time.Duration,
		roundCallback cmix.RoundEventCallback, roundList ...id.Round)
	AddHealthCallback(f func(bool)) uint64
	RemoveHealthCallback(uint64)
}

// NotificationsManager contains the methods from [notifications.Manager] that
// are required by the [Manager].
type NotificationsManager interface {
	Set(toBeNotifiedOn *id.ID, group string, metadata []byte,
		status clientNotif.NotificationState) error
	Get(toBeNotifiedOn *id.ID) (status clientNotif.NotificationState,
		metadata []byte, group string, exists bool)
	Delete(toBeNotifiedOn *id.ID) error
	RegisterUpdateCallback(group string, nu clientNotif.Update)
}

// NickNameManager interface is an object that handles the mapping of nicknames
// to cMix reception IDs.
type NickNameManager interface {
	// GetNickname gets the nickname associated with this DM user.
	GetNickname() (string, bool)
	// SetNickname sets the nickname to use for this user.
	SetNickname(nick string) error
}

// SendTracker provides facilities for tracking sent messages
type SendTracker interface {
	// Init is used by the DM Client to register trigger and
	// update functions and start send tracking
	Init(net cMixClient, trigger triggerEventFunc,
		updateStatus updateStatusFunc, rng *fastRNG.StreamGenerator)

	// DenotePendingSend registers a new message to be tracked for sending
	DenotePendingSend(partnerPublicKey, senderPubKey ed25519.PublicKey,
		partnerToken uint32,
		messageType MessageType,
		msg *DirectMessage) (uuid uint64, err error)

	// FailedSend marks a message failed
	FailedSend(uuid uint64) error

	// Sent marks a message successfully Sent
	Sent(uuid uint64, msgID cryptoMessage.ID, round rounds.Round) error

	// CheckIfSent checks if the given message was a sent message
	CheckIfSent(messageID cryptoMessage.ID, r rounds.Round) bool

	// Delivered marks a message delivered
	Delivered(msgID cryptoMessage.ID, round rounds.Round) bool

	// StopTracking stops tracking a message
	StopTracking(msgID cryptoMessage.ID, round rounds.Round) bool
}
