////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"fmt"
	"testing"
)

// Consistency test of MessageType.String.
func TestMessageType_String_Consistency(t *testing.T) {
	expectedStrings := map[MessageType]string{
		Text: "Text", AdminText: "AdminText", Reaction: "Reaction", Silent: "Silent", Invitation: "Invitation",
		Delete: "Delete", Pinned: "Pinned", Mute: "Mute",
		AdminReplay: "AdminReplay", FileTransfer: "FileTransfer",
		Invitation + 1: fmt.Sprintf("Unknown messageType %d", Invitation+1),
		Invitation + 2: fmt.Sprintf("Unknown messageType %d", Invitation+2),
	}

	for mt, expected := range expectedStrings {
		if mt.String() != expected {
			t.Errorf("Stringer failed on test.\nexpected: %s\nreceived: %s",
				expected, mt)
		}
	}
}

// Consistency test of MessageType.Bytes.
func TestMessageType_Bytes_Consistency(t *testing.T) {
	expectedBytes := [][]byte{{1, 0}, {2, 0}, {3, 0}}

	for i, expected := range expectedBytes {
		mt := MessageType(i + 1)
		if !bytes.Equal(mt.Bytes(), expected) {
			t.Errorf("Bytes failed on test %d.\nexpected: %v\nreceived: %v",
				i, expected, mt.Bytes())
		}
	}
}

// Tests that a MessageType marshalled via MessageType.Marshal and unmarshalled
// via UnmarshalMessageType matches the original.
func TestMessageType_Marshal_UnmarshalMessageType(t *testing.T) {
	tests := []MessageType{Text, AdminText, Reaction, Delete, Pinned, Mute,
		AdminReplay, FileTransfer}

	for _, mt := range tests {
		data := mt.Marshal()
		newMt := UnmarshalMessageType(data)

		if mt != newMt {
			t.Errorf("Failed to marshal and unmarshal MessageType %s."+
				"\nexpected: %d\nreceived: %d", mt, mt, newMt)
		}
	}
}
