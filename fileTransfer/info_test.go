////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"reflect"
	"testing"
)

// Tests that a TransferInfo marshalled via TransferInfo.Marshal and
// unmarshalled via UnmarshalTransferInfo matches the original.
func TestTransferInfo_Marshal_UnmarshalTransferInfo(t *testing.T) {
	ti := &TransferInfo{
		FileName: "FileName",
		FileType: "FileType",
		Key:      ftCrypto.TransferKey{1, 2, 3},
		Mac:      []byte("I am a MAC"),
		NumParts: 6,
		Size:     250,
		Retry:    2.6,
		Preview:  []byte("I am a preview"),
	}

	data, err := ti.Marshal()
	if err != nil {
		t.Errorf("Failed to marshal TransferInfo: %+v", err)
	}

	newTi, err := UnmarshalTransferInfo(data)
	if err != nil {
		t.Errorf("Failed to unmarshal TransferInfo: %+v", err)
	}

	if !reflect.DeepEqual(ti, newTi) {
		t.Errorf("Unmarshalled TransferInfo does not match original."+
			"\nexpected: %+v\nreceived: %+v", ti, newTi)
	}
}
