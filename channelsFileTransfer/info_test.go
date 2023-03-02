////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"encoding/json"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"
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
		FileName: randStringBytes(fileNameMaxLen, prng),
		FileType: randStringBytes(fileTypeMaxLen, prng),
		Preview:  []byte{},
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

	t.Logf("%s", data)

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
		FileName: randStringBytes(fileNameMaxLen, prng),
		FileType: randStringBytes(fileTypeMaxLen, prng),
		Preview:  []byte("I am a preview"),
		FileLink: FileLink{
			FileID:        ftCrypto.NewID([]byte("fileData")),
			RecipientID:   id.NewIdFromString("recipient", id.User, t),
			SentTimestamp: netTime.Now().Round(0),
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
