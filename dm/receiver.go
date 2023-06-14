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
	c           *dmClient
	api         EventModel
	sendTracker SendTracker
}

type dmProcessor struct {
	r *receiver
}

func (dp *dmProcessor) String() string {
	return "directMessage-"
}

func (dp *dmProcessor) Process(msg format.Message, _ []string, _ []byte,
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
	senderToken = directMsg.GetDMToken()

	// On a regular receive, the msgID uses my reception ID and token
	// as I am the partner to which the message was sent.
	myID := deriveReceptionID(dp.r.c.GetPublicKey().Bytes(),
		dp.r.c.GetToken())

	jww.INFO.Printf("[DM] DeriveDirectMessage(%s...) Receive", myID)

	msgID := message.DeriveDirectMessageID(myID, directMsg)

	// Check if we sent the message and ignore triggering if we sent
	// This will happen when DMing with oneself, but the receive self
	// processor will update the status to delivered, so we do nothing here.
	if dp.r.sendTracker.CheckIfSent(msgID, round) {
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// Check the round to ensure the message is not a replay
	if id.Round(directMsg.RoundID) != round.ID &&
		id.Round(directMsg.SelfRoundID) != round.ID {
		jww.WARN.Printf("The round DM %s send on %d "+
			"by %s was not the same as the "+
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

	pubSigningKey := ecdh.EcdhNike2EdwardsPublicKey(senderPublicKey)

	messageType := MessageType(directMsg.PayloadType)

	// Check if the user is blocked
	user := dp.r.c.ps.getOrSet(pubSigningKey)
	if user.Status == statusBlocked {
		jww.INFO.Printf("Dropping message from blocked user: %s",
			base64.RawStdEncoding.EncodeToString(pubSigningKey))
		return
	}

	// partner Token is the sender Token
	partnerToken := senderToken

	// Process the receivedMessage. This is already in an instanced event;
	// no new thread is needed.
	// Note that in the non-self send case, the partner key and sender
	// key are the same. This is how the UI differentiates between the two.
	uuid, err := dp.r.receiveMessage(msgID, messageType, directMsg.Nickname,
		directMsg.Payload, partnerToken,
		pubSigningKey, pubSigningKey, ts, receptionID,
		round, Received)
	if err != nil {
		jww.WARN.Printf("Error processing for "+
			"DM (UUID: %d): %+v", uuid, err)
	}
}

// selfProcessor processes a self-encrypted DM message.
type selfProcessor struct {
	r *receiver
}

func (sp *selfProcessor) String() string {
	return "directMessageSelf-"
}

func (sp *selfProcessor) Process(msg format.Message, _ []string, _ []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	ciphertext := reconstructCiphertext(msg)

	partnerPublicKey, payload, err := dm.Cipher.DecryptSelf(ciphertext,
		sp.r.c.privateKey)
	if err != nil {
		jww.ERROR.Printf("failed to decrypt direct message (self): %s",
			err)
		return
	}
	senderPublicKey := sp.r.c.publicKey

	directMsg := &DirectMessage{}
	if err := proto.Unmarshal(payload, directMsg); err != nil {
		jww.ERROR.Printf("unable to parse direct message: %+v",
			err)
		return
	}
	partnerToken := directMsg.GetDMToken()

	// On a self receive, the msgID uses the partner key and
	// the partner token stored on the message.
	// The partner is not the sender on a self send...
	partnerID := deriveReceptionID(partnerPublicKey.Bytes(),
		partnerToken)

	jww.INFO.Printf("[DM] DeriveDirectMessage(%s...) ReceiveSelf",
		partnerID)

	msgID := message.DeriveDirectMessageID(partnerID, directMsg)

	// Check if we sent the message and ignore triggering if we
	// sent, but mark the message as delivered
	if sp.r.sendTracker.CheckIfSent(msgID, round) {
		go func() {
			ok := sp.r.sendTracker.Delivered(msgID, round)
			if !ok {
				jww.WARN.Printf("[DM] Couldn't mark delivered"+
					": %s %v)",
					msgID, round)
			}
			sp.r.sendTracker.StopTracking(msgID, round)
			if !ok {
				jww.WARN.Printf("[DM] Couldn't StopTracking: "+
					"%s, %v", msgID, round)
			}
		}()
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// Check the round to ensure the message is not a replay
	if id.Round(directMsg.SelfRoundID) != round.ID {
		jww.WARN.Printf("The round self DM %s send on %d "+
			"by %s was not the same as the "+
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

	pubSigningKey := ecdh.EcdhNike2EdwardsPublicKey(senderPublicKey)
	partnerPubKey := ecdh.EcdhNike2EdwardsPublicKey(partnerPublicKey)

	messageType := MessageType(directMsg.PayloadType)

	// Process the receivedMessage. This is already in an instanced event;
	// no new thread is needed.
	uuid, err := sp.r.receiveMessage(msgID, messageType, directMsg.Nickname,
		directMsg.Payload, partnerToken,
		partnerPubKey, pubSigningKey, ts, receptionID,
		round, Received)
	if err != nil {
		jww.WARN.Printf("Error processing for "+
			"DM (UUID: %d): %+v", uuid, err)
	}
}

// GetSelfProcessor handles receiving self sent direct messages.
func (r *receiver) GetSelfProcessor() *selfProcessor {
	return &selfProcessor{r: r}
}

// GetProcessor handles receiving direct messages.
func (r *receiver) GetProcessor() *dmProcessor {
	return &dmProcessor{r: r}
}

// receiveMessage attempts to parse the message and calls the appropriate
// receiver function.
func (r *receiver) receiveMessage(msgID message.ID, messageType MessageType,
	nick string, plaintext []byte, partnerDMToken uint32,
	partnerPubKey, senderPubKey ed25519.PublicKey, ts time.Time,
	_ receptionID.EphemeralIdentity, round rounds.Round,
	status Status) (uint64, error) {
	switch messageType {
	case TextType:
		return r.receiveTextMessage(msgID, messageType,
			nick, plaintext, partnerDMToken, partnerPubKey,
			senderPubKey, 0, ts, round, status)
	case ReplyType:
		return r.receiveTextMessage(msgID, messageType,
			nick, plaintext, partnerDMToken, partnerPubKey,
			senderPubKey, 0, ts, round, status)
	case ReactionType:
		return r.receiveReaction(msgID, messageType,
			nick, plaintext, partnerDMToken, partnerPubKey,
			senderPubKey, 0, ts, round, status)
	default:
		return r.api.Receive(msgID, nick, plaintext,
			partnerPubKey, senderPubKey,
			partnerDMToken, 0, ts, round,
			messageType, status), nil
	}
}

func (r *receiver) receiveTextMessage(messageID message.ID,
	messageType MessageType, nickname string, content []byte,
	dmToken uint32, partnerPubKey, senderPubKey ed25519.PublicKey,
	codeset uint8, timestamp time.Time, round rounds.Round,
	status Status) (uint64, error) {
	txt := &Text{}

	if err := proto.Unmarshal(content, txt); err != nil {
		return 0, errors.Wrapf(err, "failed text unmarshal DM %s "+
			"from %x, type %s, ts: %s, round: %d",
			messageID, partnerPubKey, messageType,
			timestamp, round.ID)
	}

	if txt.ReplyMessageID != nil && !allZeros(txt.ReplyMessageID) {

		if len(txt.ReplyMessageID) == message.IDLen {
			var replyTo message.ID
			copy(replyTo[:], txt.ReplyMessageID)
			tag := makeDebugTag(partnerPubKey, content,
				SendReplyTag)
			jww.INFO.Printf("[%s] DM - Received reply with %s "+
				"to %s", tag,
				base64.StdEncoding.EncodeToString(
					partnerPubKey),
				base64.StdEncoding.EncodeToString(
					txt.ReplyMessageID))
			return r.api.ReceiveReply(messageID, replyTo, nickname,
				txt.Text, partnerPubKey, senderPubKey,
				dmToken, codeset, timestamp, round, status), nil
		} else {
			return 0, errors.Errorf("Failed DM reply to for "+
				"message %s with partner key %v "+
				"(codeset %d) on type %s, "+
				"ts: %s, round: %d, "+
				"returning without reply",
				messageID, partnerPubKey, codeset,
				messageType, timestamp,
				round.ID)
		}
	}

	tag := makeDebugTag(partnerPubKey, content, SendMessageTag)
	jww.INFO.Printf("[%s] DM - Received message with partner %s ",
		tag, base64.StdEncoding.EncodeToString(partnerPubKey))

	return r.api.ReceiveText(messageID, nickname, txt.Text,
		partnerPubKey, senderPubKey, dmToken, codeset,
		timestamp, round, status), nil
}

// receiveReaction is the internal function that handles the reception of
// Reactions.
//
// It does edge checking to ensure the received reaction is just a single emoji.
// If the received reaction is not, the reaction is dropped.
// If the messageID for the message the reaction is malformed, then the
// reaction is dropped.
func (r *receiver) receiveReaction(messageID message.ID,
	messageType MessageType, nickname string, content []byte,
	dmToken uint32, partnerPubKey, senderPubKey ed25519.PublicKey,
	codeset uint8, timestamp time.Time, round rounds.Round,
	status Status) (uint64, error) {
	react := &Reaction{}
	if err := proto.Unmarshal(content, react); err != nil {
		return 0, errors.Wrapf(err, "Failed to text unmarshal DM %s "+
			"with %x, type %s, ts: %s, round: %d",
			messageID, partnerPubKey, messageType, timestamp,
			round.ID)
	}

	// check that the reaction is a single emoji and ignore if it isn't
	if err := emoji.ValidateReaction(react.Reaction); err != nil {
		return 0, errors.Wrapf(err, "Failed process DM reaction %s"+
			" with %x, type %s, ts: %s, round: %d, due to "+
			"malformed reaction (%s), ignoring reaction",
			messageID, partnerPubKey, messageType, timestamp,
			round.ID, content)
	}

	if react.ReactionMessageID != nil &&
		len(react.ReactionMessageID) == message.IDLen {
		var reactTo message.ID
		copy(reactTo[:], react.ReactionMessageID)

		tag := makeDebugTag(partnerPubKey, content, SendReactionTag)
		jww.INFO.Printf("[%s] DM - Received reaction with %s "+
			"to %s", tag,
			base64.RawStdEncoding.EncodeToString(partnerPubKey),
			reactTo)

		return r.api.ReceiveReaction(messageID, reactTo, nickname,
			react.Reaction, partnerPubKey, senderPubKey,
			dmToken, codeset, timestamp,
			round, status), nil
	}
	return 0, errors.Errorf("Failed process DM reaction %s with public "+
		"key %v (codeset %d), type %s, ts: %s, "+
		"round: %d, reacting to invalid message, "+
		"ignoring reaction",
		messageID, partnerPubKey, codeset,
		messageType, timestamp,
		round.ID)
}

// This helper does the opposite of "createCMIXFields" in send.go
func reconstructCiphertext(msg format.Message) []byte {
	var res []byte
	fp := msg.GetKeyFP()
	res = append(res, fp[1:]...)
	res = append(res, msg.GetMac()[1:]...)
	res = append(res, msg.GetContents()...)
	return res
}

func allZeros(data []byte) bool {
	for i := range data {
		if data[i] != 0 {
			return false
		}
	}
	return true
}
