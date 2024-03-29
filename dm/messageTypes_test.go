////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"fmt"
	"testing"
)

// Consistency test of MessageType.String.
func TestMessageType_String_Consistency(t *testing.T) {
	expectedStrings := map[MessageType]string{
		TextType: "Text", ReplyType: "Reply", ReactionType: "Reaction",
		SilentType: "Silent", InvitationType: "Invitation", DeleteType: "Delete",
		DeleteType + 1: fmt.Sprintf("Unknown messageType %d", DeleteType+1),
		DeleteType + 2: fmt.Sprintf("Unknown messageType %d", DeleteType+2),
	}

	for mt, expected := range expectedStrings {
		if mt.String() != expected {
			t.Errorf("Stringer failed on test.\nexpected: %s\nreceived: %s",
				expected, mt)
		}
	}
}

// Tests that a MessageType marshalled via MessageType.Marshal and unmarshalled
// via UnmarshalMessageType matches the original.
func TestMessageType_Marshal_UnmarshalMessageType(t *testing.T) {
	tests := []MessageType{
		TextType, ReplyType, ReactionType, SilentType, InvitationType, DeleteType}

	for _, mt := range tests {
		data := mt.Marshal()
		newMt := UnmarshalMessageType(data)

		if mt != newMt {
			t.Errorf("Failed to marshal and unmarshal MessageType %s."+
				"\nexpected: %d\nreceived: %d", mt, mt, newMt)
		}
	}
}
