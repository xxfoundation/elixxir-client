////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

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

// Error path: Tests that UnmarshalFileInfo returns the expected error when the
// data is an invalid proto message.
func TestUnmarshalFileInfo_ProtoUnmarshalError(t *testing.T) {
	data := []byte("invalid data")
	expectedErr := fmt.Sprintf(fileInfoMsgProtoUnmarshalErr, &FileInfoMsg{})

	_, err := UnmarshalFileInfo(data)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Unexpcted error for invalid proto message."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
	}
}

// Error path: Tests that UnmarshalFileInfo returns the expected error when the
// file ID cannot be unmarshalled.
func TestUnmarshalFileInfo_FileIdUnmarshalError(t *testing.T) {
	data, err := proto.Marshal(&FileInfoMsg{Fid: []byte("invalid ID")})
	if err != nil {
		t.Fatalf("Failed to proto marshal FileInfoMsg: %+v", err)
	}

	expectedErr := fileInfoFileIdUnmarshalErr

	_, err = UnmarshalFileInfo(data)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Unexpcted error for invalid proto message."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
	}
}

// Error path: Tests that UnmarshalFileInfo returns the expected error when the
// recipient ID cannot be unmarshalled.
func TestUnmarshalFileInfo_RecipientUnmarshalError(t *testing.T) {
	data, err := proto.Marshal(&FileInfoMsg{
		Fid: make([]byte, ftCrypto.IdLen), RecipientID: []byte("invalid ID")})
	if err != nil {
		t.Fatalf("Failed to proto marshal FileInfoMsg: %+v", err)
	}

	expectedErr := fileInfoRecipientIdUnmarshalErr

	_, err = UnmarshalFileInfo(data)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Unexpcted error for invalid proto message."+
			"\nexpected: %s\nreceived: %v", expectedErr, err)
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
