////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"bytes"
	"fmt"
	"testing"
)

// Tests that a payload smaller than the max payload size encoded via
// NewSizedBroadcast and decoded via DecodeSizedBroadcast matches the original.
func TestNewSizedBroadcast_DecodeSizedBroadcast_SmallPayload(t *testing.T) {
	const maxPayloadSize = 512
	payload := []byte("This is my payload message.")

	data, err := NewSizedBroadcast(maxPayloadSize, payload)
	if err != nil {
		t.Errorf("NewSizedBroadcast returned an error: %+v", err)
	}

	decodedPayload, err := DecodeSizedBroadcast(data)
	if err != nil {
		t.Errorf("DecodeSizedBroadcast returned an error: %+v", err)
	}

	if !bytes.Equal(payload, decodedPayload) {
		t.Errorf("Decoded payload does not match original."+
			"\nexpected: %q\nreceived: %q", payload, decodedPayload)
	}
}

// Tests that a payload the same size as the max payload size encoded via
// NewSizedBroadcast and decoded via DecodeSizedBroadcast matches the original.
func TestNewSizedBroadcast_DecodeSizedBroadcast_FullSizesPayload(t *testing.T) {
	payload := []byte("This is my payload message.")
	maxPayloadSize := len(payload) + sizeSize

	data, err := NewSizedBroadcast(maxPayloadSize, payload)
	if err != nil {
		t.Errorf("NewSizedBroadcast returned an error: %+v", err)
	}

	decodedPayload, err := DecodeSizedBroadcast(data)
	if err != nil {
		t.Errorf("DecodeSizedBroadcast returned an error: %+v", err)
	}

	if !bytes.Equal(payload, decodedPayload) {
		t.Errorf("Decoded payload does not match original."+
			"\nexpected: %q\nreceived: %q", payload, decodedPayload)
	}
}

// Error path: tests that NewSizedBroadcast returns an error when the payload is
// larger than the max payload size.
func TestNewSizedBroadcast_MaxPayloadSizeError(t *testing.T) {
	payload := []byte("This is my payload message.")
	maxPayloadSize := len(payload)
	expectedErr := fmt.Sprintf(errNewSizedBroadcastMaxSize,
		len(payload)+sizedBroadcastMinSize, maxPayloadSize)

	_, err := NewSizedBroadcast(maxPayloadSize, payload)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("NewSizedBroadcast did not return the expected error when "+
			"the payload is too large.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Error path: tests that DecodeSizedBroadcast returns an error when the length
// of the data is shorter than the minimum length of a sized broadcast.
func TestDecodeSizedBroadcast_DataTooShortError(t *testing.T) {
	data := []byte{0}
	expectedErr := fmt.Sprintf(
		errDecodeSizedBroadcastDataLen, len(data), sizedBroadcastMinSize)

	_, err := DecodeSizedBroadcast(data)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("DecodeSizedBroadcast did not return the expected error "+
			"when the data is too small.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Error path: tests that DecodeSizedBroadcast returns an error when the payload
// size is larger than the actual payload contained in the data.
func TestDecodeSizedBroadcast_SizeMismatchError(t *testing.T) {
	data := []byte{255, 0, 10}
	expectedErr := fmt.Sprintf(
		errDecodeSizedBroadcastSize, data[0], len(data[sizeSize:]))

	_, err := DecodeSizedBroadcast(data)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("DecodeSizedBroadcast did not return the expected error "+
			"when the size is too large.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Tests that MaxSizedBroadcastPayloadSize returns the correct max size.
func TestMaxSizedBroadcastPayloadSize(t *testing.T) {
	maxPayloadSize := 512
	expectedSize := maxPayloadSize - sizedBroadcastMinSize
	receivedSize := MaxSizedBroadcastPayloadSize(maxPayloadSize)
	if receivedSize != expectedSize {
		t.Errorf("Incorrect max paylaod size.\nexpected: %d\nreceived: %d",
			expectedSize, receivedSize)
	}
}
