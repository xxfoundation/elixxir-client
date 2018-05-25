////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package bot functions for working with the user discovery bot (UDB)
package bot

import (
	"encoding/base64"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/crypto/format"
	"os"
	"testing"
	"time"
)

var ListenCh chan *format.Message

type dummyMessaging struct {
	listener chan *format.Message
}

// SendMessage to the server
func (d *dummyMessaging) SendMessage(recipientID uint64, message string) error {
	jww.ERROR.Printf("Sending: %s", message)
	return nil
}

// Listen for messages from a given sender
func (d *dummyMessaging) Listen(senderID uint64) chan *format.Message {
	return d.listener
}

// StopListening to a given listener (closes and deletes)
func (d *dummyMessaging) StopListening(listenerCh chan *format.Message) {}

// MessageReceiver thread to get new messages
func (d *dummyMessaging) MessageReceiver(delay time.Duration) {}

func TestMain(m *testing.M) {
	ListenCh = make(chan *format.Message, 100)
	io.Messaging = &dummyMessaging{
		listener: ListenCh,
	}
	os.Exit(m.Run())
}

// SendMsg puts a fake udb response message on the channel
func SendMsg(msg string) {
	m, _ := format.NewMessage(13, 1, msg)
	ListenCh <- &m[0]
}

// TestRegister smoke tests the registration functionality.
func TestRegister(t *testing.T) {
	pubKeyBits := []string{
		"S8KXBczy0jins9uS4LgBPt0bkFl8t00MnZmExQ6GcOcu8O7DKgAsNz" +
			"LU7a+gMTbIsS995IL/kuFF8wcBaQJBY23095PMSQ/nMuetzhk9HdXxrGIiKBo3C/n4SClp" +
			"q4H+PoF9XziEVKua8JxGM2o83KiCK3tNUpaZbAAElkjueY4=",
		"8Lg/eoeKGgPlleTYfO3JyGfnwBtLi73ti0h2dBQWW94JTqTQDr+z" +
			"xVpLzdgTt+87TkAl0yXu9mOUXqGJ+51lTcRlIdIpWpfgUbibdRme8IThg0RNCF31ESKCts" +
			"o8gJ8mSVljIXxrC+Uuoi+Gl1LNN5nPARykatx0Y70xNdJd2BQ=",
	}
	pubKey := make([]byte, 256)
	for i := range pubKeyBits {
		bytes, _ := base64.StdEncoding.DecodeString(pubKeyBits[i])
		for j := range bytes {
			pubKey[j+i*128] = bytes[j]
		}
	}

	fingerprint := "8oKh7TYG4KxQcBAymoXPBHSD/uga9pX3Mn/jKhvcD8M="

	// Send response messages in advance
	SendMsg(fmt.Sprintf("GETKEY %s NOTFOUND", fingerprint))
	SendMsg("PUSHKEY ACK NEED 128")
	SendMsg(fmt.Sprintf("PUSHKEY COMPLETE %s", fingerprint))
	SendMsg("REGISTRATION COMPLETE")

	err := Register("EMAIL", "rick@privategrity.com", pubKey)
	if err != nil {
		t.Errorf("Registration failure: %s", err.Error())
	}
}
