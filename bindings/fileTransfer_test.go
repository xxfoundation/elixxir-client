////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"testing"

	"gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

func TestFileTransfer_inputs(t *testing.T) {
	fs := &FileSend{
		Name:     "testfile.txt",
		Type:     "text file",
		Preview:  []byte("it's me a preview"),
		Contents: []byte("This is the full contents of the file in bytes"),
	}
	fsm, _ := json.Marshal(fs)
	t.Log("FileSend example json:")
	t.Log(string(fsm))
	t.Log("\n")

	tid, _ := fileTransfer.NewTransferID(csprng.NewSystemRNG())
	sid := id.NewIdFromString("zezima", id.User, t)
	rf := &ReceivedFile{
		TransferID: tid.Bytes(),
		SenderID:   sid.Marshal(),
		Preview:    []byte("it's me a preview"),
		Name:       "testfile.txt",
		Type:       "text file",
		Size:       2048,
	}
	rfm, _ := json.Marshal(rf)
	t.Log("ReceivedFile example json:")
	t.Log(string(rfm))
	t.Log("\n")

	p := &Progress{
		Completed:   false,
		Transmitted: 128,
		Total:       2048,
		Err:         nil,
	}
	pm, _ := json.Marshal(p)
	t.Log("Progress example json:")
	t.Log(string(pm))
	t.Log("\n")

	er := &EventReport{
		Priority:  1,
		Category:  "Test Events",
		EventType: "Ping",
		Details:   "This is an example of an event report",
	}
	erm, _ := json.Marshal(er)
	t.Log("EventReport example json:")
	t.Log(string(erm))
	t.Log("\n")
}
