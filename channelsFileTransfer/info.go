////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/xx_network/primitives/netTime"
	"time"

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
	// FileName is the name of the file.
	FileName string `json:"fileName"`

	// FileType indicates what type of file is being transferred.
	FileType string `json:"fileType"`

	// Preview contains a preview of the file.
	Preview []byte `json:"preview"`

	FileLink
}

// FileLink contains all the information required to download and decrypt a
// file.
type FileLink struct {
	// FileID is the file's ID.
	FileID ftCrypto.ID `json:"fileID"`

	// RecipientID is the ID to listen on to receive file.
	RecipientID *id.ID `json:"recipientID"`

	// SentTimestamp is the time when the file was first queued to send.
	SentTimestamp time.Time `json:"sentTimestamp"`

	// Key is the 256-bit encryption key.
	Key ftCrypto.TransferKey `json:"key"`

	// Mac is the transfer MAC (Message Authentication Code) for this file.
	Mac []byte `json:"mac"`

	// Size is the file size in bytes.
	Size uint32 `json:"size"`

	// NumParts is the number of file parts being transferred.
	NumParts uint16 `json:"numParts"`

	// Retry determines number of resends allowed on failure.
	Retry float32 `json:"retry"`
}

// Expired returns true if the file link is expired. A file link is expired when
// the sent timestamp (time that the first file part was sent) is greater than
// the max message life, meaning some or all the file parts no longer exist on
// the network for download.
// TODO: test
func (fl *FileLink) Expired() bool {
	return netTime.Since(fl.SentTimestamp) > channels.MessageLife
}

// GetFileID returns the file's ID.
func (fl *FileLink) GetFileID() ftCrypto.ID {
	return fl.FileID
}

// GetRecipient returns the recipient ID to download the file from.
func (fl *FileLink) GetRecipient() *id.ID {
	return fl.RecipientID
}

// GetFileSize returns the size of the entire file.
func (fl *FileLink) GetFileSize() uint32 {
	return fl.Size
}

// GetNumParts returns the total number of file parts in the transfer.
func (fl *FileLink) GetNumParts() uint16 {
	return fl.NumParts
}

// Marshal serialises the FileInfo for sending over the network.
func (fi *FileInfo) Marshal() ([]byte, error) {
	// Construct NewFileTransfer message
	protoMsg := &FileInfoMsg{
		FileName:      fi.FileName,
		FileType:      fi.FileType,
		Retry:         fi.Retry,
		Preview:       fi.Preview,
		Fid:           fi.FileID.Marshal(),
		RecipientID:   fi.RecipientID.Marshal(),
		SentTimestamp: fi.SentTimestamp.UnixNano(),
		TransferKey:   fi.Key.Bytes(),
		TransferMac:   fi.Mac,
		Size:          fi.Size,
		NumParts:      uint32(fi.NumParts),
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
		FileLink: FileLink{
			FileID:        fid,
			RecipientID:   recipientID,
			SentTimestamp: time.Unix(0, fi.SentTimestamp),
			Key:           ftCrypto.UnmarshalTransferKey(fi.GetTransferKey()),
			Mac:           fi.TransferMac,
			Size:          fi.Size,
			NumParts:      uint16(fi.NumParts),
			Retry:         fi.Retry,
		},
		FileName: fi.FileName,
		FileType: fi.FileType,
		Preview:  fi.Preview,
	}, nil
}
