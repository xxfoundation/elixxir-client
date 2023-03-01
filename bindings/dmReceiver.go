///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"crypto/ed25519"
	"time"

	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/dm"
	"gitlab.com/elixxir/crypto/message"
)

// DMReceiver is an interface which an external party which uses the dm
// system passed an object which adheres to in order to get events on the
// channel.
type DMReceiver interface {
	// Receive is called when a raw direct message is received
	// with unkown type. It may be called multiple times on the
	// same message. It is incumbent on the user of the API to
	// filter such called by message ID.
	//
	// The api user must interpret the message type and perform
	// their own message parsing.
	//
	// Parameters:
	//  - messageID - The bytes of the [dm.MessageID] of the received
	//    message.
	//  - nickname - The nickname of the sender of the message.
	//  - text - The bytes content of the message.
	//  - timestamp - Time the message was received; represented
	//    as nanoseconds since unix epoch.
	//  - partnerKey - The partners's Ed25519 public key. This is
	//    required to respond.
	//  - senderKey - The sender's Ed25519 public key.
	//  - dmToken - The senders direct messaging token. This is
	//    required to respond.
	//  - codeset - The codeset version.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundId - The ID of the round that the message was received on.
	//  - mType - the type of the message, always 1 for this call
	//  - status - the [dm.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns a non-negative unique UUID for the message that it can be
	// referenced by later with [EventModel.UpdateSentStatus].
	Receive(messageID []byte, nickname string, text []byte,
		partnerKey, senderKey []byte,
		dmToken int32, codeset int, timestamp,
		roundId, mType, status int64) int64

	// ReceiveText is called whenever a direct message is
	// received that is a text type. It may be called multiple times
	// on the same message. It is incumbent on the user of the API
	// to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply in theory can
	// arrive before the initial message. As a result, it may be
	// important to buffer replies.
	//
	// Parameters:
	//  - messageID - The bytes of the [dm.MessageID] of the received
	//    message.
	//  - nickname - The nickname of the sender of the message.
	//  - text - The content of the message.
	//  - partnerKey - The partners's Ed25519 public key. This is
	//    required to respond.
	//  - senderKey - The sender's Ed25519 public key.
	//  - dmToken - The senders direct messaging token. This is
	//    required to respond.
	//  - codeset - The codeset version.
	//  - timestamp - Time the message was received; represented
	//    as nanoseconds since unix epoch.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundId - The ID of the round that the message was received on.
	//  - status - the [dm.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns a non-negative unique UUID for the message that it can be
	// referenced by later with [EventModel.UpdateSentStatus].
	ReceiveText(messageID []byte, nickname, text string,
		partnerKey, senderKey []byte,
		dmToken int32, codeset int, timestamp,
		roundId, status int64) int64

	// ReceiveReply is called whenever a direct message is
	// received that is a reply. It may be called multiple times
	// on the same message. It is incumbent on the user of the API
	// to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply in theory can
	// arrive before the initial message. As a result, it may be
	// important to buffer replies.
	//
	// Parameters:
	//  - messageID - The bytes of the [dm.MessageID] of the received
	//    message.
	//  - reactionTo - The [dm.MessageID] for the message
	//    that received a reply.
	//  - nickname - The nickname of the sender of the message.
	//  - text - The content of the message.
	//  - partnerKey - The partners's Ed25519 public key. This is
	//    required to respond.
	//  - senderKey - The sender's Ed25519 public key.
	//  - dmToken - The senders direct messaging token. This is
	//    required to respond.
	//  - codeset - The codeset version.
	//  - timestamp - Time the message was received; represented
	//    as nanoseconds since unix epoch.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundId - The ID of the round that the message was received on.
	//  - status - the [dm.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns a non-negative unique UUID for the message that it can be
	// referenced by later with [EventModel.UpdateSentStatus].
	ReceiveReply(messageID, reactionTo []byte, nickname,
		text string, partnerKey, senderKey []byte,
		dmToken int32, codeset int,
		timestamp, roundId, status int64) int64

	// ReceiveReaction is called whenever a reaction to a direct
	// message is received. It may be called multiple times on the
	// same reaction.  It is incumbent on the user of the API to
	// filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply in theory can
	// arrive before the initial message. As a result, it may be
	// important to buffer reactions.
	//
	// Parameters:
	//  - messageID - The bytes of the [dm.MessageID] of the received
	//    message.
	//  - reactionTo - The [dm.MessageID] for the message
	//    that received a reply.
	//  - nickname - The nickname of the sender of the message.
	//  - reaction - The contents of the reaction message.
	//  - partnerKey - The partners's Ed25519 public key. This is
	//    required to respond.
	//  - senderKey - The sender's Ed25519 public key.
	//  - dmToken - The senders direct messaging token. This is
	//    required to respond.
	//  - codeset - The codeset version.
	//  - timestamp - Time the message was received; represented
	//    as nanoseconds since unix epoch.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundId - The ID of the round that the message was received on.
	//  - status - the [dm.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns a non-negative unique uuid for the message by which it can be
	// referenced later with UpdateSentStatus
	ReceiveReaction(messageID, reactionTo []byte,
		nickname, reaction string, partnerKey, senderKey []byte,
		dmToken int32,
		codeset int, timestamp, roundId,
		status int64) int64

	// UpdateSentStatus is called whenever the sent status of a message has
	// changed.
	//
	// Parameters:
	//  - messageID - The bytes of the [dm.MessageID] of the received
	//    message.
	//  - status - the [dm.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	UpdateSentStatus(uuid int64, messageID []byte, timestamp, roundID,
		status int64)
}

// dmReceiver is a wrapper which wraps an existing DMReceiver object and
// implements [dm.Receiver]
type dmReceiver struct {
	dr DMReceiver
}

// NewDMReceiver is a constructor for a dmReceiver. This will take in an
// DMReceiver and wraps it around the dmReceiver.
func NewDMReceiver(dr DMReceiver) dm.EventModel {
	return &dmReceiver{dr: dr}
}

// Receive is called whenever a direct message is received.
// It may be called multiple times on the same message. It is incumbent on the
// user of the API to filter such called by message ID.
func (dmr *dmReceiver) Receive(messageID message.ID,
	nickname string, text []byte, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32, codeset uint8, timestamp time.Time,
	round rounds.Round, mType dm.MessageType,
	status dm.Status) uint64 {

	return uint64(dmr.dr.Receive(messageID[:], nickname,
		text, partnerKey, senderKey, int32(dmToken), int(codeset),
		timestamp.UnixNano(), int64(round.ID),
		int64(mType), int64(status)))
}

// Receive is called whenever a direct message is received.
// It may be called multiple times on the same message. It is incumbent on the
// user of the API to filter such called by message ID.
func (dmr *dmReceiver) ReceiveText(messageID message.ID,
	nickname, text string, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32, codeset uint8, timestamp time.Time,
	round rounds.Round,
	status dm.Status) uint64 {

	return uint64(dmr.dr.ReceiveText(messageID[:], nickname,
		text, partnerKey, senderKey, int32(dmToken), int(codeset),
		timestamp.UnixNano(), int64(round.ID), int64(status)))
}

// ReceiveReply is called whenever a direct message is received that
// is a reply. It may be called multiple times on the same message. It
// is incumbent on the user of the API to filter such called by
// message ID.
//
// Messages may arrive our of order, so a reply in theory can arrive before the
// initial message. As a result, it may be important to buffer replies.
func (dmr *dmReceiver) ReceiveReply(messageID message.ID,
	reactionTo message.ID, nickname, text string,
	partnerKey, senderKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, status dm.Status) uint64 {

	return uint64(dmr.dr.ReceiveReply(messageID[:], reactionTo[:],
		nickname, text, partnerKey, senderKey, int32(dmToken),
		int(codeset),
		timestamp.UnixNano(), int64(round.ID), int64(status)))

}

// ReceiveReaction is called whenever a reaction to a direct message
// is received. It may be called multiple times on the same
// reaction. It is incumbent on the user of the API to filter such
// called by message ID.
//
// Messages may arrive our of order, so a reply in theory can arrive before the
// initial message. As a result, it may be important to buffer reactions.
func (dmr *dmReceiver) ReceiveReaction(messageID message.ID,
	reactionTo message.ID, nickname, reaction string,
	partnerKey, senderKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status dm.Status) uint64 {

	return uint64(dmr.dr.ReceiveReaction(messageID[:],
		reactionTo[:], nickname, reaction, partnerKey, senderKey,
		int32(dmToken),
		int(codeset), timestamp.UnixNano(),
		int64(round.ID), int64(status)))
}

// UpdateSentStatus is called whenever the sent status of a message has changed.
func (dmr *dmReceiver) UpdateSentStatus(uuid uint64,
	messageID message.ID, timestamp time.Time, round rounds.Round,
	status dm.Status) {
	dmr.dr.UpdateSentStatus(int64(uuid), messageID[:], timestamp.UnixNano(),
		int64(round.ID), int64(status))
}
