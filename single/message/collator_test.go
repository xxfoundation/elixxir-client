package message

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Happy path
func TestNewCollator(t *testing.T) {
	messageCount := uint8(10)
	expected := &Collator{
		payloads: make([][]byte, messageCount),
		maxNum:   unsetCollatorMax,
		count:    0,
	}

	c := NewCollator(messageCount)

	if !reflect.DeepEqual(expected, c) {
		t.Errorf("NewCollator failed to generate the expected Collator."+
			"\nexepcted: %+v\nreceived: %+v", expected, c)
	}
}

// Happy path.
func TestCollator_Collate(t *testing.T) {
	messageCount := 16
	msgPayloadSize := 2
	msgParts := map[int]ResponsePart{}
	expectedData := make([]byte, messageCount*msgPayloadSize)
	copy(expectedData, "This is the expected final data.")

	buff := bytes.NewBuffer(expectedData)
	for i := 0; i < messageCount; i++ {
		msgParts[i] = NewResponsePart(msgPayloadSize + 5)
		msgParts[i].SetNumParts(uint8(messageCount))
		msgParts[i].SetPartNum(uint8(i))
		msgParts[i].SetContents(buff.Next(msgPayloadSize))
	}

	c := NewCollator(uint8(messageCount))

	i := 0
	var fullPayload []byte
	for j, part := range msgParts {
		i++

		var err error
		var collated bool

		fullPayload, collated, err = c.Collate(part.Marshal())
		if err != nil {
			t.Errorf("Collate returned an error for part #%d: %+v", j, err)
		}

		if i == messageCount && (!collated || fullPayload == nil) {
			t.Errorf("Collate failed to collate a completed payload."+
				"\ncollated:    %v\nfullPayload: %+v", collated, fullPayload)
		} else if i < messageCount && (collated || fullPayload != nil) {
			t.Errorf("Collate signaled it collated an unfinished payload."+
				"\ncollated:    %v\nfullPayload: %+v", collated, fullPayload)
		}
	}

	if !bytes.Equal(expectedData, fullPayload) {
		t.Errorf("Collate failed to return the correct collated data."+
			"\nexpected: %s\nreceived: %s", expectedData, fullPayload)
	}
}

// Error path: the byte slice cannot be unmarshaled.
func TestCollator_collate_UnmarshalError(t *testing.T) {
	payloadBytes := []byte{1}
	c := NewCollator(1)
	payload, collated, err := c.Collate(payloadBytes)
	expectedErr := strings.Split(errUnmarshalResponsePart, "%")[0]

	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Collate failed to return an error for failing to "+
			"unmarshal the payload.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}

	if payload != nil || collated {
		t.Errorf("Collate signaled the payload was collated on error."+
			"\npayload:  %+v\ncollated: %+v", payload, collated)
	}
}

// Error path: max reported parts by payload larger than set in Collator.
func TestCollator_Collate_MaxPartsError(t *testing.T) {
	payloadBytes := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	messageCount := uint8(1)
	c := NewCollator(messageCount)
	_, _, err := c.Collate(payloadBytes)
	expectedErr := fmt.Sprintf(errMaxParts, 0xFF, messageCount)

	if err == nil || err.Error() != expectedErr {
		t.Errorf("Collate failed to return an error when the max number of "+
			"parts is larger than the payload size."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: the message part number is greater than the max number of parts.
func TestCollator_Collate_PartNumTooLargeError(t *testing.T) {
	payloadBytes := []byte{25, 5, 5, 5, 5}
	partNum := uint8(5)
	c := NewCollator(partNum)
	_, _, err := c.Collate(payloadBytes)
	expectedErr := fmt.Sprintf(errPartOutOfRange, partNum, c.maxNum)

	if err == nil || err.Error() != expectedErr {
		t.Errorf("Collate failed to return the expected error when the part "+
			"number is greater than the max number of parts."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: a message with the part number already exists.
func TestCollator_Collate_PartExistsError(t *testing.T) {
	payloadBytes := []byte{0, 1, 5, 0, 1, 20}
	c := NewCollator(5)
	_, _, err := c.Collate(payloadBytes)
	if err != nil {
		t.Fatalf("Collate returned an error: %+v", err)
	}
	expectedErr := fmt.Sprintf(errPartExists, payloadBytes[1])

	_, _, err = c.Collate(payloadBytes)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Collate failed to return an error when the part number "+
			"already exists.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}
