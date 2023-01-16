////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for gateways

package storage

import (
	"context"
	"crypto/ed25519"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gorm.io/gorm"
	"time"
)

const (
	// Can be provided to SqlLite to create a temporary, in-memory DB.
	temporaryDbPath = "file::memory:?cache=shared"

	// Determines maximum runtime (in seconds) of DB queries.
	dbTimeout = 3 * time.Second
)

// newContext builds a context for database operations.
func newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), dbTimeout)
}

// JoinChannel is called whenever a channel is joined locally.
// Creates the Channel.
func (i *impl) JoinChannel(channel *cryptoBroadcast.Channel) {
	parentErr := errors.New("failed to JoinChannel")

	newChannel := Channel{
		Id:          channel.ReceptionID.Marshal(),
		Name:        channel.Name,
		Description: channel.Description,
	}

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Create(newChannel).Error
	cancel()

	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to create Channel: %+v", err))
		return
	}
	jww.DEBUG.Printf("Successfully joined channel: %s", channel.ReceptionID)
}

// LeaveChannel is called whenever a channel is left locally.
// Deletes all Message associated with the given Channel.
func (i *impl) LeaveChannel(channelID *id.ID) {
	parentErr := errors.New("failed to LeaveChannel")

	// Also deletes associated Messages due to CASCADE
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Delete(Channel{Id: channelID.Marshal()}).Error
	cancel()

	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to delete Channel: %+v", err))
		return
	}
	jww.DEBUG.Printf("Successfully deleted channel: %s", channelID)
}

// ReceiveMessage is called whenever a message is received on a given channel.
// Creates the Message.
func (i *impl) ReceiveMessage(channelID *id.ID, messageID message.ID, nickname,
	text string, pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus, hidden bool) uint64 {
	textBytes := []byte(text)
	var err error

	if i.cipher != nil {
		textBytes, err = i.cipher.Encrypt([]byte(text))
		if err != nil {
			jww.ERROR.Printf("Failed to encrypt message: %+v", err)
			return 0
		}
	}

	msgToInsert := buildMessage(
		channelID.Marshal(), messageID.Bytes(), nil, nickname,
		textBytes, pubKey, dmToken, codeset, timestamp, lease, round.ID,
		messageType, false, hidden, status)

	ctx, cancel := newContext()
	err = i.db.WithContext(ctx).Create(msgToInsert).Error
	cancel()

	if err != nil {
		jww.ERROR.Printf("Failed to receive message: %+v", err)
	}
	return msgToInsert.Id
}

// ReceiveReply is called whenever a message is received that is a reply on a
// given channel. Creates the Message.
func (i *impl) ReceiveReply(channelID *id.ID, messageID, reactionTo message.ID,
	nickname, text string, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus, hidden bool) uint64 {
	textBytes := []byte(text)
	var err error

	if i.cipher != nil {
		textBytes, err = i.cipher.Encrypt([]byte(text))
		if err != nil {
			jww.ERROR.Printf("Failed to encrypt reply: %+v", err)
			return 0
		}
	}

	msgToInsert := buildMessage(
		channelID.Marshal(), messageID.Bytes(), reactionTo.Bytes(), nickname,
		textBytes, pubKey, dmToken, codeset, timestamp, lease, round.ID,
		messageType, false, hidden, status)

	ctx, cancel := newContext()
	err = i.db.WithContext(ctx).Create(msgToInsert).Error
	cancel()

	if err != nil {
		jww.ERROR.Printf("Failed to receive reply: %+v", err)
	}
	return msgToInsert.Id
}

// ReceiveReaction is called whenever a reaction to a message is received on a
// given channel. Creates the Message.
func (i *impl) ReceiveReaction(channelID *id.ID, messageID, reactionTo message.ID,
	nickname, reaction string, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus, hidden bool) uint64 {
	textBytes := []byte(reaction)
	var err error

	if i.cipher != nil {
		textBytes, err = i.cipher.Encrypt([]byte(reaction))
		if err != nil {
			jww.ERROR.Printf("Failed to encrypt reaction: %+v", err)
			return 0
		}
	}

	msgToInsert := buildMessage(
		channelID.Marshal(), messageID.Bytes(), reactionTo.Bytes(), nickname,
		textBytes, pubKey, dmToken, codeset, timestamp, lease, round.ID,
		messageType, false, hidden, status)

	ctx, cancel := newContext()
	err = i.db.WithContext(ctx).Create(msgToInsert).Error
	cancel()

	if err != nil {
		jww.ERROR.Printf("Failed to receive reaction: %+v", err)
	}
	return msgToInsert.Id
}

// UpdateFromUUID is called whenever a message at the UUID is modified.
//
// messageID, timestamp, round, pinned, and hidden are all nillable and may be
// updated based upon the UUID at a later date. If a nil value is passed, then
// make no update.
func (i *impl) UpdateFromUUID(uuid uint64, messageID *message.ID, timestamp *time.Time,
	round *rounds.Round, pinned, hidden *bool, status *channels.SentStatus) {
	parentErr := errors.New("failed to UpdateFromMessageID")

	msgToUpdate := buildMessage(
		nil, messageID.Bytes(), nil, "",
		nil, nil, 0, 0, *timestamp, 0, 0,
		0, *pinned, *hidden, *status)
	if round != nil {
		msgToUpdate.Round = uint64(round.ID)
	}
	msgToUpdate.Id = uuid

	// When updating with struct it will only update non-zero fields by default
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Updates(msgToUpdate).Error
	cancel()

	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to create Channel: %+v", err))
	}
}

// UpdateFromMessageID is called whenever a message with the message ID is
// modified.
//
// The API needs to return the UUID of the modified message that can be
// referenced at a later time.
//
// timestamp, round, pinned, and hidden are all nillable and may be updated
// based upon the UUID at a later date. If a nil value is passed, then make
// no update.
func (i *impl) UpdateFromMessageID(messageID message.ID, timestamp *time.Time,
	round *rounds.Round, pinned, hidden *bool, status *channels.SentStatus) uint64 {
	parentErr := errors.New("failed to UpdateFromMessageID")

	msgToUpdate := buildMessage(
		nil, messageID.Bytes(), nil, "",
		nil, nil, 0, 0, *timestamp, 0, 0,
		0, *pinned, *hidden, *status)
	if round != nil {
		msgToUpdate.Round = uint64(round.ID)
	}

	// Build a transaction to prevent race conditions
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		currentMessage := &Message{}
		err := tx.Take(currentMessage, "message_id = ?", messageID.Bytes()).Error
		if err != nil {
			return err
		}

		// When updating with struct it will only update non-zero fields by default
		msgToUpdate.Id = currentMessage.Id
		return tx.Updates(msgToUpdate).Error
	})
	cancel()

	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to create Channel: %+v", err))
	}
	return msgToUpdate.Id
}

// GetMessage returns the [channels.ModelMessage] with the given [message.ID].
func (i *impl) GetMessage(messageID message.ID) (channels.ModelMessage, error) {
	result := &Message{}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(result, "message_id = ?",
		messageID.Bytes()).Error
	cancel()
	if err != nil {
		return channels.ModelMessage{}, err
	}

	var channelId *id.ID
	if result.ChannelId != nil {
		channelId, err = id.Unmarshal(result.ChannelId)
		if err != nil {
			return channels.ModelMessage{}, err
		}
	}

	var parentMsgId message.ID
	if result.ParentMessageId != nil {
		parentMsgId, err = message.UnmarshalID(result.ParentMessageId)
		if err != nil {
			return channels.ModelMessage{}, err
		}
	}

	return channels.ModelMessage{
		UUID:            result.Id,
		Nickname:        result.Nickname,
		MessageID:       messageID,
		ChannelID:       channelId,
		ParentMessageID: parentMsgId,
		Timestamp:       result.Timestamp,
		Lease:           result.Lease,
		Status:          channels.SentStatus(result.Status),
		Hidden:          result.Hidden,
		Pinned:          result.Pinned,
		Content:         result.Text,
		Type:            channels.MessageType(result.Type),
		Round:           id.Round(result.Round),
		PubKey:          result.Pubkey,
		CodesetVersion:  result.CodesetVersion,
	}, nil
}

// MuteUser is called whenever a user is muted or unmuted.
func (i *impl) MuteUser(channelID *id.ID, pubKey ed25519.PublicKey, unmute bool) {
	i.muteCb(channelID, pubKey, unmute)
}

// DeleteMessage removes a message with the given messageID from storage.
func (i *impl) DeleteMessage(messageID message.ID) error {
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Where("message_id = ?",
		messageID.Bytes()).Delete(&Message{}).Error
	cancel()

	if err != nil {
		return errors.Errorf("Unable to delete Message: %+v", err)
	}
	return nil
}

// buildMessage is a private helper that converts typical [channels.EventModel]
// inputs into a basic Message structure for insertion into storage.
//
// NOTE: ID is not set inside this function because we want to use the
// autoincrement key by default. If you are trying to overwrite an existing
// message, then you need to set it manually yourself.
func buildMessage(channelID, messageID, parentID []byte, nickname string,
	text []byte, pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, lease time.Duration, round id.Round,
	mType channels.MessageType, pinned, hidden bool,
	status channels.SentStatus) *Message {
	return &Message{
		MessageId:       messageID,
		Nickname:        nickname,
		ChannelId:       channelID,
		ParentMessageId: parentID,
		Timestamp:       timestamp,
		Lease:           lease,
		Status:          uint8(status),
		Hidden:          hidden,
		Pinned:          pinned,
		Text:            text,
		Type:            uint16(mType),
		Round:           uint64(round),
		// User Identity Info
		Pubkey:         pubKey,
		DmToken:        dmToken,
		CodesetVersion: codeset,
	}
}
