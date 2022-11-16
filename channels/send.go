////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"
)

const (
	cmixChannelTextVersion     = 0
	cmixChannelReactionVersion = 0

	// SendMessageTag is the base tag used when generating a debug tag for
	// sending a message.
	SendMessageTag = "ChMessage"

	// SendReplyTag is the base tag used when generating a debug tag for
	// sending a reply.
	SendReplyTag = "ChReply"

	// SendReactionTag is the base tag used when generating a debug tag for
	// sending a reaction.
	SendReactionTag = "ChReaction"
)

// The size of the nonce used in the message ID.
const messageNonceSize = 4

// SendGeneric is used to send a raw message over a channel. In general, it
// should be wrapped in a function that defines the wire protocol.
//
// If the final message, before being sent over the wire, is too long, this will
// return an error. Due to the underlying encoding using compression, it is not
// possible to define the largest payload that can be sent, but it will always
// be possible to send a payload of 802 bytes at minimum.
func (m *manager) SendGeneric(channelID *id.ID, messageType MessageType,
	msg []byte, validUntil time.Duration, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {

	// Note: We log sends on exit, and append what happened to the message
	// this cuts down on clutter in the log.
	sendPrint := fmt.Sprintf("[%s] Sending ch %s type %d at %s",
		params.DebugTag, channelID, messageType, netTime.Now())
	defer jww.INFO.Println(sendPrint)

	// Find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	nickname, _ := m.GetNickname(channelID)

	var msgId cryptoChannel.MessageID

	// Retrieve token.
	// Note that this may be nil if DM token have not been enabled,
	// which is OK.
	dmToken := m.getDmToken(channelID)

	chMsg := &ChannelMessage{
		Lease:          validUntil.Nanoseconds(),
		PayloadType:    uint32(messageType),
		Payload:        msg,
		Nickname:       nickname,
		Nonce:          make([]byte, messageNonceSize),
		LocalTimestamp: netTime.Now().UnixNano(),
		DMToken:        dmToken,
	}

	// Generate random nonce to be used for message ID generation. This makes it
	// so two identical messages sent on the same round have different message
	// IDs.
	rng := m.rng.GetStream()
	n, err := rng.Read(chMsg.Nonce)
	rng.Close()
	if err != nil {
		sendPrint += fmt.Sprintf(", failed to generate nonce: %+v", err)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("Failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		sendPrint += fmt.Sprintf(
			", got %d bytes for %d-byte nonce", n, messageNonceSize)
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf(
				"Generated %d bytes for %d-byte nonce", n, messageNonceSize)
	}

	usrMsg := &UserMessage{
		ECCPublicKey: m.me.PubKey,
	}

	// Note: we are not checking if message is too long before trying to find a
	// round

	// Build the function pointer that will build the message
	assemble := func(rid id.Round) ([]byte, error) {
		// Build the message
		chMsg.RoundID = uint64(rid)

		// Serialize the message
		chMsgSerial, err := proto.Marshal(chMsg)
		if err != nil {
			return nil, err
		}

		// Make the messageID
		msgId = cryptoChannel.MakeMessageID(chMsgSerial, channelID)

		// Sign the message
		messageSig := ed25519.Sign(*m.me.Privkey, chMsgSerial)

		usrMsg.Message = chMsgSerial
		usrMsg.Signature = messageSig

		// Serialize the user message
		usrMsgSerial, err := proto.Marshal(usrMsg)
		if err != nil {
			return nil, err
		}

		return usrMsgSerial, nil
	}

	sendPrint += fmt.Sprintf(", pending send %s", netTime.Now())
	uuid, err := m.st.denotePendingSend(channelID, &userMessageInternal{
		userMessage:    usrMsg,
		channelMessage: chMsg,
		messageID:      msgId,
	})
	if err != nil {
		sendPrint += fmt.Sprintf(", pending send failed %s", err.Error())
		return cryptoChannel.MessageID{}, rounds.Round{},
			ephemeral.Id{}, err
	}

	sendPrint += fmt.Sprintf(", broadcasting message %s", netTime.Now())
	r, ephID, err := ch.broadcast.BroadcastWithAssembler(assemble, params)
	if err != nil {
		sendPrint += fmt.Sprintf(
			", broadcast failed %s, %s", netTime.Now(), err.Error())
		errDenote := m.st.failedSend(uuid)
		if errDenote != nil {
			sendPrint += fmt.Sprintf(
				", failed to denote failed broadcast: %s", err.Error())
		}
		return cryptoChannel.MessageID{}, rounds.Round{},
			ephemeral.Id{}, err
	}
	sendPrint += fmt.Sprintf(
		", broadcast succeeded %s, success!", netTime.Now())
	err = m.st.send(uuid, msgId, r)
	if err != nil {
		sendPrint += fmt.Sprintf(", broadcast failed: %s ", err.Error())
	}
	return msgId, r, ephID, err
}

// SendAdminGeneric is used to send a raw message over a channel encrypted with
// admin keys, identifying it as sent by the admin. In general, it should be
// wrapped in a function that defines the wire protocol.
//
// If the final message, before being sent over the wire, is too long, this will
// return an error. The message must be at most 510 bytes long.
func (m *manager) SendAdminGeneric(privKey rsa.PrivateKey, channelID *id.ID,
	messageType MessageType, msg []byte, validUntil time.Duration,
	params cmix.CMIXParams) (cryptoChannel.MessageID, rounds.Round,
	ephemeral.Id, error) {

	// Note: We log sends on exit, and append what happened to the message
	// this cuts down on clutter in the log.
	sendPrint := fmt.Sprintf("[%s] Admin sending ch %s type %d at %s",
		params.DebugTag, channelID, messageType, netTime.Now())
	defer jww.INFO.Println(sendPrint)

	// Find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	var msgId cryptoChannel.MessageID
	chMsg := &ChannelMessage{
		Lease:          validUntil.Nanoseconds(),
		PayloadType:    uint32(messageType),
		Payload:        msg,
		Nickname:       AdminUsername,
		Nonce:          make([]byte, messageNonceSize),
		LocalTimestamp: netTime.Now().UnixNano(),
	}

	// Generate random nonce to be used for message ID generation. This makes it
	// so two identical messages sent on the same round have different message
	// IDs
	rng := m.rng.GetStream()
	n, err := rng.Read(chMsg.Nonce)
	rng.Close()
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf("Failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{},
			errors.Errorf(
				"Generated %d bytes for %-byte nonce", n, messageNonceSize)
	}

	// Note: we are not checking if message is too long before trying to
	// find a round

	// Build the function pointer that will build the message
	assemble := func(rid id.Round) ([]byte, error) {
		// Build the message
		chMsg.RoundID = uint64(rid)

		// Serialize the message
		chMsgSerial, err := proto.Marshal(chMsg)
		if err != nil {
			return nil, err
		}

		msgId = cryptoChannel.MakeMessageID(chMsgSerial, channelID)

		// Check if the message is too long
		if len(chMsgSerial) > ch.broadcast.MaxRSAToPublicPayloadSize() {
			return nil, MessageTooLongErr
		}

		return chMsgSerial, nil
	}

	sendPrint += fmt.Sprintf(", pending send %s", netTime.Now())
	uuid, err := m.st.denotePendingAdminSend(channelID, chMsg)
	if err != nil {
		sendPrint += fmt.Sprintf(", pending send failed %s", err.Error())
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	sendPrint += fmt.Sprintf(", broadcasting message %s", netTime.Now())
	r, ephID, err := ch.broadcast.BroadcastRSAToPublicWithAssembler(privKey,
		assemble, params)
	if err != nil {
		sendPrint += fmt.Sprintf(
			", broadcast failed %s, %s", netTime.Now(), err.Error())
		errDenote := m.st.failedSend(uuid)
		if errDenote != nil {
			sendPrint += fmt.Sprintf(
				", failed to denote failed broadcast: %s", err.Error())
			jww.ERROR.Printf(
				"Failed to update for a failed send to %s: %+v", channelID, err)
		}
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}
	sendPrint += fmt.Sprintf(
		", broadcast succeeded %s, success!", netTime.Now())
	err = m.st.send(uuid, msgId, r)
	if err != nil {
		sendPrint += fmt.Sprintf(", broadcast failed: %s ", err.Error())
	}
	return msgId, r, ephID, err
}

// SendMessage is used to send a formatted message over a channel.
//
// Due to the underlying encoding using compression, it is not possible to
// define the largest payload that can be sent, but it will always be possible
// to send a payload of 798 bytes at minimum.
//
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if ValidForever is used.
func (m *manager) SendMessage(channelID *id.ID, msg string,
	validUntil time.Duration, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(channelID, m.me.PubKey, []byte(msg), SendMessageTag)
	jww.INFO.Printf("[%s]SendMessage(%s)", tag, channelID)

	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: nil,
	}

	params = params.SetDebugTag(tag)

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, Text, txtMarshaled, validUntil, params)
}

// SendReply is used to send a formatted message over a channel.
//
// Due to the underlying encoding using compression, it is not possible to
// define the largest payload that can be sent, but it will always be possible
// to send a payload of 766 bytes at minimum.
//
// If the message ID that the reply is sent to does not exist, then the other
// side will post the message as a normal message and not as a reply.
//
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if ValidForever is used.
func (m *manager) SendReply(channelID *id.ID, msg string,
	replyTo cryptoChannel.MessageID, validUntil time.Duration,
	params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(channelID, m.me.PubKey, []byte(msg), SendReplyTag)
	jww.INFO.Printf("[%s]SendReply(%s, to %s)", tag, channelID, replyTo)
	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: replyTo[:],
	}

	params = params.SetDebugTag(tag)

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, Text, txtMarshaled, validUntil,
		params)
}

// SendReaction is used to send a reaction to a message over a channel. The
// reaction must be a single emoji with no other characters, and will be
// rejected otherwise.
//
// Clients will drop the reaction if they do not recognize the reactTo message.
func (m *manager) SendReaction(channelID *id.ID, reaction string,
	reactTo cryptoChannel.MessageID, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	tag := makeChaDebugTag(
		channelID, m.me.PubKey, []byte(reaction), SendReactionTag)
	jww.INFO.Printf("[%s]SendReply(%s, to %s)", tag, channelID, reactTo)

	if err := ValidateReaction(reaction); err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	react := &CMIXChannelReaction{
		Version:           cmixChannelReactionVersion,
		Reaction:          reaction,
		ReactionMessageID: reactTo[:],
	}

	params = params.SetDebugTag(tag)

	reactMarshaled, err := proto.Marshal(react)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(
		channelID, Reaction, reactMarshaled, ValidForever, params)
}

// makeChaDebugTag is a debug helper that creates non-unique msg identifier.
//
// This is set as the debug tag on messages and enables some level of tracing a
// message (if its contents/chan/type are unique).
func makeChaDebugTag(channelID *id.ID, id ed25519.PublicKey,
	msg []byte, baseTag string) string {

	h, _ := blake2b.New256(nil)
	h.Write(channelID[:])
	h.Write(msg)
	h.Write(id)

	tripCode := base64.RawStdEncoding.EncodeToString(h.Sum(nil))[:12]
	return fmt.Sprintf("%s-%s", baseTag, tripCode)
}
