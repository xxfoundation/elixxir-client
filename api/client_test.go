////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"crypto/sha256"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
	"time"
)

//FIXME: write mock registration and gateway structures so test can succeed
/*func TestRegistrationGob(t *testing.T) {
	// Put some user data into a gob
	err := globals.InitStorage(&globals.RamStorage{}, "")
	if err != nil {
		t.Error(err)
	}

	// populate a gob in the store
	grp := crypto.InitCrypto()
	_, err = Register(true, "UAV6IWD6", "", []string{gwAddress}, false, grp)
	if err != nil {
		t.Error(err)
	}

	// get the gob out of there again
	sessionGob := globals.LocalStorage.Load()
	var sessionBytes bytes.Buffer
	sessionBytes.Write(sessionGob)
	dec := gob.NewDecoder(&sessionBytes)
	Session = user.SessionObj{}
	err = dec.Decode(&Session)
	if err != nil {
		t.Error(err)
	}

	VerifyRegisterGobAddress(t)
	VerifyRegisterGobKeys(t)
	VerifyRegisterGobUser(t)
}*/

func VerifyRegisterGobAddress(t *testing.T) {

	if Session.GetGWAddress() != gwAddress {
		t.Errorf("GetGWAddress() returned %v, expected %v",
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
	grp := Session.GetGroup()
	h := sha256.New()
	h.Write([]byte(string(20005)))
	expectedTransmissionBaseKey := grp.NewIntFromBytes(h.Sum(nil))
	if Session.GetKeys()[0].TransmissionKey.Cmp(
		expectedTransmissionBaseKey) != 0 {
		t.Errorf("Transmission base key was %v, expected %v",
			Session.GetKeys()[0].TransmissionKey.Text(16),
			expectedTransmissionBaseKey.Text(16))
	}
	h = sha256.New()
	h.Write([]byte(string(40005)))
	expectedReceptionBaseKey := grp.NewIntFromBytes(h.Sum(nil))
	if Session.GetKeys()[0].ReceptionKey.Cmp(
		expectedReceptionBaseKey) != 0 {
		t.Errorf("Reception base key was %v, expected %v",
			Session.GetKeys()[0].ReceptionKey.Text(16),
			expectedReceptionBaseKey.Text(16))
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

	if !reflect.DeepEqual(sndr, []byte{}) {
		t.Errorf("Sender not empty from typed message")
	}
}

func TestParsedMessage_GetPayload(t *testing.T) {
	pm := ParsedMessage{}
	payload := []byte{0, 1, 2, 3}
	pm.Payload = payload
	pld := pm.GetPayload()

	if !reflect.DeepEqual(pld, payload) {
		t.Errorf("Output payload does not match input payload: %v %v", payload, pld)
	}
}

func TestParsedMessage_GetRecipient(t *testing.T) {
	pm := ParsedMessage{}
	rcpt := pm.GetRecipient()

	if !reflect.DeepEqual(rcpt, []byte{}) {
		t.Errorf("Recipient not empty from typed message")
	}
}

func TestParsedMessage_GetMessageType(t *testing.T) {
	pm := ParsedMessage{}
	var typeTest int32
	typeTest = 6
	pm.Typed = typeTest
	typ := pm.GetMessageType()

	if typ != typeTest {
		t.Errorf("Returned type does not match")
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
