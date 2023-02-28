////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Calculates the maximum size of a marshalled FileInfo without a preview and
// calculates the maximum size of a preview that can fit in a channel message.
func TestFileInfo_Size(t *testing.T) {
	prng := rand.New(rand.NewSource(4252634))

	key, err := ftCrypto.NewTransferKey(csprng.NewSystemRNG())
	if err != nil {
		t.Fatalf("Failed to generate new transfer key: %+v", err)
	}

	fi := &FileInfo{
		FileName: randStringBytes(FileNameMaxLen, prng),
		FileType: randStringBytes(FileTypeMaxLen, prng),
		Preview:  []byte{},
		FileLink: FileLink{
			FileID:        ftCrypto.NewID([]byte("fileData")),
			RecipientID:   id.NewIdFromString("recipient", id.User, t),
			SentTimestamp: netTime.Now(),
			Key:           key,
			Mac:           make([]byte, format.MacLen),
			NumParts:      math.MaxUint16,
			Size:          math.MaxUint32,
			Retry:         math.MaxFloat32,
		},
	}

	data, err := fi.Marshal()
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

// Tests that a FileInfo marshalled via FileInfo.Marshal and unmarshalled via
// UnmarshalFileInfo matches the original.
func TestFileInfo_Marshal_UnmarshalFileInfo(t *testing.T) {
	fi := &FileInfo{
		FileName: "FileName",
		FileType: "FileType",
		Preview:  []byte("I am a preview"),
		FileLink: FileLink{
			FileID:        ftCrypto.NewID([]byte("fileData")),
			RecipientID:   id.NewIdFromString("recipient", id.User, t),
			SentTimestamp: netTime.Now(),
			Key:           ftCrypto.TransferKey{1, 2, 3},
			Mac:           []byte("I am a MAC"),
			NumParts:      6,
			Size:          250,
			Retry:         2.6,
		},
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
		FileName: "FileName",
		FileType: "FileType",
		Preview:  []byte("I am a preview"),
		FileLink: FileLink{
			FileID:        ftCrypto.NewID([]byte("fileData")),
			RecipientID:   id.NewIdFromString("recipient", id.User, t),
			SentTimestamp: netTime.Now(),
			Key:           ftCrypto.TransferKey{1, 2, 3},
			Mac:           []byte("I am a MAC"),
			NumParts:      6,
			Size:          250,
			Retry:         2.6,
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
