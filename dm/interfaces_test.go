////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"time"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

func createLinkedNets() (*mockClient, *mockClient) {
	client1 := newMockClient()
	client2 := newMockClient()

	client1.otherClient = client2
	client2.otherClient = client1
	return client1, client2
}

func newMockClient() *mockClient {
	return &mockClient{
		rndID:       uint64(0),
		processors:  make(map[id.ID]message.Processor),
		otherClient: nil,
	}
}

type mockClient struct {
	rndID       uint64
	processors  map[id.ID]message.Processor
	otherClient *mockClient
}

func (mc *mockClient) GetMaxMessageLength() int {
	return 2048
}
func (mc *mockClient) SendManyWithAssembler(recipients []*id.ID,
	assembler cmix.ManyMessageAssembler,
	params cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error) {
	mc.rndID += 1
	id1, _, _, err := ephemeral.GetId(recipients[0], 8, time.Now().Unix())
	if err != nil {
		return rounds.Round{}, nil, err
	}
	id2, _, _, err := ephemeral.GetId(recipients[1], 8, time.Now().Unix())
	if err != nil {
		return rounds.Round{}, nil, err
	}
	ids := []ephemeral.Id{id1, id2}
	rnd := rounds.Round{ID: id.Round(mc.rndID)}
	msgs, err := assembler(rnd.ID)
	clients := []*mockClient{mc.otherClient, mc}
	if mc.otherClient != nil {
		for i := 0; i < len(recipients); i++ {
			msg := format.NewMessage(2048)
			msg.SetContents(msgs[i].Payload)
			msg.SetKeyFP(msgs[i].Fingerprint)
			msg.SetMac(msgs[i].Mac)
			recID := receptionID.EphemeralIdentity{
				EphId:  ids[i],
				Source: recipients[i],
			}
			clients[i].processors[*recipients[i]].Process(
				msg, recID, rnd)
		}
	}
	return rounds.Round{ID: id.Round(mc.rndID)}, ids, nil
}

func (mc *mockClient) AddIdentity(id *id.ID, validUntil time.Time, persistent bool,
	fallthroughProcessor message.Processor) {
}
func (mc *mockClient) AddIdentityWithHistory(id *id.ID, _ time.Time, _ time.Time,
	_ bool, processor message.Processor) {
	mc.processors[*id] = processor
}
func (mc *mockClient) AddService(*id.ID, message.Service,
	message.Processor) {
	panic("cannot add server to mockClient here")
}
func (mc *mockClient) DeleteClientService(*id.ID) {}
func (mc *mockClient) RemoveIdentity(*id.ID)      {}
func (mc *mockClient) GetRoundResults(time.Duration, cmix.RoundEventCallback,
	...id.Round) {
}
func (mc *mockClient) AddHealthCallback(func(bool)) uint64 { return 0 }
func (mc *mockClient) RemoveHealthCallback(uint64)         {}

func newMockReceiver() *mockReceiver {
	return &mockReceiver{
		Msgs: make([]mockMessage, 0),
	}
}

type mockReceiver struct {
	Msgs []mockMessage
}

func (mr *mockReceiver) Receive(messageID cryptoMessage.ID,
	nickname string, text []byte, pubKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, mType MessageType, status Status) uint64 {
	return 0
}

func (mr *mockReceiver) ReceiveText(messageID cryptoMessage.ID,
	nickname, text string, pubKey ed25519.PublicKey, dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, status Status) uint64 {
	return 0
}

func (mr *mockReceiver) ReceiveReply(messageID cryptoMessage.ID,
	reactionTo cryptoMessage.ID, nickname, text string,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status Status) uint64 {
	return 0
}

func (mr *mockReceiver) ReceiveReaction(messageID cryptoMessage.ID,
	reactionTo cryptoMessage.ID, nickname, reaction string,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status Status) uint64 {
	return 0
}

func (mr *mockReceiver) UpdateSentStatus(uuid uint64, messageID cryptoMessage.ID,
	timestamp time.Time, round rounds.Round, status Status) {
}

type mockMessage struct {
	Message   string
	PubKey    ed25519.PublicKey
	DMToken   uint32
	MessageID cryptoMessage.ID
}
