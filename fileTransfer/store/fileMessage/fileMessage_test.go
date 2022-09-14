////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package fileMessage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"
)

// Tests that NewPartMessage returns a PartMessage of the expected size.
func Test_newPartMessage(t *testing.T) {
	externalPayloadSize := 256

	fm := NewPartMessage(externalPayloadSize)

	if len(fm.data) != externalPayloadSize {
		t.Errorf("Size of PartMessage data does not match payload size."+
			"\nexpected: %d\nreceived: %d", externalPayloadSize, len(fm.data))
	}
}

// Error path: tests that NewPartMessage returns the expected error when the
// external payload size is too small.
func Test_newPartMessage_SmallPayloadSizeError(t *testing.T) {
	externalPayloadSize := fmMinSize - 1
	expectedErr := fmt.Sprintf(errNewFmSize, externalPayloadSize, fmMinSize)

	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("NewPartMessage did not return the expected error when "+
				"the given external payload size is too small."+
				"\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	}()

	NewPartMessage(externalPayloadSize)
}

// Tests that mapPartMessage maps the data to the correct parts of the
// PartMessage.
func Test_mapPartMessage(t *testing.T) {
	// Generate expected values
	_, expectedData, expectedPartNum, expectedFile :=
		newRandomFileMessage()

	fm := mapPartMessage(expectedData)

	if !bytes.Equal(expectedData, fm.data) {
		t.Errorf("Incorrect data.\nexpected: %q\nreceived: %q",
			expectedData, fm.data)
	}

	if !bytes.Equal(expectedPartNum, fm.partNum) {
		t.Errorf("Incorrect part number.\nexpected: %q\nreceived: %q",
			expectedPartNum, fm.partNum)
	}

	if !bytes.Equal(expectedFile, fm.part) {
		t.Errorf("Incorrect part data.\nexpected: %q\nreceived: %q",
			expectedFile, fm.part)
	}

}

// Tests that UnmarshalPartMessage returns a PartMessage with the expected
// values.
func Test_unmarshalPartMessage(t *testing.T) {
	// Generate expected values
	_, expectedData, expectedPartNumb, expectedFile :=
		newRandomFileMessage()

	fm, err := UnmarshalPartMessage(expectedData)
	if err != nil {
		t.Errorf("UnmarshalPartMessage return an error: %+v", err)
	}

	if !bytes.Equal(expectedData, fm.data) {
		t.Errorf("Incorrect data.\nexpected: %q\nreceived: %q",
			expectedData, fm.data)
	}

	if !bytes.Equal(expectedPartNumb, fm.partNum) {
		t.Errorf("Incorrect part number.\nexpected: %q\nreceived: %q",
			expectedPartNumb, fm.partNum)
	}

	if !bytes.Equal(expectedFile, fm.part) {
		t.Errorf("Incorrect part data.\nexpected: %q\nreceived: %q",
			expectedFile, fm.part)
	}
}

// Error path: tests that UnmarshalPartMessage returns the expected error when
// the provided data is too small to be unmarshalled into a PartMessage.
func Test_unmarshalPartMessage_SizeError(t *testing.T) {
	data := make([]byte, fmMinSize-1)
	expectedErr := fmt.Sprintf(unmarshalFmSizeErr, len(data), fmMinSize)

	_, err := UnmarshalPartMessage(data)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("UnmarshalPartMessage did not return the expected error when "+
			"the given bytes are too small to be a PartMessage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that PartMessage.Marshal returns the correct data.
func Test_fileMessage_marshal(t *testing.T) {
	fm, expectedData, _, _ := newRandomFileMessage()

	data := fm.Marshal()

	if !bytes.Equal(expectedData, data) {
		t.Errorf("Marshalled data does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedData, data)
	}
}

// Tests that PartMessage.GetPartNum returns the correct part number.
func Test_fileMessage_getPartNum(t *testing.T) {
	fm, _, expectedPartNum, _ := newRandomFileMessage()

	partNum := fm.GetPartNum()
	expected := binary.LittleEndian.Uint16(expectedPartNum)

	if expected != partNum {
		t.Errorf("Part number does not match expected."+
			"\nexpected: %d\nreceived: %d", expected, partNum)
	}
}

// Tests that PartMessage.SetPartNum sets the correct part number.
func Test_fileMessage_setPartNum(t *testing.T) {
	fm := NewPartMessage(256)

	expectedPartNum := make([]byte, partNumLen)
	rand.New(rand.NewSource(42)).Read(expectedPartNum)
	expected := binary.LittleEndian.Uint16(expectedPartNum)

	fm.SetPartNum(expected)

	if expected != fm.GetPartNum() {
		t.Errorf("Failed to set correct part number.\nexpected: %d\nreceived: %d",
			expected, fm.GetPartNum())
	}
}

// Tests that PartMessage.GetPart returns the correct part data.
func Test_fileMessage_getFile(t *testing.T) {
	fm, _, _, expectedFile := newRandomFileMessage()

	file := fm.GetPart()

	if !bytes.Equal(expectedFile, file) {
		t.Errorf("File data does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedFile, file)
	}
}

// Tests that PartMessage.SetPart sets the correct part data.
func Test_fileMessage_setFile(t *testing.T) {
	fm := NewPartMessage(256)

	fileData := make([]byte, 64)
	rand.New(rand.NewSource(42)).Read(fileData)
	expectedFile := make([]byte, fm.GetPartSize())
	copy(expectedFile, fileData)

	fm.SetPart(expectedFile)

	if !bytes.Equal(expectedFile, fm.GetPart()) {
		t.Errorf("Failed to set correct part data.\nexpected: %q\nreceived: %q",
			expectedFile, fm.GetPart())
	}
}

// Error path: tests that PartMessage.SetPart returns the expected error when
// the provided part data is too large for the message.
func Test_fileMessage_setFile_FileTooLargeError(t *testing.T) {
	fm := NewPartMessage(fmMinSize + 1)

	expectedErr := fmt.Sprintf(errSetFileFm, fm.GetPartSize()+1, fm.GetPartSize())

	defer func() {
		if r := recover(); r == nil || r != expectedErr {
			t.Errorf("SetPart did not return the expected error when the "+
				"given part data is too large to fit in the PartMessage."+
				"\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	}()

	fm.SetPart(make([]byte, fm.GetPartSize()+1))
}

// Tests that PartMessage.GetPartSize returns the expected available space for
// the part data.
func Test_fileMessage_getFileSize(t *testing.T) {
	expectedSize := 256

	fm := NewPartMessage(fmMinSize + expectedSize)

	if expectedSize != fm.GetPartSize() {
		t.Errorf("File size incorrect.\nexpected: %d\nreceived: %d",
			expectedSize, fm.GetPartSize())
	}
}

// newRandomFileMessage generates a new PartMessage filled with random data and
// return the PartMessage and its individual parts.
func newRandomFileMessage() (PartMessage, []byte, []byte, []byte) {
	prng := rand.New(rand.NewSource(42))
	partNum := make([]byte, partNumLen)
	prng.Read(partNum)
	part := make([]byte, 64)
	prng.Read(part)
	data := append(partNum, part...)

	fm := mapPartMessage(data)

	return fm, data, partNum, part
}
