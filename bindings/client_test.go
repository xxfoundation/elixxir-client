////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"bytes"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/gateway"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"os"
	"reflect"
	"testing"
	"time"
)

const gwAddress = "localhost:5557"

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// Make sure NewClient returns an error when called incorrectly.
func TestNewClientNil(t *testing.T) {
	_, err := NewClient(nil, "")
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, nil) input!")
	}

	_, err = NewClient(nil, "hello")
	if err == nil {
		t.Errorf("NewClient returned nil on invalid (nil, 'hello') input!")
	}
}

func TestNewClient(t *testing.T) {
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello")
	if err != nil {
		t.Errorf("NewClient returned error: %v", err)
	} else if client == nil {
		t.Errorf("NewClient returned nil Client object")
	}
}

// BytesReceiver receives the last message and puts the data it received into
// byte slices
type BytesReceiver struct {
	receptionBuffer []byte
	lastSID         []byte
	lastRID         []byte
}

// This is the method that globals.Receive calls when you set a BytesReceiver
// as the global receiver
func (br *BytesReceiver) Receive(message Message) {
	br.receptionBuffer = append(br.receptionBuffer, message.GetPayload()...)
	br.lastRID = message.GetRecipient()
	br.lastSID = message.GetSender()
}

func TestRegister(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello")
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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
	p := large.NewInt(1)
	p.SetString(primeString, 16)
	g := large.NewInt(2)
	q := large.NewInt(3)
	grp := cyclic.NewGroup(p, g, q)
	grpJSON, err := grp.MarshalJSON()
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	regRes, err := client.Register(registrationCode, gwAddress, 1, false, string(grpJSON))
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
}

func TestRegisterBadNumNodes(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello")
	p := large.NewInt(int64(1))
	g := large.NewInt(int64(2))
	q := large.NewInt(int64(3))
	grp := cyclic.NewGroup(p, g, q)
	grpJSON, err := grp.MarshalJSON()
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	_, err = client.Register(registrationCode, gwAddress, 0, false, string(grpJSON))
	if err == nil {
		t.Errorf("Registration worked with bad numnodes! %s", err.Error())
	}
}

func TestLoginLogout(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello")
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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
	p := large.NewInt(1)
	p.SetString(primeString, 16)
	g := large.NewInt(2)
	q := large.NewInt(3)
	grp := cyclic.NewGroup(p, g, q)
	grpJSON, err := grp.MarshalJSON()
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	regRes, err := client.Register(registrationCode, gwAddress, 1, false, string(grpJSON))
	loginRes, err2 := client.Login(regRes, gwAddress, "")
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
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello")
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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
	p := large.NewInt(1)
	p.SetString(primeString, 16)
	g := large.NewInt(2)
	q := large.NewInt(3)
	grp := cyclic.NewGroup(p, g, q)
	grpJSON, err := grp.MarshalJSON()
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	regRes, _ := client.Register(registrationCode, gwAddress, 1, false, string(grpJSON))
	client.Login(regRes, gwAddress, "")
	listener := MockListener(false)
	client.Listen(id.ZeroID[:], int32(cmixproto.Type_NO_TYPE), &listener)
	client.client.GetSwitchboard().Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			MessageType: 0,
			Body:        []byte("stuff"),
		},
		Sender:   id.ZeroID,
		Receiver: id.ZeroID,
	})
	if !listener {
		t.Error("Message not received")
	}
}

func TestStopListening(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	client, err := NewClient(&d, "hello")
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
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
	p := large.NewInt(1)
	p.SetString(primeString, 16)
	g := large.NewInt(2)
	q := large.NewInt(3)
	grp := cyclic.NewGroup(p, g, q)
	grpJSON, err := grp.MarshalJSON()
	if err != nil {
		t.Errorf("Failed to marshal group JSON: %s", err)
	}

	regRes, _ := client.Register(registrationCode, gwAddress, 1, false, string(grpJSON))
	client.Login(regRes, gwAddress, "")
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
