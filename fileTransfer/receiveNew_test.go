////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"testing"
	"time"
)

// Tests that State.receiveNewFileTransfer receives the sent message and that
// it reports the correct data to the callback.
func TestManager_receiveNewFileTransfer(t *testing.T) {
	// Create new ReceiveCallback that sends the results on a channel
	receiveChan := make(chan receivedFtResults)
	receiveCB := func(tid ftCrypto.TransferID, fileName, fileType string,
		sender *id.ID, size uint32, preview []byte) {
		receiveChan <- receivedFtResults{
			tid, fileName, fileType, sender, size, preview}
	}

	// Create new manager, stoppable, and channel to receive messages
	m := newTestManager(false, nil, nil, receiveCB, nil, t)
	stop := stoppable.NewSingle(newFtStoppableName)
	rawMsgs := make(chan message.Receive, rawMessageBuffSize)

	// Start receiving thread
	go m.receiveNewFileTransfer(rawMsgs, stop)

	// Create new message.Receive with marshalled NewFileTransfer
	key, _ := ftCrypto.NewTransferKey(NewPrng(42))
	protoMsg := &NewFileTransfer{
		FileName:    "testFile",
		TransferKey: key.Bytes(),
		TransferMac: []byte("transferMac"),
		NumParts:    16,
		Size:        256,
		Retry:       1.5,
		Preview:     []byte("filePreview"),
	}
	marshalledMsg, err := proto.Marshal(protoMsg)
	if err != nil {
		t.Errorf("Failed to Marshal proto message: %+v", err)
	}
	receiveMsg := message.Receive{
		Payload:     marshalledMsg,
		MessageType: message.NewFileTransfer,
		Sender:      id.NewIdFromString("sender", id.User, t),
	}

	// Add message to channel
	rawMsgs <- receiveMsg

	// Wait to receive message
	select {
	case <-time.NewTimer(100 * time.Millisecond).C:
		t.Error("Timed out waiting to receive message.")
	case r := <-receiveChan:
		if !receiveMsg.Sender.Cmp(r.sender) {
			t.Errorf("Received sender ID does not match expected."+
				"\nexpected: %s\nreceived: %s", receiveMsg.Sender, r.sender)
		}
		if protoMsg.Size != r.size {
			t.Errorf("Received file size does not match expected."+
				"\nexpected: %d\nreceived: %d", protoMsg.Size, r.size)
		}
		if !bytes.Equal(protoMsg.Preview, r.preview) {
			t.Errorf("Received preview does not match expected."+
				"\nexpected: %q\nreceived: %q", protoMsg.Preview, r.preview)
		}
	}

	// Stop thread
	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to stop stoppable: %+v", err)
	}
}

// Tests that State.receiveNewFileTransfer stops receiving messages when the
// stoppable is triggered.
func TestManager_receiveNewFileTransfer_Stop(t *testing.T) {
	// Create new ReceiveCallback that sends the results on a channel
	receiveChan := make(chan receivedFtResults)
	receiveCB := func(tid ftCrypto.TransferID, fileName, fileType string,
		sender *id.ID, size uint32, preview []byte) {
		receiveChan <- receivedFtResults{
			tid, fileName, fileType, sender, size, preview}
	}

	// Create new manager, stoppable, and channel to receive messages
	m := newTestManager(false, nil, nil, receiveCB, nil, t)
	stop := stoppable.NewSingle(newFtStoppableName)
	rawMsgs := make(chan message.Receive, rawMessageBuffSize)

	// Start receiving thread
	go m.receiveNewFileTransfer(rawMsgs, stop)

	// Create new message.Receive with marshalled NewFileTransfer
	key, _ := ftCrypto.NewTransferKey(NewPrng(42))
	protoMsg := &NewFileTransfer{
		FileName:    "testFile",
		TransferKey: key.Bytes(),
		TransferMac: []byte("transferMac"),
		NumParts:    16,
		Size:        256,
		Retry:       1.5,
		Preview:     []byte("filePreview"),
	}
	marshalledMsg, err := proto.Marshal(protoMsg)
	if err != nil {
		t.Errorf("Failed to Marshal proto message: %+v", err)
	}
	receiveMsg := message.Receive{
		Payload:     marshalledMsg,
		MessageType: message.NewFileTransfer,
		Sender:      id.NewIdFromString("sender", id.User, t),
	}

	// Stop thread
	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to stop stoppable: %+v", err)
	}

	for !stop.IsStopped() {
	}

	// Add message to channel
	rawMsgs <- receiveMsg

	// Wait to receive message
	select {
	case <-time.NewTimer(time.Millisecond).C:
	case r := <-receiveChan:
		t.Errorf("Callback called when the thread should have quit."+
			"\nreceived: %+v", r)
	}
}

// Tests that State.receiveNewFileTransfer does not report on the callback
// when the received message is of the wrong type.
func TestManager_receiveNewFileTransfer_InvalidMessageError(t *testing.T) {
	// Create new ReceiveCallback that sends the results on a channel
	receiveChan := make(chan receivedFtResults)
	receiveCB := func(tid ftCrypto.TransferID, fileName, fileType string,
		sender *id.ID, size uint32, preview []byte) {
		receiveChan <- receivedFtResults{
			tid, fileName, fileType, sender, size, preview}
	}

	// Create new manager, stoppable, and channel to receive messages
	m := newTestManager(false, nil, nil, receiveCB, nil, t)
	stop := stoppable.NewSingle(newFtStoppableName)
	rawMsgs := make(chan message.Receive, rawMessageBuffSize)

	// Start receiving thread
	go m.receiveNewFileTransfer(rawMsgs, stop)

	// Create new message.Receive with wrong type
	receiveMsg := message.Receive{
		MessageType: message.NoType,
	}

	// Add message to channel
	rawMsgs <- receiveMsg

	// Wait to receive message
	select {
	case <-time.NewTimer(time.Millisecond).C:
	case r := <-receiveChan:
		t.Errorf("Callback called when the message is of the wrong type."+
			"\nreceived: %+v", r)
	}

	// Stop thread
	err := stop.Close()
	if err != nil {
		t.Errorf("Failed to stop stoppable: %+v", err)
	}
}

// Tests that State.readNewFileTransferMessage returns the expected sender ID,
// file size, and preview.
func TestManager_readNewFileTransferMessage(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	// Create new message.Send containing marshalled NewFileTransfer
	recipient := id.NewIdFromString("recipient", id.User, t)
	expectedFileName := "testFile"
	expectedFileType := "txt"
	key, _ := ftCrypto.NewTransferKey(NewPrng(42))
	mac := []byte("transferMac")
	numParts, expectedFileSize, retry := uint16(16), uint32(256), float32(1.5)
	expectedPreview := []byte("filePreview")
	sendMsg, err := newNewFileTransferE2eMessage(
		recipient, expectedFileName, expectedFileType, key, mac, numParts,
		expectedFileSize, retry, expectedPreview)
	if err != nil {
		t.Errorf("Failed to create new Send message: %+v", err)
	}

	// Create message.Receive with marshalled NewFileTransfer
	receiveMsg := message.Receive{
		Payload:     sendMsg.Payload,
		MessageType: message.NewFileTransfer,
		Sender:      id.NewIdFromString("sender", id.User, t),
	}

	// Read the message
	_, fileName, fileType, sender, fileSize, preview, err :=
		m.readNewFileTransferMessage(receiveMsg)
	if err != nil {
		t.Errorf("readNewFileTransferMessage returned an error: %+v", err)
	}

	if expectedFileName != fileName {
		t.Errorf("Returned file name does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedFileName, fileName)
	}

	if expectedFileType != fileType {
		t.Errorf("Returned file type does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedFileType, fileType)
	}

	if !receiveMsg.Sender.Cmp(sender) {
		t.Errorf("Returned sender ID does not match expected."+
			"\nexpected: %s\nreceived: %s", receiveMsg.Sender, sender)
	}

	if expectedFileSize != fileSize {
		t.Errorf("Returned file size does not match expected."+
			"\nexpected: %d\nreceived: %d", expectedFileSize, fileSize)
	}

	if !bytes.Equal(expectedPreview, preview) {
		t.Errorf("Returned preview does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedPreview, preview)
	}
}

// Error path: tests that State.readNewFileTransferMessage returns the
// expected error when the message.Receive has the wrong MessageType.
func TestManager_readNewFileTransferMessage_MessageTypeError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	expectedErr := receiveMessageTypeErr

	// Create message.Receive with marshalled NewFileTransfer
	receiveMsg := message.Receive{
		MessageType: message.NoType,
	}

	// Read the message
	_, _, _, _, _, _, err := m.readNewFileTransferMessage(receiveMsg)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("readNewFileTransferMessage did not return the expected "+
			"error when the message.Receive has the wrong MessageType."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that State.readNewFileTransferMessage returns the
// expected error when the payload of the message.Receive cannot be
// unmarshalled.
func TestManager_readNewFileTransferMessage_ProtoUnmarshalError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	expectedErr := strings.Split(protoUnmarshalErr, "%")[0]

	// Create message.Receive with marshalled NewFileTransfer
	receiveMsg := message.Receive{
		Payload:     []byte("invalidPayload"),
		MessageType: message.NewFileTransfer,
	}

	// Read the message
	_, _, _, _, _, _, err := m.readNewFileTransferMessage(receiveMsg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readNewFileTransferMessage did not return the expected "+
			"error when the payload could not be unmarshalled."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}
