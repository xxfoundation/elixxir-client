////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// sqlite requires cgo, which is not available in wasm
//go:build !js || !wasm

package storage

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/dm"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"testing"
	"time"
)

func dummyReceivedMessageCB(uint64, ed25519.PublicKey, bool, bool) {}

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	os.Exit(m.Run())
}

// Test simple receive of a new message for a new conversation.
func TestImpl_Receive(t *testing.T) {
	m, err := newImpl("TestImpl_Receive", nil,
		dummyReceivedMessageCB, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	testString := "test"
	testBytes := []byte(testString)
	partnerPubKey := ed25519.PublicKey(testBytes)
	testRound := id.Round(10)

	// Can use ChannelMessageID for ease, doesn't matter here
	testMsgId := message.DeriveChannelMessageID(&id.ID{1}, uint64(testRound), testBytes)

	// Receive a test message
	uuid := m.Receive(testMsgId, testString, testBytes,
		partnerPubKey, partnerPubKey, 0, 0, time.Now(),
		rounds.Round{ID: testRound}, dm.TextType, dm.Received)
	if uuid == 0 {
		t.Fatalf("Expected non-zero message uuid")
	}
	jww.DEBUG.Printf("Received test message: %d", uuid)

	// First, we expect a conversation to be created
	testConvo := m.GetConversation(partnerPubKey)
	if testConvo == nil {
		t.Fatalf("Expected conversation to be created")
	}
	// Spot check a conversation attribute
	if testConvo.Nickname != testString {
		t.Fatalf("Expected conversation nickname %s, got %s",
			testString, testConvo.Nickname)
	}

	// Next, we expect the message to be created
	testMessage := &Message{Id: uuid}
	err = m.db.Take(testMessage).Error
	if err != nil {
		t.Fatalf(err.Error())
	}
	// Spot check a message attribute
	if !bytes.Equal(testMessage.SenderPubKey, partnerPubKey) {
		t.Fatalf("Expected message attibutes to match, expected %v got %v",
			partnerPubKey, testMessage.SenderPubKey)
	}
}

// Test happy path. Insert some conversations and check they exist.
func TestImpl_GetConversations(t *testing.T) {
	m, err := newImpl("TestImpl_GetConversations", nil,
		dummyReceivedMessageCB, true)
	if err != nil {
		t.Fatal(err.Error())
	}
	numTestConvo := 10

	// Insert a test convo
	for i := 0; i < numTestConvo; i++ {
		testBytes := []byte(fmt.Sprintf("%d", i))
		testPubKey := ed25519.PublicKey(testBytes)
		err = m.upsertConversation("test", testPubKey,
			uint32(i), uint8(i), &time.Time{})
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	results := m.GetConversations()
	if len(results) != numTestConvo {
		t.Fatalf("Expected %d convos, got %d", numTestConvo, len(results))
	}

	for i, convo := range results {
		if convo.Token != uint32(i) {
			t.Fatalf("Expected %d convo token, got %d", i, convo.Token)
		}
		if convo.CodesetVersion != uint8(i) {
			t.Fatalf("Expected %d convo codeset, got %d",
				i, convo.CodesetVersion)
		}
	}
}

// Test happy path toggling between blocked/unblocked in a Conversation.
func TestImpl_BlockSender(t *testing.T) {
	m, err := newImpl("TestImpl_BlockSender", nil,
		dummyReceivedMessageCB, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Insert a test convo
	testBytes := []byte("test")
	testPubKey := ed25519.PublicKey(testBytes)
	err = m.upsertConversation("test", testPubKey, 0, 0, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Default to unblocked
	result := m.GetConversation(testPubKey)
	if result.BlockedTimestamp != nil {
		t.Fatal("Expected blocked to be nil")
	}

	// Now toggle blocked
	m.BlockSender(testPubKey)
	result = m.GetConversation(testPubKey)
	if result.BlockedTimestamp == nil {
		t.Fatal("Expected blocked to be non-nil")
	}

	// Now toggle blocked again
	m.UnblockSender(testPubKey)
	result = m.GetConversation(testPubKey)
	if result.BlockedTimestamp != nil {
		t.Fatalf("Expected blocked to be nil, got %+v", result)
	}
}
