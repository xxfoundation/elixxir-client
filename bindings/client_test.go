////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package bindings

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/format"
	"os"
	"testing"
	"time"
	"gitlab.com/privategrity/crypto/id"
	"gitlab.com/privategrity/comms/gateway"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/client/cmixproto"
	"gitlab.com/privategrity/client/switchboard"
	"gitlab.com/privategrity/client/parse"
)

const gwAddress = "localhost:5557"

var gatewayData api.TestInterface

// NOTE: These need to be set up as io.Messaging is called during Init...
var ListenCh chan *format.Message
var lastmsg []byte

type dummyMessaging struct {
	listener chan *format.Message
}

// SendMessage to the server
func (d *dummyMessaging) SendMessage(recipientID *id.UserID,
	message []byte) error {
	jww.INFO.Printf("Sending: %s", message)
	lastmsg = message
	return nil
}

// Listen for messages from a given sender
func (d *dummyMessaging) Listen(senderID *id.UserID) chan *format.Message {
	return d.listener
}

// StopListening to a given switchboard (closes and deletes)
func (d *dummyMessaging) StopListening(listenerCh chan *format.Message) {}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(delay time.Duration) {}

func TestMain(m *testing.M) {
	io.SendAddress = gwAddress
	io.ReceiveAddress = gwAddress
	ListenCh = make(chan *format.Message, 100)
	io.Messaging = &dummyMessaging{
		listener: ListenCh,
	}

	gatewayData = api.TestInterface{
		LastReceivedMessage: pb.CmixMessage{},
	}

	os.Exit(m.Run())
}

// Make sure InitClient returns an error when called incorrectly.
func TestInitClientNil(t *testing.T) {
	err := InitClient(nil, "")
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, nil) input!")
	}
	globals.LocalStorage = nil

	err = InitClient(nil, "hello")
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, 'hello') input!")
	}
	globals.LocalStorage = nil
}

func TestInitClient(t *testing.T) {
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")
	if err != nil {
		t.Errorf("InitClient returned error: %v", err)
	}
	globals.LocalStorage = nil
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
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")

	regRes, err := Register(registrationCode, gwAddress, 1, false)
	if err != nil {
		t.Errorf("Registration failed: %s", err.Error())
	}
	if len(regRes) == 0 {
		t.Errorf("Invalid registration number received: %v", regRes)
	}
	globals.LocalStorage = nil
}

func TestRegisterBadNumNodes(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")

	_, err = Register(registrationCode, gwAddress, 0, false)
	if err == nil {
		t.Errorf("Registration worked with bad numnodes! %s", err.Error())
	}
	globals.LocalStorage = nil
}

func TestLoginLogout(t *testing.T) {
	gwShutDown := gateway.StartGateway(gwAddress, gateway.NewImplementation())
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")

	regRes, err := Register(registrationCode, gwAddress, 1, false)
	loginRes, err2 := Login(regRes, gwAddress)
	if err2 != nil {
		t.Errorf("Login failed: %s", err.Error())
	}
	if len(loginRes) == 0 {
		t.Errorf("Invalid login received: %v", loginRes)
	}
	time.Sleep(2000 * time.Millisecond)
	err3 := Logout()
	if err3 != nil {
		t.Errorf("Logoutfailed: %s", err.Error())
	}
	globals.LocalStorage = nil
}

func TestDisableBlockingTransmission(t *testing.T) {
	if !io.BlockTransmissions {
		t.Errorf("BlockingTransmission not intilized properly")
	}
	DisableBlockingTransmission()
	if io.BlockTransmissions {
		t.Errorf("BlockingTransmission not disabled properly")
	}
}

func TestSetRateLimiting(t *testing.T) {
	u, _ := user.Users.GetUser(id.NewUserIDFromUint(1, t))
	nk := make([]user.NodeKeys, 1)
	user.TheSession = user.NewSession(u, gwAddress, nk)
	if io.TransmitDelay != time.Duration(1000)*time.Millisecond {
		t.Errorf("SetRateLimiting not intilized properly")
	}
	SetRateLimiting(10)
	if io.TransmitDelay != time.Duration(10)*time.Millisecond {
		t.Errorf("SetRateLimiting not updated properly")
	}
}

type MockListener bool

func (m *MockListener) Hear(msg Message, isHeardElsewhere bool) {
	*m = true
}

// Proves that a message can be received by a listener added with the bindings
func TestListen(t *testing.T) {
	listener := MockListener(false)
	Listen(id.ZeroID[:], int32(cmixproto.Type_NO_TYPE), &listener)
	switchboard.Listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: 0,
			Body: []byte("stuff"),
		},
		Sender:   id.ZeroID,
		Receiver: id.ZeroID,
	})
	if !listener {
		t.Error("Message not received")
	}
}

func TestStopListening(t *testing.T) {
	listener := MockListener(false)
	handle := Listen(id.ZeroID[:], int32(cmixproto.Type_NO_TYPE), &listener)
	StopListening(handle)
	switchboard.Listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: 0,
			Body: []byte("stuff"),
		},
		Sender:   id.ZeroID,
		Receiver: id.ZeroID,
	})
	if listener {
		t.Error("Message was received after we stopped listening for it")
	}
}
