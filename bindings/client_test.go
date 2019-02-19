////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"bytes"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"os"
	"testing"
	"time"
	"reflect"

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
func (d *dummyMessaging) SendMessage(recipientID *id.User,
	message []byte) error {
	jww.INFO.Printf("Sending: %s", message)
	lastmsg = message
	return nil
}

// Listen for messages from a given sender
func (d *dummyMessaging) Listen(senderID *id.User) chan *format.Message {
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
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
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
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
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
	gwShutDown := gateway.StartGateway(gwAddress,
		gateway.NewImplementation(), "", "")
	time.Sleep(100 * time.Millisecond)
	defer gwShutDown()
	registrationCode := "UAV6IWD6"
	d := api.DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")

	regRes, err := Register(registrationCode, gwAddress, 1, false)
	loginRes, err2 := Login(regRes, gwAddress, "")
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
	u, _ := user.Users.GetUser(id.NewUserFromUint(1, t))
	nk := make([]user.NodeKeys, 1)
	user.TheSession = user.NewSession(u, gwAddress, nk, nil)
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


func TestParse(t *testing.T){
	ms := parse.Message{}
	ms.Body = []byte{0,1,2}
	ms.Type = cmixproto.Type_NO_TYPE
	ms.Receiver = id.ZeroID
	ms.Sender = id.ZeroID

	messagePacked := ms.Pack()

	msOut, err := ParseMessage(messagePacked)

	if err!=nil{
		t.Errorf("Message failed to parse: %s", err.Error())
	}

	if msOut.GetType()!=int32(ms.Type){
		t.Errorf("Types do not match after message parse: %v vs %v", msOut.GetType(), ms.Type)
	}

	if !reflect.DeepEqual(ms.Body,msOut.GetPayload()){
		t.Errorf("Bodies do not match after message parse: %v vs %v", msOut.GetPayload(), ms.Body)
	}

}
