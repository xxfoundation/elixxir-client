////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
	"testing"

	"gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Creates example JSON outputs used in documentation.
func TestFileTransfer_inputs(t *testing.T) {
	// ReceivedFile
	tid, _ := fileTransfer.NewTransferID(csprng.NewSystemRNG())
	sid, _ := id.NewRandomID(csprng.NewSystemRNG(), id.User)
	rf := &ReceivedFile{
		TransferID: &tid,
		SenderID:   sid,
		Preview:    []byte("it's me a preview"),
		Name:       "testfile.txt",
		Type:       "text file",
		Size:       2048,
	}
	rfm, _ := json.MarshalIndent(rf, "", "  ")
	t.Log("ReceivedFile example JSON:")
	fmt.Printf("%s\n\n", rfm)

	// FileSend
	fs := &FileSend{
		Name:     "testFile",
		Type:     "txt",
		Preview:  []byte("File preview."),
		Contents: []byte("File contents."),
	}
	fsm, _ := json.MarshalIndent(fs, "", "  ")
	t.Log("FileSend example JSON:")
	fmt.Printf("%s\n\n", fsm)

	// Progress
	p := &Progress{
		TransferID:  &tid,
		Completed:   false,
		Transmitted: 128,
		Total:       2048,
	}
	pm, _ := json.MarshalIndent(p, "", "  ")
	t.Log("Progress example JSON:")
	fmt.Printf("%s\n\n", pm)

	// EventReport
	er := &EventReport{
		Priority:  1,
		Category:  "Test Events",
		EventType: "Ping",
		Details:   "This is an example of an event report",
	}
	erm, _ := json.MarshalIndent(er, "", "  ")
	t.Log("EventReport example JSON:")
	fmt.Printf("%s\n\n", erm)
}
