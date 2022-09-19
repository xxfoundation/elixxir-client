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
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"testing"
	"time"

	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"

	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
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

func (m *mockBroadcastChannel) Broadcast(payload []byte, cMixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {

	m.hasRun = true

	m.payload = payload
	m.params = cMixParams

	return rounds.Round{ID: 123}, ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastWithAssembler(assembler broadcast.Assembler, cMixParams cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	m.hasRun = true

	var err error

	m.payload, err = assembler(returnedRound)
	m.params = cMixParams

	return rounds.Round{ID: 123}, ephemeral.Id{}, err
}

func (m *mockBroadcastChannel) BroadcastRSAtoPublic(pk rsa.PrivateKey, payload []byte,
	cMixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
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

func (m *mockBroadcastChannel) RegisterListener(listenerCb broadcast.ListenerFunc, method broadcast.Method) error {
	return nil
}

func (m *mockBroadcastChannel) Stop() {
}

type mockNameService struct {
	validChMsg bool
}

func (m *mockNameService) GetUsername() string {
	return "Alice"
}

func (m *mockNameService) GetChannelValidationSignature() (signature []byte, lease time.Time) {
	return []byte("fake validation sig"), time.Now()
}

func (m *mockNameService) GetChannelPubkey() ed25519.PublicKey {
	return []byte("fake pubkey")
}

func (m *mockNameService) SignChannelMessage(message []byte) (signature []byte, err error) {
	return []byte("fake sig"), nil
}

func (m *mockNameService) ValidateChannelMessage(username string, lease time.Time,
	pubKey ed25519.PublicKey, authorIDSignature []byte) bool {
	return m.validChMsg
}

func TestSendGeneric(t *testing.T) {

	nameService := new(mockNameService)
	nameService.validChMsg = true

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		name:     nameService,
		st: loadSendTracker(&mockBroadcastClient{},
			versioned.NewKV(ekv.MakeMemstore()), func(chID *id.ID,
				umi *userMessageInternal,
				receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(chID *id.ID, cm *ChannelMessage,
				messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(messageID cryptoChannel.MessageID, status SentStatus) {}),
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

	messageId, roundId, ephemeralId, err := m.SendGeneric(
		channelID,
		messageType,
		msg,
		validUntil,
		*params)
	if err != nil {
		t.Logf("ERROR %v", err)
		t.Fail()
	}
	t.Logf("messageId %v, roundId %v, ephemeralId %v", messageId, roundId, ephemeralId)

	//verify the message was handled correctly

	//decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %s", err)
	}

	// do checks of the data
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s ",
			umi.messageID, messageId)
	}

	if !bytes.Equal(umi.GetChannelMessage().Payload, msg) {
		t.Errorf("The payload does not match. %s vs %s ",
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

	nameService := new(mockNameService)
	nameService.validChMsg = true

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		name:     nameService,
		st: loadSendTracker(&mockBroadcastClient{},
			versioned.NewKV(ekv.MakeMemstore()), func(chID *id.ID,
				umi *userMessageInternal,
				receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(chID *id.ID, cm *ChannelMessage,
				messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(messageID cryptoChannel.MessageID, status SentStatus) {}),
	}

	messageType := Text
	msg := []byte("hello world")
	validUntil := time.Hour

	rng := &csprng.SystemRNG{}
	ch, priv, err := cryptoBroadcast.NewChannel("test", "test",
		1000, rng)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}

	mbc := &mockBroadcastChannel{crypto: ch}

	m.channels[*ch.ReceptionID] = &joinedChannel{
		broadcast: mbc,
	}

	messageId, roundId, ephemeralId, err := m.SendAdminGeneric(priv,
		ch.ReceptionID, messageType, msg, validUntil, cmix.GetDefaultCMIXParams())
	if err != nil {
		t.Fatalf("Failed to SendAdminGeneric: %v", err)
	}
	t.Logf("messageId %v, roundId %v, ephemeralId %v", messageId, roundId, ephemeralId)

	//verify the message was handled correctly

	msgID := cryptoChannel.MakeMessageID(mbc.payload)

	if !msgID.Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s ",
			msgID, messageId)
	}

	//decode the channel message
	chMgs := &ChannelMessage{}
	err = proto.Unmarshal(mbc.payload, chMgs)
	if err != nil {
		t.Fatalf("Failed to decode the channel message: %s", err)
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

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		name:     nameService,
		st: loadSendTracker(&mockBroadcastClient{},
			versioned.NewKV(ekv.MakeMemstore()), func(chID *id.ID,
				umi *userMessageInternal,
				receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(chID *id.ID, cm *ChannelMessage,
				messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(messageID cryptoChannel.MessageID, status SentStatus) {}),
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

	messageId, roundId, ephemeralId, err := m.SendMessage(
		channelID,
		msg,
		validUntil,
		*params)
	if err != nil {
		t.Logf("ERROR %v", err)
		t.Fail()
	}
	t.Logf("messageId %v, roundId %v, ephemeralId %v", messageId, roundId, ephemeralId)

	//verify the message was handled correctly

	//decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %s", err)
	}

	// do checks of the data
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s ",
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

	//decode the text message
	txt := &CMIXChannelText{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not decode cmix channel text: %s", err)
	}

	if txt.Text != msg {
		t.Errorf("Content of message is incorrect: %s vs %s", txt.Text, msg)
	}

	if txt.ReplyMessageID != nil {
		t.Errorf("Reply ID on a text message is not nil")
	}
}

func TestSendReply(t *testing.T) {

	nameService := new(mockNameService)
	nameService.validChMsg = true

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		name:     nameService,
		st: loadSendTracker(&mockBroadcastClient{},
			versioned.NewKV(ekv.MakeMemstore()), func(chID *id.ID,
				umi *userMessageInternal,
				receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(chID *id.ID, cm *ChannelMessage,
				messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(messageID cryptoChannel.MessageID, status SentStatus) {}),
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

	messageId, roundId, ephemeralId, err := m.SendReply(
		channelID, msg, replyMsgID, validUntil, *params)
	if err != nil {
		t.Logf("ERROR %v", err)
		t.Fail()
	}
	t.Logf("messageId %v, roundId %v, ephemeralId %v", messageId, roundId, ephemeralId)

	//verify the message was handled correctly

	//decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %s", err)
	}

	// do checks of the data
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s ",
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

	//decode the text message
	txt := &CMIXChannelText{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not decode cmix channel text: %s", err)
	}

	if txt.Text != msg {
		t.Errorf("Content of message is incorrect: %s vs %s", txt.Text, msg)
	}

	if !bytes.Equal(txt.ReplyMessageID, replyMsgID[:]) {
		t.Errorf("The reply message ID is not what was passed in")
	}
}

func TestSendReaction(t *testing.T) {

	nameService := new(mockNameService)
	nameService.validChMsg = true

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		name:     nameService,
		st: loadSendTracker(&mockBroadcastClient{},
			versioned.NewKV(ekv.MakeMemstore()), func(chID *id.ID,
				umi *userMessageInternal,
				receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(chID *id.ID, cm *ChannelMessage,
				messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
				round rounds.Round, status SentStatus) {
			}, func(messageID cryptoChannel.MessageID, status SentStatus) {}),
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

	messageId, roundId, ephemeralId, err := m.SendReaction(
		channelID, msg, replyMsgID, *params)
	if err != nil {
		t.Logf("ERROR %v", err)
		t.Fail()
	}
	t.Logf("messageId %v, roundId %v, ephemeralId %v", messageId, roundId, ephemeralId)

	//verify the message was handled correctly

	//decode the user message
	umi, err := unmarshalUserMessageInternal(mbc.payload)
	if err != nil {
		t.Fatalf("Failed to decode the user message: %s", err)
	}

	// do checks of the data
	if !umi.GetMessageID().Equals(messageId) {
		t.Errorf("The message IDs do not match. %s vs %s ",
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

	//decode the text message
	txt := &CMIXChannelReaction{}
	err = proto.Unmarshal(umi.GetChannelMessage().Payload, txt)
	if err != nil {
		t.Fatalf("Could not decode cmix channel text: %s", err)
	}

	if txt.Reaction != msg {
		t.Errorf("Content of message is incorrect: %s vs %s", txt.Reaction, msg)
	}

	if !bytes.Equal(txt.ReactionMessageID, replyMsgID[:]) {
		t.Errorf("The reply message ID is not what was passed in")
	}
}
