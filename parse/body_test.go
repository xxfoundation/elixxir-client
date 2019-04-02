////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"bytes"
	"testing"
)

func TestParse(t *testing.T) {
	body := []byte{0x80, 0x02, 0x89, 0x02, 0x03, 0x04}
	actual, err := Parse(body)
	expected := &TypedBody{}
	expected.Body = []byte{0x89, 0x02, 0x03, 0x04}
	expected.MessageType = 256

	if err != nil {
		t.Error(err.Error())
	}

	if actual.MessageType != expected.MessageType {
		t.Errorf("Body type didn't match. Expected: %v, actual: %v",
			expected.MessageType, actual.MessageType)
	} else if !bytes.Equal(actual.Body, expected.Body) {
		t.Errorf("Body didn't match. Expected: %v, actual: %v",
			expected.Body, actual.Body)
	}
}

func TestParseTypeTooLong(t *testing.T) {
	body := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80,
		0x80, 0x80, 0x01, 0x02, 0x03, 0x04}
	_, err := Parse(body)

	if err == nil {
		t.Error("Didn't get an error from Parse(" +
			") when the body type was too long")
	}
}

func TestTypeAsBytes(t *testing.T) {
	expected := []byte{0x80, 0x02}
	actual := TypeAsBytes(256)
	if !bytes.Equal(expected, actual) {
		t.Errorf("Type magic number didn't match. Expected: %v, actual: %v",
			expected, actual)
	}
}

func TestPack(t *testing.T) {
	expected := []byte{0x01, 0x02, 0x03, 0x04}
	actual := Pack(&TypedBody{
		MessageType: 1,
		Body: []byte{0x02, 0x03, 0x04},
	})
	if !bytes.Equal(expected, actual) {
		t.Errorf("Pack didn't return correctly packed byte slice. "+
			"Expected: %v, actual: %v", expected, actual)
	}
}
