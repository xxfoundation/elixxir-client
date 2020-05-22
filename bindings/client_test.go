////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"bytes"
	"crypto"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

const NumNodes = 3
const NumGWs = NumNodes
const GWsStartPort = 7950
const RegPort = 5100
const ValidRegCode = "WTROXJ33"

var RegHandler = MockRegistration{}
var RegComms *registration.Comms

var GWComms [NumGWs]*gateway.Comms

var def *ndf.NetworkDefinition

type MockRegistration struct {
}

func (i *MockRegistration) RegisterUser(registrationCode, test string) (hash []byte, err error) {
	return nil, nil
}

func (i *MockRegistration) RegisterNode(ID *id.ID, ServerAddr, ServerTlsCert,
	GatewayAddr, GatewayTlsCert, RegistrationCode string) error {
	return nil
}

func (i *MockRegistration) GetCurrentClientVersion() (string, error) {
	return globals.SEMVER, nil
}

func (i *MockRegistration) PollNdf(clientNdfHash []byte,
	auth *connect.Auth) ([]byte,
	error) {
	ndfJson, _ := json.Marshal(def)
	return ndfJson, nil
}

func (i *MockRegistration) Poll(*pb.PermissioningPoll, *connect.Auth, string) (*pb.PermissionPollResponse, error) {
	return nil, nil
}

// Setups general testing params and calls test wrapper
func TestMain(m *testing.M) {
	os.Exit(testMainWrapper(m))
}

// Make sure NewClient returns an error when called incorrectly.
func TestNewClientNil(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	_, err := NewClient(nil, "", "", ndfStr, pubKey)
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, nil) input!")
	}

	_, err = NewClient(nil, "", "", "", "hello")
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, 'hello') input!")
	}
}

//Happy path: tests creation of valid client
func TestNewClient(t *testing.T) {
	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}

	ndfStr, pubKey := getNDFJSONStr(def, t)

	client, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("NewClient returned error: %v", err)
	} else if client == nil {
		t.Errorf("NewClient returned nil Client object")
	}
	for _, gw := range GWComms {
		gw.DisconnectAll()
	}
}

//Happy Path: Register with permissioning
func TestRegister(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = client.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = client.GenerateKeys("")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	regRes, err := client.RegisterWithPermissioning(true, ValidRegCode)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	for _, gw := range GWComms {
		gw.DisconnectAll()
	}
}

type DummyReceptionCallback struct{}

func (*DummyReceptionCallback) Callback(error) {
	return
}

//Error path: Changing username should panic before registration has happened
func TestClient_ChangeUsername_ErrorPath(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}

	testClient, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = testClient.ChangeUsername("josh420")
	if err == nil {
		t.Error("Expected error path, should not be able to change username before" +
			"regState PermissioningComplete")
	}
}

//Happy path: should have no errors when changing username
func TestClient_ChangeUsername(t *testing.T) {
	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}

	testClient, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = testClient.client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	regRes, err := testClient.RegisterWithPermissioning(false, ValidRegCode)
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}

	err = testClient.ChangeUsername("josh420")
	if err != nil {
		t.Errorf("Unexpected error, should have changed username: %v", err)
	}

}

func TestClient_StorageIsEmpty(t *testing.T) {
	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}

	testClient, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = testClient.client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	regRes, err := testClient.RegisterWithPermissioning(false, ValidRegCode)
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}

	if testClient.StorageIsEmpty() {
		t.Errorf("Unexpected empty storage!")
	}
}

//Error path: Have added no contacts, so deleting a contact should fail
func TestDeleteUsername_EmptyContactList(t *testing.T) {
	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}

	testClient, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = testClient.client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	regRes, err := testClient.RegisterWithPermissioning(false, ValidRegCode)
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	//Attempt to delete a contact from an empty contact list
	_, err = testClient.DeleteContact([]byte("typo"))
	if err != nil {
		return
	}
	t.Errorf("Expected error path, but did not get error on deleting a contact." +
		"Contact list should be empty")
}

//Happy path: Tests regState gets properly updated along the registration codepath
func TestClient_GetRegState(t *testing.T) {
	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	testClient, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = testClient.client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// Register with a valid registration code
	_, err = testClient.RegisterWithPermissioning(true, ValidRegCode)

	if err != nil {
		t.Errorf("Register with permissioning failed: %s", err.Error())
	}

	if testClient.GetRegState() != int64(user.PermissioningComplete) {
		t.Errorf("Unexpected reg state: Expected PermissioningComplete (%d), recieved: %d",
			user.PermissioningComplete, testClient.GetRegState())
	}

	err = testClient.RegisterWithNodes()
	if err != nil {
		t.Errorf("Register with nodes failed: %v", err.Error())
	}
}

//Happy path: send unencrypted message
func TestClient_Send(t *testing.T) {
	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	testClient, err := NewClient(&d, "hello", "", ndfStr, pubKey)

	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	err = testClient.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = testClient.client.GenerateKeys(nil, "password")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	// Register with a valid registration code
	userID, err := testClient.RegisterWithPermissioning(true, ValidRegCode)

	if err != nil {
		t.Errorf("Register with permissioning failed: %s", err.Error())
	}

	err = testClient.RegisterWithNodes()
	if err != nil {
		t.Errorf("Register with nodes failed: %v", err.Error())
	}

	// Login to gateway
	_, err = testClient.Login(userID, "password")

	if err != nil {
		t.Errorf("Login failed: %s", err.Error())
	}

	err = testClient.StartMessageReceiver(&DummyReceptionCallback{})

	if err != nil {
		t.Errorf("Could not start message reception: %+v", err)
	}

	receiverID := id.NewIdFromBytes(userID, t)
	receiverID.SetType(id.User)
	// Test send with invalid sender ID
	_, err = testClient.Send(
		mockMesssage{
			Sender:    id.NewIdFromUInt(12, id.User, t),
			TypedBody: parse.TypedBody{Body: []byte("test")},
			Receiver:  receiverID,
		}, false)

	if err != nil {
		// TODO: would be nice to catch the sender but we
		//  don't have the interface/mocking for that.
		t.Errorf("error on first message send: %+v", err)
	}

	// Test send with valid inputs
	_, err = testClient.Send(
		mockMesssage{
			Sender:    id.NewIdFromBytes(userID, t),
			TypedBody: parse.TypedBody{Body: []byte("test")},
			Receiver:  testClient.client.GetCurrentUser(),
		}, false)

	if err != nil {
		t.Errorf("Error sending message: %v", err)
	}

	err = testClient.Logout()

	if err != nil {
		t.Errorf("Logout failed: %v", err)
	}
	disconnectServers()
}

func TestLoginLogout(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	if err != nil {
		t.Errorf("Error starting client: %+v", err)
	}
	// InitNetwork to gateway
	err = client.InitNetwork()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = client.client.GenerateKeys(nil, "")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	regRes, err := client.RegisterWithPermissioning(true, ValidRegCode)
	loginRes, err2 := client.Login(regRes, "")
	if err2 != nil {
		t.Errorf("Login failed: %s", err2.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}

	err = client.StartMessageReceiver(&DummyReceptionCallback{})
	if err != nil {
		t.Errorf("Could not start message reciever: %+v", err)
	}
	time.Sleep(200 * time.Millisecond)
	err3 := client.Logout()
	if err3 != nil {
		t.Errorf("Logoutfailed: %s", err3.Error())
	}
	for _, gw := range GWComms {
		gw.DisconnectAll()
	}
}

type MockListener bool

func (m *MockListener) Hear(msg Message, isHeardElsewhere bool) {
	*m = true
}

// Proves that a message can be received by a listener added with the bindings
func TestListen(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	// InitNetwork to gateway
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = client.client.GenerateKeys(nil, "1234")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	regRes, _ := client.RegisterWithPermissioning(true, ValidRegCode)
	_, err = client.Login(regRes, "1234")

	if err != nil {
		t.Errorf("Could not log in: %+v", err)
	}

	listener := MockListener(false)
	client.Listen(id.ZeroUser[:], int32(cmixproto.Type_NO_TYPE), &listener)
	client.client.GetSwitchboard().Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: 0,
			Body:        []byte("stuff"),
		},
		Sender:   &id.ZeroUser,
		Receiver: client.client.GetCurrentUser(),
	})
	time.Sleep(time.Second)
	if !listener {
		t.Error("Message not received")
	}
	for _, gw := range GWComms {
		gw.DisconnectAll()
	}
}

func TestStopListening(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := DummyStorage{LocationA: "Blah", StoreA: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", "", ndfStr, pubKey)
	// InitNetwork to gateway
	err = client.InitNetwork()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	err = client.client.GenerateKeys(nil, "1234")
	if err != nil {
		t.Errorf("Could not generate Keys: %+v", err)
	}

	regRes, _ := client.RegisterWithPermissioning(true, ValidRegCode)

	_, err = client.Login(regRes, "1234")

	if err != nil {
		t.Errorf("Could not log in: %+v", err)
	}

	listener := MockListener(false)
	handle, err := client.Listen(id.ZeroUser[:], int32(cmixproto.Type_NO_TYPE), &listener)
	if err != nil {
		t.Fatal(err)
	}
	client.StopListening(handle)
	client.client.GetSwitchboard().Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: 0,
			Body:        []byte("stuff"),
		},
		Sender:   &id.ZeroUser,
		Receiver: &id.ZeroUser,
	})
	if listener {
		t.Error("Message was received after we stopped listening for it")
	}
}

type MockWriter struct {
	lastMessage []byte
}

func (mw *MockWriter) Write(msg []byte) (int, error) {
	mw.lastMessage = msg
	return len(msg), nil
}

func TestSetLogOutput(t *testing.T) {
	mw := &MockWriter{}
	SetLogOutput(mw)
	msg := "Test logging message"
	globals.Log.CRITICAL.Print(msg)
	if !bytes.Contains(mw.lastMessage, []byte(msg)) {
		t.Errorf("Mock writer didn't get the logging message")
	}
}

func TestParse(t *testing.T) {
	ms := parse.Message{}
	ms.Body = []byte{0, 1, 2}
	ms.MessageType = int32(cmixproto.Type_NO_TYPE)
	ms.Receiver = &id.ZeroUser
	ms.Sender = &id.ZeroUser

	messagePacked := ms.Pack()

	msOut, err := ParseMessage(messagePacked)

	if err != nil {
		t.Errorf("Message failed to parse: %s", err.Error())
	}

	if msOut.GetMessageType() != int32(ms.MessageType) {
		t.Errorf("Types do not match after message parse: %v vs %v", msOut.GetMessageType(), ms.MessageType)
	}

	if !reflect.DeepEqual(ms.Body, msOut.GetPayload()) {
		t.Errorf("Bodies do not match after message parse: %v vs %v", msOut.GetPayload(), ms.Body)
	}

}

func getNDFJSONStr(def *ndf.NetworkDefinition, t *testing.T) (string, string) {
	ndfBytes, err := json.Marshal(def)

	if err != nil {
		t.Errorf("Could not JSON the NDF: %+v", err)
	}

	// Load tls private key
	privKey, err := rsa.LoadPrivateKeyFromPem([]byte("-----BEGIN PRIVATE KEY-----\n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\nOaRA5ZbdE7g7vxKRV36jT3wvD7W+\n-----END PRIVATE KEY-----\n"))
	if err != nil || privKey == nil {
		t.Error("Failed to load privKey\n")
	}

	// Sign the NDF
	rsaHash := crypto.SHA256.New()
	rsaHash.Write(ndfBytes)
	signature, _ := rsa.Sign(
		crand.Reader, privKey, crypto.SHA256, rsaHash.Sum(nil), nil)

	// Compose network definition string
	ndfStr := string(ndfBytes) + "\n" + base64.StdEncoding.EncodeToString(signature) + "\n"

	return ndfStr, "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n"
}

// Handles initialization of mock registration server,
// gateways used for registration and gateway used for session
func testMainWrapper(m *testing.M) int {

	def = getNDF()

	// Initialize permissioning server
	// TODO(nan) We shouldn't need to start registration servers twice, right?
	pAddr := def.Registration.Address
	regId := new(id.ID)
	copy(regId[:], "testRegServer")
	regId.SetType(id.Generic)
	RegComms = registration.StartRegistrationServer(regId, pAddr,
		&RegHandler, nil, nil)

	// Start mock gateways used by registration and defer their shutdown (may not be needed)
	//the ports used are colliding between tests in GoLand when running full suite, this is a dumb fix
	bump := rand.Intn(10) * 10
	for i := 0; i < NumGWs; i++ {
		gwId := new(id.ID)
		copy(gwId[:], "testGateway")
		gwId.SetType(id.Gateway)

		gw := ndf.Gateway{
			ID:      gwId.Marshal(),
			Address: fmtAddress(GWsStartPort + i + bump),
		}

		def.Gateways = append(def.Gateways, gw)
		GWComms[i] = gateway.StartGateway(gwId, gw.Address,
			gateway.NewImplementation(), nil, nil)
	}

	// Start mock registration server and defer its shutdown
	def.Registration = ndf.Registration{
		Address: fmtAddress(RegPort),
	}
	RegComms = registration.StartRegistrationServer(regId, def.Registration.Address,
		&RegHandler, nil, nil)

	for i := 0; i < NumNodes; i++ {
		nId := new(id.ID)
		nId[0] = byte(i)
		nId.SetType(id.Node)
		n := ndf.Node{
			ID: nId[:],
		}
		def.Nodes = append(def.Nodes, n)
	}

	defer testWrapperShutdown()
	return m.Run()
}

func testWrapperShutdown() {
	for _, gw := range GWComms {
		gw.Shutdown()
	}
	RegComms.Shutdown()

}

func fmtAddress(port int) string { return fmt.Sprintf("localhost:%d", port) }

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B" +
				"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE" +
				"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F" +
				"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041" +
				"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45" +
				"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209" +
				"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29" +
				"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E" +
				"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2" +
				"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696" +
				"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E" +
				"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873" +
				"847AEF49F66E43873",
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
				"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
				"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
				"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
				"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
				"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
				"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
				"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
				"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
				"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
				"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
				"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
				"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
				"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
				"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
		Registration: ndf.Registration{
			Address:        fmt.Sprintf("0.0.0.0:%d", 5000+rand.Intn(1000)),
			TlsCertificate: "",
		},
	}
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

func (d *DummyStorage) GetLocation() string {
	return fmt.Sprintf("%s,%s", d.LocationA, d.LocationB)
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

type mockMesssage struct {
	parse.TypedBody
	// The crypto type is inferred from the message's contents
	InferredType parse.CryptoType
	Sender       *id.ID
	Receiver     *id.ID
	Nonce        []byte
	Timestamp    time.Time
}

// Returns the message's sender ID
func (m mockMesssage) GetSender() []byte {
	return m.Sender.Bytes()
}

// Returns the message payload
// Parse this with protobuf/whatever according to the type of the message
func (m mockMesssage) GetPayload() []byte {
	return m.TypedBody.Body
}

// Returns the message's recipient ID
func (m mockMesssage) GetRecipient() []byte {
	return m.Receiver.Bytes()
}

// Returns the message's type
func (m mockMesssage) GetMessageType() int32 {
	return m.TypedBody.MessageType
}

// Returns the message's timestamp in seconds since unix epoc
func (m mockMesssage) GetTimestamp() int64 {
	return m.Timestamp.Unix()
}

// Returns the message's timestamp in ns since unix epoc
func (m mockMesssage) GetTimestampNano() int64 {
	return m.Timestamp.UnixNano()
}

func disconnectServers() {
	for _, gw := range GWComms {
		gw.DisconnectAll()

	}
	RegComms.DisconnectAll()
}
