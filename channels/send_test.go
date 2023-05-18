////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"crypto/ed25519"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/collective"
	"math/rand"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/netTime"

	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"

	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/cmix"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
)

func Test_manager_SendGeneric(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		kv:              kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity, rounds.Round,
			SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round, *bool, *bool,
			*SentStatus) error {
			return nil
		}, crng),
	}

	rng := crng.GetStream()
	defer rng.Close()
	channelID, _ := id.NewRandomID(rng, id.User)
	messageType := Text
	msg := []byte("hello world")
	validUntil := time.Hour
	params := cmix.CMIXParams{DebugTag: "ChannelTest"}
	mbc := &mockBroadcastChannel{}
	m.channels[*channelID] = &joinedChannel{broadcast: mbc}

	messageID, _, _, err :=
		m.SendGeneric(channelID, messageType, msg, validUntil, true, params, nil)
	if err != nil {
		t.Fatalf("SendGeneric error: %+v", err)
	}

	// Verify the message was handled correctly

	// Decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %+v", err)
	}

	// Do checks of the data
	if !umi.GetMessageID().Equals(messageID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			messageID, umi.messageID)
	}

	if !bytes.Equal(umi.GetChannelMessage().Payload, msg) {
		t.Errorf("Incorrect payload.\nexpected: %q\nreceived: %q",
			msg, umi.GetChannelMessage().Payload)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, umi.GetChannelMessage().RoundID)
	}
}

func Test_manager_SendAdminGeneric(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		kv:              kv,
		rng:             crng,
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string)},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity, rounds.Round,
			SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round, *bool, *bool,
			*SentStatus) error {
			return nil
		}, crng),
	}

	messageType := Text
	msg := []byte("hello world")
	validUntil := time.Hour

	ch, _, err := m.generateChannel("abc", "abc", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	mbc := &mockBroadcastChannel{crypto: ch}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	messageID, _, _, err := m.SendAdminGeneric(ch.ReceptionID, messageType, msg,
		validUntil, true, cmix.GetDefaultCMIXParams())
	if err != nil {
		t.Fatalf("Failed to SendAdminGeneric: %v", err)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Fatalf("Could not proto unmarshal ChannelMessage: %+v", err)
	}

	if !bytes.Equal(chMgs.Payload, msg) {
		t.Errorf("Incorrect message.\nexpected: %q\nreceived: %q",
			msg, chMgs.Payload)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, chMgs.RoundID)
	}

	msgID := message.DeriveChannelMessageID(ch.ReceptionID, chMgs.RoundID,
		mbc.payload)

	if !msgID.Equals(messageID) {
		t.Errorf("The message IDs do not match. %s vs %s", msgID,
			messageID)
	}

}

func Test_manager_SendMessage(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := collective.TestingKV(t, ekv.MakeMemstore(), collective.StandardPrefexs, nil)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		kv:              kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity, rounds.Round,
			SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round, *bool, *bool,
			*SentStatus) error {
			return nil
		}, crng),
	}

	rng := crng.GetStream()
	defer rng.Close()
	channelID, _ := id.NewRandomID(rng, id.User)
	msg := "hello world"
	validUntil := time.Hour
	params := cmix.CMIXParams{DebugTag: "ChannelTest"}
	mbc := &mockBroadcastChannel{}
	m.channels[*channelID] = &joinedChannel{broadcast: mbc}

	messageID, _, _, err := m.SendMessage(channelID, msg, validUntil, params, nil)
	if err != nil {
		t.Fatalf("SendMessage error: %+v", err)
	}

	// Verify the message was handled correctly

	// Decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %+v", err)
	}

	// Do checks of the data
	if !umi.GetMessageID().Equals(messageID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			messageID, umi.messageID)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, umi.GetChannelMessage().RoundID)
	}

	// Decode the text message
	txt := &CMIXChannelText{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelText: %+v", err)
	}

	if txt.Text != msg {
		t.Errorf("Incorrect message contents.\nexpected: %s\nreceived: %s",
			msg, txt.Text)
	}

	if txt.ReplyMessageID != nil {
		t.Errorf("Incorrect ReplyMessageID.\nexpected: %v\nreceived: %v",
			nil, txt.ReplyMessageID)
	}
}

func Test_manager_SendReply(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		kv:              kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity, rounds.Round,
			SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round, *bool, *bool,
			*SentStatus) error {
			return nil
		}, crng),
	}

	rng := crng.GetStream()
	defer rng.Close()
	channelID, _ := id.NewRandomID(rng, id.User)
	msg := "hello world"
	validUntil := time.Hour
	params := new(cmix.CMIXParams)
	replyMsgID := message.ID{69}
	mbc := &mockBroadcastChannel{}
	m.channels[*channelID] = &joinedChannel{broadcast: mbc}

	messageID, _, _, err :=
		m.SendReply(channelID, msg, replyMsgID, validUntil, *params, nil)
	if err != nil {
		t.Fatalf("SendReply error: %+v", err)
	}

	// Verify the message was handled correctly

	// Decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %+v", err)
	}

	// Do checks of the data
	if !umi.GetMessageID().Equals(messageID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			messageID, umi.messageID)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, umi.GetChannelMessage().RoundID)
	}

	// Decode the text message
	txt := &CMIXChannelText{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelText: %+v", err)
	}

	if txt.Text != msg {
		t.Errorf("Incorrect message contents.\nexpected: %s\nreceived: %s",
			msg, txt.Text)
	}

	if !bytes.Equal(txt.ReplyMessageID, replyMsgID[:]) {
		t.Errorf("Incorrect ReplyMessageID.\nexpected: %v\nreceived: %v",
			replyMsgID[:], txt.ReplyMessageID)
	}
}

func Test_manager_SendReaction(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		kv:              kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity,
			rounds.Round, SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round,
			*bool, *bool, *SentStatus) error {
			return nil
		}, crng),
	}

	rng := crng.GetStream()
	defer rng.Close()
	channelID, _ := id.NewRandomID(rng, id.User)
	msg := "🍆"
	params := new(cmix.CMIXParams)
	replyMsgID := message.ID{69}
	mbc := &mockBroadcastChannel{}
	m.channels[*channelID] = &joinedChannel{broadcast: mbc}

	messageID, _, _, err := m.SendReaction(
		channelID, msg, replyMsgID, ValidForever, *params)
	if err != nil {
		t.Fatalf("SendReaction error: %+v", err)
	}

	// Verify the message was handled correctly

	// Decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %+v", err)
	}

	// Do checks of the data
	if !umi.GetMessageID().Equals(messageID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			messageID, umi.messageID)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, umi.GetChannelMessage().RoundID)
	}

	// Decode the text message
	txt := &CMIXChannelReaction{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelReaction: %+v", err)
	}

	if txt.Reaction != msg {
		t.Errorf("Incorrect reaction.\nexpected: %s\nreceived: %s",
			msg, txt.Reaction)
	}

	if !bytes.Equal(txt.ReactionMessageID, replyMsgID[:]) {
		t.Errorf("Incorrect ReactionMessageID.\nexpected: %v\nreceived: %v",
			replyMsgID, txt.ReactionMessageID)
	}
}

func Test_manager_SendSilent(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())
	pi, err := cryptoChannel.GenerateIdentity(prng)
	require.NoError(t, err)

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		kv:              kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity,
			rounds.Round, SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round,
			*bool, *bool, *SentStatus) error {
			return nil
		}, crng),
	}

	rng := crng.GetStream()
	defer rng.Close()

	ch, _, err := m.generateChannel("abc", "abc", cryptoBroadcast.Public, 1000)
	require.NoError(t, err)

	params := new(cmix.CMIXParams)
	mbc := &mockBroadcastChannel{
		crypto: ch,
	}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	// Send message
	messageID, _, _, err := m.SendSilent(ch.ReceptionID, ValidForever, *params)
	require.NoError(t, err)

	// Verify the message was handled correctly

	// Decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload, ch.ReceptionID)
	require.NoError(t, err)

	// Do checks of the data
	require.True(t, umi.GetMessageID().Equals(messageID))

	// Decode the text message
	txt := &CMIXChannelSilentMessage{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	require.NoError(t, err)

}

func Test_manager_DeleteMessage(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	kv := versioned.NewKV(ekv.MakeMemstore())

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv,
			func(*id.ID, *userMessageInternal, MessageType, []byte, time.Time,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
				message.ID, receptionID.EphemeralIdentity,
				rounds.Round, SentStatus) (uint64, error) {
				return 0, nil
			}, func(uint64, *message.ID, *time.Time, *rounds.Round,
				*bool, *bool, *SentStatus) error {
				return nil
			}, crng),
	}

	ch, _, err := m.generateChannel("abc", "abc", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	targetedMessageID := message.ID{56}
	mbc := &mockBroadcastChannel{}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	messageID, round, _, err :=
		m.DeleteMessage(ch.ReceptionID, targetedMessageID, cmix.CMIXParams{})
	if err != nil {
		t.Fatalf("SendReaction error: %+v", err)
	}

	// Verify the message was handled correctly
	expectedMessageID := message.
		DeriveChannelMessageID(ch.ReceptionID, uint64(round.ID), mbc.payload)
	if !expectedMessageID.Equals(messageID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			expectedMessageID, messageID)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Fatalf("Could not proto unmarshal ChannelMessage: %+v", err)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, chMgs.RoundID)
	}

	// Decode the text message
	deleteMsg := &CMIXChannelDelete{}
	err = proto.Unmarshal(chMgs.Payload, deleteMsg)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelDelete: %+v", err)
	}

	if !bytes.Equal(deleteMsg.MessageID, targetedMessageID[:]) {
		t.Errorf("Incorrect MessageID.\nexpected: %v\nreceived: %v",
			targetedMessageID, deleteMsg.MessageID)
	}
}

func Test_manager_PinMessage(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	kv := versioned.NewKV(ekv.MakeMemstore())

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity, rounds.Round,
			SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round, *bool, *bool,
			*SentStatus) error {
			return nil
		}, crng),
	}

	ch, _, err := m.generateChannel("abc", "abc", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	targetedMessageID := message.ID{56}
	mbc := &mockBroadcastChannel{}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	messageID, round, _, err := m.PinMessage(ch.ReceptionID, targetedMessageID,
		false, 24*time.Hour, cmix.CMIXParams{})
	if err != nil {
		t.Fatalf("SendReaction error: %+v", err)
	}

	// Verify the message was handled correctly
	expectedMessageID := message.
		DeriveChannelMessageID(ch.ReceptionID, uint64(round.ID), mbc.payload)
	if !expectedMessageID.Equals(messageID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			expectedMessageID, messageID)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Fatalf("Could not proto unmarshal ChannelMessage: %+v", err)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, chMgs.RoundID)
	}

	// Decode the text message
	pinnedMsg := &CMIXChannelPinned{}
	err = proto.Unmarshal(chMgs.Payload, pinnedMsg)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelPinned: %+v", err)
	}

	if !bytes.Equal(pinnedMsg.MessageID, targetedMessageID[:]) {
		t.Errorf("Incorrect MessageID.\nexpected: %v\nreceived: %v",
			targetedMessageID, pinnedMsg.MessageID)
	}
}

func Test_manager_MuteUser(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, MessageType, []byte, time.Time,
			receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
			uint64, error) {
			return 0, nil
		}, func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
			message.ID, receptionID.EphemeralIdentity, rounds.Round,
			SentStatus) (uint64, error) {
			return 0, nil
		}, func(uint64, *message.ID, *time.Time, *rounds.Round, *bool, *bool,
			*SentStatus) error {
			return nil
		}, crng),
	}

	ch, _, err := m.generateChannel("abc", "abc", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	mbc := &mockBroadcastChannel{}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	messageID, round, _, err := m.MuteUser(
		ch.ReceptionID, pi.PubKey, false, 24*time.Hour, cmix.CMIXParams{})
	if err != nil {
		t.Fatalf("SendReaction error: %+v", err)
	}

	// Verify the message was handled correctly
	expectedMessageID := message.
		DeriveChannelMessageID(ch.ReceptionID, uint64(round.ID), mbc.payload)
	if !expectedMessageID.Equals(messageID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			expectedMessageID, messageID)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Errorf("Failed to unmarshal: %s", err)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			returnedRound, chMgs.RoundID)
	}

	// Decode the text message
	muteMsg := &CMIXChannelMute{}
	err = proto.Unmarshal(chMgs.Payload, muteMsg)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelMute: %+v", err)
	}

	if !bytes.Equal(muteMsg.PubKey, pi.PubKey) {
		t.Errorf("Incorrect PubKey.\nexpected: %x\nreceived: %x",
			pi.PubKey, muteMsg.PubKey)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Mock Interfaces                                                            //
////////////////////////////////////////////////////////////////////////////////

const returnedRound = 42

// mockBroadcastChannel adheres to the [broadcast.Channel] interface and is used
// for testing.
type mockBroadcastChannel struct {
	hasRun  bool
	payload []byte
	pk      rsa.PrivateKey
	crypto  *cryptoBroadcast.Channel
	params  cmix.CMIXParams
}

func (m *mockBroadcastChannel) MaxPayloadSize() int            { return 1024 }
func (m *mockBroadcastChannel) MaxRSAToPublicPayloadSize() int { return 512 }
func (m *mockBroadcastChannel) Get() *cryptoBroadcast.Channel  { return m.crypto }

func (m *mockBroadcastChannel) Broadcast(payload []byte, _ []string, _ uint16,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	m.hasRun = true
	m.payload = payload
	m.params = cMixParams
	return rounds.Round{ID: returnedRound}, ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastWithAssembler(
	assembler broadcast.Assembler, _ []string, _ uint16,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	m.hasRun = true
	var err error
	m.payload, err = assembler(returnedRound)
	m.params = cMixParams
	return rounds.Round{ID: returnedRound}, ephemeral.Id{}, err
}

func (m *mockBroadcastChannel) BroadcastRSAtoPublic(pk rsa.PrivateKey,
	payload []byte, _ []string, _ uint16, cMixParams cmix.CMIXParams) (
	[]byte, rounds.Round, ephemeral.Id, error) {
	m.hasRun = true
	m.payload = payload
	m.pk = pk
	m.params = cMixParams
	return nil, rounds.Round{ID: returnedRound}, ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastRSAToPublicWithAssembler(
	pk rsa.PrivateKey, assembler broadcast.Assembler, _ []string, _ uint16,
	cMixParams cmix.CMIXParams) ([]byte, rounds.Round, ephemeral.Id, error) {
	m.hasRun = true
	var err error
	m.payload, err = assembler(returnedRound)
	m.params = cMixParams
	m.pk = pk
	return nil, rounds.Round{ID: returnedRound}, ephemeral.Id{}, err
}

func (m *mockBroadcastChannel) RegisterRSAtoPublicListener(
	broadcast.ListenerFunc, []string) (broadcast.Processor, error) {
	panic("implement me")
}

func (m *mockBroadcastChannel) RegisterSymmetricListener(
	broadcast.ListenerFunc, []string) (broadcast.Processor, error) {
	panic("implement me")
}

func (m *mockBroadcastChannel) Stop() {}

// mockNameService adheres to the NameService interface and is used for testing.
type mockNameService struct {
	validChMsg bool
}

func (m *mockNameService) GetUsername() string { return "Alice" }
func (m *mockNameService) GetChannelValidationSignature() ([]byte, time.Time) {
	return []byte("fake validation sig"), netTime.Now()
}
func (m *mockNameService) GetChannelPubkey() ed25519.PublicKey {
	return []byte("fake pubkey")
}
func (m *mockNameService) SignChannelMessage([]byte) ([]byte, error) {
	return []byte("fake sig"), nil
}
func (m *mockNameService) ValidateChannelMessage(string, time.Time,
	ed25519.PublicKey, []byte) bool {
	return m.validChMsg
}
