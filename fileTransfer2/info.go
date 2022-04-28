////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"encoding/json"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

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

func (ti *TransferInfo) Marshal() ([]byte, error) {
	return json.Marshal(ti)
}

func UnmarshalTransferInfo(data []byte) (*TransferInfo, error) {
	var ti TransferInfo
	return &ti, json.Unmarshal(data, &ti)
}
