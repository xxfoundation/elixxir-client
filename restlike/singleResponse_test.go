////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import (
	"bytes"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

// Test happy path
func TestSingleResponse_Callback(t *testing.T) {
	resultChan := make(chan *Message, 1)
	cb := func(input *Message) {
		resultChan <- input
	}
	testPath := "test/path"
	testMethod := Get
	testMessage := &Message{
		Content: []byte("test"),
		Headers: nil,
		Method:  uint32(testMethod),
		Uri:     testPath,
		Error:   "",
	}

	response := singleResponse{cb}

	testPayload, err := proto.Marshal(testMessage)
	if err != nil {
		t.Errorf(err.Error())
	}
	response.Callback(testPayload, receptionID.EphemeralIdentity{}, nil, nil)

	select {
	case result := <-resultChan:
		if result.Uri != testPath {
			t.Errorf("Mismatched uri")
		}
		if result.Method != uint32(testMethod) {
			t.Errorf("Mismatched method")
		}
		if !bytes.Equal(testMessage.Content, result.Content) {
			t.Errorf("Mismatched content")
		}
	case <-time.After(3 * time.Second):
		t.Errorf("Test SingleResponse timed out!")
	}
}

// Test error input path
func TestSingleResponse_Callback_Err(t *testing.T) {
	resultChan := make(chan *Message, 1)
	cb := func(input *Message) {
		resultChan <- input
	}
	response := singleResponse{cb}

	response.Callback(nil, receptionID.EphemeralIdentity{}, nil, errors.New("test"))

	select {
	case result := <-resultChan:
		if len(result.Error) == 0 {
			t.Errorf("Expected cb error!")
		}
	case <-time.After(3 * time.Second):
		t.Errorf("Test SingleResponse input error timed out!")
	}
}

// Test proto error path
func TestSingleResponse_Callback_ProtoErr(t *testing.T) {
	resultChan := make(chan *Message, 1)
	cb := func(input *Message) {
		resultChan <- input
	}
	response := singleResponse{cb}

	response.Callback([]byte("test"), receptionID.EphemeralIdentity{}, nil, nil)

	select {
	case result := <-resultChan:
		if len(result.Error) == 0 {
			t.Errorf("Expected cb proto error!")
		}
	case <-time.After(3 * time.Second):
		t.Errorf("Test SingleResponse proto error timed out!")
	}
}
