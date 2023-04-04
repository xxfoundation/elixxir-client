////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for gateways

package storage

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/dm"
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

// buildMessage is a private helper that converts typical dm.EventModel inputs
// into a basic Message structure for insertion into storage.
//
// NOTE: ID is not set inside this function because we want to use the
// autoincrement key by default. If you are trying to overwrite an existing
// message, then you need to set it manually yourself.
func buildMessage(messageID, parentID, text []byte, partnerKey,
	senderKey ed25519.PublicKey, timestamp time.Time, round id.Round,
	mType dm.MessageType, codeset uint8, status dm.Status) *Message {
	return &Message{
		MessageId:          messageID,
		ConversationPubKey: partnerKey[:],
		ParentMessageId:    parentID,
		Timestamp:          timestamp,
		SenderPubKey:       senderKey[:],
		Status:             uint8(status),
		CodesetVersion:     codeset,
		Text:               text,
		Type:               uint16(mType),
		Round:              uint64(round),
	}
}

func (i *impl) Receive(messageID message.ID, nickname string, text []byte,
	partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, mType dm.MessageType, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to Receive"
	jww.TRACE.Printf("[DM SQL] Receive(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, nil, nickname, string(text),
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round, mType, status)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(err, parentErr))
		return 0
	}
	return uuid
}

func (i *impl) ReceiveText(messageID message.ID, nickname, text string,
	partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to ReceiveText"
	jww.TRACE.Printf("[DM SQL] ReceiveText(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, nil, nickname, text,
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round,
		dm.TextType, status)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(err, parentErr))
		return 0
	}
	return uuid
}

func (i *impl) ReceiveReply(messageID message.ID, reactionTo message.ID, nickname,
	text string, partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to ReceiveReply"
	jww.TRACE.Printf("[DM SQL] ReceiveReply(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, &reactionTo, nickname, text,
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round,
		dm.ReplyType, status)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(err, parentErr))
		return 0
	}
	return uuid
}

func (i *impl) ReceiveReaction(messageID message.ID, reactionTo message.ID,
	nickname, reaction string, partnerPubKey, senderPubKey ed25519.PublicKey,
	dmToken uint32, codeset uint8, timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to ReceiveReaction"
	jww.TRACE.Printf("[DM SQL] ReceiveReaction(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, &reactionTo, nickname, reaction,
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round,
		dm.ReactionType, status)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(err, parentErr))
		return 0
	}
	return uuid
}

func (i *impl) UpdateSentStatus(uuid uint64, messageID message.ID,
	timestamp time.Time, round rounds.Round, status dm.Status) {
	parentErr := errors.New("[DM SQL] failed to UpdateSentStatus")
	jww.TRACE.Printf(
		"[DM SQL] UpdateSentStatus(%d, %s, ...)", uuid, messageID)

	// Use the uuid to get the existing Message
	currentMessage := &Message{Id: uuid}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(currentMessage).Error
	cancel()
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessagef(parentErr,
			"Unable to get message: %+v", err))
		return
	}

	// Update the fields, if needed
	currentMessage.Status = uint8(status)
	if !messageID.Equals(message.ID{}) {
		currentMessage.MessageId = messageID.Bytes()
	}
	if round.ID != 0 {
		currentMessage.Round = uint64(round.ID)
	}
	if !timestamp.Equal(time.Time{}) {
		currentMessage.Timestamp = timestamp
	}

	// Store the updated Message
	_, err = i.upsertMessage(currentMessage)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.Wrap(parentErr, err.Error()))
		return
	}

	jww.TRACE.Printf("[DM SQL] Calling ReceiveMessageCB(%v, %v, t, f)",
		uuid, currentMessage.ConversationPubKey)
	go i.receivedMessageCB(uuid, currentMessage.ConversationPubKey,
		true, false)
}

func (i *impl) BlockSender(senderPubKey ed25519.PublicKey) {
	parentErr := "failed to BlockSender"
	err := i.setBlocked(senderPubKey, true)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessage(err, parentErr))
	}
}

func (i *impl) UnblockSender(senderPubKey ed25519.PublicKey) {
	parentErr := "failed to UnblockSender"
	err := i.setBlocked(senderPubKey, false)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessage(err, parentErr))
	}
}

// setBlocked is a helper for blocking/unblocking a given Conversation.
func (i *impl) setBlocked(senderPubKey ed25519.PublicKey, isBlocked bool) error {
	resultConvo, err := i.getConversation(senderPubKey)
	if err != nil {
		return err
	}

	return i.updateConversation(resultConvo.Nickname, resultConvo.Pubkey,
		resultConvo.Token, resultConvo.CodesetVersion, &isBlocked)
}

func (i *impl) GetConversation(senderPubKey ed25519.PublicKey) *dm.ModelConversation {
	parentErr := "failed to GetConversation"
	resultConvo, err := i.getConversation(senderPubKey)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessage(err, parentErr))
		return nil
	}

	return &dm.ModelConversation{
		Pubkey:         resultConvo.Pubkey,
		Nickname:       resultConvo.Nickname,
		Token:          resultConvo.Token,
		CodesetVersion: resultConvo.CodesetVersion,
		Blocked:        *resultConvo.Blocked,
	}
}

func (i *impl) GetConversations() []dm.ModelConversation {
	parentErr := "failed to GetConversations"

	var results []*Conversation
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Find(&results).Error
	cancel()
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessage(err, parentErr))
		return nil
	}

	conversations := make([]dm.ModelConversation, len(results))
	for i := range results {
		resultConvo := results[i]
		conversations[i] = dm.ModelConversation{
			Pubkey:         resultConvo.Pubkey,
			Nickname:       resultConvo.Nickname,
			Token:          resultConvo.Token,
			CodesetVersion: resultConvo.CodesetVersion,
			Blocked:        *resultConvo.Blocked,
		}
	}
	return conversations
}

// receiveWrapper is a higher-level wrapper of upsertMessage.
func (i *impl) receiveWrapper(messageID message.ID, parentID *message.ID, nickname,
	data string, partnerKey, senderKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, mType dm.MessageType, status dm.Status) (uint64, error) {

	// Keep track of whether Conversation was altered
	conversationUpdated := false
	result, err := i.getConversation(partnerKey)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		} else {
			// If there is no extant Conversation, create one.
			jww.DEBUG.Printf(
				"[DM SQL] Joining conversation with %s", nickname)
			err = i.createConversation(nickname, partnerKey, dmToken,
				codeset, false)
			if err != nil {
				return 0, err
			}
			conversationUpdated = true
		}
	} else {
		jww.DEBUG.Printf(
			"[DM SQL] Conversation with %s already joined", nickname)

		// Update Conversation if nickname was altered
		isFromPartner := bytes.Equal(result.Pubkey, senderKey)
		nicknameChanged := result.Nickname != nickname
		if isFromPartner && nicknameChanged {
			jww.DEBUG.Printf(
				"[DM SQL] Updating from nickname %s to %s",
				result.Nickname, nickname)
			err = i.updateConversation(nickname, result.Pubkey, result.Token,
				result.CodesetVersion, result.Blocked)
			if err != nil {
				return 0, err
			}
			conversationUpdated = true
		}
	}

	// Handle encryption, if it is present
	textBytes := []byte(data)
	if i.cipher != nil {
		textBytes, err = i.cipher.Encrypt(textBytes)
		if err != nil {
			return 0, err
		}
	}

	var parentIdBytes []byte
	if parentID != nil {
		parentIdBytes = parentID.Marshal()
	}

	msgToInsert := buildMessage(messageID.Bytes(), parentIdBytes, textBytes,
		partnerKey, senderKey, timestamp, round.ID, mType, codeset, status)

	uuid, err := i.upsertMessage(msgToInsert)
	if err != nil {
		return 0, err
	}

	jww.TRACE.Printf("[DM SQL] Calling ReceiveMessageCB(%v, %v, f, %t)",
		uuid, partnerKey, conversationUpdated)
	go i.receivedMessageCB(uuid, partnerKey, false, conversationUpdated)
	return uuid, nil
}

// upsertMessage is a helper function that will update an existing record
// if Message.ID is specified. Otherwise, it will perform an insert.
func (i *impl) upsertMessage(msg *Message) (uint64, error) {
	var err error
	ctx, cancel := newContext()
	if msg.Id != 0 {
		err = i.db.WithContext(ctx).Create(msg).Error
	} else {
		err = i.db.WithContext(ctx).Updates(msg).Error
	}
	cancel()
	if err != nil {
		return 0, err
	}

	jww.DEBUG.Printf("[DM SQL] Successfully stored message %d", msg.Id)
	return msg.Id, nil
}

// getConversation is a helper that returns the Conversation with the given senderPubKey.
func (i *impl) getConversation(senderPubKey ed25519.PublicKey) (*Conversation, error) {
	result := &Conversation{Pubkey: senderPubKey}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(result).Error
	cancel()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// createConversation is used for joining a Conversation.
func (i *impl) createConversation(nickname string,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8, blocked bool) error {
	newConvo := Conversation{
		Pubkey:         pubKey,
		Nickname:       nickname,
		Token:          dmToken,
		CodesetVersion: codeset,
		Blocked:        &blocked,
	}

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Create(newConvo).Error
	cancel()
	if err != nil {
		return errors.Errorf("[DM SQL] failed to createConversation: %+v", err)
	}

	return nil
}

// updateConversation is used for updating an extant Conversation to the given fields.
func (i *impl) updateConversation(nickname string,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8, blocked *bool) error {
	newConvo := Conversation{
		Pubkey:         pubKey,
		Nickname:       nickname,
		Token:          dmToken,
		CodesetVersion: codeset,
		Blocked:        blocked,
	}
	jww.DEBUG.Printf("[DM SQL] Attempting to updateConversation: %+v", newConvo)

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Updates(newConvo).Error
	cancel()
	if err != nil {
		return errors.Errorf("[DM SQL] failed to updateConversation: %+v", err)
	}
	return nil
}
