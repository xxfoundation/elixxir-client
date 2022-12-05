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
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
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

const returnedRound = 42

type mockBroadcastChannel struct {
	hasRun bool

	payload []byte
	params  cmix.CMIXParams

	pk rsa.PrivateKey

	crypto *cryptoBroadcast.Channel
}

func (m *mockBroadcastChannel) MaxPayloadSize() int {
	return 1024
}

func (m *mockBroadcastChannel) MaxRSAToPublicPayloadSize() int {
	return 512
}

func (m *mockBroadcastChannel) Get() *cryptoBroadcast.Channel {
	return m.crypto
}

func (m *mockBroadcastChannel) Broadcast(payload []byte,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {

	m.hasRun = true

	m.payload = payload
	m.params = cMixParams

	return rounds.Round{ID: 123}, ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastWithAssembler(
	assembler broadcast.Assembler, cMixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	m.hasRun = true

	var err error

	m.payload, err = assembler(returnedRound)
	m.params = cMixParams

	return rounds.Round{ID: 123}, ephemeral.Id{}, err
}

func (m *mockBroadcastChannel) BroadcastRSAtoPublic(pk rsa.PrivateKey,
	payload []byte, cMixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	m.hasRun = true

	m.payload = payload
	m.params = cMixParams

	m.pk = pk
	return rounds.Round{ID: 123}, ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastRSAToPublicWithAssembler(
	pk rsa.PrivateKey, assembler broadcast.Assembler,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {

	m.hasRun = true

	var err error

	m.payload, err = assembler(returnedRound)
	m.params = cMixParams

	m.pk = pk

	return rounds.Round{ID: 123}, ephemeral.Id{}, err
}

func (m *mockBroadcastChannel) RegisterListener(
	broadcast.ListenerFunc, broadcast.Method) error {
	return nil
}
func (m *mockBroadcastChannel) Stop() {}

type mockNameService struct {
	validChMsg bool
}

func (m *mockNameService) GetUsername() string {
	return "Alice"
}

func (m *mockNameService) GetChannelValidationSignature() (
	signature []byte, lease time.Time) {
	return []byte("fake validation sig"), netTime.Now()
}

func (m *mockNameService) GetChannelPubkey() ed25519.PublicKey {
	return []byte("fake pubkey")
}

func (m *mockNameService) SignChannelMessage([]byte) (signature []byte, err error) {
	return []byte("fake sig"), nil
}

func (m *mockNameService) ValidateChannelMessage(
	string, time.Time, ed25519.PublicKey, []byte) bool {
	return m.validChMsg
}

func TestSendGeneric(t *testing.T) {
	nameService := new(mockNameService)
	nameService.validChMsg = true

	rng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	m := &manager{
		me:       pi,
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		events:   initEvents(&mockEventModel{}, kv),
		nicknameManager: &nicknameManager{
			byChannel: make(map[id.ID]string),
			kv:        nil,
		},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(chID *id.ID,
			umi *userMessageInternal, ts time.Time,
			receptionID receptionID.EphemeralIdentity,
			round rounds.Round, status SentStatus) (uint64, error) {
			return 0, nil
		}, func(chID *id.ID, cm *ChannelMessage, ts time.Time,
			messageID cryptoChannel.MessageID,
			receptionID receptionID.EphemeralIdentity, round rounds.Round,
			status SentStatus) (uint64, error) {
			return 0, nil
		}, func(uuid uint64, messageID *cryptoChannel.MessageID,
			timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
			status *SentStatus) {
		}, crng),
	}

	channelID := new(id.ID)
	messageType := Text
	msg := []byte("hello world")
	validUntil := time.Hour
	params := new(cmix.CMIXParams)

	mbc := &mockBroadcastChannel{}

	m.channels[*channelID] = &joinedChannel{
		broadcast: mbc,
	}

	messageId, _, _, err :=
		m.SendGeneric(channelID, messageType, msg, validUntil, *params)
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
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s",
			umi.messageID, messageId)
	}

	if !bytes.Equal(umi.GetChannelMessage().Payload, msg) {
		t.Errorf("The payload does not match. %s vs %s",
			umi.GetChannelMessage().Payload, msg)
	}

	if MessageType(umi.GetChannelMessage().PayloadType) != messageType {
		t.Fatalf("Message types do not match, %s vs %s",
			MessageType(umi.GetChannelMessage().PayloadType), messageType)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			umi.GetChannelMessage().RoundID, returnedRound)
	}
}

func TestAdminGeneric(t *testing.T) {
	prng := rand.New(rand.NewSource(64))
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	m := &manager{
		me:              pi,
		channels:        make(map[id.ID]*joinedChannel),
		kv:              kv,
		rng:             rng,
		nicknameManager: &nicknameManager{byChannel: make(map[id.ID]string)},
		st: loadSendTracker(&mockBroadcastClient{}, kv,
			func(*id.ID, *userMessageInternal, time.Time,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(*id.ID, *ChannelMessage, time.Time, cryptoChannel.MessageID,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(uint64, *cryptoChannel.MessageID, *time.Time, *rounds.Round,
				*bool, *bool, *SentStatus) {
			},
			rng),
	}

	messageType := Text
	msg := []byte("hello world")
	validUntil := time.Hour

	ch, _, err :=
		m.generateChannel("test", "test", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}

	mbc := &mockBroadcastChannel{crypto: ch}

	m.channels[*ch.ReceptionID] = &joinedChannel{
		broadcast: mbc,
	}

	messageId, _, _, err := m.SendAdminGeneric(ch.ReceptionID, messageType, msg,
		validUntil, cmix.GetDefaultCMIXParams())
	if err != nil {
		t.Fatalf("Failed to SendAdminGeneric: %v", err)
	}

	// Verify the message was handled correctly

	msgID := cryptoChannel.MakeMessageID(mbc.payload, ch.ReceptionID)

	if !msgID.Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s", msgID, messageId)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Fatalf("Failed to decode the channel message: %+v", err)
	}

	if !bytes.Equal(chMgs.Payload, msg) {
		t.Errorf("Messages do not match, %s vs %s", chMgs.Payload, msg)
	}

	if MessageType(chMgs.PayloadType) != messageType {
		t.Errorf("Message types do not match, %s vs %s",
			MessageType(chMgs.PayloadType), messageType)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			chMgs.RoundID, returnedRound)
	}
}

func TestSendMessage(t *testing.T) {
	nameService := new(mockNameService)
	nameService.validChMsg = true

	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())

	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	m := &manager{
		me:       pi,
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		events:   initEvents(&mockEventModel{}, kv),
		nicknameManager: &nicknameManager{
			byChannel: make(map[id.ID]string),
			kv:        nil,
		},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(chID *id.ID,
			umi *userMessageInternal, ts time.Time,
			receptionID receptionID.EphemeralIdentity,
			round rounds.Round, status SentStatus) (uint64, error) {
			return 0, nil
		}, func(chID *id.ID, cm *ChannelMessage, ts time.Time,
			messageID cryptoChannel.MessageID,
			receptionID receptionID.EphemeralIdentity, round rounds.Round,
			status SentStatus) (uint64, error) {
			return 0, nil
		}, func(uuid uint64, messageID *cryptoChannel.MessageID,
			timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
			status *SentStatus) {
		}, crng),
	}

	channelID := new(id.ID)
	messageType := Text
	msg := "hello world"
	validUntil := time.Hour
	params := new(cmix.CMIXParams)

	mbc := &mockBroadcastChannel{}

	m.channels[*channelID] = &joinedChannel{
		broadcast: mbc,
	}

	messageId, _, _, err := m.SendMessage(channelID, msg, validUntil, *params)
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
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s",
			umi.messageID, messageId)
	}

	if MessageType(umi.GetChannelMessage().PayloadType) != messageType {
		t.Fatalf("Message types do not match, %s vs %s",
			MessageType(umi.GetChannelMessage().PayloadType), messageType)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			umi.GetChannelMessage().RoundID, returnedRound)
	}

	// Decode the text message
	txt := &CMIXChannelText{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not decode cmix channel text: %+v", err)
	}

	if txt.Text != msg {
		t.Errorf("Content of message is incorrect: %s vs %s", txt.Text, msg)
	}

	if txt.ReplyMessageID != nil {
		t.Errorf("Reply ID on a text message is not nil")
	}
}

func TestSendReply(t *testing.T) {
	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())

	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	m := &manager{
		me:       pi,
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		events:   initEvents(&mockEventModel{}, kv),
		nicknameManager: &nicknameManager{
			byChannel: make(map[id.ID]string),
			kv:        nil,
		},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(chID *id.ID,
			umi *userMessageInternal, ts time.Time,
			receptionID receptionID.EphemeralIdentity,
			round rounds.Round, status SentStatus) (uint64, error) {
			return 0, nil
		}, func(chID *id.ID, cm *ChannelMessage, ts time.Time,
			messageID cryptoChannel.MessageID,
			receptionID receptionID.EphemeralIdentity, round rounds.Round,
			status SentStatus) (uint64, error) {
			return 0, nil
		}, func(uuid uint64, messageID *cryptoChannel.MessageID,
			timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
			status *SentStatus) {
		}, crng),
	}

	channelID := new(id.ID)
	messageType := Text
	msg := "hello world"
	validUntil := time.Hour
	params := new(cmix.CMIXParams)

	replyMsgID := cryptoChannel.MessageID{}
	replyMsgID[0] = 69

	mbc := &mockBroadcastChannel{}

	m.channels[*channelID] = &joinedChannel{
		broadcast: mbc,
	}

	messageId, _, _, err :=
		m.SendReply(channelID, msg, replyMsgID, validUntil, *params)
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
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s",
			umi.messageID, messageId)
	}

	if MessageType(umi.GetChannelMessage().PayloadType) != messageType {
		t.Fatalf("Message types do not match, %s vs %s",
			MessageType(umi.GetChannelMessage().PayloadType), messageType)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			umi.GetChannelMessage().RoundID, returnedRound)
	}

	// Decode the text message
	txt := &CMIXChannelText{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not decode cmix channel text: %+v", err)
	}

	if txt.Text != msg {
		t.Errorf("Content of message is incorrect: %s vs %s", txt.Text, msg)
	}

	if !bytes.Equal(txt.ReplyMessageID, replyMsgID[:]) {
		t.Errorf("The reply message ID is not what was passed in")
	}
}

func TestSendReaction(t *testing.T) {
	prng := rand.New(rand.NewSource(64))
	kv := versioned.NewKV(ekv.MakeMemstore())

	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	m := &manager{
		me:       pi,
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		events:   initEvents(&mockEventModel{}, kv),
		nicknameManager: &nicknameManager{
			byChannel: make(map[id.ID]string),
			kv:        nil,
		},
		st: loadSendTracker(&mockBroadcastClient{}, kv, func(chID *id.ID,
			umi *userMessageInternal, ts time.Time,
			receptionID receptionID.EphemeralIdentity,
			round rounds.Round, status SentStatus) (uint64, error) {
			return 0, nil
		}, func(chID *id.ID, cm *ChannelMessage, ts time.Time,
			messageID cryptoChannel.MessageID,
			receptionID receptionID.EphemeralIdentity, round rounds.Round,
			status SentStatus) (uint64, error) {
			return 0, nil
		}, func(uuid uint64, messageID *cryptoChannel.MessageID,
			timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
			status *SentStatus) {
		}, crng),
	}

	channelID := new(id.ID)
	messageType := Reaction
	msg := "🍆"
	params := new(cmix.CMIXParams)

	replyMsgID := cryptoChannel.MessageID{}
	replyMsgID[0] = 69

	mbc := &mockBroadcastChannel{}

	m.channels[*channelID] = &joinedChannel{
		broadcast: mbc,
	}

	messageId, _, _, err := m.SendReaction(channelID, msg, replyMsgID, *params)
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
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s",
			umi.messageID, messageId)
	}

	if MessageType(umi.GetChannelMessage().PayloadType) != messageType {
		t.Fatalf("Message types do not match, %s vs %s",
			MessageType(umi.GetChannelMessage().PayloadType), messageType)
	}

	if umi.GetChannelMessage().RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			umi.GetChannelMessage().RoundID, returnedRound)
	}

	// Decode the text message
	txt := &CMIXChannelReaction{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not decode cmix channel text: %+v", err)
	}

	if txt.Reaction != msg {
		t.Errorf("Content of message is incorrect: %s vs %s", txt.Reaction, msg)
	}

	if !bytes.Equal(txt.ReactionMessageID, replyMsgID[:]) {
		t.Errorf("The reply message ID is not what was passed in")
	}
}

func Test_manager_DeleteMessage(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	kv := versioned.NewKV(ekv.MakeMemstore())

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv,
			func(*id.ID, *userMessageInternal, time.Time,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(*id.ID, *ChannelMessage, time.Time, cryptoChannel.MessageID,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(uint64, *cryptoChannel.MessageID, *time.Time, *rounds.Round,
				*bool, *bool, *SentStatus) {
			},
			crng),
	}

	ch, _, err :=
		m.generateChannel("test", "test", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	targetedMessageID := cryptoChannel.MessageID{56}
	mbc := &mockBroadcastChannel{}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	messageId, _, _, err := m.DeleteMessage(
		ch.ReceptionID, targetedMessageID, false, cmix.CMIXParams{})
	if err != nil {
		t.Fatalf("SendReaction error: %+v", err)
	}

	// Verify the message was handled correctly
	msgID := cryptoChannel.MakeMessageID(mbc.payload, ch.ReceptionID)
	if !msgID.Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s", msgID, messageId)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Fatalf("Failed to decode the channel message: %+v", err)
	}

	if MessageType(chMgs.PayloadType) != Delete {
		t.Errorf("Message types do not match, %s vs %s",
			MessageType(chMgs.PayloadType), Delete)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			chMgs.RoundID, returnedRound)
	}

	// Decode the text message
	deleteMsg := &CMIXChannelDelete{}
	err = proto.Unmarshal(chMgs.Payload, deleteMsg)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelDelete: %+v", err)
	}

	if !bytes.Equal(deleteMsg.MessageID, targetedMessageID[:]) {
		t.Errorf("The reply message ID is not what was passed in")
	}
}

func Test_manager_PinMessage(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	kv := versioned.NewKV(ekv.MakeMemstore())

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv,
			func(*id.ID, *userMessageInternal, time.Time,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(*id.ID, *ChannelMessage, time.Time, cryptoChannel.MessageID,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(uint64, *cryptoChannel.MessageID, *time.Time, *rounds.Round,
				*bool, *bool, *SentStatus) {
			},
			crng),
	}

	ch, _, err :=
		m.generateChannel("test", "test", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	targetedMessageID := cryptoChannel.MessageID{56}
	mbc := &mockBroadcastChannel{}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	messageId, _, _, err := m.PinMessage(
		ch.ReceptionID, targetedMessageID, false, cmix.CMIXParams{})
	if err != nil {
		t.Fatalf("SendReaction error: %+v", err)
	}

	// Verify the message was handled correctly
	msgID := cryptoChannel.MakeMessageID(mbc.payload, ch.ReceptionID)
	if !msgID.Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s", msgID, messageId)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Fatalf("Failed to decode the channel message: %+v", err)
	}

	if MessageType(chMgs.PayloadType) != Pinned {
		t.Errorf("Message types do not match, %s vs %s",
			MessageType(chMgs.PayloadType), Pinned)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			chMgs.RoundID, returnedRound)
	}

	// Decode the text message
	deleteMsg := &CMIXChannelPinned{}
	err = proto.Unmarshal(chMgs.Payload, deleteMsg)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelPinned: %+v", err)
	}

	if !bytes.Equal(deleteMsg.MessageID, targetedMessageID[:]) {
		t.Errorf("The reply message ID is not what was passed in")
	}
}

func Test_manager_MuteUser(t *testing.T) {
	prng := rand.New(rand.NewSource(6784))
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	kv := versioned.NewKV(ekv.MakeMemstore())

	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		kv:       kv,
		rng:      crng,
		st: loadSendTracker(&mockBroadcastClient{}, kv,
			func(*id.ID, *userMessageInternal, time.Time,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(*id.ID, *ChannelMessage, time.Time, cryptoChannel.MessageID,
				receptionID.EphemeralIdentity, rounds.Round, SentStatus) (
				uint64, error) {
				return 0, nil
			},
			func(uint64, *cryptoChannel.MessageID, *time.Time, *rounds.Round,
				*bool, *bool, *SentStatus) {
			},
			crng),
	}

	ch, _, err :=
		m.generateChannel("test", "test", cryptoBroadcast.Public, 1000)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	mbc := &mockBroadcastChannel{}
	m.channels[*ch.ReceptionID] = &joinedChannel{broadcast: mbc}

	messageId, _, _, err :=
		m.MuteUser(ch.ReceptionID, pi.PubKey, false, cmix.CMIXParams{})
	if err != nil {
		t.Fatalf("SendReaction error: %+v", err)
	}

	// Verify the message was handled correctly
	msgID := cryptoChannel.MakeMessageID(mbc.payload, ch.ReceptionID)
	if !msgID.Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s", msgID, messageId)
	}

	// Decode the channel message
	chMgs := &ChannelMessage{}
	if err = proto.Unmarshal(mbc.payload, chMgs); err != nil {
		t.Fatalf("Failed to decode the channel message: %+v", err)
	}

	if MessageType(chMgs.PayloadType) != Mute {
		t.Errorf("Message types do not match, %s vs %s",
			MessageType(chMgs.PayloadType), Mute)
	}

	if chMgs.RoundID != returnedRound {
		t.Errorf("The returned round is incorrect, %d vs %d",
			chMgs.RoundID, returnedRound)
	}

	// Decode the text message
	deleteMsg := &CMIXChannelMute{}
	err = proto.Unmarshal(chMgs.Payload, deleteMsg)
	if err != nil {
		t.Fatalf("Could not proto unmarshal CMIXChannelPinned: %+v", err)
	}

	if !bytes.Equal(deleteMsg.PubKey, pi.PubKey) {
		t.Errorf("The reply public key is not what was passed in")
	}
}
