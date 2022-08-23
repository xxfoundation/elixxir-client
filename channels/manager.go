package channels

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
)

type manager struct {
	//List of all channels
	channels map[*id.ID]*joinedChannel
	mux      sync.RWMutex

	//External references
	kv     *versioned.KV
	client broadcast.Client
	rng    *fastRNG.StreamGenerator
	name   NameService

	//Events model
	events
	broadcastMaker broadcast.NewBroadcastChannelFunc
}

func (m *manager) Send(channelID *id.ID, msg []byte, validUntil time.Duration,
	messageType MessageType, params cmix.CMIXParams) (cryptoChannel.MessageID, id.Round, ephemeral.Id,
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
