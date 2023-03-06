////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"time"

	"gitlab.com/elixxir/client/v4/channels"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
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
func (fl *FileLink) Expired() bool {
	return netTime.Since(fl.SentTimestamp) >= channels.MessageLife
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
