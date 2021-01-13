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
	"testing"
)

// Happy path.
func Test_newResponseMessagePart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	payloadSize := prng.Intn(2000)
	expected := responseMessagePart{
		data:     make([]byte, payloadSize),
		partNum:  make([]byte, partNumLen),
		maxParts: make([]byte, maxPartsLen),
		payload:  make([]byte, payloadSize-partNumLen-maxPartsLen),
	}

	rmp := newResponseMessagePart(payloadSize)

	if !reflect.DeepEqual(expected, rmp) {
		t.Errorf("newResponseMessagePart() did not return the expected "+
			"responseMessagePart.\nexpected: %+v\nreceived: %v", expected, rmp)
	}
}

// Error path: provided payload size is not large enough.
func Test_newResponseMessagePart_PayloadSizeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("newResponseMessagePart() did not panic when the size of " +
				"the payload is smaller than the required size.")
		}
	}()

	_ = newResponseMessagePart(1)
}

// Happy path.
func Test_mapResponseMessagePart(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedPartNum := uint8(prng.Uint32())
	expectedMaxParts := uint8(prng.Uint32())
	expectedPayload := make([]byte, prng.Intn(2000))
	prng.Read(expectedPayload)
	var data []byte
	data = append(data, expectedPartNum, expectedMaxParts)
	data = append(data, expectedPayload...)

	rmp := mapResponseMessagePart(data)

	if expectedPartNum != rmp.partNum[0] {
		t.Errorf("mapResponseMessagePart() did not correctly map partNum."+
			"\nexpected: %d\nreceived: %d", expectedPartNum, rmp.partNum[0])
	}

	if expectedMaxParts != rmp.maxParts[0] {
		t.Errorf("mapResponseMessagePart() did not correctly map maxParts."+
			"\nexpected: %d\nreceived: %d", expectedMaxParts, rmp.maxParts[0])
	}

	if !bytes.Equal(expectedPayload, rmp.payload) {
		t.Errorf("mapResponseMessagePart() did not correctly map payload."+
			"\nexpected: %+v\nreceived: %+v", expectedPayload, rmp.payload)
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
func TestResponseMessagePart_SetPayload_GetPayload_GetPayloadSize(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	externalPayloadSize := prng.Intn(2000)
	payloadSize := externalPayloadSize - partNumLen - maxPartsLen
	expectedPayload := make([]byte, payloadSize)
	prng.Read(expectedPayload)
	rmp := newResponseMessagePart(externalPayloadSize)
	rmp.SetPayload(expectedPayload)

	if !bytes.Equal(expectedPayload, rmp.GetPayload()) {
		t.Errorf("GetPayload() failed to return the expected payload."+
			"\nexpected: %+v\nrecieved: %+v", expectedPayload, rmp.GetPayload())
	}

	if payloadSize != rmp.GetPayloadSize() {
		t.Errorf("GetPayloadSize() failed to return the expected payload size."+
			"\nexpected: %d\nrecieved: %d", payloadSize, rmp.GetPayloadSize())
	}
}

// Error path: size of supplied payload does not match message payload size.
func TestResponseMessagePart_SetPayload_PayloadSizeError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetPayload() did not panic when the size of the supplied " +
				"bytes is not the same as the payload content size.")
		}
	}()

	rmp := newResponseMessagePart(255)
	rmp.SetPayload([]byte{1, 2, 3})
}
