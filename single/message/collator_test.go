package message

import (
	"bytes"
	"fmt"
	"reflect"
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
	msgParts := map[int]mockPart{}
	expectedData := make([]byte, messageCount*msgPayloadSize)
	copy(expectedData, "This is the expected final data.")

	buff := bytes.NewBuffer(expectedData)
	for i := 0; i < messageCount; i++ {
		msgParts[i] = mockPart{
			numParts: uint8(messageCount),
			partNum:  uint8(i),
			contents: buff.Next(msgPayloadSize),
		}
	}

	c := NewCollator(uint8(messageCount))

	i := 0
	var fullPayload []byte
	for j, part := range msgParts {
		i++

		var err error
		var collated bool

		fullPayload, collated, err = c.Collate(part)
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

// Error path: max reported parts by payload larger than set in Collator.
func TestCollator_Collate_MaxPartsError(t *testing.T) {
	p := mockPart{0xFF, 0xFF, []byte{0xFF, 0xFF, 0xFF}}
	messageCount := uint8(1)
	c := NewCollator(messageCount)
	_, _, err := c.Collate(p)
	expectedErr := fmt.Sprintf(errMaxParts, 0xFF, messageCount)

	if err == nil || err.Error() != expectedErr {
		t.Errorf("Collate failed to return an error when the max number of "+
			"parts is larger than the payload size."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: the message part number is greater than the max number of parts.
func TestCollator_Collate_PartNumTooLargeError(t *testing.T) {
	p := mockPart{35, 5, []byte{5, 5, 5}}
	messageCount := uint8(1)
	c := NewCollator(messageCount)
	_, _, err := c.Collate(p)
	expectedErr := fmt.Sprintf(errMaxParts, p.numParts, messageCount)

	if err == nil || err.Error() != expectedErr {
		t.Errorf("Collate failed to return the expected error when the part "+
			"number is greater than the max number of parts."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: a message with the part number already exists.
func TestCollator_Collate_PartExistsError(t *testing.T) {
	p := mockPart{5, 1, []byte{5, 0, 1, 20}}
	c := NewCollator(5)
	_, _, err := c.Collate(p)
	if err != nil {
		t.Fatalf("Collate returned an error: %+v", err)
	}
	expectedErr := fmt.Sprintf(errPartExists, p.partNum)

	_, _, err = c.Collate(p)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("Collate failed to return an error when the part number "+
			"already exists.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

type mockPart struct {
	numParts uint8
	partNum  uint8
	contents []byte
}

func (m mockPart) GetNumParts() uint8  { return m.numParts }
func (m mockPart) GetPartNum() uint8   { return m.partNum }
func (m mockPart) GetContents() []byte { return m.contents }
func (m mockPart) Marshal() []byte     { return append([]byte{m.numParts, m.partNum}, m.contents...) }
