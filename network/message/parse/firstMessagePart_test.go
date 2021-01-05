///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package parse

import (
	"bytes"
	"gitlab.com/elixxir/client/interfaces/message"
	"reflect"
	"testing"
	"time"
)

// Expected firstMessagePart for checking against, generated by fmp in TestNewFirstMessagePart
var efmp = firstMessagePart{
	messagePart: messagePart{
		Data: []byte{0, 0, 4, 53, 0, 0, 13, 2, 0, 0, 0, 2, 1, 0, 0, 0, 14, 215, 133, 90, 117, 0, 0, 0, 0, 254, 32,
			116, 101, 115, 116, 105, 110, 103, 115, 116, 114, 105, 110, 103},
		Id:       []byte{0, 0, 4, 53},
		Part:     []byte{0},
		Len:      []byte{0, 13},
		Contents: []byte{116, 101, 115, 116, 105, 110, 103, 115, 116, 114, 105, 110, 103},
	},
	NumParts:  []byte{2},
	Type:      []byte{0, 0, 0, 2},
	Timestamp: []byte{1, 0, 0, 0, 14, 215, 133, 90, 117, 0, 0, 0, 0, 254, 32},
}

// Test that newFirstMessagePart returns a correctly made firstMessagePart
func TestNewFirstMessagePart(t *testing.T) {
	fmp := newFirstMessagePart(
		message.Text,
		1077,
		2,
		time.Unix(1609786229, 0),
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g',
			's', 't', 'r', 'i', 'n', 'g'},
	)

	if !reflect.DeepEqual(fmp, efmp) {
		t.Errorf("Expected and got firstMessagePart did not match.\r\tGot: %#v\r\tExected: %#v", fmp, efmp)
	}
}

// Test that FirstMessagePartFromBytes returns a correctly made firstMessagePart from the bytes of one
func TestFirstMessagePartFromBytes(t *testing.T) {
	fmp := FirstMessagePartFromBytes(efmp.Data)

	if !reflect.DeepEqual(fmp, efmp) {
		t.Error("Expected and got firstMessagePart did not match")
	}
}

// Test that GetType returns the correct type for a firstMessagePart
func TestFirstMessagePart_GetType(t *testing.T) {
	if efmp.GetType() != message.Text {
		t.Errorf("Got %v, expected %v", efmp.GetType(), message.Text)
	}
}

// Test that GetNumParts returns the correct number of parts for a firstMessagePart
func TestFirstMessagePart_GetNumParts(t *testing.T) {
	if efmp.GetNumParts() != 2 {
		t.Errorf("Got %v, expected %v", efmp.GetNumParts(), 2)
	}
}

// Test that GetTimestamp returns the correct timestamp for a firstMessagePart
func TestFirstMessagePart_GetTimestamp(t *testing.T) {
	et, err := efmp.GetTimestamp()
	if err != nil {
		t.Error(err)
	}
	if !time.Unix(1609786229, 0).Equal(et) {
		t.Errorf("Got %v, expected %v", et, time.Unix(1609786229, 0))
	}
}

// Test that GetTimestamp returns the correct bytes for a firstMessagePart
func TestFirstMessagePart_Bytes(t *testing.T) {
	if bytes.Compare(efmp.Bytes(), efmp.Data) != 0 {
		t.Errorf("Got %v, expected %v", efmp.Bytes(), efmp.Data)
	}
}
