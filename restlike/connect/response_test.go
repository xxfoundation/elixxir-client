////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"bytes"
	"gitlab.com/elixxir/client/v5/e2e/receive"
	"gitlab.com/elixxir/client/v5/restlike"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

// Test happy path
func TestSingleResponse_Callback(t *testing.T) {
	resultChan := make(chan *restlike.Message, 1)
	cb := func(input *restlike.Message) {
		resultChan <- input
	}
	testPath := "test/path"
	testMethod := restlike.Get
	testMessage := &restlike.Message{
		Content: []byte("test"),
		Headers: nil,
		Method:  uint32(testMethod),
		Uri:     testPath,
		Error:   "",
	}

	r := &response{cb}

	testPayload, err := proto.Marshal(testMessage)
	if err != nil {
		t.Errorf(err.Error())
	}
	r.Hear(receive.Message{Payload: testPayload})

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

// Test proto error path
func TestSingleResponse_Callback_ProtoErr(t *testing.T) {
	resultChan := make(chan *restlike.Message, 1)
	cb := func(input *restlike.Message) {
		resultChan <- input
	}
	r := &response{cb}

	r.Hear(receive.Message{Payload: []byte("test")})

	select {
	case result := <-resultChan:
		if len(result.Error) == 0 {
			t.Errorf("Expected cb proto error!")
		}
	case <-time.After(3 * time.Second):
		t.Errorf("Test SingleResponse proto error timed out!")
	}
}
