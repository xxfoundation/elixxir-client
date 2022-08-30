package channels

import (
	"bytes"
	"crypto/ed25519"
	"gitlab.com/xx_network/crypto/csprng"
	"testing"
	"time"

	"gitlab.com/xx_network/crypto/multicastRSA"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"

	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
)

type mockBroadcastChannel struct {
	hasRun bool

	payload []byte
	params  cmix.CMIXParams

	pk multicastRSA.PrivateKey

	crypto *cryptoBroadcast.Channel
}

func (m *mockBroadcastChannel) MaxPayloadSize() int {
	return 1024
}

func (m *mockBroadcastChannel) MaxAsymmetricPayloadSize() int {
	return 512
}

func (m *mockBroadcastChannel) Get() *cryptoBroadcast.Channel {
	return m.crypto
}

func (m *mockBroadcastChannel) Broadcast(payload []byte, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {

	m.hasRun = true

	m.payload = payload
	m.params = cMixParams

	return id.Round(123), ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastWithAssembler(assembler broadcast.Assembler, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	m.hasRun = true

	var err error

	m.payload, err = assembler(42)
	m.params = cMixParams

	return id.Round(123), ephemeral.Id{}, err
}

func (m *mockBroadcastChannel) BroadcastAsymmetric(pk multicastRSA.PrivateKey, payload []byte,
	cMixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	m.hasRun = true

	m.payload = payload
	m.params = cMixParams

	m.pk = pk
	return id.Round(123), ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastAsymmetricWithAssembler(
	pk multicastRSA.PrivateKey, assembler broadcast.Assembler,
	cMixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {

	m.hasRun = true

	var err error

	m.payload, err = assembler(42)
	m.params = cMixParams

	m.pk = pk

	return id.Round(123), ephemeral.Id{}, err
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

	//Unsize the broadcast
	unsized, err := broadcast.DecodeSizedBroadcast(mbc.payload)
	if err != nil {
		t.Fatalf("Failed to decode the sized broadcast: %s", err)
	}

	//decode the user message
	umi, err := unmarshalUserMessageInternal(unsized)
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
}

func TestAdminGeneric(t *testing.T) {

	nameService := new(mockNameService)
	nameService.validChMsg = true

	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		name:     nameService,
	}

	messageType := Text
	msg := []byte("hello world")
	validUntil := time.Hour

	rng := &csprng.SystemRNG{}
	ch, priv, err := cryptoBroadcast.NewChannel("test", "test", rng)
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

	//Unsize the broadcast
	unsized, err := broadcast.DecodeSizedBroadcast(mbc.payload)
	if err != nil {
		t.Fatalf("Failed to decode the sized broadcast: %s", err)
	}

	//decode the channel message

	umi, err := unmarshalUserMessageInternal(unsized)
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
}
