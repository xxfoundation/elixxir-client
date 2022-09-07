////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"testing"
	"time"
)

// Unit test of newInternalMsg.
func Test_newInternalMsg(t *testing.T) {
	maxDataSize := 2 * internalMinLen
	im, err := newInternalMsg(maxDataSize)
	if err != nil {
		t.Errorf("newInternalMsg() returned an error: %+v", err)
	}

	if len(im.data) != maxDataSize {
		t.Errorf("newInternalMsg() set data to the wrong length."+
			"\nexpected: %d\nreceived: %d", maxDataSize, len(im.data))
	}
}

// Error path: the maxDataSize is smaller than the minimum size.
func Test_newInternalMsg_PayloadSizeError(t *testing.T) {
	maxDataSize := internalMinLen - 1
	expectedErr := fmt.Sprintf(newInternalSizeErr, maxDataSize, internalMinLen)

	_, err := newInternalMsg(maxDataSize)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("newInternalMsg() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of mapInternalMsg.
func Test_mapInternalMsg(t *testing.T) {
	// Create all the expected data
	timestamp := make([]byte, timestampLen)
	binary.LittleEndian.PutUint64(timestamp, uint64(netTime.Now().UnixNano()))
	senderID := id.NewIdFromString("test sender ID", id.User, t).Marshal()
	payload := []byte("Sample payload contents.")
	size := make([]byte, internalPayloadSizeLen)
	binary.LittleEndian.PutUint16(size, uint16(len(payload)))

	// Construct data into single slice
	data := bytes.NewBuffer(nil)
	data.Write(timestamp)
	data.Write(senderID)
	data.Write(size)
	data.Write(payload)

	// Map data
	im := mapInternalMsg(data.Bytes())

	// Check that the mapped values match the expected values
	if !bytes.Equal(timestamp, im.timestamp) {
		t.Errorf("mapInternalMsg() did not correctly map timestamp."+
			"\nexpected: %+v\nreceived: %+v", timestamp, im.timestamp)
	}

	if !bytes.Equal(senderID, im.senderID) {
		t.Errorf("mapInternalMsg() did not correctly map senderID."+
			"\nexpected: %+v\nreceived: %+v", senderID, im.senderID)
	}

	if !bytes.Equal(size, im.size) {
		t.Errorf("mapInternalMsg() did not correctly map size."+
			"\nexpected: %+v\nreceived: %+v", size, im.size)
	}

	if !bytes.Equal(payload, im.payload) {
		t.Errorf("mapInternalMsg() did not correctly map payload."+
			"\nexpected: %+v\nreceived: %+v", payload, im.payload)
	}
}

// Tests that a marshaled and unmarshalled internalMsg matches the original.
func TestInternalMsg_Marshal_unmarshalInternalMsg(t *testing.T) {
	im, _ := newInternalMsg(internalMinLen * 2)
	im.SetTimestamp(netTime.Now())
	im.SetSenderID(id.NewIdFromString("test sender ID", id.User, t))
	im.SetPayload([]byte("Sample payload message."))

	data := im.Marshal()

	newIm, err := unmarshalInternalMsg(data)
	if err != nil {
		t.Errorf("unmarshalInternalMsg() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(im, newIm) {
		t.Errorf("unmarshalInternalMsg() did not return the expected internalMsg."+
			"\nexpected: %s\nreceived: %s", im, newIm)
	}
}

// Error path: error is returned when the data is too short.
func Test_unmarshalInternalMsg_DataLengthError(t *testing.T) {
	expectedErr := fmt.Sprintf(unmarshalInternalSizeErr, 0, internalMinLen)

	_, err := unmarshalInternalMsg(nil)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("unmarshalInternalMsg() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Happy path.
func TestInternalMsg_SetTimestamp_GetTimestamp(t *testing.T) {
	im, _ := newInternalMsg(internalMinLen * 2)
	timestamp := netTime.Now()
	im.SetTimestamp(timestamp)
	testTimestamp := im.GetTimestamp()

	if !timestamp.Equal(testTimestamp) {
		t.Errorf("Failed to get original timestamp."+
			"\nexpected: %s\nreceived: %s", timestamp, testTimestamp)
	}
}

// Happy path.
func TestInternalMsg_SetSenderID_GetSenderID(t *testing.T) {
	im, _ := newInternalMsg(internalMinLen * 2)
	sid := id.NewIdFromString("testSenderID", id.User, t)
	im.SetSenderID(sid)
	testID, err := im.GetSenderID()
	if err != nil {
		t.Errorf("GetSenderID() returned an error: %+v", err)
	}

	if !sid.Cmp(testID) {
		t.Errorf("Failed to get original sender ID."+
			"\nexpected: %s\nreceived: %s", sid, testID)
	}
}

// Tests that the original payload matches the saved one.
func TestInternalMsg_SetPayload_GetPayload(t *testing.T) {
	im, _ := newInternalMsg(internalMinLen * 2)
	payload := []byte("Test payload message.")
	im.SetPayload(payload)
	testPayload := im.GetPayload()

	if !bytes.Equal(payload, testPayload) {
		t.Errorf("Failed to get original sender payload."+
			"\nexpected: %s\nreceived: %s", payload, testPayload)
	}
}

// Happy path.
func TestInternalMsg_GetPayloadSize(t *testing.T) {
	im, _ := newInternalMsg(internalMinLen * 2)
	payload := []byte("Test payload message.")
	im.SetPayload(payload)

	if len(payload) != im.GetPayloadSize() {
		t.Errorf("GetPayloadSize() failed to return the correct size."+
			"\nexpected: %d\nreceived: %d", len(payload), im.GetPayloadSize())
	}
}

// Happy path.
func TestInternalMsg_GetPayloadMaxSize(t *testing.T) {
	im, _ := newInternalMsg(internalMinLen * 2)

	if internalMinLen != im.GetPayloadMaxSize() {
		t.Errorf("GetPayloadSize() failed to return the correct size."+
			"\nexpected: %d\nreceived: %d", internalMinLen, im.GetPayloadMaxSize())
	}
}

// Happy path.
func TestInternalMsg_String(t *testing.T) {
	im, _ := newInternalMsg(internalMinLen * 2)
	im.SetTimestamp(time.Date(1955, 11, 5, 12, 0, 0, 0, time.UTC))
	im.SetSenderID(id.NewIdFromString("test sender ID", id.User, t))
	payload := []byte("Sample payload message.")
	payload = append(payload, 0, 1, 2)
	im.SetPayload(payload)

	expected := "{timestamp:" + im.GetTimestamp().String() +
		", senderID:dGVzdCBzZW5kZXIgSUQAAAAAAAAAAAAAAAAAAAAAAAAD, " +
		"size:26, payload:\"Sample payload message.\\x00\\x01\\x02\"}"

	if im.String() != expected {
		t.Errorf("String() failed to return the expected value."+
			"\nexpected: %s\nreceived: %s", expected, im.String())
	}
}

// Happy path: tests that String returns the expected string for a nil
// internalMsg.
func TestInternalMsg_String_NilInternalMessage(t *testing.T) {
	im := internalMsg{}

	expected := "{timestamp:<nil>, senderID:<nil>, size:<nil>, payload:<nil>}"

	if im.String() != expected {
		t.Errorf("String() failed to return the expected value."+
			"\nexpected: %s\nreceived: %s", expected, im.String())
	}
}
