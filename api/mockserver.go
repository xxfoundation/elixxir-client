////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
<<<<<<< HEAD
=======

// This sets up a dummy/mock server instance for testing purposes
>>>>>>> Make unit tests work
package api

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"time"
)

// Blank struct implementing ServerHandler interface for testing purposes (Passing to StartServer)
type TestInterface struct {
	LastReceivedMessage pb.CmixMessage
}

func (m *TestInterface) NewRound(roundId string) {}

func (m *TestInterface) SetPublicKey(roundId string, pkey []byte) {}

func (m *TestInterface) PrecompDecrypt(message *pb.PrecompDecryptMessage) {}

func (m *TestInterface) PrecompEncrypt(message *pb.PrecompEncryptMessage) {}

func (m *TestInterface) PrecompReveal(message *pb.PrecompRevealMessage) {}

func (m *TestInterface) PrecompPermute(message *pb.PrecompPermuteMessage) {}

func (m *TestInterface) PrecompShare(message *pb.PrecompShareMessage) {}

func (m *TestInterface) PrecompShareInit(message *pb.PrecompShareInitMessage) {}

func (m *TestInterface) PrecompShareCompare(message *pb.
	PrecompShareCompareMessage) {
}

func (m *TestInterface) PrecompShareConfirm(message *pb.
	PrecompShareConfirmMessage) {
}

func (m *TestInterface) RealtimeDecrypt(message *pb.RealtimeDecryptMessage) {}

func (m *TestInterface) RealtimeEncrypt(message *pb.RealtimeEncryptMessage) {}

func (m *TestInterface) RealtimePermute(message *pb.RealtimePermuteMessage) {}

func (m *TestInterface) ClientPoll(message *pb.ClientPollMessage) *pb.CmixMessage {
	jww.ERROR.Printf("Sending Msg from: %d, %d, %d", m.LastReceivedMessage.SenderID,
		len(m.LastReceivedMessage.MessagePayload),
		len(m.LastReceivedMessage.RecipientID))
	return &m.LastReceivedMessage
}

func (m *TestInterface) RequestContactList(message *pb.ContactPoll) *pb.
	ContactMessage {
	return &pb.ContactMessage{
		Contacts: []*pb.Contact{
			{
				UserID: 3,
				Nick:   "Snicklefritz",
			}, {
				UserID: 5786,
				Nick:   "Jonwayne",
			},
		},
	}
}

func (m *TestInterface) UserUpsert(message *pb.UpsertUserMessage) {}

func (m *TestInterface) SetNick(message *pb.Contact) {
	nick = message.Nick
}

func (m *TestInterface) ReceiveMessageFromClient(message *pb.CmixMessage) {
	jww.ERROR.Printf("Received Msg from: %d, %d, %d", message.SenderID,
		len(message.MessagePayload), len(message.RecipientID))
	m.LastReceivedMessage = *message
}
func (m *TestInterface) StartRound(message *pb.InputMessages) {}

func (m *TestInterface) RoundtripPing(message *pb.TimePing) {}

func (m *TestInterface) ServerMetrics(message *pb.ServerMetricsMessage) {}

func (m *TestInterface) PollRegistrationStatus(message *pb.
	RegistrationPoll) *pb.RegistrationConfirmation {
	return &pb.RegistrationConfirmation{}
}

// Mock dummy storage interface for testing.
type DummyStorage struct {
	Location string
	LastSave []byte
}

func (d *DummyStorage) SetLocation(l string) error {
	d.Location = l
	return nil
}

func (d *DummyStorage) GetLocation() string {
	return d.Location
}

func (d *DummyStorage) Save(b []byte) error {
	d.LastSave = make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		d.LastSave[i] = b[i]
	}
	return nil
}

func (d *DummyStorage) Load() []byte {
	return d.LastSave
}

type DummyReceiver struct {
	LastMessage APIMessage
}

func (d *DummyReceiver) Receive(message APIMessage) {
	d.LastMessage = message
}
