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
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/dm"
	"gitlab.com/elixxir/crypto/message"
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

func (i impl) Receive(messageID message.ID, nickname string, text []byte,
	partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, mType dm.MessageType, status dm.Status) uint64 {
	//TODO implement me
	panic("implement me")
}

func (i impl) ReceiveText(messageID message.ID, nickname, text string,
	partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	//TODO implement me
	panic("implement me")
}

func (i impl) ReceiveReply(messageID message.ID, reactionTo message.ID, nickname,
	text string, partnerPubKey, senderPubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	//TODO implement me
	panic("implement me")
}

func (i impl) ReceiveReaction(messageID message.ID, reactionTo message.ID,
	nickname, reaction string, partnerPubKey, senderPubKey ed25519.PublicKey,
	dmToken uint32, codeset uint8, timestamp time.Time, round rounds.Round, status dm.Status) uint64 {
	//TODO implement me
	panic("implement me")
}

func (i impl) UpdateSentStatus(uuid uint64, messageID message.ID,
	timestamp time.Time, round rounds.Round, status dm.Status) {
	//TODO implement me
	panic("implement me")
}

func (i impl) BlockSender(senderPubKey ed25519.PublicKey) {
	//TODO implement me
	panic("implement me")
}

func (i impl) UnblockSender(senderPubKey ed25519.PublicKey) {
	//TODO implement me
	panic("implement me")
}

func (i impl) GetConversation(senderPubKey ed25519.PublicKey) *dm.ModelConversation {
	//TODO implement me
	panic("implement me")
}

func (i impl) GetConversations() []dm.ModelConversation {
	//TODO implement me
	panic("implement me")
}
