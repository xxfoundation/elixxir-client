////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/xxdk"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Smoke test of the entire file transfer system.
func Test_FileTransfer_Smoke(t *testing.T) {

	////////////////////////////////////////////////////////////////////////////
	// Set Up                                                                 //
	////////////////////////////////////////////////////////////////////////////
	jww.SetStdoutThreshold(jww.LevelDebug)
	timeout := 15 * time.Second
	cMixHandler := newMockCmixHandler()
	prng := rand.New(rand.NewSource(1978))
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	params := DefaultParams()
	params.ResendWait = 15 * time.Millisecond
	params.MaxThroughput = 0

	// Set up the first client
	myID1 := id.NewIdFromString("myID1", id.User, t)
	storage1 := newMockStorage()
	cMix1 := newMockCmix(myID1, cMixHandler, storage1)
	user1 := newMockE2e(myID1, cMix1, storage1, rngGen)

	w1, eb1, err := NewWrapper(user1, params)
	if err != nil {
		t.Fatalf("Failed to create new file transfer manager 1: %+v", err)
	}

	me1, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("Failed to create new private identity 1: %+v", err)
	}

	ch1, err := newMockChannelsManager(me1)
	if err != nil {
		t.Fatalf("Failed to create new mock channel manager 1: %+v", err)
	}

	evFileCh1 := make(chan ModelFile, 100)
	evMsgCh1 := make(chan channels.ModelMessage, 100)
	ev1 := newMockEventModel(func(msg ModelFile) { evFileCh1 <- msg },
		func(msg channels.ModelMessage) { evMsgCh1 <- msg }, t)

	// Set up the second client
	myID2 := id.NewIdFromString("myID2", id.User, t)
	storage2 := newMockStorage()
	cMix2 := newMockCmix(myID2, cMixHandler, storage2)
	user2 := newMockE2e(myID2, cMix2, storage2, rngGen)

	w2, eb2, err := NewWrapper(user2, params)
	if err != nil {
		t.Fatalf("Failed to create new file transfer manager 2: %+v", err)
	}

	me2, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatalf("Failed to create new private identity 2: %+v", err)
	}

	ch2, err := newMockChannelsManager(me2)
	if err != nil {
		t.Fatalf("Failed to create new mock channel manager 2: %+v", err)
	}

	evFileCh2 := make(chan ModelFile, 100)
	evMsgCh2 := make(chan channels.ModelMessage, 100)
	ev2 := newMockEventModel(func(msg ModelFile) { evFileCh2 <- msg },
		func(msg channels.ModelMessage) { evMsgCh2 <- msg }, t)

	emh1, err := eb1(ev1, ch1, me1)
	if err != nil {
		t.Fatal(err)
	}
	emh2, err := eb2(ev2, ch2, me2)
	if err != nil {
		t.Fatal(err)
	}

	ch1.addEMH(emh1[0], emh2[0])
	ch2.addEMH(emh1[0], emh2[0])

	stop1, err := w1.StartProcesses()
	if err != nil {
		t.Fatalf("Failed to start processes for manager 1: %+v", err)
	}

	stop2, err := w2.StartProcesses()
	if err != nil {
		t.Fatalf("Failed to start processes for manager 2: %+v", err)
	}

	////////////////////////////////////////////////////////////////////////////
	// Upload/Download New File                                               //
	////////////////////////////////////////////////////////////////////////////

	// Define details of file to send
	channelID := id.NewIdFromString("channel", id.User, t)
	fileName, fileType := "myFile", "txt"
	fileData := []byte(loremIpsum)
	preview := []byte("Lorem ipsum dolor sit amet")
	retry := float32(2.0)

	uploadCh := make(chan ftCrypto.ID, 10)
	uploadCB := func(completed bool, s, r, tt uint16, st SentTransfer, _ FilePartTracker, err error) {
		if err != nil {
			t.Fatalf("File transfer error: %+v", err)
		} else if completed {
			uploadCh <- st.GetFileID()
		}
	}

	go func() {
		select {
		case <-uploadCh:
		case <-time.After(500 * time.Millisecond):
			t.Errorf("Timed out waiting for callback to complete.")
		}
	}()

	// Upload file
	fid, err := w1.Upload(fileData, retry, uploadCB, 0)
	if err != nil {
		t.Fatalf("Failed to upload file: %+v", err)
	}

	// Check that file is added to the event model with expected values
	select {
	case f := <-evFileCh1:
		expected := ModelFile{
			FileID:    fid,
			FileLink:  nil,
			FileData:  fileData,
			Timestamp: f.Timestamp,
			Status:    Uploading,
		}
		if !reflect.DeepEqual(f, expected) {
			t.Errorf("Unexpected data stored in event model."+
				"\nexpected: %+v\nreceived: %+v", expected, f)
		}
	case <-time.After(timeout):
		t.Fatalf("Timed out after %s waiting for upload to begin.", timeout)
	}

	// Check that, once the upload is complete, the status is correctly changed,
	// the file info is added to the event model, and the transfer was deleted
	// from the sent transfers in the file manager
	var fileLink []byte
	select {
	case f := <-evFileCh1:
		if f.Status != Complete {
			t.Errorf("Uploaded file not marked as complete."+
				"\nexpected: %s\nreceived: %s", Complete, f.Status)
		} else if f.FileLink == nil {
			t.Errorf("File link not set: %v", f.FileLink)
		} else if st, exists := w1.m.sent.GetTransfer(f.FileID); exists {
			t.Errorf("Transfer not removed from sent transfers: %+v", st)
		}
		fileLink = f.FileLink
	case <-time.After(timeout):
		t.Fatalf("Timed out after %s waiting for file to upload.", timeout)
	}

	// Send the file to the channel
	msgID, _, _, err := w1.Send(channelID, fileLink, fileName, fileType, preview, 0,
		xxdk.GetDefaultCMixParams())
	if err != nil {
		t.Fatalf("Failed to send file: %+v", err)
	}

	var fl FileLink
	if err = json.Unmarshal(fileLink, &fl); err != nil {
		t.Fatal(err)
	}

	expectedFI := FileInfo{fileName, fileType, preview, fl}
	expectedContent, err := json.Marshal(expectedFI)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the file info is added to the event model
	var fileInfo []byte
	select {
	case f := <-evMsgCh2:
		expected := channels.ModelMessage{
			MessageID: msgID,
			ChannelID: channelID,
			Timestamp: f.Timestamp,
			Lease:     0,
			Status:    channels.Delivered,
			Content:   expectedContent,
			Type:      channels.FileTransfer,
			PubKey:    me1.PubKey,
			DmToken:   me1.GetDMToken(),
		}
		if !reflect.DeepEqual(f, expected) {
			t.Errorf("Unexpected data stored in event model."+
				"\nexpected: %+v\nreceived: %+v", expected, f)
		}
		fileInfo = f.Content
	case <-time.After(timeout):
		t.Fatalf("Timed out after %s waiting for file info to be added to the "+
			"event model.", timeout)
	}

	// Download the file
	_, err = w2.Download(fileInfo, nil, 0)
	if err != nil {
		t.Fatalf("Failed to download file: %+v", err)
	}

	// Check that the download has started
	select {
	case f := <-evFileCh2:
		if f.Status != Downloading {
			t.Errorf("Download file not marked as started."+
				"\nexpected: %s\nreceived: %s", Downloading, f.Status)
		}
	case <-time.After(timeout):
		t.Fatalf("Timed out after %s waiting for download to start.", timeout)
	}

	// Check that the completed file is added to the event model
	select {
	case f := <-evFileCh2:
		expected := ModelFile{
			FileID:    fid,
			FileLink:  fileLink,
			FileData:  fileData,
			Timestamp: f.Timestamp,
			Status:    Complete,
		}
		if !reflect.DeepEqual(f, expected) {
			t.Errorf("Unexpected data stored in event model."+
				"\nexpected: %+v\nreceived: %+v", expected, f)
		}
		if rt, exists := w2.m.received.GetTransfer(f.FileID); exists {
			t.Errorf("Transfer not removed from received transfers: %+v", rt)
		}

	case <-time.After(timeout):
		t.Fatalf("Timed out after %s waiting for file to download.", timeout)
	}

	////////////////////////////////////////////////////////////////////////////
	// Resume partially uploaded file                                         //
	////////////////////////////////////////////////////////////////////////////

	// Define details of file to send
	fileData2 := jellyBeans
	retry2 := float32(1.5)
	fileName2 := "jellyBeans"
	fileType2 := "png"
	preview2 := []byte("Some jelly beans")

	// Upload file and then stop in the middle
	fid, err = w1.Upload(fileData2, retry2, nil, 0)
	if err != nil {
		t.Fatalf("Failed to upload file: %+v", err)
	}

	time.Sleep(5 * time.Millisecond)

	err = stop1.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 1: %+v", err)
	}

	err = stop2.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 2: %+v", err)
	}

	// Set up the first client
	w1, eb1, err = NewWrapper(user1, params)
	if err != nil {
		t.Fatalf("Failed to create new file transfer manager 1: %+v", err)
	}

	// Set up the second client
	w2, eb2, err = NewWrapper(user2, params)
	if err != nil {
		t.Fatalf("Failed to create new file transfer manager 2: %+v", err)
	}

	emh1, err = eb1(ev1, ch1, me1)
	if err != nil {
		t.Fatal(err)
	}
	emh2, err = eb2(ev2, ch2, me2)
	if err != nil {
		t.Fatal(err)
	}

	ch1.addEMH(emh1[0], emh2[0])
	ch2.addEMH(emh1[0], emh2[0])

	stop1, err = w1.StartProcesses()
	if err != nil {
		t.Fatalf("Failed to start processes for manager 1: %+v", err)
	}

	stop2, err = w2.StartProcesses()
	if err != nil {
		t.Fatalf("Failed to start processes for manager 2: %+v", err)
	}

	cbChan := make(chan ftCrypto.ID, 2)
	cb := func(completed bool, s, r, tt uint16, st SentTransfer,
		_ FilePartTracker, _ error) {
		// t.Logf("completed:%t sent:%d received:%d total:%d  %s", completed, s, r, tt, st.GetFileID())
		if completed {
			cbChan <- st.GetFileID()
		}
	}

	// Upload the new file again to resume the upload
	fid, err = w1.Upload(fileData2, 5, cb, 0)
	if err != nil {
		t.Fatalf("Failed to upload completed file: %+v", err)
	}

	select {
	case receivedFID := <-cbChan:
		if fid != receivedFID {
			t.Errorf("Callback called for wrong file ID."+
				"\nexpected: %s\nreceivesd: %s", fid, receivedFID)
		}
	case <-time.After(350 * time.Millisecond):
		t.Errorf("Timed out waiting for callback to be called indicating " +
			"that the file transfer is complete.")
	}

	var uploadedFileData2 []byte
	for uploadedFileData2 == nil {
		select {
		case f := <-evFileCh1:
			if f.FileID == fid && f.Status == Complete {
				uploadedFileData2 = f.FileData
			}
		case <-time.After(15 * time.Millisecond):
			t.Fatalf("Timed out waiting to receive uplaoded file data.")
		}
	}
	if !bytes.Equal(uploadedFileData2, fileData2) {
		t.Errorf("Event model has incorrect file data for %s."+
			"\nexpected: %q\nreceived: %q",
			fid, fileData2[:24], uploadedFileData2[:24])
	}

	// Upload the same initial file again to verify the callback returns
	// completed immediately
	cbChan2 := make(chan ftCrypto.ID, 2)
	cb2 := func(completed bool, _, _, _ uint16, st SentTransfer,
		_ FilePartTracker, _ error) {
		if completed {
			cbChan2 <- st.GetFileID()
		}
	}
	fid, err = w1.Upload(fileData, 5, cb2, 0)
	if err != nil {
		t.Fatalf("Failed to upload completed file: %+v", err)
	}

	select {
	case receivedFID := <-cbChan2:
		if fid != receivedFID {
			t.Errorf("Callback called for wrong file ID."+
				"\nexpected: %s\nreceived: %s", fid, receivedFID)
		}
	case <-time.After(15 * time.Millisecond):
		t.Errorf("Timed out waiting for callback to be called indicating " +
			"that the file transfer is complete.")
	}

	////////////////////////////////////////////////////////////////////////////
	// Resume partially downloaded file                                       //
	////////////////////////////////////////////////////////////////////////////

	fileLink2 := ev1.files[fid].FileLink
	msgID, _, _, err = w1.Send(channelID, fileLink2, fileName2, fileType2,
		preview2, 0, xxdk.GetDefaultCMixParams())
	if err != nil {
		t.Fatal(err)
	}

	var fileInfo2 []byte
	for fileInfo2 == nil {
		select {
		case m := <-evMsgCh2:
			t.Logf("%q", m.Content)
			if m.MessageID == msgID {
				fileInfo2 = m.Content
			}
		case <-time.After(15 * time.Millisecond):
			t.Fatalf("Timed out waiting to receive file info.")
		}
	}

	// Download the file
	_, err = w2.Download(fileInfo2, nil, 0)
	if err != nil {
		t.Fatalf("Failed to download file: %+v", err)
	}

	err = stop1.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 1: %+v", err)
	}

	err = stop2.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 2: %+v", err)
	}

	// Set up the first client
	w1, eb1, err = NewWrapper(user1, params)
	if err != nil {
		t.Fatalf("Failed to create new file transfer manager 1: %+v", err)
	}

	// Set up the second client
	w2, eb2, err = NewWrapper(user2, params)
	if err != nil {
		t.Fatalf("Failed to create new file transfer manager 2: %+v", err)
	}

	emh1, err = eb1(ev1, ch1, me1)
	if err != nil {
		t.Fatal(err)
	}
	emh2, err = eb2(ev2, ch2, me2)
	if err != nil {
		t.Fatal(err)
	}

	ch1.addEMH(emh1[0], emh2[0])
	ch2.addEMH(emh1[0], emh2[0])

	stop1, err = w1.StartProcesses()
	if err != nil {
		t.Fatalf("Failed to start processes for manager 1: %+v", err)
	}

	stop2, err = w2.StartProcesses()
	if err != nil {
		t.Fatalf("Failed to start processes for manager 2: %+v", err)
	}

	// Download the same file again
	downloadCh := make(chan ftCrypto.ID, 10)
	downloadCB := func(completed bool, r, tt uint16, rt ReceivedTransfer,
		_ FilePartTracker, err error) {
		t.Logf("completed:%t received:%d total:%d  %s", completed, r, tt, rt.GetFileID())
		if completed {
			downloadCh <- rt.GetFileID()
		}
	}

	fid, err = w2.Download(fileInfo2, downloadCB, 0)
	if err != nil {
		t.Fatalf("Failed to download file: %+v", err)
	}

	select {
	case receivedFID := <-downloadCh:
		if fid != receivedFID {
			t.Errorf("Callback called for wrong file ID."+
				"\nexpected: %s\nreceived: %s", fid, receivedFID)
		}
	case <-time.After(15 * time.Millisecond):
		t.Errorf("Timed out waiting for callback to be called indicating " +
			"that the file transfer is complete.")
	}

	// Download the same initial file again to verify the callback returns
	// completed immediately
	downloadCh = make(chan ftCrypto.ID, 10)
	downloadCB = func(completed bool, r, tt uint16, rt ReceivedTransfer,
		_ FilePartTracker, err error) {
		t.Logf("completed:%t received:%d total:%d  %s", completed, r, tt, rt.GetFileID())
		if completed {
			downloadCh <- rt.GetFileID()
		}
	}
	fid, err = w2.Download(fileInfo, downloadCB, 0)
	if err != nil {
		t.Fatalf("Failed to download completed file: %+v", err)
	}

	select {
	case receivedFID := <-downloadCh:
		if fid != receivedFID {
			t.Errorf("Callback called for wrong file ID."+
				"\nexpected: %s\nreceivesd: %s", fid, receivedFID)
		}
	case <-time.After(15 * time.Millisecond):
		t.Errorf("Timed out waiting for callback to be called indicating " +
			"that the file transfer is complete.")
	}
}