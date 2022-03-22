////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	ftStorage "gitlab.com/elixxir/client/storage/fileTransfer"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/diffieHellman"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests that newManager does not return errors, that the sent and received
// transfer lists are new, and that the callback works.
func Test_newManager(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))

	cbChan := make(chan bool)
	cb := func(ftCrypto.TransferID, string, string, *id.ID, uint32, []byte) {
		cbChan <- true
	}

	m, err := newManager(nil, nil, nil, nil, nil, nil, kv, cb, DefaultParams())
	if err != nil {
		t.Errorf("newManager returned an error: %+v", err)
	}

	// Check that the SentFileTransfersStore is new and correct
	expectedSent, _ := ftStorage.NewSentFileTransfersStore(kv)
	if !reflect.DeepEqual(expectedSent, m.sent) {
		t.Errorf("SentFileTransfersStore in manager incorrect."+
			"\nexpected: %+v\nreceived: %+v", expectedSent, m.sent)
	}

	// Check that the ReceivedFileTransfersStore is new and correct
	expectedReceived, _ := ftStorage.NewReceivedFileTransfersStore(kv)
	if !reflect.DeepEqual(expectedReceived, m.received) {
		t.Errorf("ReceivedFileTransfersStore in manager incorrect."+
			"\nexpected: %+v\nreceived: %+v", expectedReceived, m.received)
	}

	// Check that the callback is called
	go m.receiveCB(ftCrypto.TransferID{}, "", "", nil, 0, nil)
	select {
	case <-cbChan:
	case <-time.NewTimer(time.Millisecond).C:
		t.Error("Timed out waiting for callback to be called")
	}
}

// Tests that Manager.Send adds a new sent transfer, sends the NewFileTransfer
// E2E message, and adds all the file parts to the queue.
func TestManager_Send(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	fileName := "testFile"
	fileType := "txt"
	numParts := uint16(16)
	partSize, _ := m.getPartSize()
	fileData, _ := newFile(numParts, partSize, prng, t)
	preview := []byte("filePreview")
	retry := float32(1.5)
	numFps := calcNumberOfFingerprints(numParts, retry)

	rng := csprng.NewSystemRNG()
	dhKey := m.store.E2e().GetGroup().NewInt(42)
	pubKey := diffieHellman.GeneratePublicKey(dhKey, m.store.E2e().GetGroup())
	_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhB, rng)
	p := params.GetDefaultE2ESessionParams()

	err := m.store.E2e().AddPartner(recipient, pubKey, dhKey,
		mySidhPriv, theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipient, err)
	}

	tid, err := m.Send(
		fileName, fileType, fileData, recipient, retry, preview, nil, 0)
	if err != nil {
		t.Errorf("Send returned an error: %+v", err)
	}

	////
	// Check if the transfer exists
	////
	transfer, err := m.sent.GetTransfer(tid)
	if err != nil {
		t.Errorf("Failed to get transfer %s: %+v", tid, err)
	}

	if !recipient.Cmp(transfer.GetRecipient()) {
		t.Errorf("New transfer has incorrect recipient."+
			"\nexpected: %s\nreceoved: %s", recipient, transfer.GetRecipient())
	}
	if transfer.GetNumParts() != numParts {
		t.Errorf("New transfer has incorrect number of parts."+
			"\nexpected: %d\nreceived: %d", numParts, transfer.GetNumParts())
	}
	if transfer.GetNumFps() != numFps {
		t.Errorf("New transfer has incorrect number of fingerprints."+
			"\nexpected: %d\nreceived: %d", numFps, transfer.GetNumFps())
	}

	////
	// get NewFileTransfer E2E message
	////
	sendMsg := m.net.(*testNetworkManager).GetE2eMsg(0)
	if sendMsg.MessageType != message.NewFileTransfer {
		t.Errorf("E2E message has wrong MessageType.\nexpected: %d\nreceived: %d",
			message.NewFileTransfer, sendMsg.MessageType)
	}
	if !sendMsg.Recipient.Cmp(recipient) {
		t.Errorf("E2E message has wrong Recipient.\nexpected: %s\nreceived: %s",
			recipient, sendMsg.Recipient)
	}
	receivedNFT := &NewFileTransfer{}
	err = proto.Unmarshal(sendMsg.Payload, receivedNFT)
	if err != nil {
		t.Errorf("Failed to unmarshal received NewFileTransfer: %+v", err)
	}
	expectedNFT := &NewFileTransfer{
		FileName:    fileName,
		FileType:    fileType,
		TransferKey: transfer.GetTransferKey().Bytes(),
		TransferMac: ftCrypto.CreateTransferMAC(fileData, transfer.GetTransferKey()),
		NumParts:    uint32(numParts),
		Size:        uint32(len(fileData)),
		Retry:       retry,
		Preview:     preview,
	}
	if !proto.Equal(expectedNFT, receivedNFT) {
		t.Errorf("Received NewFileTransfer message does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expectedNFT, receivedNFT)
	}

	////
	// Check queued parts
	////
	if len(m.sendQueue) != int(numParts) {
		t.Errorf("Failed to add all file parts to queue."+
			"\nexpected: %d\nreceived: %d", numParts, len(m.sendQueue))
	}
}

// Error path: tests that Manager.Send returns the expected error when the
// network is not healthy.
func TestManager_Send_NetworkHealthError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	fileName := "MySentFile"
	expectedErr := fmt.Sprintf(sendNetworkHealthErr, fileName)

	m.net.(*testNetworkManager).health.healthy = false

	recipient := id.NewIdFromString("recipient", id.User, t)
	_, err := m.Send(fileName, "", nil, recipient, 0, nil, nil, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Send did not return the expected error when the network is "+
			"not healthy.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that Manager.Send returns the expected error when the
// provided file name is longer than FileNameMaxLen.
func TestManager_Send_FileNameLengthError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	fileName := strings.Repeat("A", FileNameMaxLen+1)
	expectedErr := fmt.Sprintf(fileNameSizeErr, len(fileName), FileNameMaxLen)

	_, err := m.Send(fileName, "", nil, nil, 0, nil, nil, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Send did not return the expected error when the file name "+
			"is too long.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that Manager.Send returns the expected error when the
// provided file type is longer than FileTypeMaxLen.
func TestManager_Send_FileTypeLengthError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	fileType := strings.Repeat("A", FileTypeMaxLen+1)
	expectedErr := fmt.Sprintf(fileTypeSizeErr, len(fileType), FileTypeMaxLen)

	_, err := m.Send("", fileType, nil, nil, 0, nil, nil, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Send did not return the expected error when the file type "+
			"is too long.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that Manager.Send returns the expected error when the
// provided file is larger than FileMaxSize.
func TestManager_Send_FileSizeError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	fileData := make([]byte, FileMaxSize+1)
	expectedErr := fmt.Sprintf(fileSizeErr, len(fileData), FileMaxSize)

	_, err := m.Send("", "", fileData, nil, 0, nil, nil, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Send did not return the expected error when the file data "+
			"is too large.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that Manager.Send returns the expected error when the
// provided preview is larger than PreviewMaxSize.
func TestManager_Send_PreviewSizeError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	previewData := make([]byte, PreviewMaxSize+1)
	expectedErr := fmt.Sprintf(previewSizeErr, len(previewData), PreviewMaxSize)

	_, err := m.Send("", "", nil, nil, 0, previewData, nil, 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Send did not return the expected error when the preview "+
			"data is too large.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that Manager.Send returns the expected error when the E2E
// message fails to send.
func TestManager_Send_SendE2eError(t *testing.T) {
	m := newTestManager(true, nil, nil, nil, nil, t)
	prng := NewPrng(42)
	recipient := id.NewIdFromString("recipient", id.User, t)
	fileName := "testFile"
	fileType := "bytes"
	numParts := uint16(16)
	partSize, _ := m.getPartSize()
	fileData, _ := newFile(numParts, partSize, prng, t)
	preview := []byte("filePreview")
	retry := float32(1.5)

	rng := csprng.NewSystemRNG()
	dhKey := m.store.E2e().GetGroup().NewInt(42)
	pubKey := diffieHellman.GeneratePublicKey(dhKey, m.store.E2e().GetGroup())
	_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhB, rng)
	p := params.GetDefaultE2ESessionParams()

	err := m.store.E2e().AddPartner(recipient, pubKey, dhKey,
		mySidhPriv, theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipient, err)
	}

	expectedErr := fmt.Sprintf(newFtSendE2eErr, recipient, "")

	_, err = m.Send(
		fileName, fileType, fileData, recipient, retry, preview, nil, 0)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Send did not return the expected error when the E2E message "+
			"failed to send.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that Manager.RegisterSentProgressCallback calls the callback when it is
// added to the transfer and that the callback is associated with the expected
// transfer and is called when calling from the transfer.
func TestManager_RegisterSentProgressCallback(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, false, nil, nil, nil, t)
	expectedErr := errors.New("CallbackError")

	// Create new callback and channel for the callback to trigger
	cbChan := make(chan sentProgressResults, 6)
	cb := func(completed bool, sent, arrived, total uint16,
		tr interfaces.FilePartTracker, err error) {
		cbChan <- sentProgressResults{completed, sent, arrived, total, tr, err}
	}

	// Start thread waiting for callback to be called
	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(20 * time.Millisecond).C:
				t.Errorf("Timed out waiting for callback call #%d.", i)
			case r := <-cbChan:
				switch i {
				case 0:
					err := checkSentProgress(r.completed, r.sent, r.arrived,
						r.total, false, 0, 0, sti[0].numParts)
					if err != nil {
						t.Errorf("%d: %+v", i, err)
					}
					if r.err != nil {
						t.Errorf("Callback returned an error (%d): %+v", i, r.err)
					}
					done0 <- true
				case 1:
					if r.err == nil || r.err != expectedErr {
						t.Errorf("Callback did not return the expected error (%d)."+
							"\nexpected: %v\nreceived: %+v", i, expectedErr, r.err)
					}
					done1 <- true
				}
			}
		}
	}()

	err := m.RegisterSentProgressCallback(sti[0].tid, cb, 1*time.Millisecond)
	if err != nil {
		t.Errorf("RegisterSentProgressCallback returned an error: %+v", err)
	}
	<-done0

	transfer, _ := m.sent.GetTransfer(sti[0].tid)

	transfer.CallProgressCB(expectedErr)

	<-done1
}

// Error path: tests that Manager.RegisterSentProgressCallback returns an error
// when no transfer with the ID exists.
func TestManager_RegisterSentProgressCallback_NoTransferError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	tid := ftCrypto.UnmarshalTransferID([]byte("invalidID"))

	err := m.RegisterSentProgressCallback(tid, nil, 0)
	if err == nil {
		t.Error("RegisterSentProgressCallback did not return an error when " +
			"no transfer with the ID exists.")
	}
}

func TestManager_Resend(t *testing.T) {

}

// Error path: tests that Manager.Resend returns an error when no transfer with
// the ID exists.
func TestManager_Resend_NoTransferError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	tid := ftCrypto.UnmarshalTransferID([]byte("invalidID"))

	err := m.Resend(tid)
	if err == nil {
		t.Error("Resend did not return an error when no transfer with the " +
			"ID exists.")
	}
}

// Error path: tests that Manager.Resend returns the error when the transfer has
// not run out of fingerprints.
func TestManager_Resend_NoFingerprints(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{16}, false, false, nil, nil, nil, t)
	expectedErr := fmt.Sprintf(transferNotFailedErr, sti[0].tid)
	// Delete the transfer
	err := m.Resend(sti[0].tid)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Resend did not return the expected error when the transfer "+
			"has not run out of fingerprints.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Tests that Manager.CloseSend deletes the transfer when it has run out of
// fingerprints but is not complete.
func TestManager_CloseSend_NoFingerprints(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{16}, false, false, nil, nil, nil, t)
	partSize, _ := m.getPartSize()

	// Use up all the fingerprints in the transfer
	transfer, _ := m.sent.GetTransfer(sti[0].tid)
	for fpNum := uint16(0); fpNum < sti[0].numFps; fpNum++ {
		partNum := fpNum % sti[0].numParts
		_, _, _, err := transfer.GetEncryptedPart(partNum, partSize+2)
		if err != nil {
			t.Errorf("Failed to encrypt part %d (%d): %+v", partNum, fpNum, err)
		}
	}

	// Delete the transfer
	err := m.CloseSend(sti[0].tid)
	if err != nil {
		t.Errorf("CloseSend returned an error: %+v", err)
	}

	// Check that the transfer was deleted
	_, err = m.sent.GetTransfer(sti[0].tid)
	if err == nil {
		t.Errorf("Failed to delete transfer %s.", sti[0].tid)
	}
}

// Tests that Manager.CloseSend deletes the transfer when it completed but has
// fingerprints.
func TestManager_CloseSend_Complete(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{3}, false, false, nil, nil, nil, t)

	// Set all parts to finished
	transfer, _ := m.sent.GetTransfer(sti[0].tid)
	_, err := transfer.SetInProgress(0, 0, 1, 2)
	if err != nil {
		t.Errorf("Failed to set parts to in-progress: %+v", err)
	}
	complete, err := transfer.FinishTransfer(0)
	if err != nil {
		t.Errorf("Failed to set parts to finished: %+v", err)
	}

	// Ensure that FinishTransfer reported the transfer as complete
	if !complete {
		t.Error("FinishTransfer did not report the transfer as complete.")
	}

	// Delete the transfer
	err = m.CloseSend(sti[0].tid)
	if err != nil {
		t.Errorf("CloseSend returned an error: %+v", err)
	}

	// Check that the transfer was deleted
	_, err = m.sent.GetTransfer(sti[0].tid)
	if err == nil {
		t.Errorf("Failed to delete transfer %s.", sti[0].tid)
	}
}

// Error path: tests that Manager.CloseSend returns an error when no transfer
// with the ID exists.
func TestManager_CloseSend_NoTransferError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	tid := ftCrypto.UnmarshalTransferID([]byte("invalidID"))

	err := m.CloseSend(tid)
	if err == nil {
		t.Error("CloseSend did not return an error when no transfer with the " +
			"ID exists.")
	}
}

// Error path: tests that Manager.CloseSend returns an error when the transfer
// has not run out of fingerprints and is not complete
func TestManager_CloseSend_NotCompleteErr(t *testing.T) {
	m, sti, _ := newTestManagerWithTransfers(
		[]uint16{16}, false, false, nil, nil, nil, t)
	expectedErr := fmt.Sprintf(transferInProgressErr, sti[0].tid)

	err := m.CloseSend(sti[0].tid)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("CloseSend did not return the expected error when the transfer"+
			"is not complete.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that Manager.Receive returns an error when no transfer with
// the ID exists.
func TestManager_Receive_NoTransferError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	tid := ftCrypto.UnmarshalTransferID([]byte("invalidID"))

	_, err := m.Receive(tid)
	if err == nil {
		t.Error("Receive did not return an error when no transfer with the ID " +
			"exists.")
	}
}

// Error path: tests that Manager.Receive returns an error when the file is
// incomplete.
func TestManager_Receive_GetFileError(t *testing.T) {
	m, _, rti := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, false, nil, nil, nil, t)

	_, err := m.Receive(rti[0].tid)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Error("Receive did not return the expected error when no transfer " +
			"with the ID exists.")
	}
}

// Tests that Manager.RegisterReceivedProgressCallback calls the callback when
// it is added to the transfer and that the callback is associated with the
// expected transfer and is called when calling from the transfer.
func TestManager_RegisterReceivedProgressCallback(t *testing.T) {
	m, _, rti := newTestManagerWithTransfers(
		[]uint16{12, 4, 1}, false, false, nil, nil, nil, t)
	expectedErr := errors.New("CallbackError")

	// Create new callback and channel for the callback to trigger
	cbChan := make(chan receivedProgressResults, 6)
	cb := func(completed bool, received, total uint16,
		tr interfaces.FilePartTracker, err error) {
		cbChan <- receivedProgressResults{completed, received, total, tr, err}
	}

	// Start thread waiting for callback to be called
	done0, done1 := make(chan bool), make(chan bool)
	go func() {
		for i := 0; i < 2; i++ {
			select {
			case <-time.NewTimer(20 * time.Millisecond).C:
				t.Errorf("Timed out waiting for callback call #%d.", i)
			case r := <-cbChan:
				switch i {
				case 0:
					err := checkReceivedProgress(r.completed, r.received,
						r.total, false, 0, rti[0].numParts)
					if err != nil {
						t.Errorf("%d: %+v", i, err)
					}
					if r.err != nil {
						t.Errorf("Callback returned an error (%d): %+v", i, r.err)
					}
					done0 <- true
				case 1:
					if r.err == nil || r.err != expectedErr {
						t.Errorf("Callback did not return the expected error (%d)."+
							"\nexpected: %v\nreceived: %+v", i, expectedErr, r.err)
					}
					done1 <- true
				}
			}
		}
	}()

	err := m.RegisterReceivedProgressCallback(rti[0].tid, cb, time.Millisecond)
	if err != nil {
		t.Errorf("RegisterReceivedProgressCallback returned an error: %+v", err)
	}
	<-done0

	transfer, _ := m.received.GetTransfer(rti[0].tid)

	transfer.CallProgressCB(expectedErr)

	<-done1
}

// Error path: tests that Manager.RegisterReceivedProgressCallback returns an
// error when no transfer with the ID exists.
func TestManager_RegisterReceivedProgressCallback_NoTransferError(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)
	tid := ftCrypto.UnmarshalTransferID([]byte("invalidID"))

	err := m.RegisterReceivedProgressCallback(tid, nil, 0)
	if err == nil {
		t.Error("RegisterReceivedProgressCallback did not return an error " +
			"when no transfer with the ID exists.")
	}
}

// Tests that calcNumberOfFingerprints matches some manually calculated
// results.
func Test_calcNumberOfFingerprints(t *testing.T) {
	testValues := []struct {
		numParts uint16
		retry    float32
		result   uint16
	}{
		{12, 0.5, 18},
		{13, 0.6667, 21},
		{1, 0.89, 1},
		{2, 0.75, 3},
		{119, 0.45, 172},
	}

	for i, val := range testValues {
		result := calcNumberOfFingerprints(val.numParts, val.retry)

		if val.result != result {
			t.Errorf("calcNumberOfFingerprints(%3d, %3.2f) result is "+
				"incorrect (%d).\nexpected: %d\nreceived: %d",
				val.numParts, val.retry, i, val.result, result)
		}
	}
}

// Tests that Manager satisfies the interfaces.FileTransfer interface.
func TestManager_FileTransferInterface(t *testing.T) {
	var _ interfaces.FileTransfer = Manager{}
}

// Sets up a mock file transfer with two managers that sends one file from one
// to another.
func Test_FileTransfer(t *testing.T) {
	var wg sync.WaitGroup

	// Create callback with channel for receiving new file transfer
	receiveNewCbChan := make(chan receivedFtResults, 100)
	receiveNewCB := func(tid ftCrypto.TransferID, fileName, fileType string,
		sender *id.ID, size uint32, preview []byte) {
		receiveNewCbChan <- receivedFtResults{
			tid, fileName, fileType, sender, size, preview}
	}

	// Create reception channels for both managers
	newFtChan1 := make(chan message.Receive, rawMessageBuffSize)
	filePartChan1 := make(chan message.Receive, rawMessageBuffSize)
	newFtChan2 := make(chan message.Receive, rawMessageBuffSize)
	filePartChan2 := make(chan message.Receive, rawMessageBuffSize)

	// Generate sending and receiving managers
	m1 := newTestManager(false, filePartChan2, newFtChan2, nil, nil, t)
	m2 := newTestManager(false, filePartChan1, newFtChan1, receiveNewCB, nil, t)

	// Add partner
	dhKey := m1.store.E2e().GetGroup().NewInt(42)
	pubKey := diffieHellman.GeneratePublicKey(dhKey, m1.store.E2e().GetGroup())
	p := params.GetDefaultE2ESessionParams()
	recipient := id.NewIdFromString("recipient", id.User, t)

	rng := csprng.NewSystemRNG()
	_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA,
		rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(
		sidh.KeyVariantSidhB, rng)

	err := m1.store.E2e().AddPartner(recipient, pubKey, dhKey,
		mySidhPriv, theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipient, err)
	}

	stop1, err := m1.startProcesses(newFtChan1, filePartChan1)
	if err != nil {
		t.Errorf("Failed to start processes for sending manager: %+v", err)
	}

	stop2, err := m2.startProcesses(newFtChan2, filePartChan2)
	if err != nil {
		t.Errorf("Failed to start processes for receving manager: %+v", err)
	}

	// Create progress tracker for sending
	sentCbChan := make(chan sentProgressResults, 20)
	sentCb := func(completed bool, sent, arrived, total uint16,
		tr interfaces.FilePartTracker, err error) {
		sentCbChan <- sentProgressResults{completed, sent, arrived, total, tr, err}
	}

	// Start threads that tracks sent progress until complete
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			select {
			case <-time.NewTimer(250 * time.Millisecond).C:
				t.Errorf("Timed out waiting for sent progress callback %d.", i)
			case r := <-sentCbChan:
				if r.completed {
					return
				}
			}
		}
		t.Error("Sent progress callback never reported file finishing to send.")
	}()

	// Create file and parameters
	prng := NewPrng(42)
	partSize, _ := m1.getPartSize()
	fileName := "testFile"
	fileType := "file"
	file, parts := newFile(32, partSize, prng, t)
	preview := parts[0]

	// Send file
	sendTid, err := m1.Send(fileName, fileType, file, recipient, 0.5, preview,
		sentCb, time.Millisecond)
	if err != nil {
		t.Errorf("Send returned an error: %+v", err)
	}

	// Wait for the receiving manager to get E2E message and call callback to
	// get transfer ID of received transfer
	var receiveTid ftCrypto.TransferID
	select {
	case <-time.NewTimer(10 * time.Millisecond).C:
		t.Error("Timed out waiting for receive callback: ")
	case r := <-receiveNewCbChan:
		if !bytes.Equal(preview, r.preview) {
			t.Errorf("File preview received from callback incorrect."+
				"\nexpected: %q\nreceived: %q", preview, r.preview)
		}
		if len(file) != int(r.size) {
			t.Errorf("File size received from callback incorrect."+
				"\nexpected: %d\nreceived: %d", len(file), r.size)
		}
		receiveTid = r.tid
	}

	// Register progress callback with receiving manager
	receiveCbChan := make(chan receivedProgressResults, 100)
	receiveCb := func(completed bool, received, total uint16,
		tr interfaces.FilePartTracker, err error) {
		receiveCbChan <- receivedProgressResults{completed, received, total, tr, err}
	}

	// Start threads that tracks received progress until complete
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			select {
			case <-time.NewTimer(450 * time.Millisecond).C:
				t.Errorf("Timed out waiting for receive progress callback %d.", i)
			case r := <-receiveCbChan:
				if r.completed {
					// Count the number of parts marked as received
					count := 0
					for j := uint16(0); j < r.total; j++ {
						if r.tracker.GetPartStatus(j) == interfaces.FpReceived {
							count++
						}
					}

					// Ensure that the number of parts received reported by the
					// callback matches the number marked received
					if count != int(r.received) {
						t.Errorf("Number of parts marked received does not "+
							"match number reported by callback."+
							"\nmarked:   %d\ncallback: %d", count, r.received)
					}

					return
				}
			}
		}
		t.Error("Receive progress callback never reported file finishing to receive.")
	}()

	err = m2.RegisterReceivedProgressCallback(
		receiveTid, receiveCb, time.Millisecond)
	if err != nil {
		t.Errorf("Failed to register receive progress callback: %+v", err)
	}

	wg.Wait()

	// Check that the file can be received
	receivedFile, err := m2.Receive(receiveTid)
	if err != nil {
		t.Errorf("Failed to receive file: %+v", err)
	}

	// Check that the received file matches the sent file
	if !bytes.Equal(file, receivedFile) {
		t.Errorf("Received file does not match sent."+
			"\nexpected: %q\nrecevied: %q", file, receivedFile)
	}

	// Check that the received transfer was deleted
	_, err = m2.received.GetTransfer(receiveTid)
	if err == nil {
		t.Error("Failed to delete received file transfer once file has been " +
			"received.")
	}

	// Close the transfer on the sending manager
	err = m1.CloseSend(sendTid)
	if err != nil {
		t.Errorf("Failed to close the send: %+v", err)
	}

	// Check that the sent transfer was deleted
	_, err = m1.sent.GetTransfer(sendTid)
	if err == nil {
		t.Error("Failed to delete sent file transfer once file has been sent " +
			"and closed.")
	}

	if err = stop1.Close(); err != nil {
		t.Errorf("Failed to close sending manager threads: %+v", err)
	}

	if err = stop2.Close(); err != nil {
		t.Errorf("Failed to close receiving manager threads: %+v", err)
	}
}
