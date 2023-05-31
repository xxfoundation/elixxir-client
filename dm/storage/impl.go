////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/dm"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"gorm.io/gorm"
	"strings"
	"time"
)

const (
	// Can be provided to SqlLite to create a temporary, in-memory DB.
	temporaryDbPath = "file:%s?mode=memory&cache=shared"

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
func buildMessage(messageID, parentID []byte, text string, partnerKey []byte,
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
		Text:               []byte(text),
		Type:               uint16(mType),
		Round:              int64(round),
	}
}

func (i *impl) Receive(messageID message.ID, nickname string, text []byte,
	partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, mType dm.MessageType, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to Receive: %+v"
	jww.TRACE.Printf("[DM SQL] Receive(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, nil, nickname, string(text),
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round, mType, status)
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return 0
	}
	return uuid
}

func (i *impl) ReceiveText(messageID message.ID, nickname, text string,
	partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to ReceiveText: %+v"
	jww.TRACE.Printf("[DM SQL] ReceiveText(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, nil, nickname, text,
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round,
		dm.TextType, status)
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return 0
	}
	return uuid
}

func (i *impl) ReceiveReply(messageID message.ID, reactionTo message.ID, nickname,
	text string, partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to ReceiveReply: %+v"
	jww.TRACE.Printf("[DM SQL] ReceiveReply(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, &reactionTo, nickname, text,
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round,
		dm.ReplyType, status)
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return 0
	}
	return uuid
}

func (i *impl) ReceiveReaction(messageID message.ID, reactionTo message.ID,
	nickname, reaction string, partnerPubKey, senderPubKey ed25519.PublicKey,
	dmToken uint32, codeset uint8, timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	parentErr := "[DM SQL] failed to ReceiveReaction: %+v"
	jww.TRACE.Printf("[DM SQL] ReceiveReaction(%s)", messageID)

	uuid, err := i.receiveWrapper(messageID, &reactionTo, nickname, reaction,
		partnerPubKey, senderPubKey, dmToken, codeset, timestamp, round,
		dm.ReactionType, status)
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return 0
	}
	return uuid
}

func (i *impl) UpdateSentStatus(uuid uint64, messageID message.ID,
	timestamp time.Time, round rounds.Round, status dm.Status) {
	parentErr := "[DM SQL] failed to UpdateSentStatus: %+v"
	jww.TRACE.Printf(
		"[DM SQL] UpdateSentStatus(%d, %s, ...)", uuid, messageID)

	// Use the uuid to get the existing Message
	currentMessage := &Message{Id: int64(uuid)}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(currentMessage).Error
	cancel()
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return
	}

	// Update the fields, if needed
	currentMessage.Status = uint8(status)
	if !messageID.Equals(message.ID{}) {
		currentMessage.MessageId = messageID.Bytes()
	}
	if round.ID != 0 {
		currentMessage.Round = int64(round.ID)
	}
	if !timestamp.Equal(time.Time{}) {
		currentMessage.Timestamp = timestamp
	}

	// Store the updated Message
	_, err = i.upsertMessage(currentMessage)
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return
	}

	jww.TRACE.Printf("[DM SQL] Calling ReceiveMessageCB(%v, %v, t, f)",
		uuid, currentMessage.ConversationPubKey)
	go i.receivedMessageCB(uuid, currentMessage.ConversationPubKey,
		true, false)
}

func (i *impl) BlockSender(senderPubKey ed25519.PublicKey) {
	parentErr := "Failed to BlockSender: %+v"

	err := i.setBlocked(senderPubKey, true)
	if err != nil {
		jww.ERROR.Printf("%+v", errors.WithMessage(err, parentErr))
	}
}

func (i *impl) UnblockSender(senderPubKey ed25519.PublicKey) {
	parentErr := "Failed to UnblockSender: %+v"
	err := i.setBlocked(senderPubKey, false)
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
	}
}

// setBlocked is a helper for blocking/unblocking a given Conversation.
func (i *impl) setBlocked(senderPubKey ed25519.PublicKey, isBlocked bool) error {
	resultConvo, err := i.getConversation(senderPubKey)
	if err != nil {
		return err
	}

	var timeBlocked *time.Time = nil
	if isBlocked {
		blockUser := netTime.Now()
		timeBlocked = &blockUser
	}

	return i.upsertConversation(resultConvo.Nickname, resultConvo.Pubkey,
		resultConvo.Token, resultConvo.CodesetVersion, timeBlocked)
}

func (i *impl) GetConversation(senderPubKey ed25519.PublicKey) *dm.ModelConversation {
	parentErr := "Failed to GetConversation: %+v"
	resultConvo, err := i.getConversation(senderPubKey)
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return nil
	}

	return &dm.ModelConversation{
		Pubkey:           resultConvo.Pubkey,
		Nickname:         resultConvo.Nickname,
		Token:            resultConvo.Token,
		CodesetVersion:   resultConvo.CodesetVersion,
		BlockedTimestamp: resultConvo.BlockedTimestamp,
	}
}

func (i *impl) GetConversations() []dm.ModelConversation {
	parentErr := "Failed to GetConversations: %+v"

	var results []*Conversation
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Find(&results).Error
	cancel()
	if err != nil {
		jww.ERROR.Printf(parentErr, err)
		return nil
	}

	conversations := make([]dm.ModelConversation, len(results))
	for i := range results {
		resultConvo := results[i]
		conversations[i] = dm.ModelConversation{
			Pubkey:           resultConvo.Pubkey,
			Nickname:         resultConvo.Nickname,
			Token:            resultConvo.Token,
			CodesetVersion:   resultConvo.CodesetVersion,
			BlockedTimestamp: resultConvo.BlockedTimestamp,
		}
	}
	return conversations
}

// receiveWrapper is a higher-level wrapper of upsertMessage.
func (i *impl) receiveWrapper(messageID message.ID, parentID *message.ID, nickname,
	data string, partnerKey, senderKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, mType dm.MessageType, status dm.Status) (uint64, error) {
	partnerKeyStr := base64.StdEncoding.EncodeToString(partnerKey)

	// Keep track of whether a Conversation was altered
	var convoToUpdate *Conversation

	// Determine whether Conversation needs to be created
	result, err := i.getConversation(partnerKey)
	if err != nil {
		if !strings.Contains(err.Error(), gorm.ErrRecordNotFound.Error()) {
			return 0, err
		} else {
			// If there is no extant Conversation, create one.
			jww.DEBUG.Printf(
				"[DM SQL] Joining conversation with %s", partnerKeyStr)
			convoToUpdate = &Conversation{
				Pubkey:           partnerKey,
				Nickname:         nickname,
				Token:            dmToken,
				CodesetVersion:   codeset,
				BlockedTimestamp: nil,
			}
		}
	} else {
		jww.DEBUG.Printf(
			"[DM SQL] Conversation with %s already joined", partnerKeyStr)

		// Update Conversation if nickname was altered
		isFromPartner := bytes.Equal(result.Pubkey, partnerKey)
		nicknameChanged := result.Nickname != nickname
		if isFromPartner && nicknameChanged {
			jww.DEBUG.Printf("[DM SQL] Updating from nickname %s to %s",
				result.Nickname, nickname)
			convoToUpdate = result
			convoToUpdate.Nickname = nickname
		}

		// Fix conversation if dmToken is altered
		dmTokenChanged := result.Token != dmToken
		if isFromPartner && dmTokenChanged {
			jww.WARN.Printf(
				"[DM indexedDB] Updating from dmToken %d to %d",
				result.Token, dmToken)
			convoToUpdate = result
			convoToUpdate.Token = dmToken
		}
	}

	// Update the conversation in storage, if needed
	conversationUpdated := convoToUpdate != nil

	if conversationUpdated {
		err = i.upsertConversation(convoToUpdate.Nickname, convoToUpdate.Pubkey,
			convoToUpdate.Token, convoToUpdate.CodesetVersion,
			convoToUpdate.BlockedTimestamp)
		if err != nil {
			return 0, err
		}
	}

	var parentIdBytes []byte
	if parentID != nil {
		parentIdBytes = parentID.Marshal()
	}

	msgToInsert := buildMessage(messageID.Bytes(), parentIdBytes, data,
		partnerKey, senderKey, timestamp, round.ID, mType, codeset, status)

	uuid, err := i.upsertMessage(msgToInsert)
	if err != nil {
		return 0, err
	}

	jww.TRACE.Printf("[DM SQL] Calling ReceiveMessageCB(%v, %v, f, %t)",
		uuid, partnerKeyStr, conversationUpdated)
	go i.receivedMessageCB(uuid, partnerKey,
		false, conversationUpdated)
	return uuid, nil
}

// upsertMessage is a helper function that will update an existing record
// if Message.ID is specified. Otherwise, it will perform an insert.
func (i *impl) upsertMessage(msg *Message) (uint64, error) {
	jww.DEBUG.Printf("[DM SQL] Attempting to upsertMessage: %+v", msg)

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Save(msg).Error
	cancel()
	if err != nil {
		return 0, errors.Errorf("failed to upsertMessage: %+v", err)
	}

	jww.DEBUG.Printf("[DM SQL] Successfully stored message %d", msg.Id)
	return uint64(msg.Id), nil
}

// getConversation is a helper that returns the Conversation with the given senderPubKey.
func (i *impl) getConversation(senderPubKey ed25519.PublicKey) (*Conversation, error) {
	result := &Conversation{Pubkey: senderPubKey}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(result).Error
	cancel()
	if err != nil {
		return nil, errors.Errorf("failed to getConversation: %+v", err)
	}
	return result, nil
}

// upsertConversation is used for updating or creating a Conversation with the given fields.
func (i *impl) upsertConversation(nickname string,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timeBlocked *time.Time) error {

	newConvo := Conversation{
		Pubkey:           pubKey,
		Nickname:         nickname,
		Token:            dmToken,
		CodesetVersion:   codeset,
		BlockedTimestamp: timeBlocked,
	}
	jww.DEBUG.Printf("[DM SQL] Attempting to upsertConversation: %+v", newConvo)

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Save(newConvo).Error
	cancel()
	if err != nil {
		return errors.Errorf("[DM SQL] failed to upsertConversation: %+v", err)
	}
	return nil
}
