////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"github.com/golang/protobuf/proto"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

// TransferInfo contains all the information for a new transfer. This is the
// information sent in the initial file transfer so the recipient can prepare
// for the incoming file transfer parts.
type TransferInfo struct {
	FileName string               // Name of the file
	FileType string               // String that indicates type of file
	Key      ftCrypto.TransferKey // 256-bit encryption key
	Mac      []byte               // 256-bit MAC of the entire file
	NumParts uint16               // Number of file parts
	Size     uint32               // The size of the file, in bytes
	Retry    float32              // Determines how many times to retry sending
	Preview  []byte               // A preview of the file
}

// Marshal serialises the TransferInfo for sending over the network.
func (ti *TransferInfo) Marshal() ([]byte, error) {
	// Construct NewFileTransfer message
	protoMsg := &NewFileTransfer{
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
	var newFT NewFileTransfer
	err := proto.Unmarshal(data, &newFT)
	if err != nil {
		return nil, err
	}
	transferKey := ftCrypto.UnmarshalTransferKey(newFT.GetTransferKey())

	return &TransferInfo{
		FileName: newFT.FileName,
		FileType: newFT.FileType,
		Key:      transferKey,
		Mac:      newFT.TransferMac,
		NumParts: uint16(newFT.NumParts),
		Size:     newFT.Size,
		Retry:    newFT.Retry,
		Preview:  newFT.Preview,
	}, nil
}
