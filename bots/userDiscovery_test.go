////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package bot functions for working with the user discovery bot (UDB)
package bots

import (
	"encoding/base64"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/crypto/format"
	"os"
	"testing"
	"time"
	"gitlab.com/privategrity/crypto/id"
)

var ListenCh chan *format.Message

type dummyMessaging struct {
	listener chan *format.Message
}

// SendMessage to the server
func (d *dummyMessaging) SendMessage(recipientID *id.UserID,
	message []byte) error {
	jww.INFO.Printf("Sending: %s", string(message))
	return nil
}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(delay time.Duration) {}

var pubKeyBits string
var keyFingerprint string
var pubKey []byte

func TestMain(m *testing.M) {
	// Make the reception channels buffered for this test
	pushKeyResponseListener = make(udbResponseListener, 100)
	getKeyResponseListener = make(udbResponseListener, 100)
	registerResponseListener = make(udbResponseListener, 100)
	searchResponseListener = make(udbResponseListener, 100)

	io.Messaging = &dummyMessaging{}

	io.Messaging = &dummyMessaging{
		listener: ListenCh,
	}

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

	err := Register("EMAIL", "rick@privategrity.com", pubKey)
	if err != nil {
		t.Errorf("Registration failure: %s", err.Error())
	}
}

// TestSearch smoke tests the search function
func TestSearch(t *testing.T) {

	// Send response messages from fake UDB in advance
	searchResponseListener <- fmt.Sprintf("SEARCH %s FOUND %d %s",
		"blah@privategrity.com", 26, keyFingerprint)
	getKeyResponseListener <- fmt.Sprintf("GETKEY %s %s", keyFingerprint,
		pubKeyBits)

	contacts, err := Search("EMAIL", "blah@privategrity.com")
	if err != nil {
		t.Errorf("Error on Search: %s", err.Error())
	}
	_, ok := contacts[26]
	if !ok {
		t.Errorf("Search did not return user ID 26!")
	}
}
