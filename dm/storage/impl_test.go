////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// sqlite requires cgo, which is not available in wasm
//go:build !js || !wasm

package storage

import (
	"crypto/ed25519"
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"testing"
)

func dummyReceivedMessageCB(uint64, ed25519.PublicKey, bool, bool) {}

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	os.Exit(m.Run())
}

// Test happy path toggling between blocked/unblocked in a Conversation.
func TestWasmModel_BlockSender(t *testing.T) {
	m, err := newImpl("test", nil, dummyReceivedMessageCB)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Insert a test convo
	testPubKey := ed25519.PublicKey{}
	err = m.createConversation("test", testPubKey, 0, 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Default to unblocked
	result := m.GetConversation(testPubKey)
	if result.Blocked {
		t.Fatal("Expected blocked to be false")
	}

	// Now toggle blocked
	m.BlockSender(testPubKey)
	result = m.GetConversation(testPubKey)
	if !result.Blocked {
		t.Fatal("Expected blocked to be true")
	}

	// Now toggle blocked again
	m.UnblockSender(testPubKey)
	result = m.GetConversation(testPubKey)
	if result.Blocked {
		t.Fatal("Expected blocked to be false")
	}
}
