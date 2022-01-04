///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"
)

// Tests that newPartMessage returns a partMessage of the expected size.
func Test_newPartMessage(t *testing.T) {
	externalPayloadSize := 256

	fm, err := newPartMessage(externalPayloadSize)
	if err != nil {
		t.Errorf("newPartMessage returned an error: %+v", err)
	}

	if len(fm.data) != externalPayloadSize {
		t.Errorf("Size of partMessage data does not match payload size."+
			"\nexpected: %d\nreceived: %d", externalPayloadSize, len(fm.data))
	}
}

// Error path: tests that newPartMessage returns the expected error when the
// external payload size is too small.
func Test_newPartMessage_SmallPayloadSizeError(t *testing.T) {
	externalPayloadSize := fmMinSize - 1
	expectedErr := fmt.Sprintf(newFmSizeErr, externalPayloadSize, fmMinSize)

	_, err := newPartMessage(externalPayloadSize)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("newPartMessage did not return the expected error when the "+
			"given external payload size is too small."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that mapPartMessage maps the data to the correct parts of the
// partMessage.
func Test_mapPartMessage(t *testing.T) {
	// Generate expected values
	_, expectedData, expectedPadding, expectedPartNum, expectedFile :=
		newRandomFileMessage()

	fm := mapPartMessage(expectedData)

	if !bytes.Equal(expectedData, fm.data) {
		t.Errorf("Incorrect data.\nexpected: %q\nreceived: %q",
			expectedData, fm.data)
	}

	if !bytes.Equal(expectedPadding, fm.padding) {
		t.Errorf("Incorrect padding data.\nexpected: %q\nreceived: %q",
			expectedPadding, fm.padding)
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

// Tests that unmarshalPartMessage returns a partMessage with the expected
// values.
func Test_unmarshalPartMessage(t *testing.T) {
	// Generate expected values
	_, expectedData, expectedPadding, expectedPartNumb, expectedFile :=
		newRandomFileMessage()

	fm, err := unmarshalPartMessage(expectedData)
	if err != nil {
		t.Errorf("unmarshalPartMessage return an error: %+v", err)
	}

	if !bytes.Equal(expectedData, fm.data) {
		t.Errorf("Incorrect data.\nexpected: %q\nreceived: %q",
			expectedData, fm.data)
	}

	if !bytes.Equal(expectedPadding, fm.padding) {
		t.Errorf("Incorrect padding data.\nexpected: %q\nreceived: %q",
			expectedPadding, fm.padding)
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

// Error path: tests that unmarshalPartMessage returns the expected error when
// the provided data is too small to be unmarshalled into a partMessage.
func Test_unmarshalPartMessage_SizeError(t *testing.T) {
	data := make([]byte, fmMinSize-1)
	expectedErr := fmt.Sprintf(unmarshalFmSizeErr, len(data), fmMinSize)

	_, err := unmarshalPartMessage(data)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("unmarshalPartMessage did not return the expected error when "+
			"the given bytes are too small to be a partMessage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that partMessage.marshal returns the correct data.
func Test_fileMessage_marshal(t *testing.T) {
	fm, expectedData, _, _, _ := newRandomFileMessage()

	data := fm.marshal()

	if !bytes.Equal(expectedData, data) {
		t.Errorf("Marshalled data does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedData, data)
	}
}

// Tests that partMessage.getPadding returns the correct padding data.
func Test_fileMessage_getPadding(t *testing.T) {
	fm, _, expectedPadding, _, _ := newRandomFileMessage()

	padding := fm.getPadding()

	if !bytes.Equal(expectedPadding, padding) {
		t.Errorf("Padding data does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedPadding, padding)
	}
}

// Tests that partMessage.setPadding sets the correct data.
func Test_fileMessage_setPadding(t *testing.T) {
	fm, err := newPartMessage(256)
	if err != nil {
		t.Errorf("Failed to create new partMessage: %+v", err)
	}

	expectedPadding := make([]byte, paddingLen)
	rand.New(rand.NewSource(42)).Read(expectedPadding)

	fm.setPadding(expectedPadding)

	if !bytes.Equal(expectedPadding, fm.getPadding()) {
		t.Errorf("Failed to set correct padding.\nexpected: %q\nreceived: %q",
			expectedPadding, fm.getPadding())
	}
}

// Tests that partMessage.getPartNum returns the correct part number.
func Test_fileMessage_getPartNum(t *testing.T) {
	fm, _, _, expectedPartNum, _ := newRandomFileMessage()

	partNum := fm.getPartNum()
	expected := binary.LittleEndian.Uint16(expectedPartNum)

	if expected != partNum {
		t.Errorf("Part number does not match expected."+
			"\nexpected: %d\nreceived: %d", expected, partNum)
	}
}

// Tests that partMessage.setPartNum sets the correct part number.
func Test_fileMessage_setPartNum(t *testing.T) {
	fm, err := newPartMessage(256)
	if err != nil {
		t.Errorf("Failed to create new partMessage: %+v", err)
	}

	expectedPartNum := make([]byte, partNumLen)
	rand.New(rand.NewSource(42)).Read(expectedPartNum)
	expected := binary.LittleEndian.Uint16(expectedPartNum)

	fm.setPartNum(expected)

	if expected != fm.getPartNum() {
		t.Errorf("Failed to set correct part number.\nexpected: %d\nreceived: %d",
			expected, fm.getPartNum())
	}
}

// Tests that partMessage.getPart returns the correct part data.
func Test_fileMessage_getFile(t *testing.T) {
	fm, _, _, _, expectedFile := newRandomFileMessage()

	file := fm.getPart()

	if !bytes.Equal(expectedFile, file) {
		t.Errorf("File data does not match expected."+
			"\nexpected: %q\nreceived: %q", expectedFile, file)
	}
}

// Tests that partMessage.setPart sets the correct part data.
func Test_fileMessage_setFile(t *testing.T) {
	fm, err := newPartMessage(256)
	if err != nil {
		t.Errorf("Failed to create new partMessage: %+v", err)
	}

	fileData := make([]byte, 64)
	rand.New(rand.NewSource(42)).Read(fileData)
	expectedFile := make([]byte, fm.getPartSize())
	copy(expectedFile, fileData)

	err = fm.setPart(expectedFile)
	if err != nil {
		t.Errorf("setPart returned an error: %+v", err)
	}

	if !bytes.Equal(expectedFile, fm.getPart()) {
		t.Errorf("Failed to set correct part data.\nexpected: %q\nreceived: %q",
			expectedFile, fm.getPart())
	}
}

// Error path: tests that partMessage.setPart returns the expected error when
// the provided part data is too large for the message.
func Test_fileMessage_setFile_FileTooLargeError(t *testing.T) {
	fm, err := newPartMessage(fmMinSize + 1)
	if err != nil {
		t.Errorf("Failed to create new partMessage: %+v", err)
	}

	expectedErr := fmt.Sprintf(setFileFmErr, fm.getPartSize()+1, fm.getPartSize())

	err = fm.setPart(make([]byte, fm.getPartSize()+1))
	if err == nil || err.Error() != expectedErr {
		t.Errorf("setPart did not return the expected error when the given "+
			"part data is too large to fit in the partMessage."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that partMessage.getPartSize returns the expected available space for
// the part data.
func Test_fileMessage_getFileSize(t *testing.T) {
	expectedSize := 256

	fm, err := newPartMessage(fmMinSize + expectedSize)
	if err != nil {
		t.Errorf("Failed to create new partMessage: %+v", err)
	}

	if expectedSize != fm.getPartSize() {
		t.Errorf("File size incorrect.\nexpected: %d\nreceived: %d",
			expectedSize, fm.getPartSize())
	}
}

// newRandomFileMessage generates a new partMessage filled with random data and
// return the partMessage and its individual parts.
func newRandomFileMessage() (partMessage, []byte, []byte, []byte, []byte) {
	prng := rand.New(rand.NewSource(42))
	padding := make([]byte, paddingLen)
	prng.Read(padding)
	partNum := make([]byte, partNumLen)
	prng.Read(partNum)
	part := make([]byte, 64)
	prng.Read(part)
	data := append(append(padding, partNum...), part...)

	fm := mapPartMessage(data)

	return fm, data, padding, partNum, part
}
