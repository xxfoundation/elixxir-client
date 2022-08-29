package channels

import (
	"github.com/forPelevin/gomoji"
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

func (m *manager) SendGeneric(channelID *id.ID, msg []byte, validUntil time.Duration,
	messageType MessageType, params cmix.CMIXParams) (cryptoChannel.MessageID,
	id.Round, ephemeral.Id, error) {

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
			UsernameLease:       unameLease.Unix(),
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

		msgId = cryptoChannel.MakeMessageID(usrMsgSerialSized)

		return usrMsgSerialSized, nil
	}

	//TODO: send the send message over to reception manually so it is added to
	//the database early
	rid, ephid, err := ch.broadcast.BroadcastWithAssembler(assemble, params)
	return msgId, rid, ephid, err
}

func (m *manager) SendAdminGeneric(privKey *rsa.PrivateKey, channelID *id.ID,
	msg []byte, validUntil time.Duration, messageType MessageType,
	params cmix.CMIXParams) (cryptoChannel.MessageID, id.Round, ephemeral.Id,
	error) {

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

		msgId = cryptoChannel.MakeMessageID(chMsgSerialSized)

		return chMsgSerialSized, nil
	}

	//TODO: send the send message over to reception manually so it is added to
	//the database early
	rid, ephid, err := ch.broadcast.BroadcastAsymmetricWithAssembler(privKey,
		assemble, params)
	return msgId, rid, ephid, err
}

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

	return m.SendGeneric(channelID, txtMarshaled, validUntil, Text, params)
}

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

	return m.SendGeneric(channelID, txtMarshaled, validUntil, Text, params)
}

func (m *manager) SendReaction(channelID *id.ID, reaction string,
	replyTo cryptoChannel.MessageID, validUntil time.Duration,
	params cmix.CMIXParams) (cryptoChannel.MessageID, id.Round, ephemeral.Id,
	error) {

	if len(reaction) != 1 {
		return error
	}

	txt := &CMIXChannelReaction{
		Version:           cmixChannelReactionVersion,
		Reaction:          reaction,
		ReactionMessageID: replyTo[:],
	}

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoChannel.MessageID{}, 0, ephemeral.Id{}, err
	}

	return m.SendGeneric(channelID, txtMarshaled, validUntil, Text, params)
}
