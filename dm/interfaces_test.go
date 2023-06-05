////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"bytes"
	"crypto/ed25519"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// createLinkedNets links 2 clients together.
func createLinkedNets() (*mockClient, *mockClient) {
	client1 := newMockClient()
	client2 := newMockClient()

	client1.otherClient = client2
	client2.otherClient = client1
	return client1, client2
}

// newMockClient creates a client that can send messages
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
	tmpMsg := format.NewMessage(2048)
	return tmpMsg.ContentsSize()
}

// This calls the assembler (encryption) function and returns mocked
// but valid round IDs, etc.
// When otherClient is not nil, this sends the messages to the linked
// receiver.
func (mc *mockClient) SendManyWithAssembler(recipients []*id.ID,
	assembler cmix.ManyMessageAssembler,
	params cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error) {
	jww.INFO.Printf("SendManyWithAssembler: %s, %s", recipients[0],
		recipients[1])
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
	if err != nil {
		panic(err.Error())
	}
	clients := []*mockClient{mc.otherClient, mc}
	if mc.otherClient != nil {
		for i := 0; i < len(recipients); i++ {
			msg := format.NewMessage(2048)
			msg.SetKeyFP(msgs[i].Fingerprint)
			msg.SetContents(msgs[i].Payload)
			msg.SetMac(msgs[i].Mac)
			SIH, err := msgs[i].Service.Hash(nil, msg.GetContents())
			if err != nil {
				panic(err)
			}
			msg.SetSIH(SIH)
			recID := receptionID.EphemeralIdentity{
				EphId:  ids[i],
				Source: recipients[i],
			}
			clients[i].processors[*recipients[i]].Process(
				msg, []string{}, []byte{}, recID, rnd)
		}
	}
	return rounds.Round{ID: id.Round(mc.rndID)}, ids, nil
}

func (mc *mockClient) AddIdentity(id *id.ID, validUntil time.Time, persistent bool,
	fallthroughProcessor message.Processor) {
}
func (mc *mockClient) AddIdentityWithHistory(id *id.ID, _ time.Time, _ time.Time,
	_ bool, processor message.Processor) {
	jww.INFO.Printf("AddIdentityWithHistory: %s", id)
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

// mockReceiver stores the messages sent to it for testing/debugging
// NOTE: when sending remember the sender sees the sent message twice.
//
//	the receiver receives it only once. See TestE2EDMs test in dm_test.go
//	for details
func newMockReceiver() *mockReceiver {
	return &mockReceiver{
		Msgs:    make([]mockMessage, 0),
		uuid:    0,
		blocked: make([]ed25519.PublicKey, 0),
	}
}

type mockReceiver struct {
	Msgs    []mockMessage
	uuid    uint64
	blocked []ed25519.PublicKey
}

func (mr *mockReceiver) Receive(messageID cryptoMessage.ID,
	nickname string, text []byte, pubKey, senderPubKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, mType MessageType, status Status) uint64 {
	jww.INFO.Printf("Receive: %s", messageID)
	mr.Msgs = append(mr.Msgs, mockMessage{
		Message:   string(text),
		PubKey:    pubKey,
		DMToken:   dmToken,
		MessageID: messageID,
		ReplyTo:   cryptoMessage.ID{},
	})
	mr.uuid += 1
	return mr.uuid
}

func (mr *mockReceiver) ReceiveText(messageID cryptoMessage.ID,
	nickname, text string, pubKey, senderPubKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, status Status) uint64 {
	jww.INFO.Printf("ReceiveText: %s", messageID)
	mr.Msgs = append(mr.Msgs, mockMessage{
		Message:   text,
		PubKey:    pubKey,
		DMToken:   dmToken,
		MessageID: messageID,
		ReplyTo:   cryptoMessage.ID{},
	})
	mr.uuid += 1
	return mr.uuid
}

func (mr *mockReceiver) ReceiveReply(messageID cryptoMessage.ID,
	reactionTo cryptoMessage.ID, nickname, text string,
	pubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status Status) uint64 {
	jww.INFO.Printf("ReceiveReply: %s", messageID)
	mr.Msgs = append(mr.Msgs, mockMessage{
		Message:   text,
		PubKey:    pubKey,
		DMToken:   dmToken,
		MessageID: messageID,
		ReplyTo:   reactionTo,
	})
	mr.uuid += 1
	return mr.uuid
}

func (mr *mockReceiver) ReceiveReaction(messageID cryptoMessage.ID,
	reactionTo cryptoMessage.ID, nickname, reaction string,
	pubKey, senderPubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status Status) uint64 {
	jww.INFO.Printf("ReceiveReaction: %s", messageID)
	mr.Msgs = append(mr.Msgs, mockMessage{
		Message:   reaction,
		PubKey:    pubKey,
		DMToken:   dmToken,
		MessageID: messageID,
		ReplyTo:   reactionTo,
	})
	mr.uuid += 1
	return mr.uuid
}

func (mr *mockReceiver) UpdateSentStatus(uuid uint64, messageID cryptoMessage.ID,
	timestamp time.Time, round rounds.Round, status Status) {
	jww.INFO.Printf("UpdateSentStatus: %s", messageID)
}

func (mr *mockReceiver) GetConversation(pubKey ed25519.PublicKey) *ModelConversation {
	convo := ModelConversation{}
	convo.Pubkey = pubKey
	for i := range mr.blocked {
		if bytes.Equal(mr.blocked[i][:], pubKey[:]) {
			convo.BlockedTimestamp = &time.Time{}
			return &convo
		}
	}
	return &convo
}

func (mr *mockReceiver) GetConversations() []ModelConversation {
	convos := make([]ModelConversation, len(mr.blocked))
	for i := range mr.blocked {
		convos[i] = ModelConversation{
			Pubkey:           mr.blocked[i],
			BlockedTimestamp: &time.Time{},
		}
	}
	return convos
}

type mockMessage struct {
	Message   string
	PubKey    ed25519.PublicKey
	DMToken   uint32
	MessageID cryptoMessage.ID
	ReplyTo   cryptoMessage.ID
}
