////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"github.com/golang/protobuf/proto"

	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// TransferInfo contains all the information for a new transfer. This is the
// information sent in the initial file transfer so the recipient can prepare
// for the incoming file transfer parts.
type TransferInfo struct {
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

// Marshal serialises the TransferInfo for sending over the network.
func (ti *TransferInfo) Marshal() ([]byte, error) {
	// Construct NewFileTransfer message
	protoMsg := &TransferInfoMsg{
		Fid:         ti.FID.Marshal(),
		RecipientID: ti.RecipientID.Marshal(),
		FileName:    ti.FileName,
		FileType:    ti.FileType,
		TransferKey: ti.Key.Bytes(),
		TransferMac: ti.Mac,
		NumParts:    uint32(ti.NumParts),
		Size:        ti.Size,
		Retry:       ti.Retry,
		Preview:     ti.Preview,
	}

	return proto.Marshal(protoMsg)
}

// UnmarshalTransferInfo deserializes the TransferInfo.
func UnmarshalTransferInfo(data []byte) (*TransferInfo, error) {
	// Unmarshal the request message
	var ti TransferInfoMsg
	err := proto.Unmarshal(data, &ti)
	if err != nil {
		return nil, err
	}

	fid, err := ftCrypto.UnmarshalID(ti.Fid)
	if err != nil {
		return nil, err
	}

	transferKey := ftCrypto.UnmarshalTransferKey(ti.GetTransferKey())

	recipientID, err := id.Unmarshal(ti.RecipientID)
	if err != nil {
		return nil, err
	}

	return &TransferInfo{
		FID:         fid,
		RecipientID: recipientID,
		FileName:    ti.FileName,
		FileType:    ti.FileType,
		Key:         transferKey,
		Mac:         ti.TransferMac,
		NumParts:    uint16(ti.NumParts),
		Size:        ti.Size,
		Retry:       ti.Retry,
		Preview:     ti.Preview,
	}, nil
}
