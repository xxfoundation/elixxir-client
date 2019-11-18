////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	"encoding/json"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"sync"
	"time"
)

var def *ndf.NetworkDefinition

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

func (m APIMessage) GetTimestamp() time.Time {
	return time.Now()
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
	msgId, ipaddr string) (*pb.Slot, error) {
	return &pb.Slot{}, nil
}

// Return any MessageIDs in the globals for this User
func (m *TestInterface) CheckMessages(userId *id.User,
	messageID, ipaddr string) ([]string, error) {
	return make([]string, 0), nil
}

// PutMessage adds a message to the outgoing queue and
// calls SendBatch when it's size is the batch size
func (m *TestInterface) PutMessage(msg *pb.Slot, ipaddr string) error {
	m.LastReceivedMessage = *msg
	return nil
}

func (m *TestInterface) ConfirmNonce(message *pb.RequestRegistrationConfirmation, ipaddr string) (*pb.RegistrationConfirmation, error) {
	regConfirmation := &pb.RegistrationConfirmation{
		ClientSignedByServer: &pb.RSASignature{},
	}

	return regConfirmation, nil
}

// Blank struct implementing Registration Handler interface for testing purposes (Passing to StartServer)
type MockRegistration struct {
	//LastReceivedMessage pb.CmixMessage
}

func (s *MockRegistration) RegisterNode(ID []byte,
	NodeTLSCert, GatewayTLSCert, RegistrationCode, Addr, Addr2 string) error {
	return nil
}

func (s *MockRegistration) GetUpdatedNDF(clientNdfHash []byte) ([]byte, error) {

	ndfData := def

	ndfJson, _ := json.Marshal(ndfData)
	return ndfJson, nil
}

// Registers a user and returns a signed public key
func (s *MockRegistration) RegisterUser(registrationCode,
	key string) (hash []byte, err error) {
	return nil, nil
}

func (s *MockRegistration) GetCurrentClientVersion() (version string, err error) {
	return globals.SEMVER, nil
}

func getDHPubKey() *cyclic.Int {
	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString("9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48"+
			"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F"+
			"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5"+
			"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2"+
			"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41"+
			"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE"+
			"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15"+
			"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B", 16),
		large.NewIntFromString("5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613"+
			"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4"+
			"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472"+
			"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5"+
			"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA"+
			"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71"+
			"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0"+
			"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16))

	dh := cmixGrp.RandomCoprime(cmixGrp.NewMaxInt())
	return cmixGrp.ExpG(dh, cmixGrp.NewMaxInt())
}

// Pass-through for Registration Nonce Communication
func (m *TestInterface) RequestNonce(message *pb.NonceRequest, ipaddr string) (*pb.Nonce, error) {
	dh := getDHPubKey().Bytes()
	return &pb.Nonce{
		DHPubKey: dh,
	}, nil
}

// Mock dummy storage interface for testing.
type DummyStorage struct {
	LocationA string
	LocationB string
	StoreA    []byte
	StoreB    []byte
	mutex     sync.Mutex
}

func (d *DummyStorage) IsEmpty() bool {
	return d.StoreA == nil && d.StoreB == nil
}

func (d *DummyStorage) SetLocation(lA, lB string) error {
	d.LocationA = lA
	d.LocationB = lB
	return nil
}

func (d *DummyStorage) GetLocation() (string, string) {
	//return fmt.Sprintf("%s,%s", d.LocationA, d.LocationB)
	return d.LocationA, d.LocationB
}

func (d *DummyStorage) SaveA(b []byte) error {
	d.StoreA = make([]byte, len(b))
	copy(d.StoreA, b)
	return nil
}

func (d *DummyStorage) SaveB(b []byte) error {
	d.StoreB = make([]byte, len(b))
	copy(d.StoreB, b)
	return nil
}

func (d *DummyStorage) Lock() {
	d.mutex.Lock()
}

func (d *DummyStorage) Unlock() {
	d.mutex.Unlock()
}

func (d *DummyStorage) LoadA() []byte {
	return d.StoreA
}

func (d *DummyStorage) LoadB() []byte {
	return d.StoreB
}

type DummyReceiver struct {
	LastMessage APIMessage
}

func (d *DummyReceiver) Receive(message APIMessage) {
	d.LastMessage = message
}
