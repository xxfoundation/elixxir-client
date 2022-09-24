////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"google.golang.org/protobuf/proto"
	"time"
)

const (
	cmixChannelTextVersion     = 0
	cmixChannelReactionVersion = 0
)

// SendGeneric is used to send a raw message over a channel. In general, it
// should be wrapped in a function which defines the wire protocol
// If the final message, before being sent over the wire, is too long, this will
// return an error. Due to the underlying encoding using compression, it isn't
// possible to define the largest payload that can be sent, but
// it will always be possible to send a payload of 802 bytes at minimum
func (m *manager) SendGeneric(channelID *id.ID, messageType MessageType,
	msg []byte, validUntil time.Duration, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {

	//find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	nickname, _ := m.nicknameManager.GetNickname(channelID)

	var msgId cryptoChannel.MessageID

	chMsg := &ChannelMessage{
		Lease:       validUntil.Nanoseconds(),
		PayloadType: uint32(messageType),
		Payload:     msg,
		Nickname:    nickname,
	}

	usrMsg := &UserMessage{
		ECCPublicKey: m.me.PubKey,
	}

	//Note: we are not checking check if message is too long before trying to
	//find a round

	//Build the function pointer that will build the message
	assemble := func(rid id.Round) ([]byte, error) {

		//Build the message
		chMsg.RoundID = uint64(rid)

		//Serialize the message
		chMsgSerial, err := proto.Marshal(chMsg)
		if err != nil {
			return nil, err
		}

		//make the messageID
		msgId = cryptoChannel.MakeMessageID(chMsgSerial)

		//Sign the message
		messageSig := ed25519.Sign(*m.me.Privkey, chMsgSerial)

		usrMsg.Message = chMsgSerial
		usrMsg.Signature = messageSig

		//Serialize the user message
		usrMsgSerial, err := proto.Marshal(usrMsg)
		if err != nil {
			return nil, err
		}

		return usrMsgSerial, nil
	}

	uuid, err := m.st.denotePendingSend(channelID, &userMessageInternal{
		userMessage:    usrMsg,
		channelMessage: chMsg,
		messageID:      msgId,
	})

	r, ephid, err := ch.broadcast.BroadcastWithAssembler(assemble, params)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}
	err = m.st.send(uuid, msgId, r)
	return msgId, r, ephid, err
}

// SendAdminGeneric is used to send a raw message over a channel encrypted
// with admin keys, identifying it as sent by the admin. In general, it
// should be wrapped in a function which defines the wire protocol
// If the final message, before being sent over the wire, is too long, this will
// return an error. The message must be at most 510 bytes long.
func (m *manager) SendAdminGeneric(privKey rsa.PrivateKey, channelID *id.ID,
	messageType MessageType, msg []byte, validUntil time.Duration,
	params cmix.CMIXParams) (cryptoChannel.MessageID, rounds.Round, ephemeral.Id,
	error) {

	//find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	var msgId cryptoChannel.MessageID
	chMsg := &ChannelMessage{
		Lease:       validUntil.Nanoseconds(),
		PayloadType: uint32(messageType),
		Payload:     msg,
		Nickname:    AdminUsername,
	}
	//Note: we are not checking check if message is too long before trying to
	//find a round

	//Build the function pointer that will build the message
	assemble := func(rid id.Round) ([]byte, error) {

		//Build the message
		chMsg.RoundID = uint64(rid)

		//Serialize the message
		chMsgSerial, err := proto.Marshal(chMsg)
		if err != nil {
			return nil, err
		}

		msgId = cryptoChannel.MakeMessageID(chMsgSerial)

		//check if the message is too long
		if len(chMsgSerial) > ch.broadcast.MaxRSAToPublicPayloadSize() {
			return nil, MessageTooLongErr
		}

		return chMsgSerial, nil
	}

	uuid, err := m.st.denotePendingAdminSend(channelID, chMsg)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	r, ephid, err := ch.broadcast.BroadcastRSAToPublicWithAssembler(privKey,
		assemble, params)

	err = m.st.sendAdmin(uuid, msgId, r)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}
	return msgId, r, ephid, err
}

// SendMessage is used to send a formatted message over a channel.
// Due to the underlying encoding using compression, it isn't
// possible to define the largest payload that can be sent, but
// it will always be possible to send a payload of 798 bytes at minimum
func (m *manager) SendMessage(channelID *id.ID, msg string,
	validUntil time.Duration, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {
	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: nil,
	}

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, Text, txtMarshaled, validUntil, params)
}

// SendReply is used to send a formatted message over a channel.
// Due to the underlying encoding using compression, it isn't
// possible to define the largest payload that can be sent, but
// it will always be possible to send a payload of 766 bytes at minimum.
// If the message ID the reply is sent to doesnt exist, the other side will
// post the message as a normal message and not a reply.
func (m *manager) SendReply(channelID *id.ID, msg string,
	replyTo cryptoChannel.MessageID, validUntil time.Duration,
	params cmix.CMIXParams) (cryptoChannel.MessageID, rounds.Round,
	ephemeral.Id, error) {
	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: replyTo[:],
	}

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, Text, txtMarshaled, validUntil, params)
}

// SendReaction is used to send a reaction to a message over a channel.
// The reaction must be a single emoji with no other characters, and will
// be rejected otherwise.
// Clients will drop the reaction if they do not recognize the reactTo message
func (m *manager) SendReaction(channelID *id.ID, reaction string,
	reactTo cryptoChannel.MessageID, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {

	if err := ValidateReaction(reaction); err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	react := &CMIXChannelReaction{
		Version:           cmixChannelReactionVersion,
		Reaction:          reaction,
		ReactionMessageID: reactTo[:],
	}

	reactMarshaled, err := proto.Marshal(react)
	if err != nil {
		return cryptoChannel.MessageID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, Reaction, reactMarshaled, ValidForever,
		params)
}
