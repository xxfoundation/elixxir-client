///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package groupChat

import (
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Unit test of MessageReceive.String.
func TestMessageReceive_String(t *testing.T) {
	msg := MessageReceive{
		GroupID:   id.NewIdFromString("GroupID", id.Group, t),
		ID:        group.MessageID{0, 1, 2, 3},
		Payload:   []byte("Group message."),
		SenderID:  id.NewIdFromString("SenderID", id.User, t),
		Timestamp: time.Date(1955, 11, 5, 12, 0, 0, 0, time.UTC),
	}

	expected := "{" +
		"GroupID:R3JvdXBJRAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE " +
		"ID:AAECAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= " +
		"Payload:\"Group message.\" " +
		"SenderID:U2VuZGVySUQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD " +
		"Timestamp:" + msg.Timestamp.String() +
		"}"

	if msg.String() != expected {
		t.Errorf("String() returned the incorrect string."+
			"\nexpected: %s\nreceived: %s", expected, msg.String())
	}
}

// Tests that MessageReceive.String returns the expected value for a message
// with nil values.
func TestMessageReceive_String_NilMessageReceive(t *testing.T) {
	msg := MessageReceive{}

	expected := "{" +
		"GroupID:<nil> " +
		"ID:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= " +
		"Payload:<nil> " +
		"SenderID:<nil> " +
		"Timestamp:0001-01-01 00:00:00 +0000 UTC" +
		"}"

	if msg.String() != expected {
		t.Errorf("String() returned the incorrect string."+
			"\nexpected: %s\nreceived: %s", expected, msg.String())
	}
}
