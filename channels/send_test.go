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
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
)

func Test_manager_SendGeneric(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	kv := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		local:           kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID, messageType)
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
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	remote, err := remote.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		local:           kv,
		rng:             crng,
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string)},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		local:           kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID, Text)
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
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		local:           kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID, Text)
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
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		local:           kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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
	umi, err := unmarshalUserMessageInternal(mbc.payload, channelID, Reaction)
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
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	remote, err := remote.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	require.NoError(t, err)

	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		local:           kv,
		rng:             crng,
		events:          initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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
	umi, err := unmarshalUserMessageInternal(mbc.payload, ch.ReceptionID, Silent)
	require.NoError(t, err)

	// Do checks of the data
	require.True(t, umi.GetMessageID().Equals(messageID))

	// Decode the text message
	txt := &CMIXChannelSilentMessage{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	require.NoError(t, err)

}

func Test_manager_SendInvite(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	prng := rand.New(rand.NewSource(64))
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	remote, err := remote.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	require.NoError(t, err)

	m := &manager{
		me:               pi,
		channels:         make(map[id.ID]*joinedChannel),
		local:            kv,
		rng:              crng,
		events:           initEvents(&mockEventModel{}, 512, kv, crng),
		nicknameManager:  &nicknameManager{byChannel: make(map[id.ID]string), remote: nil},
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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

	invitedChannelID, inviteeChannel := ch.ReceptionID, ch

	msg := "Dude check out this channel!"
	params := new(cmix.CMIXParams)
	mbc := &mockBroadcastChannel{
		crypto: ch,
	}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}
	host := "https://internet.speakeasy.tech/"
	maxUses := 0
	messageID, _, _, err := m.SendInvite(invitedChannelID, msg,
		inviteeChannel, host, ValidForever, *params, nil)
	require.NoError(t, err)

	// Verify the message was handled correctly

	// Decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload, invitedChannelID, Invitation)
	require.NoError(t, err)

	// Do checks of the data
	require.True(t, umi.GetMessageID().Equals(messageID))

	// Decode the text message
	txt := &CMIXChannelInvitation{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	require.NoError(t, err)

	// Ensure invite URL matches expected
	expectedLink, expectedPassword, err := ch.ShareURL(host, maxUses, rng)
	require.NoError(t, err)
	require.Equal(t, expectedLink, txt.InviteLink)
	require.Equal(t, expectedPassword, txt.Password)

}

func Test_manager_DeleteMessage(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	remote, err := remote.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		local:    kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv,
			func(*id.ID, *userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	remote, err := remote.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		local:    kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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
	mem := ekv.MakeMemstore()
	kv := versioned.NewKV(mem)
	remote := collective.TestingKV(t, mem, collective.StandardPrefexs, nil)
	remote, err := remote.Prefix(collective.StandardRemoteSyncPrefix)
	require.NoError(t, err)
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		local:    kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(*id.ID,
			*userMessageInternal, []byte, time.Time,
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
		adminKeysManager: newAdminKeysManager(remote, func(ch *id.ID, isAdmin bool) {}),
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

// Unit test of makeUserPingTags.
func Test_makeUserPingTags(t *testing.T) {
	prng := rand.New(rand.NewSource(579498))
	types := []PingType{ReplyPing, MentionPing}

	for i := 0; i < 6; i++ {
		pings := map[PingType][]ed25519.PublicKey{
			ReplyPing: {}, MentionPing: {},
		}
		expected := make([]string, i)
		for j := 0; j < i; j++ {
			pt := types[prng.Intn(len(types))]
			user, _, _ := ed25519.GenerateKey(prng)
			pings[pt] = append(pings[pt], user)
			expected[j] = makeUserPingTag(user, pt)
		}
		if i == 0 {
			expected = nil
		}
		tags := makeUserPingTags(pings)
		sort.Strings(expected)
		sort.Strings(tags)
		if !reflect.DeepEqual(expected, tags) {
			t.Errorf("Unexpected tags (%d).\nexpected: %#v\nreceived: %#v",
				i, expected, tags)
		}
	}
}

// Consistency test of makeUserPingTag.
func TestManager_makeUserPingTag_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(579498))
	expected := []string{
		"32b15a6bd6db85bf239a7fe6aeb84072c2646b96188c9c4f7dec6d35928436c2-usrReply",
		"88ab667cdc60e041849a924ed7b6e60fa2de62eca4b67336c86e7620a19d270f-usrReply",
		"ddab884d040d9182950aca0eab2899977caf62eef098d0973f99aa1b4c97f541-usrMention",
		"76341ec7e5da6fe34b80021c4e0f6362d3a1a37091126f8cb74ec8ee5bb375fb-usrMention",
		"d0f05d30b6d62460788ac7a7bde3b0de8065c6b1673ba3a724af90e87f1a152e-usrMention",
		"69a8a814d254be6d3652c5f013d063803c6542a94c0dcd52cb9d9f68bcf14041-usrMention",
		"215a17025a25d892baa595d6677046fe74d625505ad65a64d367975c94477831-usrMention",
		"9b6400094a53921ff041c024265748df635843cb6a50e8a4d6f68f7c52f9e766-usrMention",
		"456e99e886cf889b6b9ce63ecece752267b199032a2236239aaedf9d519a491b-usrReply",
		"2cf8839b549c5b360fb5434de944918d5680e203e173889fd48dd4c91099762e-usrMention",
		"a0a48cffe820a8f7e213b6bb99b99c1270fc8126bbde165b756449895f6f6603-usrReply",
		"d1086f945874607d89d9cc59c6e940217bd69c8d71dc45a0f3728cb6146ae154-usrReply",
		"db60467fc127405470c73cbc4734d1098d43e85c61904a114d95466cfb30781c-usrReply",
		"7bd7b36cf1393fdf8576df1182e5b668bb3f021aa4a1b562d885711153a30395-usrReply",
		"61a25b1fbb5dd9490d43b64d6899ae3f1fac2c2e7bf8cade5fbbcced897f10fe-usrReply",
		"689c8cc76cb15c457f6666457b360701d4f1a6312f6f1a1f1c910353a5d0ce10-usrMention",
		"64c7b1b343f38077d30cb9b99a0ed2844974d984fbd9630999a4a59069a964a7-usrReply",
		"f56ee72f86ee39d52887a08927606bce8a9d21726a82792655aee098705c48a9-usrReply",
		"008da28a9f8f70dafe533283acabbed77cb92bf5058f044bb50e1167ce6f05b4-usrMention",
		"70a659261ad311cf6111639ab6be9677f5afc0ab9e13546b3afc64597ccaabfd-usrReply",
	}
	types := []PingType{ReplyPing, MentionPing}

	for i, exp := range expected {
		pubKey, _, _ := ed25519.GenerateKey(prng)
		tag := makeUserPingTag(pubKey, types[prng.Intn(len(types))])
		if exp != tag {
			t.Errorf("Unexpected tag for key %X (%d)."+
				"\nexpected: %s\nexpected: %s", pubKey, i, exp, tag)
		}
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

func (m *mockBroadcastChannel) Broadcast(payload []byte, _ []string, _ [2]byte,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	m.hasRun = true
	m.payload = payload
	m.params = cMixParams
	return rounds.Round{ID: returnedRound}, ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastWithAssembler(
	assembler broadcast.Assembler, _ []string, _ [2]byte,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	m.hasRun = true
	var err error
	m.payload, err = assembler(returnedRound)
	m.params = cMixParams
	return rounds.Round{ID: returnedRound}, ephemeral.Id{}, err
}

func (m *mockBroadcastChannel) BroadcastRSAtoPublic(pk rsa.PrivateKey,
	payload []byte, _ []string, _ [2]byte, cMixParams cmix.CMIXParams) (
	[]byte, rounds.Round, ephemeral.Id, error) {
	m.hasRun = true
	m.payload = payload
	m.pk = pk
	m.params = cMixParams
	return nil, rounds.Round{ID: returnedRound}, ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastRSAToPublicWithAssembler(
	pk rsa.PrivateKey, assembler broadcast.Assembler, _ []string, _ [2]byte,
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

func (m *mockBroadcastChannel) AsymmetricIdentifier() []byte { panic("implement me") }
func (m *mockBroadcastChannel) SymmetricIdentifier() []byte  { panic("implement me") }

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
