///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// Happy path.
func Test_newResponseMessagePart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	expected := responseMessagePart{
		data:     make([]byte, payloadSize),
		version:  make([]byte, receptionMessageVersionLen),
		partNum:  make([]byte, partNumLen),
		maxParts: make([]byte, maxPartsLen),
		size:     make([]byte, sizeSize),
		contents: make([]byte, payloadSize-partNumLen-maxPartsLen-sizeSize-receptionMessageVersionLen),
	}

	rmp := newResponseMessagePart(payloadSize)

	if !reflect.DeepEqual(expected, rmp) {
		t.Errorf("newResponseMessagePart() did not return the expected "+
			"responseMessagePart.\nexpected: %+v\nreceived: %+v", expected, rmp)
	}
}

// Error path: provided contents size is not large enough.
func Test_newResponseMessagePart_PayloadSizeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "size of external payload") {
			t.Error("newResponseMessagePart() did not panic when the size of " +
				"the payload is smaller than the required size.")
		}
	}()

	_ = newResponseMessagePart(1)
}

// Happy path.
func Test_mapResponseMessagePart(t *testing.T) {
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

	rmp := mapResponseMessagePart(data)

	if expectedPartNum != rmp.partNum[0] {
		t.Errorf("mapResponseMessagePart() did not correctly map partNum."+
			"\nexpected: %d\nreceived: %d", expectedPartNum, rmp.partNum[0])
	}

	if expectedMaxParts != rmp.maxParts[0] {
		t.Errorf("mapResponseMessagePart() did not correctly map maxParts."+
			"\nexpected: %d\nreceived: %d", expectedMaxParts, rmp.maxParts[0])
	}

	if !bytes.Equal(expectedContents, rmp.contents) {
		t.Errorf("mapResponseMessagePart() did not correctly map contents."+
			"\nexpected: %+v\nreceived: %+v", expectedContents, rmp.contents)
	}

	if !bytes.Equal(data, rmp.data) {
		t.Errorf("mapResponseMessagePart() did not save the data correctly."+
			"\nexpected: %+v\nreceived: %+v", data, rmp.data)
	}
}

// Happy path.
func TestResponseMessagePart_Marshal_Unmarshal(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payload := make([]byte, prng.Intn(2000))
	prng.Read(payload)
	rmp := newResponseMessagePart(prng.Intn(2000))

	data := rmp.Marshal()

	newRmp, err := unmarshalResponseMessage(data)
	if err != nil {
		t.Errorf("unmarshalResponseMessage() produced an error: %+v", err)
	}

	if !reflect.DeepEqual(rmp, newRmp) {
		t.Errorf("Failed to Marshal() and unmarshal() the responseMessagePart."+
			"\nexpected: %+v\nrecieved: %+v", rmp, newRmp)
	}
}

// Error path: provided bytes are too small.
func Test_unmarshalResponseMessage(t *testing.T) {
	_, err := unmarshalResponseMessage([]byte{1})
	if err == nil {
		t.Error("unmarshalResponseMessage() did not produce an error when the " +
			"byte slice is smaller required.")
	}
}

// Happy path.
func TestResponseMessagePart_SetPartNum_GetPartNum(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedPartNum := uint8(prng.Uint32())
	rmp := newResponseMessagePart(prng.Intn(2000))

	rmp.SetPartNum(expectedPartNum)

	if expectedPartNum != rmp.GetPartNum() {
		t.Errorf("GetPartNum() failed to return the expected part number."+
			"\nexpected: %d\nrecieved: %d", expectedPartNum, rmp.GetPartNum())
	}
}

// Happy path.
func TestResponseMessagePart_SetMaxParts_GetMaxParts(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedMaxParts := uint8(prng.Uint32())
	rmp := newResponseMessagePart(prng.Intn(2000))

	rmp.SetMaxParts(expectedMaxParts)

	if expectedMaxParts != rmp.GetMaxParts() {
		t.Errorf("GetMaxParts() failed to return the expected max parts."+
			"\nexpected: %d\nrecieved: %d", expectedMaxParts, rmp.GetMaxParts())
	}
}

// Happy path.
func TestResponseMessagePart_SetContents_GetContents_GetContentsSize_GetMaxContentsSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	contentSize := externalPayloadSize - responseMinSize - 10
	expectedContents := make([]byte, contentSize)
	prng.Read(expectedContents)
	rmp := newResponseMessagePart(externalPayloadSize)
	rmp.SetContents(expectedContents)

	if !bytes.Equal(expectedContents, rmp.GetContents()) {
		t.Errorf("GetContents() failed to return the expected contents."+
			"\nexpected: %+v\nrecieved: %+v", expectedContents, rmp.GetContents())
	}

	if contentSize != rmp.GetContentsSize() {
		t.Errorf("GetContentsSize() failed to return the expected contents size."+
			"\nexpected: %d\nrecieved: %d", contentSize, rmp.GetContentsSize())
	}

	if externalPayloadSize-responseMinSize != rmp.GetMaxContentsSize() {
		t.Errorf("GetMaxContentsSize() failed to return the expected max contents size."+
			"\nexpected: %d\nrecieved: %d",
			externalPayloadSize-responseMinSize, rmp.GetMaxContentsSize())
	}
}

// Error path: size of supplied contents does not match message contents size.
func TestResponseMessagePart_SetContents_ContentsSizeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "max size of message contents") {
			t.Error("SetContents() did not panic when the size of the supplied " +
				"bytes is larger than the content size.")
		}
	}()

	rmp := newResponseMessagePart(255)
	rmp.SetContents(make([]byte, 500))
}
