////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"math"
	"time"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// ValidForever is used as a validUntil lease when sending to denote the message
// or operation never expires.
//
// Note: A message relay must be present to enforce this otherwise things expire
// after 3 weeks due to network retention.
var ValidForever = time.Duration(math.MaxInt64)

// Manager provides an interface to manager channels.
type Manager interface {
	// GetIdentity returns the public identity associated with this channel
	// manager.
	GetIdentity() cryptoChannel.Identity

	// ExportPrivateIdentity encrypts and exports the private identity to a
	// portable string.
	ExportPrivateIdentity(password string) ([]byte, error)

	// GetStorageTag returns the tag at where this manager is stored. To be used
	// when loading the manager. The storage tag is derived from the public key.
	GetStorageTag() string

	// JoinChannel joins the given channel. It will fail if the channel has
	// already been joined.
	JoinChannel(channel *cryptoBroadcast.Channel) error

	// LeaveChannel leaves the given channel. It will return an error if the
	// channel was not previously joined.
	LeaveChannel(channelID *id.ID) error

	// EnableDirectMessageToken enables the token for direct messaging for this
	// channel.
	EnableDirectMessageToken(chId *id.ID) error

	// DisableDirectMessageToken removes the token for direct messaging for a
	// given channel.
	DisableDirectMessageToken(chId *id.ID) error

	// SendGeneric is used to send a raw message over a channel. In general, it
	// should be wrapped in a function that defines the wire protocol.
	//
	// If the final message, before being sent over the wire, is too long, this
	// will return an error. Due to the underlying encoding using compression,
	// it is not possible to define the largest payload that can be sent, but it
	// will always be possible to send a payload of 802 bytes at minimum.
	//
	// The meaning of validUntil depends on the use case.
	SendGeneric(channelID *id.ID, messageType MessageType,
		msg []byte, validUntil time.Duration, params cmix.CMIXParams) (
		cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error)

	// SendAdminGeneric is used to send a raw message over a channel encrypted
	// with admin keys, identifying it as sent by the admin. In general, it
	// should be wrapped in a function that defines the wire protocol.
	//
	// If the final message, before being sent over the wire, is too long, this
	// will return an error. The message must be at most 510 bytes long.
	SendAdminGeneric(privKey rsa.PrivateKey, channelID *id.ID,
		messageType MessageType, msg []byte, validUntil time.Duration,
		params cmix.CMIXParams) (cryptoChannel.MessageID,
		rounds.Round, ephemeral.Id, error)

	// SendMessage is used to send a formatted message over a channel.
	//
	// Due to the underlying encoding using compression, it is not possible to
	// define the largest payload that can be sent, but it will always be
	// possible to send a payload of 798 bytes at minimum.
	//
	// The message will auto delete validUntil after the round it is sent in,
	// lasting forever if ValidForever is used.
	SendMessage(channelID *id.ID, msg string, validUntil time.Duration,
		params cmix.CMIXParams) (
		cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error)

	// SendReply is used to send a formatted message over a channel.
	//
	// Due to the underlying encoding using compression, it is not possible to
	// define the largest payload that can be sent, but it will always be
	// possible to send a payload of 766 bytes at minimum.
	//
	// If the message ID that the reply is sent to does not exist, then the
	// other side will post the message as a normal message and not as a reply.
	//
	// The message will auto delete validUntil after the round it is sent in,
	// lasting forever if ValidForever is used.
	SendReply(channelID *id.ID, msg string, replyTo cryptoChannel.MessageID,
		validUntil time.Duration, params cmix.CMIXParams) (
		cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error)

	// SendReaction is used to send a reaction to a message over a channel. The
	// reaction must be a single emoji with no other characters, and will be
	// rejected otherwise.
	//
	// Clients will drop the reaction if they do not recognize the reactTo
	// message.
	SendReaction(channelID *id.ID, reaction string,
		reactTo cryptoChannel.MessageID, params cmix.CMIXParams) (
		cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error)

	// RegisterReceiveHandler is used to register handlers for non default
	// message types so that they can be processed by modules. It is important
	// that such modules sync up with the event model implementation.
	//
	// There can only be one handler per message type, and this will return an
	// error on a multiple registration.
	RegisterReceiveHandler(
		messageType MessageType, listener MessageTypeReceiveMessage) error

	// GetChannels returns the IDs of all channels that have been joined. Use
	// getChannelsUnsafe if you already have taken the mux.
	GetChannels() []*id.ID

	// GetChannel returns the underlying cryptographic structure for a given
	// channel.
	GetChannel(chID *id.ID) (*cryptoBroadcast.Channel, error)

	// ReplayChannel replays all messages from the channel within the network's
	// memory (~3 weeks) over the event model. It does this by wiping the
	// underlying state tracking for message pickup for the channel, causing all
	// messages to be re-retrieved from the network.
	ReplayChannel(chID *id.ID) error

	// SetNickname sets the nickname for a channel after checking that the
	// nickname is valid using [IsNicknameValid].
	SetNickname(newNick string, chID *id.ID) error

	// DeleteNickname removes the nickname for a given channel, using the
	// codename for that channel instead.
	DeleteNickname(chID *id.ID) error

	// GetNickname returns the nickname for the given channel, if it exists.
	GetNickname(chID *id.ID) (nickname string, exists bool)
}
