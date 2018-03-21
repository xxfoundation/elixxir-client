package bindings

import (
	"testing"
	"strings"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/format"
	"bytes"
	"gitlab.com/privategrity/crypto/cyclic"
)

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

type BytesReceiver struct {
	receptionBuffer []byte
	lastSID         []byte
	lastRID         []byte
	receiveMethod   func(messageInterface format.MessageInterface)
}

func (br *BytesReceiver) Receive(message Message) {
	br.receptionBuffer = append(br.receptionBuffer, message.GetPayload()...)
	br.lastRID = message.GetRecipient()
	br.lastSID = message.GetSender()
}

func TestReceiveMessageByInterface(t *testing.T) {
	receiver := BytesReceiver{}
	SetReceiver(&receiver)
	payload := "hello there"
	senderID := cyclic.NewIntFromUInt(50).LeftpadBytes(format.SID_LEN)
	recipientID := cyclic.NewIntFromUInt(60).LeftpadBytes(format.RID_LEN)
	msg, err := format.NewMessage(cyclic.NewIntFromBytes(senderID).Uint64(),
		cyclic.NewIntFromBytes(recipientID).Uint64(), payload)
	if err != nil {
		t.Errorf("Couldn't create messages: %v", err.Error())
	}
	globals.Receive(msg[0])
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
}
