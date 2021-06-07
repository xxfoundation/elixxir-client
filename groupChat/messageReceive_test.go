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
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"testing"
	"time"
)

// Unit test of MessageReceive.String.
func TestMessageReceive_String(t *testing.T) {
	msg := MessageReceive{
		GroupID:        id.NewIdFromString("GroupID", id.Group, t),
		ID:             group.MessageID{0, 1, 2, 3},
		Payload:        []byte("Group message."),
		SenderID:       id.NewIdFromString("SenderID", id.User, t),
		RecipientID:    id.NewIdFromString("RecipientID", id.User, t),
		EphemeralID:    ephemeral.Id{0, 1, 2, 3},
		Timestamp:      time.Date(1955, 11, 5, 12, 0, 0, 0, time.UTC),
		RoundID:        42,
		RoundTimestamp: time.Date(1955, 11, 5, 12, 1, 0, 0, time.UTC),
	}

	expected := "{" +
		"GroupID:R3JvdXBJRAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE " +
		"ID:AAECAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA= " +
		"Payload:\"Group message.\" " +
		"SenderID:U2VuZGVySUQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD " +
		"RecipientID:UmVjaXBpZW50SUQAAAAAAAAAAAAAAAAAAAAAAAAAAAAD " +
		"EphemeralID:141843442434048 " +
		"Timestamp:" + msg.Timestamp.String() + " " +
		"RoundID:42 " +
		"RoundTimestamp:" + msg.RoundTimestamp.String() +
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
		"RecipientID:<nil> " +
		"EphemeralID:0 " +
		"Timestamp:0001-01-01 00:00:00 +0000 UTC " +
		"RoundID:0 " +
		"RoundTimestamp:0001-01-01 00:00:00 +0000 UTC" +
		"}"

	if msg.String() != expected {
		t.Errorf("String() returned the incorrect string."+
			"\nexpected: %s\nreceived: %s", expected, msg.String())
	}
}
