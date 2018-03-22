////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package bindings

import (
	"strings"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/format"
	"bytes"
	"gitlab.com/privategrity/crypto/cyclic"
	"testing"
)

// Mock dummy storage interface for testing.
type DummyStorage struct {
	Location string
	LastSave []byte
}

func (d *DummyStorage) SetLocation(l string) (error) {
	d.Location = l
	return nil
}

func (d *DummyStorage) GetLocation() string {
	return d.Location
}

func (d *DummyStorage) Save(b []byte) (error) {
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
	LastMessage Message
}

func (d *DummyReceiver) Receive(message Message) {
	d.LastMessage = message
}


// Make sure InitClient returns an error when called incorrectly.
func TestInitClientNil(t *testing.T) {
	err := InitClient(nil, "", nil)
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, nil) input!")
	}
	globals.LocalStorage = nil

	err = InitClient(nil, "hello", nil)
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, 'hello') input!")
	}
	globals.LocalStorage = nil
}

func TestInitClient(t *testing.T) {
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello", nil)
	if err != nil {
		t.Errorf("InitClient returned error: %v", err)
	}
	globals.LocalStorage = nil
}

func TestGetContactListJSON(t *testing.T) {
	// This call includes validating the JSON against the schema
	result, err := GetContactListJSON()

	if err != nil {
		t.Error(err.Error())
	}

	// But, just in case,
	// let's make sure that we got the error out of validateContactList anyway
	err = validateContactListJSON(result)

	if err != nil {
		t.Error(err.Error())
	}

	// Finally, make sure that all the names we expect are in the JSON
	expected := []string{"Ben", "Rick", "Jake", "Mario", "Allan", "David",
		"Jim", "Spencer", "Will", "Jono"}

	actual := string(result)

	for _, nick := range expected {
		if !strings.Contains(actual, nick) {
			t.Errorf("Error: Expected name %v wasn't in JSON %v", nick, actual)
		}
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

// This test creates a struct that implements the Receiver interface, then makes
// sure that that struct can receive a message when it's set as the global
// Receiver.
func TestReceiveMessageByInterface(t *testing.T) {
	// set up the receiver
	receiver := BytesReceiver{}
	err := InitClient(&globals.RamStorage{}, "", &receiver)
	if err != nil {
		t.Error(err.Error())
	}

	// set up the message
	payload := "hello there"
	senderID := cyclic.NewIntFromUInt(50).LeftpadBytes(format.SID_LEN)
	recipientID := cyclic.NewIntFromUInt(60).LeftpadBytes(format.RID_LEN)
	msg, err := format.NewMessage(cyclic.NewIntFromBytes(senderID).Uint64(),
		cyclic.NewIntFromBytes(recipientID).Uint64(), payload)
	if err != nil {
		t.Errorf("Couldn't create messages: %v", err.Error())
	}

	// receive the message
	globals.Receive(msg[0])

	// verify that the message was correctly received
	if !bytes.Equal(receiver.receptionBuffer, []byte(payload)) {
		t.Errorf("Message payload didn't match. Got: %v, expected %v",
			string(receiver.receptionBuffer), payload)
	}
	if !bytes.Equal(senderID, receiver.lastSID) {
		t.Errorf("Sender ID didn't match. Got: %v, expected %v",
			cyclic.NewIntFromBytes(receiver.lastSID).Uint64(),
			cyclic.NewIntFromBytes(senderID).Uint64())
	}
	if !bytes.Equal(recipientID, receiver.lastRID) {
		t.Errorf("Recipient ID didn't match. Got: %v, expected %v",
			cyclic.NewIntFromBytes(receiver.lastRID).Uint64(),
			cyclic.NewIntFromBytes(recipientID).Uint64())
	}
	globals.LocalStorage = nil
}
