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
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/channels"
	cft "gitlab.com/elixxir/client/v4/channelsFileTransfer"
	"gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Happy path test for receiving, updating, getting, and deleting a File.
func TestImpl_ReceiveFile(t *testing.T) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	testCb := func(uuid uint64, channelID *id.ID, update bool) {}

	testString := "TestImpl_ReceiveFile"
	m, err := newImpl("", nil,
		testCb, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	testTs := time.Now()
	testBytes := []byte(testString)
	testStatus := cft.Downloading

	// Insert a test row
	fId := fileTransfer.NewID(testBytes)
	err = m.ReceiveFile(fId, testBytes, testBytes, testTs, testStatus)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt to get stored row
	storedFile, err := m.GetFile(fId)
	if err != nil {
		t.Fatal(err)
	}
	// Spot check stored attribute
	if !bytes.Equal(storedFile.Link, testBytes) {
		t.Fatalf("Got unequal FileLink values")
	}

	// Attempt to updated stored row
	newTs := time.Now()
	newBytes := []byte("test")
	newStatus := cft.Complete
	err = m.UpdateFile(fId, nil, newBytes, &newTs, &newStatus)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the update took
	updatedFile, err := m.GetFile(fId)
	if err != nil {
		t.Fatal(err)
	}
	// Link should not have changed
	if !bytes.Equal(updatedFile.Link, testBytes) {
		t.Fatalf("Link should not have changed")
	}
	// Other attributes should have changed
	if !bytes.Equal(updatedFile.Data, newBytes) {
		t.Fatalf("Data should have updated")
	}
	if !updatedFile.Timestamp.Equal(newTs) {
		t.Fatalf("TS should have updated, expected %s got %s",
			newTs, updatedFile.Timestamp)
	}
	if updatedFile.Status != newStatus {
		t.Fatalf("Status should have updated")
	}

	// Delete the row
	err = m.DeleteFile(fId)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the delete operation took and get provides the expected error
	_, err = m.GetFile(fId)
	if err == nil || !errors.Is(channels.NoMessageErr, err) {
		t.Fatal(err)
	}
}

// Test error does not exist path
func TestImpl_DeleteMessage_Error(t *testing.T) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	testCb := func(uuid uint64, channelID *id.ID, update bool) {}

	testString := "TestImpl_DeleteMessage_Error"
	m, err := newImpl("", nil,
		testCb, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	testBytes := []byte(testString)
	fId := fileTransfer.NewID(testBytes)

	// Attempt to delete the row
	err = m.DeleteFile(fId)
	// Check that the delete operation failed and provides the expected error
	if err == nil || !errors.Is(channels.NoMessageErr, err) {
		t.Fatal(err)
	}
}
