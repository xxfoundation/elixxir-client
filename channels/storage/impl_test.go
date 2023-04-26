////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// sqlite requires cgo, which is not available in wasm
//go:build !js || !wasm

package storage

import (
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
)

// Series of interdependent smoke tests of the impl object and its methods.
func TestImpl(t *testing.T) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	testCb := func(uuid uint64, channelID *id.ID, update bool) {}

	model, err := newImpl("", nil, testCb, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Join a Channel
	testString := "test"
	testChannelId := &id.DummyUser
	testChannel := &cryptoBroadcast.Channel{
		ReceptionID: testChannelId,
		Name:        testString,
		Description: testString,
	}
	model.JoinChannel(testChannel)

	// Receive a Message
	testBytes := []byte(testString)
	testRoundId := uint64(10)
	testMsgId := message.DeriveChannelMessageID(testChannelId,
		testRoundId, testBytes)
	testRound := rounds.Round{ID: id.Round(testRoundId)}
	newId := model.ReceiveMessage(testChannelId, testMsgId, testString, testString, testBytes,
		0, 0, time.Now(), 0, testRound, 0, 0, false)
	t.Logf("Inserted message with ID: %d", newId)

	// Update the Message
	testInt := 1
	testTime := time.Now()
	testBool := true
	testStatus := channels.SentStatus(testInt)
	updatedId, err := model.UpdateFromMessageID(testMsgId, &testTime, nil,
		&testBool, &testBool, &testStatus)
	if err != nil {
		t.Fatal(err)
	} else if updatedId != newId {
		t.Fatalf("UUIDs differ, Got %d Expected %d", updatedId, newId)
	}

	// Compare updated Message with the original
	gotMsg, err := model.GetMessage(testMsgId)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Got Message: %v", gotMsg)
	if gotMsg.UUID != newId {
		t.Fatalf("Params differ, Got %d Expected %d", gotMsg.UUID, newId)
	}
	if !gotMsg.Timestamp.Equal(testTime) {
		t.Fatalf("Params differ, Got %T Expected %T", gotMsg.Timestamp, testTime)
	}
	if gotMsg.Hidden != testBool {
		t.Fatalf("Params differ, Got %t Expected %t", gotMsg.Hidden, testBool)
	}
	if gotMsg.Pinned != testBool {
		t.Fatalf("Params differ, Got %t Expected %t", gotMsg.Pinned, testBool)
	}
	if gotMsg.Status != testStatus {
		t.Fatalf("Params differ, Got %d Expected %d", gotMsg.Status, testStatus)
	}

	// Leave a channel and ensure its Messages are deleted
	model.LeaveChannel(testChannelId)
	gotMsg, err = model.GetMessage(testMsgId)
	if err == nil {
		t.Fatal("Expected to be unable to get deleted Message")
	}
}
