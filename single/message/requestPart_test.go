///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

// Happy path.
func Test_NewRequestPart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	expected := RequestPart{
		data:     make([]byte, payloadSize),
		partNum:  make([]byte, reqPartPartNumLen),
		size:     make([]byte, reqPartSizeLen),
		contents: make([]byte, payloadSize-reqPartMinSize),
	}

	rmp := NewRequestPart(payloadSize)

	if !reflect.DeepEqual(expected, rmp) {
		t.Errorf("NewRequestPart did not return the expected "+
			"RequestPart.\nexpected: %+v\nreceived: %+v", expected, rmp)
	}
}

// Error path: provided contents size is not large enough.
func Test_NewRequestPart_PayloadSizeError(t *testing.T) {
	externalPayloadSize := 1
	expectedErr := fmt.Sprintf(
		errReqPartPayloadSize, externalPayloadSize, reqPartMinSize)
	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("NewRequestPart did not panic with the expected error "+
				"when the size of the payload is smaller than the required "+
				"size.\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	}()

	_ = NewRequestPart(externalPayloadSize)
}

// Happy path.
func Test_mapRequestPart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedPartNum := uint8(prng.Uint32())
	size := []byte{uint8(prng.Uint64()), uint8(prng.Uint64())}
	expectedContents := make([]byte, prng.Intn(2000))
	prng.Read(expectedContents)
	var data []byte
	data = append(data, expectedPartNum)
	data = append(data, size...)
	data = append(data, expectedContents...)

	rmp := mapRequestPart(data)

	if expectedPartNum != rmp.partNum[0] {
		t.Errorf("mapRequestPart did not correctly map partNum."+
			"\nexpected: %d\nreceived: %d", expectedPartNum, rmp.partNum[0])
	}

	if !bytes.Equal(expectedContents, rmp.contents) {
		t.Errorf("mapRequestPart did not correctly map contents."+
			"\nexpected: %+v\nreceived: %+v", expectedContents, rmp.contents)
	}

	if !bytes.Equal(data, rmp.data) {
		t.Errorf("mapRequestPart did not save the data correctly."+
			"\nexpected: %+v\nreceived: %+v", data, rmp.data)
	}
}

// Happy path.
func TestRequestPart_Marshal_UnmarshalRequestPart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payload := make([]byte, prng.Intn(2000))
	prng.Read(payload)
	rmp := NewRequestPart(prng.Intn(2000))

	data := rmp.Marshal()

	newRmp, err := UnmarshalRequestPart(data)
	if err != nil {
		t.Errorf("UnmarshalRequestPart produced an error: %+v", err)
	}

	if !reflect.DeepEqual(rmp, newRmp) {
		t.Errorf("Failed to Marshal and unmarshal the RequestPart."+
			"\nexpected: %+v\nrecieved: %+v", rmp, newRmp)
	}
}

// Error path: provided bytes are too small.
func Test_UnmarshalRequestPart_Error(t *testing.T) {
	data := []byte{1}
	expectedErr := fmt.Sprintf(errReqPartDataSize, len(data), reqPartMinSize)
	_, err := UnmarshalRequestPart([]byte{1})
	if err == nil || err.Error() != expectedErr {
		t.Errorf("UnmarshalRequestPart did not produce the expected error "+
			"when the byte slice is smaller required."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Happy path.
func TestRequestPart_SetPartNum_GetPartNum(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedPartNum := uint8(prng.Uint32())
	rmp := NewRequestPart(prng.Intn(2000))

	rmp.SetPartNum(expectedPartNum)

	if expectedPartNum != rmp.GetPartNum() {
		t.Errorf("GetPartNum failed to return the expected part number."+
			"\nexpected: %d\nrecieved: %d", expectedPartNum, rmp.GetPartNum())
	}
}

// Happy path.
func TestRequestPart_GetMaxParts(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedMaxParts := uint8(0)
	rmp := NewRequestPart(prng.Intn(2000))

	if expectedMaxParts != rmp.GetNumParts() {
		t.Errorf("GetNumParts failed to return the expected max parts."+
			"\nexpected: %d\nrecieved: %d", expectedMaxParts, rmp.GetNumParts())
	}
}

// Happy path.
func TestRequestPart_SetContents_GetContents_GetContentsSize_GetMaxContentsSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	contentSize := externalPayloadSize - reqPartMinSize - 10
	expectedContents := make([]byte, contentSize)
	prng.Read(expectedContents)
	rmp := NewRequestPart(externalPayloadSize)
	rmp.SetContents(expectedContents)

	if !bytes.Equal(expectedContents, rmp.GetContents()) {
		t.Errorf("GetContents failed to return the expected contents."+
			"\nexpected: %+v\nrecieved: %+v", expectedContents, rmp.GetContents())
	}

	if contentSize != rmp.GetContentsSize() {
		t.Errorf("GetContentsSize failed to return the expected contents size."+
			"\nexpected: %d\nrecieved: %d", contentSize, rmp.GetContentsSize())
	}

	if externalPayloadSize-reqPartMinSize != rmp.GetMaxContentsSize() {
		t.Errorf("GetMaxContentsSize failed to return the expected max "+
			"contents size.\nexpected: %d\nrecieved: %d",
			externalPayloadSize-reqPartMinSize, rmp.GetMaxContentsSize())
	}
}

// Error path: size of supplied contents does not match message contents size.
func TestRequestPart_SetContents_ContentsSizeError(t *testing.T) {
	payloadSize, contentsLen := 255, 500
	expectedErr := fmt.Sprintf(
		errReqPartContentsSize, contentsLen, payloadSize-reqPartMinSize)
	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("SetContents did not panic with the expected error when "+
				"the size of the supplied bytes is larger than the content "+
				"size.\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	}()

	rmp := NewRequestPart(payloadSize)
	rmp.SetContents(make([]byte, contentsLen))
}
