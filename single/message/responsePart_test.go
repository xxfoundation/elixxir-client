////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

// Happy path.
func Test_NewResponsePart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	expected := ResponsePart{
		data:     make([]byte, payloadSize),
		version:  make([]byte, resPartVersionLen),
		partNum:  make([]byte, resPartPartNumLen),
		maxParts: make([]byte, resPartMaxPartsLen),
		size:     make([]byte, resPartSizeLen),
		contents: make([]byte, payloadSize-resPartMinSize),
	}

	rmp := NewResponsePart(payloadSize)

	if !reflect.DeepEqual(expected, rmp) {
		t.Errorf("NewResponsePart did not return the expected "+
			"ResponsePart.\nexpected: %+v\nreceived: %+v", expected, rmp)
	}
}

// Error path: provided contents size is not large enough.
func Test_NewResponsePart_PayloadSizeError(t *testing.T) {
	externalPayloadSize := 1
	expectedErr := fmt.Sprintf(
		errResPartPayloadSize, externalPayloadSize, resPartMinSize)
	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("NewResponsePart did not panic with the expected error "+
				"when the size of the payload is smaller than the required "+
				"size.\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	}()

	_ = NewResponsePart(externalPayloadSize)
}

// Happy path.
func Test_mapResponsePart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedVersion := uint8(0)
	expectedPartNum := uint8(prng.Uint32())
	expectedMaxParts := uint8(prng.Uint32())
	size := []byte{uint8(prng.Uint64()), uint8(prng.Uint64())}
	expectedContents := make([]byte, prng.Intn(2000))
	prng.Read(expectedContents)
	var data []byte
	data = append(data, expectedVersion, expectedPartNum, expectedMaxParts)
	data = append(data, size...)
	data = append(data, expectedContents...)

	rmp := mapResponsePart(data)

	if expectedPartNum != rmp.partNum[0] {
		t.Errorf("mapResponsePart did not correctly map partNum."+
			"\nexpected: %d\nreceived: %d", expectedPartNum, rmp.partNum[0])
	}

	if expectedMaxParts != rmp.maxParts[0] {
		t.Errorf("mapResponsePart did not correctly map maxResponseParts."+
			"\nexpected: %d\nreceived: %d", expectedMaxParts, rmp.maxParts[0])
	}

	if !bytes.Equal(expectedContents, rmp.contents) {
		t.Errorf("mapResponsePart did not correctly map contents."+
			"\nexpected: %+v\nreceived: %+v", expectedContents, rmp.contents)
	}

	if !bytes.Equal(data, rmp.data) {
		t.Errorf("mapResponsePart did not save the data correctly."+
			"\nexpected: %+v\nreceived: %+v", data, rmp.data)
	}
}

// Happy path.
func TestResponsePart_Marshal_UnmarshalResponsePart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payload := make([]byte, prng.Intn(2000))
	prng.Read(payload)
	rmp := NewResponsePart(prng.Intn(2000))

	data := rmp.Marshal()

	newRmp, err := UnmarshalResponsePart(data)
	if err != nil {
		t.Errorf("UnmarshalResponsePart produced an error: %+v", err)
	}

	if !reflect.DeepEqual(rmp, newRmp) {
		t.Errorf("Failed to Marshal and unmarshal the ResponsePart."+
			"\nexpected: %+v\nrecieved: %+v", rmp, newRmp)
	}
}

// Error path: provided bytes are too small.
func Test_UnmarshalResponsePart_Error(t *testing.T) {
	data := []byte{1}
	expectedErr := fmt.Sprintf(errResPartDataSize, len(data), resPartMinSize)
	_, err := UnmarshalResponsePart([]byte{1})
	if err == nil || err.Error() != expectedErr {
		t.Errorf("UnmarshalResponsePart did not produce the expected error "+
			"when the byte slice is smaller required."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Happy path.
func TestResponsePart_SetPartNum_GetPartNum(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedPartNum := uint8(prng.Uint32())
	rmp := NewResponsePart(prng.Intn(2000))

	rmp.SetPartNum(expectedPartNum)

	if expectedPartNum != rmp.GetPartNum() {
		t.Errorf("GetPartNum failed to return the expected part number."+
			"\nexpected: %d\nrecieved: %d", expectedPartNum, rmp.GetPartNum())
	}
}

// Happy path.
func TestResponsePart_SetMaxParts_GetMaxParts(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedMaxParts := uint8(prng.Uint32())
	rmp := NewResponsePart(prng.Intn(2000))

	rmp.SetNumParts(expectedMaxParts)

	if expectedMaxParts != rmp.GetNumParts() {
		t.Errorf("GetNumParts failed to return the expected max parts."+
			"\nexpected: %d\nrecieved: %d", expectedMaxParts, rmp.GetNumParts())
	}
}

// Happy path.
func TestResponsePart_SetContents_GetContents_GetContentsSize_GetMaxContentsSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	contentSize := externalPayloadSize - resPartMinSize - 10
	expectedContents := make([]byte, contentSize)
	prng.Read(expectedContents)
	rmp := NewResponsePart(externalPayloadSize)
	rmp.SetContents(expectedContents)

	if !bytes.Equal(expectedContents, rmp.GetContents()) {
		t.Errorf("GetContents failed to return the expected contents."+
			"\nexpected: %+v\nrecieved: %+v", expectedContents, rmp.GetContents())
	}

	if contentSize != rmp.GetContentsSize() {
		t.Errorf("GetContentsSize failed to return the expected contents size."+
			"\nexpected: %d\nrecieved: %d", contentSize, rmp.GetContentsSize())
	}

	if externalPayloadSize-resPartMinSize != rmp.GetMaxContentsSize() {
		t.Errorf("GetMaxResponsePartSize failed to return the expected max "+
			"contents size.\nexpected: %d\nrecieved: %d",
			externalPayloadSize-resPartMinSize, rmp.GetMaxContentsSize())
	}
}

// Error path: size of supplied contents does not match message contents size.
func TestResponsePart_SetContents_ContentsSizeError(t *testing.T) {
	payloadSize, contentsLen := 255, 500
	expectedErr := fmt.Sprintf(
		errResPartContentsSize, contentsLen, payloadSize-resPartMinSize)
	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("SetContents did not panic with the expected error when "+
				"the size of the supplied bytes is larger than the content "+
				"size.\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	}()

	rmp := NewResponsePart(payloadSize)
	rmp.SetContents(make([]byte, contentsLen))
}
