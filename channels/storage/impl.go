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
	go i.eventUpdate(channels.ChannelUpdate, channels.ChannelUpdateJson{
		ChannelID: channel.ReceptionID,
		Deleted:   false,
	})
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
	go i.eventUpdate(channels.ChannelUpdate, channels.ChannelUpdateJson{
		ChannelID: channelID,
		Deleted:   true,
	})
}

// ReceiveMessage is called whenever a message is received on a given channel.
// Creates the Message.
func (i *impl) ReceiveMessage(channelID *id.ID, messageID message.ID, nickname,
	text string, pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus, hidden bool) uint64 {
	uuid, err := i.receiveHelper(channelID, messageID, nil,
		nickname, text, pubKey, dmToken, codeset, timestamp, lease,
		round, messageType, status, hidden)
	if err != nil {
		jww.ERROR.Printf("Failed to receive message: %+v", err)
	}
	return uuid
}

// ReceiveReply is called whenever a message is received that is a reply on a
// given channel. Creates the Message.
func (i *impl) ReceiveReply(channelID *id.ID, messageID, reactionTo message.ID,
	nickname, text string, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus, hidden bool) uint64 {
	uuid, err := i.receiveHelper(channelID, messageID, reactionTo.Bytes(),
		nickname, text, pubKey, dmToken, codeset, timestamp, lease,
		round, messageType, status, hidden)
	if err != nil {
		jww.ERROR.Printf("Failed to receive reply: %+v", err)
	}
	return uuid
}

// ReceiveReaction is called whenever a reaction to a message is received on a
// given channel. Creates the Message.
func (i *impl) ReceiveReaction(channelID *id.ID, messageID, reactionTo message.ID,
	nickname, reaction string, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus, hidden bool) uint64 {
	uuid, err := i.receiveHelper(channelID, messageID, reactionTo.Bytes(),
		nickname, reaction, pubKey, dmToken, codeset, timestamp, lease,
		round, messageType, status, hidden)
	if err != nil {
		jww.ERROR.Printf("Failed to receive message: %+v", err)
	}
	return uuid
}

// UpdateFromUUID is called whenever a message at the UUID is modified.
//
// messageID, timestamp, round, pinned, and hidden are all nillable and may be
// updated based upon the UUID at a later date. If a nil value is passed, then
// make no update.
//
// Returns an error if the message cannot be updated. It must return
// channels.NoMessageErr if the message does not exist.
func (i *impl) UpdateFromUUID(uuid uint64, messageID *message.ID, timestamp *time.Time,
	round *rounds.Round, pinned, hidden *bool, status *channels.SentStatus) error {
	parentErr := "failed to UpdateFromUUID"

	msgToUpdate := &Message{
		Id:     int64(uuid),
		Hidden: hidden,
		Pinned: pinned,
	}
	if messageID != nil {
		msgToUpdate.MessageId = messageID.Marshal()
	}
	if round != nil {
		msgToUpdate.Round = int64(round.ID)
	}
	if timestamp != nil {
		msgToUpdate.Timestamp = *timestamp
	}
	if status != nil {
		msgToUpdate.Status = uint8(*status)
	}
	currentMessage := &Message{Id: msgToUpdate.Id}

	// Build a transaction to prevent race conditions
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Take(currentMessage).Error
		if err != nil {
			return err
		}

		// When updating with struct it will only update non-zero fields by default
		return tx.Updates(msgToUpdate).Error
	})
	cancel()

	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return errors.WithMessage(channels.NoMessageErr, parentErr)
		}
		return errors.WithMessage(err, parentErr)
	}
	channelId := &id.ID{}
	copy(channelId[:], currentMessage.ChannelId)

	go i.eventUpdate(channels.MessageReceived, channels.MessageReceivedJson{
		Uuid:      msgToUpdate.Id,
		ChannelID: channelId,
		Update:    true,
	})

	return nil
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
//
// Returns an error if the message cannot be updated. It must return
// channels.NoMessageErr if the message does not exist.
func (i *impl) UpdateFromMessageID(messageID message.ID, timestamp *time.Time,
	round *rounds.Round, pinned, hidden *bool, status *channels.SentStatus) (
	uint64, error) {
	parentErr := "failed to UpdateFromMessageID"

	msgToUpdate := &Message{
		MessageId: messageID.Marshal(),
		Hidden:    hidden,
		Pinned:    pinned,
	}
	if round != nil {
		msgToUpdate.Round = int64(round.ID)
	}
	if timestamp != nil {
		msgToUpdate.Timestamp = *timestamp
	}
	if status != nil {
		msgToUpdate.Status = uint8(*status)
	}
	currentMessage := &Message{}

	// Build a transaction to prevent race conditions
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Take(currentMessage, "message_id = ?", messageID.Marshal()).Error
		if err != nil {
			return err
		}

		// When updating with struct it will only update non-zero fields by default
		msgToUpdate.Id = currentMessage.Id
		return tx.Updates(msgToUpdate).Error
	})
	cancel()

	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return 0, errors.WithMessage(channels.NoMessageErr, parentErr)
		}
		return 0, errors.WithMessage(err, parentErr)
	}
	channelId := &id.ID{}
	copy(channelId[:], currentMessage.ChannelId)

	go i.eventUpdate(channels.MessageReceived, channels.MessageReceivedJson{
		Uuid:      msgToUpdate.Id,
		ChannelID: channelId,
		Update:    true,
	})
	return uint64(msgToUpdate.Id), nil
}

// GetMessage returns the [channels.ModelMessage] with the given [message.ID].
//
// Returns an error if the message cannot be gotten. It must return
// channels.NoMessageErr if the message does not exist.
func (i *impl) GetMessage(messageID message.ID) (channels.ModelMessage, error) {
	parentErr := "failed to GetMessage"

	result := &Message{MessageId: messageID.Bytes()}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(result).Error
	cancel()
	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return channels.ModelMessage{}, errors.WithMessage(channels.NoMessageErr, parentErr)
		}
		return channels.ModelMessage{}, errors.WithMessage(err, parentErr)
	}

	var channelId *id.ID
	if result.ChannelId != nil {
		channelId, err = id.Unmarshal(result.ChannelId)
		if err != nil {
			return channels.ModelMessage{}, errors.WithMessage(err, parentErr)
		}
	}

	var parentMsgId message.ID
	if result.ParentMessageId != nil {
		parentMsgId, err = message.UnmarshalID(result.ParentMessageId)
		if err != nil {
			return channels.ModelMessage{}, errors.WithMessage(err, parentErr)
		}
	}

	return channels.ModelMessage{
		UUID:            uint64(result.Id),
		Nickname:        result.Nickname,
		MessageID:       messageID,
		ChannelID:       channelId,
		ParentMessageID: parentMsgId,
		Timestamp:       result.Timestamp,
		Lease:           result.Lease,
		Status:          channels.SentStatus(result.Status),
		Hidden:          *result.Hidden,
		Pinned:          *result.Pinned,
		Content:         result.Text,
		Type:            channels.MessageType(result.Type),
		Round:           id.Round(result.Round),
		PubKey:          result.Pubkey,
		CodesetVersion:  result.CodesetVersion,
	}, nil
}

// MuteUser is called whenever a user is muted or unmuted.
func (i *impl) MuteUser(channelID *id.ID, pubKey ed25519.PublicKey, unmute bool) {
	go i.eventUpdate(channels.UserMuted, channels.UserMutedJson{
		ChannelID: channelID,
		PubKey:    pubKey,
		Unmute:    unmute,
	})
}

// DeleteMessage removes a message with the given messageID from storage.
//
// Returns an error if the message cannot be deleted. It must return
// channels.NoMessageErr if the message does not exist.
func (i *impl) DeleteMessage(messageID message.ID) error {
	parentErr := "failed to DeleteMessage: %+v"

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Where("message_id = ?",
		messageID.Bytes()).Delete(&Message{}).Error
	cancel()

	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return errors.Errorf(parentErr, channels.NoMessageErr)
		}
		return errors.Errorf(parentErr, err)
	}

	go i.eventUpdate(channels.MessageDeleted,
		channels.MessageDeletedJson{MessageID: messageID})
	return nil
}

// receiveHelper is a generic helper for receiving a Message.
// Returns UUID of the received Message as defined by the database.
func (i *impl) receiveHelper(channelID *id.ID, messageID message.ID,
	parentMsgId []byte, nickname, text string,
	pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time,
	lease time.Duration, round rounds.Round,
	messageType channels.MessageType,
	status channels.SentStatus, hidden bool) (uint64, error) {

	msgToInsert := buildMessage(
		channelID.Marshal(), messageID.Bytes(), parentMsgId, nickname,
		text, pubKey, dmToken, codeset, timestamp, lease, round.ID,
		messageType, false, hidden, status)

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Create(msgToInsert).Error
	cancel()

	if err != nil {
		return 0, errors.Errorf("Failed to insert message: %+v", err)
	}

	go i.eventUpdate(channels.MessageReceived, channels.MessageReceivedJson{
		Uuid:      msgToInsert.Id,
		ChannelID: channelID,
		Update:    false,
	})
	return uint64(msgToInsert.Id), nil
}

// buildMessage is a private helper that converts typical [channels.EventModel]
// inputs into a basic Message structure for insertion into storage.
//
// NOTE: ID is not set inside this function because we want to use the
// autoincrement key by default. If you are trying to overwrite an existing
// message, then you need to set it manually yourself.
func buildMessage(channelID, messageID, parentID []byte, nickname,
	text string, pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
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
		Hidden:          &hidden,
		Pinned:          &pinned,
		Text:            []byte(text),
		Type:            uint16(mType),
		Round:           int64(round),
		// User Identity Info
		Pubkey:         pubKey,
		DmToken:        dmToken,
		CodesetVersion: codeset,
	}
}
