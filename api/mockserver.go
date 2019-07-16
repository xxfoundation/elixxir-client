////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/parse"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

// APIMessage are an implementation of the interface in bindings and API
// easy to use from Go
type APIMessage struct {
	Payload     []byte
	SenderID    *id.User
	RecipientID *id.User
}

func (m APIMessage) GetSender() *id.User {
	return m.SenderID
}

func (m APIMessage) GetRecipient() *id.User {
	return m.RecipientID
}

func (m APIMessage) GetPayload() []byte {
	return m.Payload
}

func (m APIMessage) GetMessageType() int32 {
	return int32(cmixproto.Type_NO_TYPE)
}

func (m APIMessage) GetCryptoType() parse.CryptoType {
	return parse.None
}

func (m APIMessage) Pack() []byte {
	// assuming that the type is independently populated.
	// that's probably a bad idea
	// there's no good reason to have the same method body for each of these
	// two methods!
	return m.Payload
}

// Blank struct implementing ServerHandler interface for testing purposes (Passing to StartServer)
type TestInterface struct {
	LastReceivedMessage pb.Slot
}

// Returns message contents for MessageID, or a null/randomized message
// if that ID does not exist of the same size as a regular message
func (m *TestInterface) GetMessage(userId *id.User,
	msgId string) (*pb.Slot, bool) {
	return &pb.Slot{}, true
}

// Return any MessageIDs in the globals for this User
func (m *TestInterface) CheckMessages(userId *id.User,
	messageID string) ([]string, bool) {
	return make([]string, 0), true
}

// PutMessage adds a message to the outgoing queue and
// calls SendBatch when it's size is the batch size
func (m *TestInterface) PutMessage(msg *pb.Slot) bool {
	m.LastReceivedMessage = *msg
	return true
}

func (m *TestInterface) ConfirmNonce(message *pb.DSASignature) (*pb.RegistrationConfirmation, error) {
	regConfirmation := &pb.RegistrationConfirmation{
		Server: &pb.DSAPublicKey{},
	}

	regConfirmation.Server.P = large.NewInt(1).Bytes()
	regConfirmation.Server.Q = large.NewInt(1).Bytes()
	regConfirmation.Server.G = large.NewInt(1).Bytes()
	regConfirmation.Server.Y = large.NewInt(1).Bytes()

	return regConfirmation, nil
}

// Blank struct implementing Registration Handler interface for testing purposes (Passing to StartServer)
type MockRegistration struct {
	//LastReceivedMessage pb.CmixMessage
}

func (s *MockRegistration) RegisterNode(ID []byte,
	NodeTLSCert, GatewayTLSCert, RegistrationCode, Addr string) error {
	return nil
}

// Registers a user and returns a signed public key
func (s *MockRegistration) RegisterUser(registrationCode string,
	Y, P, Q, G []byte) (hash, R, S []byte, err error) {
	return nil, nil, nil, nil
}

// Pass-through for Registration Nonce Communication
func (m *TestInterface) RequestNonce(message *pb.NonceRequest) (*pb.Nonce, error) {
	return &pb.Nonce{}, nil
}

// Mock dummy storage interface for testing.
type DummyStorage struct {
	Location string
	LastSave []byte
	mutex    sync.Mutex
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

func (d *DummyStorage) Lock() {
	d.mutex.Lock()
}

func (d *DummyStorage) Unlock() {
	d.mutex.Unlock()
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
