////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"testing"
)

// Consistency test of MessageType.String.
func TestMessageType_String_Consistency(t *testing.T) {
	expectedStrings := map[MessageType]string{
		Text: "Text", AdminText: "AdminText", Reaction: "Reaction",
		Delete: "Delete", Pinned: "Pinned", Mute: "Mute",
		4: "Unknown messageType 4", 5: "Unknown messageType 5",
		6: "Unknown messageType 6", 7: "Unknown messageType 7",
		8: "Unknown messageType 8", 9: "Unknown messageType 9",
		10: "Unknown messageType 10",
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
	expectedBytes := [][]byte{{1, 0, 0, 0}, {2, 0, 0, 0}, {3, 0, 0, 0}}

	for i, expected := range expectedBytes {
		mt := MessageType(i + 1)
		if !bytes.Equal(mt.Bytes(), expected) {
			t.Errorf("Bytes failed on test %d.\nexpected: %v\nreceived: %v",
				i, expected, mt.Bytes())
		}
	}
}
