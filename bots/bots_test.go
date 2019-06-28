////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package bot functions for working with the user discovery bot (UDB)
package bots

import (
	"encoding/base64"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"os"
	"testing"
	"time"
)

var ListenCh chan *format.Message

type dummyMessaging struct {
	listener chan *format.Message
}

// SendMessage to the server
func (d *dummyMessaging) SendMessage(sess user.Session,
	recipientID *id.User,
	cryptoType parse.CryptoType,
	message []byte) error {
	jww.INFO.Printf("Sending: %s", string(message))
	return nil
}

// SendMessage without partitions to the server
func (d *dummyMessaging) SendMessageNoPartition(sess user.Session,
	recipientID *id.User,
	cryptoType parse.CryptoType,
	message []byte) error {
	jww.INFO.Printf("Sending: %s", string(message))
	return nil
}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(session user.Session,
	delay time.Duration) {
}

var pubKeyBits string
var keyFingerprint string
var pubKey []byte

func TestMain(m *testing.M) {
	u := &user.User{
		User: id.NewUserFromUints(&[4]uint64{0, 0, 0, 18}),
		Nick: "Bernie",
	}
	fakeSession := user.NewSession(&globals.RamStorage{},
		u, nil, nil, nil, nil)
	fakeComm := &dummyMessaging{
		listener: ListenCh,
	}
	InitBots(fakeSession, fakeComm)

	// Make the reception channels buffered for this test
	// which overwrites the channels registered in InitBots
	pushKeyResponseListener = make(channelResponseListener, 100)
	getKeyResponseListener = make(channelResponseListener, 100)
	registerResponseListener = make(channelResponseListener, 100)
	searchResponseListener = make(channelResponseListener, 100)

	pubKeyBits = "S8KXBczy0jins9uS4LgBPt0bkFl8t00MnZmExQ6GcOcu8O7DKgAsNzLU7a+gMTbIsS995IL/kuFF8wcBaQJBY23095PMSQ/nMuetzhk9HdXxrGIiKBo3C/n4SClpq4H+PoF9XziEVKua8JxGM2o83KiCK3tNUpaZbAAElkjueY7wuD96h4oaA+WV5Nh87cnIZ+fAG0uLve2LSHZ0FBZb3glOpNAOv7PFWkvN2BO37ztOQCXTJe72Y5ReoYn7nWVNxGUh0ilal+BRuJt1GZ7whOGDRE0IXfURIoK2yjyAnyZJWWMhfGsL5S6iL4aXUs03mc8BHKRq3HRjvTE10l3YFA=="
	pubKey, _ = base64.StdEncoding.DecodeString(pubKeyBits)

	keyFingerprint = "8oKh7TYG4KxQcBAymoXPBHSD/uga9pX3Mn/jKhvcD8M="
	os.Exit(m.Run())
}

// TestRegister smoke tests the registration functionality.
func TestRegister(t *testing.T) {

	// Send response messages from fake UDB in advance
	getKeyResponseListener <- fmt.Sprintf("GETKEY %s NOTFOUND", keyFingerprint)
	pushKeyResponseListener <- fmt.Sprintf("PUSHKEY COMPLETE %s", keyFingerprint)
	registerResponseListener <- "REGISTRATION COMPLETE"

	err := Register("EMAIL", "rick@elixxir.io", pubKey)
	if err != nil {
		t.Errorf("Registration failure: %s", err.Error())
	}
}

// TestSearch smoke tests the search function
func TestSearch(t *testing.T) {

	// Send response messages from fake UDB in advance
	searchResponseListener <- fmt.Sprintf("SEARCH %s FOUND %s %s",
		"blah@elixxir.io",
		base64.StdEncoding.EncodeToString(id.NewUserFromUint(26, t)[:]),
		keyFingerprint)
	getKeyResponseListener <- fmt.Sprintf("GETKEY %s %s", keyFingerprint,
		pubKeyBits)

	searchedUser, _, err := Search("EMAIL", "blah@elixxir.io")
	if err != nil {
		t.Errorf("Error on Search: %s", err.Error())
	}
	if *searchedUser != *id.NewUserFromUint(26, t) {
		t.Errorf("Search did not return user ID 26!")
	}
}

// Test NICKNAME_REQUEST and NICKNAME_RESPONSE
// messages using switchboard
// Test LookupNick function
func TestNicknameFunctions(t *testing.T) {
	// Test receiving a nickname request
	msg := &parse.Message{
		Sender: session.GetCurrentUser().User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_NICKNAME_REQUEST),
			Body:        []byte{},
		},
		InferredType: parse.Unencrypted,
		Receiver:     session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)

	// Test nickname lookup

	// Spawn lookup on goroutine
	go func() {
		nick, err := LookupNick(session.GetCurrentUser().User)
		if err != nil {
			t.Errorf("Error on LookupNick: %s", err.Error())
		}
		if nick != session.GetCurrentUser().Nick {
			t.Errorf("LookupNick returned wrong value. Expected %s,"+
				" Got %s", session.GetCurrentUser().Nick, nick)
		}
	}()

	// send response to switchboard
	msg = &parse.Message{
		Sender: session.GetCurrentUser().User,
		TypedBody: parse.TypedBody{
			MessageType: int32(cmixproto.Type_NICKNAME_RESPONSE),
			Body:        []byte(session.GetCurrentUser().Nick),
		},
		InferredType: parse.Unencrypted,
		Receiver:     session.GetCurrentUser().User,
	}
	session.GetSwitchboard().Speak(msg)
}

type errorMessaging struct{}

// SendMessage that just errors out
func (e *errorMessaging) SendMessage(sess user.Session,
	recipientID *id.User,
	cryptoType parse.CryptoType,
	message []byte) error {
	return errors.New("This is an error")
}

// SendMessage no partition that just errors out
func (e *errorMessaging) SendMessageNoPartition(sess user.Session,
	recipientID *id.User,
	cryptoType parse.CryptoType,
	message []byte) error {
	return errors.New("This is an error")
}

// MessageReceiver thread to get new messages
func (e *errorMessaging) MessageReceiver(session user.Session,
	delay time.Duration) {
}

// Test LookupNick returns error on sending problem
func TestLookupNick_error(t *testing.T) {
	// Replace comms with errorMessaging
	messaging = &errorMessaging{}
	_, err := LookupNick(session.GetCurrentUser().User)
	if err == nil {
		t.Errorf("LookupNick should have returned an error")
	}
}
