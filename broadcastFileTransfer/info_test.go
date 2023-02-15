////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"encoding/json"
	"reflect"
	"testing"

	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// Tests that a FileInfo marshalled via FileInfo.Marshal and unmarshalled via
// UnmarshalFileInfo matches the original.
func TestFileInfo_Marshal_UnmarshalFileInfo(t *testing.T) {
	fi := &FileInfo{
		FID:         ftCrypto.NewID([]byte("fileData")),
		RecipientID: id.NewIdFromString("recipient", id.User, t),
		FileName:    "FileName",
		FileType:    "FileType",
		Key:         ftCrypto.TransferKey{1, 2, 3},
		Mac:         []byte("I am a MAC"),
		NumParts:    6,
		Size:        250,
		Retry:       2.6,
		Preview:     []byte("I am a preview"),
	}

	data, err := fi.Marshal()
	if err != nil {
		t.Errorf("Failed to marshal FileInfo: %+v", err)
	}

	newTi, err := UnmarshalFileInfo(data)
	if err != nil {
		t.Errorf("Failed to unmarshal FileInfo: %+v", err)
	}

	if !reflect.DeepEqual(fi, newTi) {
		t.Errorf("Unmarshalled FileInfo does not match original."+
			"\nexpected: %+v\nreceived: %+v", fi, newTi)
	}
}

// Tests that a FileInfo JSON marshalled and unmarshalled matches the original.
func TestFileInfo_JSON_Marshal_Unmarshal(t *testing.T) {
	fi := &FileInfo{
		FID:      ftCrypto.NewID([]byte("fileData")),
		FileName: "FileName",
		FileType: "FileType",
		Key:      ftCrypto.TransferKey{1, 2, 3},
		Mac:      []byte("I am a MAC"),
		NumParts: 6,
		Size:     250,
		Retry:    2.6,
		Preview:  []byte("I am a preview"),
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
