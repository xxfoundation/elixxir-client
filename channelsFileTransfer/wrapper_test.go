////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/channels"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/xxdk"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Smoke test of the entire file transfer system.
func Test_FileTransfer_Smoke2(t *testing.T) {
	// jww.SetStdoutThreshold(jww.LevelDebug)
	timeout := 15 * time.Second
	cMixHandler := newMockCmixHandler()
	prng := rand.New(rand.NewSource(1978))
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	params := DefaultParams()
	params.ResendWait = 5 * time.Second

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

	// Define details of file to send
	channelID := id.NewIdFromString("channel", id.User, t)
	fileName, fileType := "myFile", "txt"
	fileData := []byte(loremIpsum)
	preview := []byte("Lorem ipsum dolor sit amet")
	retry := float32(2.0)

	// Upload file
	fid, err := w1.Upload(fileData, retry, nil, 0)
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
	_, _, _, err = w1.Send(channelID, fileLink, fileName, fileType, preview, 0,
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

	err = stop1.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 1: %+v", err)
	}

	err = stop2.Close()
	if err != nil {
		t.Errorf("Failed to close processes for manager 2: %+v", err)
	}
}
