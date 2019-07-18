////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/gateway"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

const NumGWs = 1
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
	_, err := NewClient(nil, "", "", "")
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
	client, err := NewClient(&d, "hello", "", "")
	if err != nil {
		t.Errorf("NewClient returned error: %v", err)
	} else if client == nil {
		t.Errorf("NewClient returned nil Client object")
	}
}

func TestRegister(t *testing.T) {

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", string(def.Serialize()), "")

	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, err := client.Register(true, ValidRegCode,
		"", "")
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
}

func TestConnectBadNumNodes(t *testing.T) {
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", string(def.Serialize()), "")

	// Connect to empty gw
	err = client.Connect()

	if err == nil {
		t.Errorf("Connect should have returned an error when no gateway is passed")
	}
}

func TestLoginLogout(t *testing.T) {

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", string(def.Serialize()), "")

	// Connect to gateway
	err = client.Connect()
	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, err := client.Register(true, ValidRegCode,
		"", "")
	loginRes, err2 := client.Login(regRes)
	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}
	time.Sleep(2000 * time.Millisecond)
	err3 := client.Logout()
	if err3 != nil {
		t.Errorf("Logoutfailed: %s", err.Error())
	}
}

type MockListener bool

func (m *MockListener) Hear(msg Message, isHeardElsewhere bool) {
	*m = true
}

// Proves that a message can be received by a listener added with the bindings
func TestListen(t *testing.T) {

	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", string(def.Serialize()), "")

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, _ := client.Register(true, ValidRegCode,
		"", "")
	_, err = client.Login(regRes)

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
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello", string(def.Serialize()), "")

	// Connect to gateway
	err = client.Connect()

	if err != nil {
		t.Errorf("Could not connect: %+v", err)
	}

	regRes, _ := client.Register(true, ValidRegCode,
		"", "")

	_, err = client.Login(regRes)

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
			gateway.NewImplementation(), "", "")
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
			Prime: "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
				"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
				"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
				"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
				"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
				"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
				"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
				"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
				"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
				"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
				"15728E5A8AACAA68FFFFFFFFFFFFFFFF",
			SmallPrime: "2",
			Generator:  "2",
		},
	}
}
