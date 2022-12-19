////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"encoding/base64"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/emoji"
	"gitlab.com/elixxir/crypto/dm"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/proto"
)

// receiver struct for message handling
type receiver struct {
	c         *dmClient
	api       Receiver
	checkSent messageReceiveFunc
}

type dmProcessor struct {
	r *receiver
}

func (dp *dmProcessor) String() string {
	return "directMessage-"
}

func (dp *dmProcessor) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	ciphertext := reconstructCiphertext(msg)

	var payload []byte
	var err error
	var senderToken uint32
	var partnerPublicKey, senderPublicKey nike.PublicKey
	partnerPublicKey, payload, err = dm.Cipher.Decrypt(ciphertext,
		dp.r.c.privateKey)
	senderToken = 0
	senderPublicKey = partnerPublicKey
	if err != nil {
		jww.ERROR.Printf("failed to decrypt direct message: %s", err)
		return
	}

	directMsg := &DirectMessage{}
	if err := proto.Unmarshal(payload, directMsg); err != nil {
		jww.ERROR.Printf("unable to parse direct message: %+v",
			err)
		return
	}
	senderToken = directMsg.DMToken

	msgID := message.DeriveDirectMessageID(dp.r.c.receptionID, directMsg)

	// Check if we sent the message and ignore triggering if we sent
	if dp.r.checkSent(msgID, round) {
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// Check the round to ensure the message is not a replay
	if id.Round(directMsg.RoundID) != round.ID &&
		id.Round(directMsg.SelfRoundID) != round.ID {
		jww.WARN.Printf("The round DM %s send on %d"+
			"by %s was not the same as the"+
			"round the message was found on (%d)", msgID,
			round.ID, partnerPublicKey, directMsg.RoundID)
		return
	}

	// NOTE: There's no signature here, that kind of thing is done
	// by Noise in the layer doing decryption.
	//
	// There also are no admin commands for direct messages, but there
	// may be control messages (i.e., disappearing messages).

	// Replace the timestamp on the message if it is outside the
	// allowable range
	ts := message.VetTimestamp(time.Unix(0, directMsg.LocalTimestamp),
		round.Timestamps[states.QUEUED], msgID)

	pubSigningKey := ecdh.ECDHNIKE2EdwardsPublicKey(senderPublicKey)

	messageType := MessageType(directMsg.PayloadType)

	// Process the receivedMessage. This is already in an instanced event;
	// no new thread is needed.
	uuid, err := dp.r.receiveMessage(msgID, messageType, directMsg.Nickname,
		directMsg.Payload, senderToken,
		*pubSigningKey, ts, receptionID,
		round, Received)
	if err != nil {
		jww.WARN.Printf("Error processing for "+
			"DM (UUID: %d): %+v", uuid, err)
	}
}

// selfProcessor processes a self Encrypted DM message.
type selfProcessor struct {
	r *receiver
}

func (sp *selfProcessor) String() string {
	return "directMessageSelf-"
}

func (sp *selfProcessor) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	jww.INFO.Printf("selfProcessor: %v", msg)

	ciphertext := reconstructCiphertext(msg)

	partnerPublicKey, payload, err := dm.Cipher.DecryptSelf(ciphertext,
		sp.r.c.privateKey)
	if err != nil {
		jww.ERROR.Printf("failed to decrypt direct message (self): %s",
			err)
		return
	}
	senderPublicKey := sp.r.c.publicKey
	senderToken := sp.r.c.myToken

	directMsg := &DirectMessage{}
	if err := proto.Unmarshal(payload, directMsg); err != nil {
		jww.ERROR.Printf("unable to parse direct message: %+v",
			err)
		return
	}

	msgID := message.DeriveDirectMessageID(sp.r.c.receptionID, directMsg)

	// Check if we sent the message and ignore triggering if we sent
	if sp.r.checkSent(msgID, round) {
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// Check the round to ensure the message is not a replay
	if id.Round(directMsg.SelfRoundID) != round.ID {
		jww.WARN.Printf("The round self DM %s send on %d"+
			"by %s was not the same as the"+
			"round the message was found on (%d)", msgID,
			round.ID, partnerPublicKey, directMsg.RoundID)
		return
	}

	// NOTE: There's no signature here, that kind of thing is done
	// by Noise in the layer doing decryption.
	//
	// There also are no admin commands for direct messages, but there
	// may be control messages (i.e., disappearing messages).

	// Replace the timestamp on the message if it is outside the
	// allowable range
	ts := message.VetTimestamp(time.Unix(0, directMsg.LocalTimestamp),
		round.Timestamps[states.QUEUED], msgID)

	pubSigningKey := ecdh.ECDHNIKE2EdwardsPublicKey(senderPublicKey)

	messageType := MessageType(directMsg.PayloadType)

	// Process the receivedMessage. This is already in an instanced event;
	// no new thread is needed.
	uuid, err := sp.r.receiveMessage(msgID, messageType, directMsg.Nickname,
		directMsg.Payload, senderToken,
		*pubSigningKey, ts, receptionID,
		round, Received)
	if err != nil {
		jww.WARN.Printf("Error processing for "+
			"DM (UUID: %d): %+v", uuid, err)
	}
}

// GetSelfProcessor handles receiving self sent direct messages
func (r *receiver) GetSelfProcessor() *selfProcessor {
	return &selfProcessor{r: r}
}

// GetSelfProcessor handles receiving direct messages
func (r *receiver) GetProcessor() *dmProcessor {
	return &dmProcessor{r: r}
}

// receiveMessage attempts to parse the message and calls the appropriate
// receiver function.
func (r *receiver) receiveMessage(msgID message.ID, messageType MessageType,
	nick string, plaintext []byte, dmToken uint32,
	partnerPubKey ed25519.PublicKey, ts time.Time,
	_ receptionID.EphemeralIdentity, round rounds.Round,
	status Status) (uint64, error) {
	switch messageType {
	case TextType:
		return r.receiveTextMessage(msgID, messageType,
			nick, plaintext, dmToken, partnerPubKey,
			0, ts, round, status)
	case ReactionType:
		return r.receiveReaction(msgID, messageType,
			nick, plaintext, dmToken, partnerPubKey,
			0, ts, round, status)
	default:
		return r.api.Receive(msgID, nick, plaintext,
			partnerPubKey, dmToken, 0, ts, round,
			messageType, status), nil
	}
}

func (r *receiver) receiveTextMessage(messageID message.ID,
	messageType MessageType, nickname string, content []byte,
	dmToken uint32, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status Status) (uint64, error) {
	txt := &Text{}

	if err := proto.Unmarshal(content, txt); err != nil {
		return 0, errors.Wrapf(err, "failed text unmarshal DM %s "+
			"from %x, type %s, ts: %s, round: %d",
			messageID, pubKey, messageType, timestamp,
			round.ID)
	}

	if txt.ReplyMessageID != nil {

		if len(txt.ReplyMessageID) == message.IDLen {
			var replyTo message.ID
			copy(replyTo[:], txt.ReplyMessageID)
			tag := makeDebugTag(pubKey, content, SendReplyTag)
			jww.INFO.Printf("[%s] DM - Received reply from %s "+
				"to %s", tag,
				base64.StdEncoding.EncodeToString(pubKey),
				base64.StdEncoding.EncodeToString(
					txt.ReplyMessageID))
			return r.api.ReceiveReply(messageID, replyTo, nickname,
				txt.Text, pubKey, dmToken, codeset, timestamp,
				round, status), nil
		} else {
			return 0, errors.Errorf("Failed DM reply to for "+
				"message %s from public key %v "+
				"(codeset %d) on type %s, "+
				"ts: %s, round: %d, "+
				"returning without reply",
				messageID, pubKey, codeset,
				messageType, timestamp,
				round.ID)
			// Still process the message, but drop the
			// reply because it is malformed
		}
	}

	tag := makeDebugTag(pubKey, content, SendMessageTag)
	jww.INFO.Printf("[%s] DM - Received message from %s ",
		tag, base64.StdEncoding.EncodeToString(pubKey))

	return r.api.ReceiveText(messageID, nickname, txt.Text,
		pubKey, dmToken, codeset, timestamp, round, status), nil
}

// receiveReaction is the internal function that handles the reception of
// Reactions.
//
// It does edge checking to ensure the received reaction is just a single emoji.
// If the received reaction is not, the reaction is dropped.
// If the messageID for the message the reaction is to is malformed, the
// reaction is dropped.
func (r *receiver) receiveReaction(messageID message.ID,
	messageType MessageType, nickname string, content []byte,
	dmToken uint32, pubKey ed25519.PublicKey, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status Status) (uint64, error) {
	react := &Reaction{}
	if err := proto.Unmarshal(content, react); err != nil {
		return 0, errors.Wrapf(err, "Failed to text unmarshal DM %s "+
			"from %x, type %s, ts: %s, round: %d",
			messageID, pubKey, messageType, timestamp,
			round.ID)
	}

	// check that the reaction is a single emoji and ignore if it isn't
	if err := emoji.ValidateReaction(react.Reaction); err != nil {
		return 0, errors.Wrapf(err, "Failed process DM reaction %s"+
			" from %x, type %s, ts: %s, round: %d, due to "+
			"malformed reaction (%s), ignoring reaction",
			messageID, pubKey, messageType, timestamp,
			round.ID, content)
	}

	if react.ReactionMessageID != nil &&
		len(react.ReactionMessageID) == message.IDLen {
		var reactTo message.ID
		copy(reactTo[:], react.ReactionMessageID)

		tag := makeDebugTag(pubKey, content, SendReactionTag)
		jww.INFO.Printf("[%s] DM - Received reaction from %s "+
			"to %s", tag,
			base64.StdEncoding.EncodeToString(pubKey),
			base64.StdEncoding.EncodeToString(
				react.ReactionMessageID))

		return r.api.ReceiveReaction(messageID, reactTo, nickname,
			react.Reaction, pubKey, dmToken, codeset, timestamp,
			round, status), nil
	}
	return 0, errors.Errorf("Failed process DM reaction %s from public "+
		"key %v (codeset %d), type %s, ts: %s, "+
		"round: %d, reacting to invalid message, "+
		"ignoring reaction",
		messageID, pubKey, codeset, messageType, timestamp,
		round.ID)
}

// This helper does the opposite of "createCMIXFields" in send.go
func reconstructCiphertext(msg format.Message) []byte {
	fp := msg.GetKeyFP()
	res := fp[1:]
	res = append(res, msg.GetMac()[1:]...)
	res = append(res, msg.GetContents()...)
	return res
}
