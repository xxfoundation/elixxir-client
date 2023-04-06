////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"encoding/json"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/channels"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Calculates the maximum size of a marshalled FileInfo without a preview and
// calculates the maximum size of a preview that can fit in a channel message.
func TestFileInfo_Size(t *testing.T) {
	prng := rand.New(rand.NewSource(4252634))

	key, err := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	if err != nil {
		t.Fatalf("Failed to generate new transfer key: %+v", err)
	}

	fi := FileInfo{
		Name:    randStringBytes(fileNameMaxLen, prng),
		Type:    randStringBytes(fileTypeMaxLen, prng),
		Preview: []byte{},
		FileLink: FileLink{
			FileID:        ftCrypto.NewID([]byte("fileData")),
			RecipientID:   id.NewIdFromString("recipient", id.User, t),
			SentTimestamp: time.Date(2006, 1, 2, 15, 04, 05, 999999999, time.FixedZone("", 7)),
			Key:           key,
			Mac:           make([]byte, 32),
			NumParts:      math.MaxUint16,
			Size:          math.MaxUint32,
			Retry:         math.MaxFloat32,
		},
	}

	data, err := json.Marshal(fi)
	if err != nil {
		t.Errorf("Failed to marshal FileInfo: %+v", err)
	}

	// The maximum payload size of a channel message according to
	// channels.Manager.SendGeneric
	const maxPayload = 802

	maxPreviewSize := maxPayload - len(data)
	t.Logf("FileInfo data size:\n%-30s %d\n%-30s %d\n%-30s %d\n",
		"Max Payload:", maxPayload,
		"Max data size without preview:", len(data),
		"Max preview size:", maxPreviewSize)
}

// Tests that a FileInfo JSON marshalled and unmarshalled matches the original.
func TestFileInfo_JSON_Marshal_Unmarshal(t *testing.T) {
	prng := rand.New(rand.NewSource(6415))
	fi := &FileInfo{
		Name:    randStringBytes(fileNameMaxLen, prng),
		Type:    randStringBytes(fileTypeMaxLen, prng),
		Preview: []byte("I am a preview"),
		FileLink: FileLink{
			FileID:        ftCrypto.NewID([]byte("fileData")),
			RecipientID:   id.NewIdFromString("recipient", id.User, t),
			SentTimestamp: netTime.Now().Round(0).UTC(),
			Key:           ftCrypto.TransferKey{1, 2, 3},
			Mac:           make([]byte, 32),
			NumParts:      math.MaxUint16,
			Size:          math.MaxUint32,
			Retry:         math.MaxFloat32,
		},
	}

	data, err := json.MarshalIndent(fi, "", "\t")
	if err != nil {
		t.Errorf("Failed to JSON marshal FileInfo: %+v", err)
	}

	var newTi FileInfo
	err = json.Unmarshal(data, &newTi)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal FileInfo: %+v", err)
	}

	if !reflect.DeepEqual(*fi, newTi) {
		t.Errorf("JSON marshalled and unmarshalled FileInfo does not match "+
			"original.\nexpected: %+v\nreceived: %+v", *fi, newTi)
	}
}

// Tests that FileLink.Expired returns true only for expired timestamps.
func TestFileLink_Expired(t *testing.T) {
	fl := FileLink{SentTimestamp: netTime.Now()}
	if fl.Expired() {
		t.Errorf("FileLink is not expired: %s", netTime.Since(fl.SentTimestamp))
	}

	fl = FileLink{SentTimestamp: netTime.Now().Add(-channels.MessageLife)}
	if !fl.Expired() {
		t.Errorf("FileLink is expired: %s", netTime.Since(fl.SentTimestamp))
	}
}

// Units test of FileLink.GetFileID.
func TestFileLink_GetFileID(t *testing.T) {
	fl := FileLink{FileID: ftCrypto.NewID([]byte("fileData"))}

	if fl.GetFileID() != fl.FileID {
		t.Errorf("Incorrect file ID.\nexpected: %s\nreceived: %s",
			fl.GetFileID(), fl.FileID)
	}
}

// Units test of FileLink.GetRecipient.
func TestFileLink_GetRecipient(t *testing.T) {
	fl := FileLink{RecipientID: id.NewIdFromString("recipient", id.User, t)}

	if fl.GetRecipient() != fl.RecipientID {
		t.Errorf("Incorrect recipient ID.\nexpected: %s\nreceived: %s",
			fl.GetRecipient(), fl.RecipientID)
	}
}

// Units test of FileLink.GetFileSize.
func TestFileLink_GetFileSize(t *testing.T) {
	fl := FileLink{Size: math.MaxUint32}

	if fl.GetFileSize() != fl.Size {
		t.Errorf("Incorrect recipient ID.\nexpected: %d\nreceived: %d",
			fl.GetFileSize(), fl.Size)
	}
}

// Units test of FileLink.GetNumParts.
func TestFileLink_GetNumParts(t *testing.T) {
	fl := FileLink{NumParts: math.MaxUint16}

	if fl.GetNumParts() != fl.NumParts {
		t.Errorf("Incorrect recipient ID.\nexpected: %d\nreceived: %d",
			fl.GetNumParts(), fl.NumParts)
	}
}
