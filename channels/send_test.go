package channels

import (
	"crypto/ed25519"
	"testing"
	"time"

	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/multicastRSA"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"

	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

type mockBroadcastChannel struct{}

func (m *mockBroadcastChannel) MaxPayloadSize() int {
	return 12345
}

func (m *mockBroadcastChannel) MaxAsymmetricPayloadSize() int {
	return 123
}

func (m *mockBroadcastChannel) Get() *cryptoBroadcast.Channel {
	return &cryptoBroadcast.Channel{}
}

func (m *mockBroadcastChannel) Broadcast(payload []byte, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	return id.Round(123), ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastWithAssembler(assembler broadcast.Assembler, cMixParams cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	return id.Round(123), ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastAsymmetric(pk multicastRSA.PrivateKey, payload []byte,
	cMixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	return id.Round(123), ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) BroadcastAsymmetricWithAssembler(
	pk multicastRSA.PrivateKey, assembler broadcast.Assembler,
	cMixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {

	return id.Round(123), ephemeral.Id{}, nil
}

func (m *mockBroadcastChannel) RegisterListener(listenerCb broadcast.ListenerFunc, method broadcast.Method) error {
	return nil
}

func (m *mockBroadcastChannel) Stop() {
}

type mockBroadcastClient struct{}

func (m *mockBroadcastClient) GetMaxMessageLength() int {
	return 123
}

func (m *mockBroadcastClient) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {

	ephemeralId := ephemeral.Id{}

	return id.Round(567), ephemeralId, nil

}

func (m *mockBroadcastClient) IsHealthy() bool {
	return true
}

func (m *mockBroadcastClient) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {

}

func (m *mockBroadcastClient) AddService(clientID *id.ID, newService message.Service,
	response message.Processor) {

}

func (m *mockBroadcastClient) DeleteClientService(clientID *id.ID) {}

func (m *mockBroadcastClient) RemoveIdentity(id *id.ID) {}

type mockNameService struct {
	validChMsg bool
}

func (m *mockNameService) GetUsername() string {
	return "Alice"
}

func (m *mockNameService) GetChannelValidationSignature() (signature []byte, lease time.Time) {
	return nil, time.Now()
}

func (m *mockNameService) GetChannelPubkey() ed25519.PublicKey {
	return nil
}

func (m *mockNameService) SignChannelMessage(message []byte) (signature []byte, err error) {
	return nil, nil
}

func (m *mockNameService) ValidateChannelMessage(username string, lease time.Time,
	pubKey ed25519.PublicKey, authorIDSignature []byte) bool {
	return m.validChMsg
}

type mockEventModel struct{}

func (m *mockEventModel) JoinChannel(channel *cryptoBroadcast.Channel) {}

func (m *mockEventModel) LeaveChannel(channelID *id.ID) {}

func (m *mockEventModel) ReceiveMessage(channelID *id.ID, messageID cryptoChannel.MessageID,
	senderUsername string, text string,
	timestamp time.Time, lease time.Duration, round rounds.Round) {
}

func (m *mockEventModel) ReceiveReply(ChannelID *id.ID, messageID cryptoChannel.MessageID,
	replyTo cryptoChannel.MessageID, SenderUsername string,
	text string, timestamp time.Time, lease time.Duration,
	round rounds.Round) {

}

func (m *mockEventModel) ReceiveReaction(channelID *id.ID, messageID cryptoChannel.MessageID,
	reactionTo cryptoChannel.MessageID, senderUsername string,
	reaction string, timestamp time.Time, lease time.Duration,
	round rounds.Round) {

}

func TestSendGeneric(t *testing.T) {

	kv := versioned.NewKV(ekv.MakeMemstore())
	client := new(mockBroadcastClient)
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	nameService := new(mockNameService)
	nameService.validChMsg = true
	model := new(mockEventModel)

	mm := NewManager(kv, client, rngGen, nameService, model)
	m := mm.(*manager)

	channelID := new(id.ID)
	messageType := MessageType(Text)
	msg := []byte("hello world")
	validUntil := time.Hour
	params := new(cmix.CMIXParams)

	m.channels[*channelID] = &joinedChannel{
		broadcast: &mockBroadcastChannel{},
	}

	messageId, roundId, ephemeralId, err := mm.SendGeneric(
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

}
