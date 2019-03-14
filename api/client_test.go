////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"testing"
	"time"
	"reflect"
	"gitlab.com/elixxir/client/parse"
)

func TestRegistrationGob(t *testing.T) {
	// Put some user data into a gob
	globals.InitStorage(&globals.RamStorage{}, "")

	// populate a gob in the store
	Register("UAV6IWD6", gwAddress, 1, false)

	// get the gob out of there again
	sessionGob := globals.LocalStorage.Load()
	var sessionBytes bytes.Buffer
	sessionBytes.Write(sessionGob)
	dec := gob.NewDecoder(&sessionBytes)
	Session = user.SessionObj{}
	dec.Decode(&Session)

	VerifyRegisterGobAddress(t)
	VerifyRegisterGobKeys(t)
	VerifyRegisterGobUser(t)
}

func VerifyRegisterGobAddress(t *testing.T) {

	if Session.GetGWAddress() != gwAddress {
		t.Errorf("GetNodeAddress() returned %v, expected %v",
			Session.GetGWAddress(), gwAddress)
	}
}

func VerifyRegisterGobUser(t *testing.T) {
	if *Session.GetCurrentUser().User != *id.NewUserFromUint(5, t) {
		t.Errorf("User's ID was %q, expected %v",
			Session.GetCurrentUser().User, 5)
	}
}

func VerifyRegisterGobKeys(t *testing.T) {
	if Session.GetPublicKey().Cmp(cyclic.NewIntFromBytes([]byte(
		"this is not a real public key"))) != 0 {
		t.Errorf("Public key was %v, expected %v",
			string(Session.GetPublicKey().Bytes()),
			"this is not a real public key")
	}
	h := sha256.New()
	h.Write([]byte(string(30005)))
	expectedTransmissionRecursiveKey := cyclic.NewIntFromBytes(h.Sum(nil))
	if Session.GetKeys()[0].TransmissionKeys.Recursive.Cmp(
		expectedTransmissionRecursiveKey) != 0 {
		t.Errorf("Transmission recursive key was %v, expected %v",
			Session.GetKeys()[0].TransmissionKeys.Recursive.Text(16),
			expectedTransmissionRecursiveKey.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(20005)))
	expectedTransmissionBaseKey := cyclic.NewIntFromBytes(h.Sum(nil))
	if Session.GetKeys()[0].TransmissionKeys.Base.Cmp(
		expectedTransmissionBaseKey) != 0 {
		t.Errorf("Transmission base key was %v, expected %v",
			Session.GetKeys()[0].TransmissionKeys.Base.Text(16),
			expectedTransmissionBaseKey.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(50005)))
	expectedReceptionRecursiveKey := cyclic.NewIntFromBytes(h.Sum(nil))
	if Session.GetKeys()[0].ReceptionKeys.Recursive.Cmp(
		expectedReceptionRecursiveKey) != 0 {
		t.Errorf("Reception recursive key was %v, expected %v",
			Session.GetKeys()[0].ReceptionKeys.Recursive.Text(16),
			expectedReceptionRecursiveKey.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(40005)))
	expectedReceptionBaseKey := cyclic.NewIntFromBytes(h.Sum(nil))
	if Session.GetKeys()[0].ReceptionKeys.Base.Cmp(
		expectedReceptionBaseKey) != 0 {
		t.Errorf("Reception base key was %v, expected %v",
			Session.GetKeys()[0].ReceptionKeys.Base.Text(16),
			expectedReceptionBaseKey.Text(16))
	}

	if Session.GetKeys()[0].ReturnKeys.Recursive == nil {
		t.Logf("warning: return recursive key is nil")
	} else {
		t.Logf("return recursive key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if Session.GetKeys()[0].ReturnKeys.Base == nil {
		t.Logf("warning: return base key is nil")
	} else {
		t.Logf("return base key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if Session.GetKeys()[0].ReceiptKeys.Recursive == nil {
		t.Logf("warning: receipt recursive key is nil")
	} else {
		t.Logf("receipt recursive key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
	if Session.GetKeys()[0].ReceiptKeys.Base == nil {
		t.Logf("warning: receipt recursive key is nil")
	} else {
		t.Logf("receipt base key is not nil. " +
			"update gob test to ensure that it's serialized to storage, " +
			"if needed")
	}
}

// Make sure that a formatted text message can deserialize to the text
// message we would expect
func TestFormatTextMessage(t *testing.T) {
	msgText := "Hello"
	msg := FormatTextMessage(msgText)
	parsed := cmixproto.TextMessage{}
	err := proto.Unmarshal(msg, &parsed)
	// Make sure it parsed correctly
	if err != nil {
		t.Errorf("Got error parsing text message: %v", err.Error())
	}
	// Check the field that we explicitly set by calling the method
	if parsed.Message != msgText {
		t.Errorf("Got wrong text from parsing message. Got %v, expected %v",
			parsed.Message, msgText)
	}
	// Make sure that timestamp is reasonable
	timeDifference := time.Now().Unix() - parsed.Time
	if timeDifference > 2 || timeDifference < -2 {
		t.Errorf("Message timestamp was off by more than one second. "+
			"Original time: %x, parsed time: %x", time.Now().Unix(), parsed.Time)
	}
	t.Logf("message: %q", msg)
}

func TestParsedMessage_GetSender(t *testing.T) {
	pm := ParsedMessage{}
	sndr := pm.GetSender()

	if !reflect.DeepEqual(sndr,[]byte{}){
		t.Errorf("Sender not empty from typed message")
	}
}

func TestParsedMessage_GetPayload(t *testing.T) {
	pm := ParsedMessage{}
	payload := []byte{0,1,2,3}
	pm.Payload = payload
	pld := pm.GetPayload()

	if !reflect.DeepEqual(pld,payload){
		t.Errorf("Output payload does not match input payload: %v %v", payload, pld)
	}
}

func TestParsedMessage_GetRecipient(t *testing.T) {
	pm := ParsedMessage{}
	rcpt := pm.GetRecipient()

	if !reflect.DeepEqual(rcpt,[]byte{}){
		t.Errorf("Recipient not empty from typed message")
	}
}

func TestParsedMessage_GetType(t *testing.T) {
	pm := ParsedMessage{}
	var typeTest int32
	typeTest = 6
	pm.Typed = typeTest
	typ := pm.GetType()

	if typ!=typeTest{
		t.Errorf("Returned type does not match")
	}
}

func TestParse(t *testing.T){
	ms := parse.Message{}
	ms.Body = []byte{0,1,2}
	ms.Type = int32(cmixproto.Type_NO_TYPE)
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

// FIXME Reinstate tests for the UDB api
//var ListenCh chan *format.Message
//var lastmsg string

//type dummyMessaging struct {
//	listener chan *format.Message
//}

// SendMessage to the server
//func (d *dummyMessaging) SendMessage(recipientID id.User,
//	message string) error {
//	jww.INFO.Printf("Sending: %s", message)
//	lastmsg = message
//	return nil
//}

// Listen for messages from a given sender
//func (d *dummyMessaging) Listen(senderID id.User) chan *format.Message {
//	return d.listener
//}

// StopListening to a given switchboard (closes and deletes)
//func (d *dummyMessaging) StopListening(listenerCh chan *format.Message) {}

// MessageReceiver thread to get new messages
//func (d *dummyMessaging) MessageReceiver(delay time.Duration) {}

//var pubKeyBits []string
//var keyFingerprint string
//var pubKey []byte

// SendMsg puts a fake udb response message on the channel
//func SendMsg(msg string) {
//	m, _ := format.NewMessage(13, 1, msg)
//	ListenCh <- &m[0]
//}

//func TestRegisterPubKeyByteLen(t *testing.T) {
//	ListenCh = make(chan *format.Message, 100)
//	io.Messaging = &dummyMessaging{
//		switchboard: ListenCh,
//	}
//	pubKeyBits = []string{
//		"S8KXBczy0jins9uS4LgBPt0bkFl8t00MnZmExQ6GcOcu8O7DKgAsNz" +
//			"LU7a+gMTbIsS995IL/kuFF8wcBaQJBY23095PMSQ/nMuetzhk9HdXxrGIiKBo3C/n4SClp" +
//			"q4H+PoF9XziEVKua8JxGM2o83KiCK3tNUpaZbAAElkjueY4=",
//		"8Lg/eoeKGgPlleTYfO3JyGfnwBtLi73ti0h2dBQWW94JTqTQDr+z" +
//			"xVpLzdgTt+87TkAl0yXu9mOUXqGJ+51lTcRlIdIpWpfgUbibdRme8IThg0RNCF31ESKCts" +
//			"o8gJ8mSVljIXxrC+Uuoi+Gl1LNN5nPARykatx0Y70xNdJd2BQ=",
//	}
//	pubKey = make([]byte, 256)
//	for i := range pubKeyBits {
//		pubkeyBytes, _ := base64.StdEncoding.DecodeString(pubKeyBits[i])
//		for j := range pubkeyBytes {
//			pubKey[j+i*128] = pubkeyBytes[j]
//		}
//	}
//
//	keyFingerprint = "8oKh7TYG4KxQcBAymoXPBHSD/uga9pX3Mn/jKhvcD8M="
//	//SendMsg("SEARCH blah@privategrity.com NOTFOUND")
//	SendMsg(fmt.Sprintf("GETKEY %s NOTFOUND", keyFingerprint))
//	SendMsg("PUSHKEY ACK NEED 128")
//	SendMsg(fmt.Sprintf("PUSHKEY COMPLETE %s", keyFingerprint))
//	SendMsg("REGISTRATION COMPLETE")
//
//	err := bots.Register("EMAIL", "blah@privategrity.com", pubKey)
//
//	if err != nil {
//		t.Errorf("Unexpected error: %s", err.Error())
//	}
//	if len(lastmsg) != 81 {
//		t.Errorf("Message wrong length: %d v. expected 81", len(lastmsg))
//	}
//}
