////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/gateway"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

const NumNodes = 1
const NumGWs = NumNodes
const GWsStartPort = 10000

const ValidRegCode = "UAV6IWD6"

var GWComms [NumGWs]*gateway.GatewayComms

var def *ndf.NetworkDefinition

// Setups general testing params and calls test wrapper
func TestMain(m *testing.M) {

	os.Exit(testMainWrapper(m))
}

// Make sure NewClient returns an error when called incorrectly.
func TestNewClientNil(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	_, err := NewClient(nil, "", ndfStr, pubKey)
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, nil) input!")
	}

	_, err = NewClient(nil, "hello", "", "")
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, 'hello') input!")
	}
}

func TestNewClient(t *testing.T) {
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}

	ndfStr, pubKey := getNDFJSONStr(def, t)

	client, err := NewClient(&d, "hello", ndfStr, pubKey)
	if err != nil {
		t.Errorf("NewClient returned error: %v", err)
	} else if client == nil {
		t.Errorf("NewClient returned nil Client object")
	}
}

func TestRegister(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey)

	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, err := client.Register(true, ValidRegCode,
		"", "", "")
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
}

func TestConnectBadNumNodes(t *testing.T) {

	newDef := *def

	newDef.Gateways = make([]ndf.Gateway, 0)

	ndfStr, pubKey := getNDFJSONStr(&newDef, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey)

	// Connect to empty gw
	err = client.Connect()

	if err == nil {
		t.Errorf("Connect should have returned an error when no gateway is passed")
	}
}

func TestLoginLogout(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey)

	// Connect to gateway
	err = client.Connect()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, err := client.Register(true, ValidRegCode,
		"", "", "")
	loginRes, err2 := client.Login(regRes, "")
	if err2 != nil {
		t.Errorf("Login failed: %s", err2.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}
	time.Sleep(2000 * time.Millisecond)
	err3 := client.Logout()
	if err3 != nil {
		t.Errorf("Logoutfailed: %s", err3.Error())
	}
}

type MockListener bool

func (m *MockListener) Hear(msg Message, isHeardElsewhere bool) {
	*m = true
}

// Proves that a message can be received by a listener added with the bindings
func TestListen(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey)

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, _ := client.Register(true, ValidRegCode,
		"", "", "")
	_, err = client.Login(regRes, "")

	if err != nil {
		t.Errorf("Could not log in: %+v", err)
	}

	listener := MockListener(false)
	client.Listen(id.ZeroID[:], int32(cmixproto.Type_NO_TYPE), &listener)
	client.client.GetSwitchboard().Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: 0,
			Body:        []byte("stuff"),
		},
		Sender:   id.ZeroID,
		Receiver: client.client.GetCurrentUser(),
	})
	if !listener {
		t.Error("Message not received")
	}
}

func TestStopListening(t *testing.T) {

	ndfStr, pubKey := getNDFJSONStr(def, t)

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", ndfStr, pubKey)

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, _ := client.Register(true, ValidRegCode,
		"", "", "")

	_, err = client.Login(regRes, "")

	if err != nil {
		t.Errorf("Could not log in: %+v", err)
	}

	listener := MockListener(false)
	handle := client.Listen(id.ZeroID[:], int32(cmixproto.Type_NO_TYPE), &listener)
	client.StopListening(handle)
	client.client.GetSwitchboard().Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: 0,
			Body:        []byte("stuff"),
		},
		Sender:   id.ZeroID,
		Receiver: id.ZeroID,
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
	ms.Receiver = id.ZeroID
	ms.Sender = id.ZeroID

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

func getNDFJSONStr(netDef *ndf.NetworkDefinition, t *testing.T) (string, string) {
	ndfBytes, err := json.Marshal(netDef)

	if err != nil {
		t.Errorf("Could not JSON the NDF: %+v", err)
	}

	privateKey, _ := rsa.GenerateKey(crand.Reader, 768)
	publicKey := &rsa.PublicKey{PublicKey: privateKey.PublicKey}
	publicKeyPem := string(rsa.CreatePublicKeyPem(publicKey))

	// Sign the NDF
	opts := rsa.NewDefaultOptions()
	rsaHash := opts.Hash.New()
	rsaHash.Write(netDef.Serialize())
	signature, _ := rsa.Sign(
		crand.Reader, privateKey, opts.Hash, rsaHash.Sum(nil), nil)

	// Compose network definition string
	ndfStr := string(ndfBytes) + "\n" + base64.StdEncoding.EncodeToString(signature)

	return ndfStr, publicKeyPem
}

// Handles initialization of mock registration server,
// gateways used for registration and gateway used for session
func testMainWrapper(m *testing.M) int {

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	rndPort := int(rng.Uint64() % 10000)

	def = getNDF()

	// Start mock gateways used by registration and defer their shutdown (may not be needed)
	for i := 0; i < NumGWs; i++ {

		gw := ndf.Gateway{
			Address: fmtAddress(GWsStartPort + i + rndPort),
		}

		def.Gateways = append(def.Gateways, gw)
		GWComms[i] = gateway.StartGateway(gw.Address,
			gateway.NewImplementation(), nil, nil)
	}

	for i := 0; i < NumNodes; i++ {
		nIdBytes := make([]byte, id.NodeIdLen)
		nIdBytes[0] = byte(i)
		n := ndf.Node{
			ID: nIdBytes,
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
			SmallPrime: "2",
			Generator:  "2",
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
			SmallPrime: "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
				"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
				"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
				"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
				"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
				"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
				"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
				"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
	}
}
