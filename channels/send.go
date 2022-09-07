////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/crypto/signature/rsa"
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
	cryptoChannel.MessageID, id.Round, ephemeral.Id, error) {

	//find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, err
	}

	var msgId cryptoChannel.MessageID
	//Note: we are not checking check if message is too long before trying to
	//find a round

	//Build the function pointer that will build the message
	assemble := func(rid id.Round) ([]byte, error) {

		//Build the message
		chMsg := &ChannelMessage{
			Lease:       validUntil.Nanoseconds(),
			RoundID:     uint64(rid),
			PayloadType: uint32(messageType),
			Payload:     msg,
		}

		//Serialize the message
		chMsgSerial, err := proto.Marshal(chMsg)
		if err != nil {
			return nil, err
		}

		//make the messageID
		msgId = cryptoChannel.MakeMessageID(chMsgSerial)

		//Sign the message
		messageSig, err := m.name.SignChannelMessage(chMsgSerial)
		if err != nil {
			return nil, err
		}

		//Build the user message
		validationSig, unameLease := m.name.GetChannelValidationSignature()

		usrMsg := &UserMessage{
			Message:             chMsgSerial,
			ValidationSignature: validationSig,
			Signature:           messageSig,
			Username:            m.name.GetUsername(),
			ECCPublicKey:        m.name.GetChannelPubkey(),
			UsernameLease:       unameLease.UnixNano(),
		}

		//Serialize the user message
		usrMsgSerial, err := proto.Marshal(usrMsg)
		if err != nil {
			return nil, err
		}

		//Fill in any extra bits in the payload to ensure it is the right size
		usrMsgSerialSized, err := broadcast.NewSizedBroadcast(
			ch.broadcast.MaxAsymmetricPayloadSize(), usrMsgSerial)
		if err != nil {
			return nil, err
		}

		return usrMsgSerialSized, nil
	}

	// TODO: send the send message over to reception manually so it is added to
	// the database early This requires an entire project in order to track
	// round state.
	rid, ephid, err := ch.broadcast.BroadcastWithAssembler(assemble, params)
	return msgId, rid, ephid, err
}

// SendAdminGeneric is used to send a raw message over a channel encrypted
// with admin keys, identifying it as sent by the admin. In general, it
// should be wrapped in a function which defines the wire protocol
// If the final message, before being sent over the wire, is too long, this will
// return an error. The message must be at most 510 bytes long.
func (m *manager) SendAdminGeneric(privKey *rsa.PrivateKey, channelID *id.ID,
	messageType MessageType, msg []byte, validUntil time.Duration,
	params cmix.CMIXParams) (cryptoChannel.MessageID, id.Round, ephemeral.Id,
	error) {

	//find the channel
	ch, err := m.getChannel(channelID)
	if err != nil {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, err
	}

	//verify the private key is correct
	if ch.broadcast.Get().RsaPubKey.N.Cmp(privKey.GetPublic().N) != 0 {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, WrongPrivateKey
	}

	var msgId cryptoChannel.MessageID
	//Note: we are not checking check if message is too long before trying to
	//find a round

	//Build the function pointer that will build the message
	assemble := func(rid id.Round) ([]byte, error) {

		//Build the message
		chMsg := &ChannelMessage{
			Lease:       validUntil.Nanoseconds(),
			RoundID:     uint64(rid),
			PayloadType: uint32(messageType),
			Payload:     msg,
		}

		//Serialize the message
		chMsgSerial, err := proto.Marshal(chMsg)
		if err != nil {
			return nil, err
		}

		msgId = cryptoChannel.MakeMessageID(chMsgSerial)

		//check if the message is too long
		if len(chMsgSerial) > broadcast.MaxSizedBroadcastPayloadSize(privKey.Size()) {
			return nil, MessageTooLongErr
		}

		//Fill in any extra bits in the payload to ensure it is the right size
		chMsgSerialSized, err := broadcast.NewSizedBroadcast(
			ch.broadcast.MaxAsymmetricPayloadSize(), chMsgSerial)
		if err != nil {
			return nil, err
		}

		return chMsgSerialSized, nil
	}

	// TODO: send the send message over to reception manually so it is added to
	// the database early. This requires an entire project in order to track
	// round state.
	rid, ephid, err := ch.broadcast.BroadcastAsymmetricWithAssembler(privKey,
		assemble, params)
	return msgId, rid, ephid, err
}

// SendMessage is used to send a formatted message over a channel.
// Due to the underlying encoding using compression, it isn't
// possible to define the largest payload that can be sent, but
// it will always be possible to send a payload of 798 bytes at minimum
func (m *manager) SendMessage(channelID *id.ID, msg string,
	validUntil time.Duration, params cmix.CMIXParams) (
	cryptoChannel.MessageID, id.Round, ephemeral.Id, error) {
	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: nil,
	}

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, err
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
	params cmix.CMIXParams) (cryptoChannel.MessageID, id.Round, ephemeral.Id,
	error) {
	txt := &CMIXChannelText{
		Version:        cmixChannelTextVersion,
		Text:           msg,
		ReplyMessageID: replyTo[:],
	}

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, Text, txtMarshaled, validUntil, params)
}

// SendReaction is used to send a reaction to a message over a channel.
// The reaction must be a single emoji with no other characters, and will
// be rejected otherwise.
// Clients will drop the reaction if they do not recognize the reactTo message
func (m *manager) SendReaction(channelID *id.ID, reaction string,
	reactTo cryptoChannel.MessageID, params cmix.CMIXParams) (
	cryptoChannel.MessageID, id.Round, ephemeral.Id, error) {

	if err := ValidateReaction(reaction); err != nil {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, err
	}

	react := &CMIXChannelReaction{
		Version:           cmixChannelReactionVersion,
		Reaction:          reaction,
		ReactionMessageID: reactTo[:],
	}

	reactMarshaled, err := proto.Marshal(react)
	if err != nil {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, Reaction, reactMarshaled, ValidForever,
		params)
}
