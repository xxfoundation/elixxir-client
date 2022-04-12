package message

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Happy path
func Test_newCollator(t *testing.T) {
	messageCount := uint8(10)
	expected := &Collator{
		payloads: make([][]byte, messageCount),
		maxNum:   unsetCollatorMax,
		count:    0,
	}
	c := NewCollator(messageCount)

	if !reflect.DeepEqual(expected, c) {
		t.Errorf("NewCollator() failed to generated the expected Collator."+
			"\nexepcted: %+v\nreceived: %+v", expected, c)
	}
}

// Happy path.
func TestCollator_collate(t *testing.T) {
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
			t.Errorf("Collate() returned an error for part #%d: %+v", j, err)
		}

		if i == messageCount && (!collated || fullPayload == nil) {
			t.Errorf("Collate() failed to Collate a completed payload."+
				"\ncollated:    %v\nfullPayload: %+v", collated, fullPayload)
		} else if i < messageCount && (collated || fullPayload != nil) {
			t.Errorf("Collate() signaled it collated an unfinished payload."+
				"\ncollated:    %v\nfullPayload: %+v", collated, fullPayload)
		}
	}

	if !bytes.Equal(expectedData, fullPayload) {
		t.Errorf("Collate() failed to return the correct collated data."+
			"\nexpected: %s\nreceived: %s", expectedData, fullPayload)
	}
}

// Error path: the byte slice cannot be unmarshaled.
func TestCollator_collate_UnmarshalError(t *testing.T) {
	payloadBytes := []byte{1}
	c := NewCollator(1)
	payload, collated, err := c.Collate(payloadBytes)

	if err == nil || !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Collate() failed to return an error for failing to "+
			"unmarshal the payload.\nerror: %+v", err)
	}

	if payload != nil || collated {
		t.Errorf("Collate() signaled the payload was collated on error."+
			"\npayload:  %+v\ncollated: %+v", payload, collated)
	}
}

// Error path: max reported parts by payload larger then set in Collator
func TestCollator_collate_MaxPartsError(t *testing.T) {
	payloadBytes := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	c := NewCollator(1)
	payload, collated, err := c.Collate(payloadBytes)

	if err == nil || !strings.Contains(err.Error(), "Max number of parts") {
		t.Errorf("Collate() failed to return an error when the max number of "+
			"parts is larger than the payload size.\nerror: %+v", err)
	}

	if payload != nil || collated {
		t.Errorf("Collate() signaled the payload was collated on error."+
			"\npayload:  %+v\ncollated: %+v", payload, collated)
	}
}

// Error path: the message part number is greater than the max number of parts.
func TestCollator_collate_PartNumTooLargeError(t *testing.T) {
	payloadBytes := []byte{25, 5, 5, 5, 5}
	c := NewCollator(5)
	payload, collated, err := c.Collate(payloadBytes)

	if err == nil || !strings.Contains(err.Error(), "greater than max number of expected parts") {
		t.Errorf("Collate() failed to return an error when the part number is "+
			"greater than the max number of parts.\nerror: %+v", err)
	}

	if payload != nil || collated {
		t.Errorf("Collate() signaled the payload was collated on error."+
			"\npayload:  %+v\ncollated: %+v", payload, collated)
	}
}

// Error path: a message with the part number already exists.
func TestCollator_collate_PartExistsError(t *testing.T) {
	payloadBytes := []byte{0, 1, 5, 0, 1, 20}
	c := NewCollator(5)
	payload, collated, err := c.Collate(payloadBytes)
	if err != nil {
		t.Fatalf("Collate() returned an error: %+v", err)
	}

	payload, collated, err = c.Collate(payloadBytes)
	if err == nil || !strings.Contains(err.Error(), "A payload for the part number") {
		t.Errorf("Collate() failed to return an error when the part number "+
			"already exists.\nerror: %+v", err)
	}

	if payload != nil || collated {
		t.Errorf("Collate() signaled the payload was collated on error."+
			"\npayload:  %+v\ncollated: %+v", payload, collated)
	}
}
