////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This sets up a dummy/mock server instance for testing purposes
package api

import (
	"encoding/json"
	"fmt"
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

func (m *TestInterface) ConfirmNonce(message *pb.RequestRegistrationConfirmation) (*pb.RegistrationConfirmation, error) {
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
	fmt.Println("GetUpdated!")

	ndfData := buildMockNDF()
	globals.Log.INFO.Printf("retndf serialization in mock updateNDF: %v", ndfData.Serialize())
	ndfJson, _ := json.Marshal(ndfData)
	return ndfJson, nil
}

func buildMockNDF() ndf.NetworkDefinition {
	reg := ndf.Registration{Address: "localhost:5000", TlsCertificate: "CERT"}
	var Nodes []ndf.Node
	for i := 0; i < 3; i++ {
		nIdBytes := make([]byte, id.NodeIdLen)
		nIdBytes[0] = byte(i)
		n := ndf.Node{
			ID: nIdBytes,
		}
		Nodes = append(Nodes, n)
	}
	var grp ndf.Group
	grp.Prime = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"
	grp.Generator = "2"
	grp.SmallPrime = "2"
	retNDF := ndf.NetworkDefinition{Timestamp: time.Now(), Registration: reg, Nodes: Nodes, CMIX: grp, E2E: grp}
	return retNDF
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
			"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16),
		large.NewIntFromString("F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F", 16))

	dh := cmixGrp.RandomCoprime(cmixGrp.NewMaxInt())
	return cmixGrp.ExpG(dh, cmixGrp.NewMaxInt())
}

// Pass-through for Registration Nonce Communication
func (m *TestInterface) RequestNonce(message *pb.NonceRequest) (*pb.Nonce, error) {
	dh := getDHPubKey().Bytes()
	return &pb.Nonce{
		DHPubKey: dh,
	}, nil
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
