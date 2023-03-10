////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/catalog"
)

// Expected firstMessagePart for checking against, generated by fmp in
// TestNewFirstMessagePart.
var expectedFMP = firstMessagePart{
	messagePart: messagePart{
		Data: []byte{0, 0, 4, 53, 0, 0, 13, 2, 0, 0, 0, 2, 22, 87, 28, 11, 215,
			220, 82, 0, 116, 101, 115, 116, 105, 110, 103, 115, 116, 114, 105,
			110, 103, 0, firstMessagePartCurrentVersion},
		Id:   []byte{0, 0, 4, 53},
		Part: []byte{0},
		Len:  []byte{0, 13},
		Contents: []byte{116, 101, 115, 116, 105, 110, 103, 115, 116, 114, 105,
			110, 103},
	},
	NumParts:  []byte{2},
	Type:      []byte{0, 0, 0, 2},
	Timestamp: []byte{22, 87, 28, 11, 215, 220, 82, 0},
	Version:   []byte{firstMessagePartCurrentVersion},
}

// Test that newFirstMessagePart returns a correctly made firstMessagePart.
func Test_newFirstMessagePart(t *testing.T) {
	fmp := newFirstMessagePart(
		catalog.XxMessage,
		1077,
		2,
		time.Unix(1609786229, 0).UTC(),
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g', 's', 't', 'r', 'i', 'n', 'g'}, len(expectedFMP.Data),
	)

	gotTime := fmp.getTimestamp()
	expectedTime := time.Unix(1609786229, 0).UTC()
	if !gotTime.Equal(expectedTime) {
		t.Errorf("Failed to get expected timestamp."+
			"\nexpected: %s\nreceived: %s", expectedTime, gotTime)
	}

	if !reflect.DeepEqual(fmp, expectedFMP) {
		t.Errorf("Expected and got firstMessagePart did not match."+
			"\nexpected: %+v\nrecieved: %+v", expectedFMP, fmp)
	}
}

// Test that firstMessagePartFromBytes returns a correctly made firstMessagePart
// from the bytes of one.
func Test_firstMessagePartFromBytes(t *testing.T) {
	fmp := firstMessagePartFromBytes(expectedFMP.Data)

	if !reflect.DeepEqual(fmp, expectedFMP) {
		t.Error("Expected and got firstMessagePart did not match")
	}
}

// Test that firstMessagePart.getType returns the correct type.
func Test_firstMessagePart_getType(t *testing.T) {
	if expectedFMP.getType() != catalog.XxMessage {
		t.Errorf("Got %v, expected %v", expectedFMP.getType(), catalog.XxMessage)
	}
}

// Test that firstMessagePart.getNumParts returns the correct number of parts.
func Test_firstMessagePart_getNumParts(t *testing.T) {
	if expectedFMP.getNumParts() != 2 {
		t.Errorf("Got %v, expected %v", expectedFMP.getNumParts(), 2)
	}
}

// Test that firstMessagePart.getTimestamp returns the correct timestamp.
func Test_firstMessagePart_getTimestamp(t *testing.T) {
	et := expectedFMP.getTimestamp()
	if !time.Unix(1609786229, 0).Equal(et) {
		t.Errorf("Got %v, expected %v", et, time.Unix(1609786229, 0))
	}
}

// Test that firstMessagePart.bytes returns the correct bytes.
func Test_firstMessagePart_bytes(t *testing.T) {
	if !bytes.Equal(expectedFMP.bytes(), expectedFMP.Data) {
		t.Errorf("Got %v, expected %v", expectedFMP.bytes(), expectedFMP.Data)
	}
}
