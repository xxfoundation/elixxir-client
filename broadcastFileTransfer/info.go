////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	fileInfoMsgProtoUnmarshalErr    = "error proto unmarshalling %T"
	fileInfoFileIdUnmarshalErr      = "error unmarshalling file ID"
	fileInfoRecipientIdUnmarshalErr = "error unmarshalling recipient ID"
)

// FileInfo contains all the information for a new transfer. This is the
// information sent in the initial file transfer so the recipient can prepare
// for the incoming file transfer parts.
type FileInfo struct {
	// FID is the file's ID.
	FID ftCrypto.ID `json:"fid"`

	// RecipientID is the ID to listen on to receive file.
	RecipientID *id.ID `json:"recipientID"`

	// FileName is the name of the file.
	FileName string `json:"fileName"`

	// FileType indicates what type of file is being transferred.
	FileType string `json:"fileType"`

	// Key is the 256-bit encryption key.
	Key ftCrypto.TransferKey `json:"key"`

	// Mac is the message MAC.
	Mac []byte `json:"mac"`

	// NumParts is the number of file parts being transferred.
	NumParts uint16 `json:"numParts"`

	// Size is the file size in bytes.
	Size uint32 `json:"size"`

	// Retry determines number of resends allowed on failure.
	Retry float32 `json:"retry"`

	// Preview contains a preview of the file.
	Preview []byte `json:"preview"`
}

// Marshal serialises the FileInfo for sending over the network.
func (fi *FileInfo) Marshal() ([]byte, error) {
	// Construct NewFileTransfer message
	protoMsg := &FileInfoMsg{
		Fid:         fi.FID.Marshal(),
		RecipientID: fi.RecipientID.Marshal(),
		FileName:    fi.FileName,
		FileType:    fi.FileType,
		TransferKey: fi.Key.Bytes(),
		TransferMac: fi.Mac,
		NumParts:    uint32(fi.NumParts),
		Size:        fi.Size,
		Retry:       fi.Retry,
		Preview:     fi.Preview,
	}

	return proto.Marshal(protoMsg)
}

// UnmarshalFileInfo deserializes the FileInfo.
func UnmarshalFileInfo(data []byte) (*FileInfo, error) {
	// Unmarshal the request message
	var fi FileInfoMsg
	err := proto.Unmarshal(data, &fi)
	if err != nil {
		return nil, errors.Wrapf(err, fileInfoMsgProtoUnmarshalErr, &fi)
	}

	fid, err := ftCrypto.UnmarshalID(fi.Fid)
	if err != nil {
		return nil, errors.Wrap(err, fileInfoFileIdUnmarshalErr)
	}

	recipientID, err := id.Unmarshal(fi.RecipientID)
	if err != nil {
		return nil, errors.Wrap(err, fileInfoRecipientIdUnmarshalErr)
	}

	return &FileInfo{
		FID:         fid,
		RecipientID: recipientID,
		FileName:    fi.FileName,
		FileType:    fi.FileType,
		Key:         ftCrypto.UnmarshalTransferKey(fi.GetTransferKey()),
		Mac:         fi.TransferMac,
		NumParts:    uint16(fi.NumParts),
		Size:        fi.Size,
		Retry:       fi.Retry,
		Preview:     fi.Preview,
	}, nil
}
