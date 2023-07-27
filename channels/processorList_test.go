////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"fmt"
	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Tests that newProcessorList returns the expected new processorList.
func Test_newProcessorList(t *testing.T) {
	expected := &processorList{
		list: make(map[id.ID]map[processorTag]broadcast.Processor),
	}

	received := newProcessorList()
	if !reflect.DeepEqual(expected, received) {
		t.Errorf("New processorList does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that processorList.addProcessor add the processors to the list, and
// that calling Process works.
func Test_processorList_addProcessor(t *testing.T) {
	pl := newProcessorList()

	channelID := id.NewIdFromString("channelID", id.User, t)
	tag := userProcessor
	c := make(chan struct{})
	pl.addProcessor(channelID, tag, &testProcessor{c})

	p, exists := pl.list[*channelID][tag]
	if !exists {
		t.Fatalf("Failed to find added processor at tag %s.", tag)
	}

	go p.Process(
		format.Message{}, nil, nil, receptionID.EphemeralIdentity{}, rounds.Round{})

	select {
	case <-c:
	case <-time.After(10 * time.Millisecond):
		t.Errorf("Timed out waiting for the processor to be called.")
	}
}

// Tests that processorList.removeProcessors removes the processors for the
// channel.
func Test_processorList_removeProcessors(t *testing.T) {
	pl := newProcessorList()

	channelID := id.NewIdFromString("channelID", id.User, t)
	pl.addProcessor(channelID, userProcessor, &testProcessor{})

	pl.removeProcessors(channelID)
	processors, exists := pl.list[*channelID]
	if exists {
		t.Fatalf("Loaded processors for %s that should have been deleted: %s",
			channelID, processors)
	}

	// Test that removing the same channel does nothing
	pl.removeProcessors(channelID)
}

// Tests that processorList.getProcessor returns the expected processor from
// list and that calling Process works.
func Test_processorList_getProcessor(t *testing.T) {
	pl := newProcessorList()

	channelID := id.NewIdFromString("channelID", id.User, t)
	tag := userProcessor
	c := make(chan struct{})
	pl.addProcessor(channelID, tag, &testProcessor{c})

	p, err := pl.getProcessor(channelID, tag)
	if err != nil {
		t.Fatalf("Failed to get processor: %+v", err)
	}
	go p.Process(
		format.Message{}, nil, nil, receptionID.EphemeralIdentity{}, rounds.Round{})

	select {
	case <-c:
	case <-time.After(10 * time.Millisecond):
		t.Errorf("Timed out waiting for the processor to be called.")
	}
}

// Error path: Tests that processorList.getProcessor returns the expected error
// when there are no processors registered with the provided channel ID.
func Test_processorList_getProcessor_InvalidChannelError(t *testing.T) {
	pl := newProcessorList()

	channelID := id.NewIdFromString("channelID", id.User, t)
	tag := adminProcessor
	expectedErr := fmt.Sprintf(noProcessorChannelErr, channelID)
	_, err := pl.getProcessor(channelID, tag)
	if err == nil || err.Error() != expectedErr {
		t.Fatalf("Did not get expected error when no processors exist with "+
			"the channel.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: Tests that processorList.getProcessor returns the expected error
// when there is no processor registered with the provided tag.
func Test_processorList_getProcessor_InvalidTagError(t *testing.T) {
	pl := newProcessorList()

	channelID := id.NewIdFromString("channelID", id.User, t)
	pl.addProcessor(channelID, userProcessor, &testProcessor{})

	tag := adminProcessor
	expectedErr := fmt.Sprintf(noProcessorTagErr, tag, channelID)
	_, err := pl.getProcessor(channelID, tag)
	if err == nil || err.Error() != expectedErr {
		t.Fatalf("Did not get expected error when no processor exists with "+
			"the tag.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests the consistency of processorTag.String.
func Test_processorTag_String(t *testing.T) {
	tests := map[processorTag]string{
		userProcessor:  "userProcessor",
		adminProcessor: "adminProcessor",
		64:             "INVALID PROCESSOR TAG 64",
	}

	for tag, expected := range tests {
		if tag.String() != expected {
			t.Errorf("Unexpected string for tag %d.\nexpected: %s\nreceived: %s",
				tag, expected, tag)
		}
	}
}

// testProcessor is used as the broadcast.Processor in testing.
type testProcessor struct {
	c chan struct{}
}

func (tp *testProcessor) Process(
	format.Message, []string, []byte, receptionID.EphemeralIdentity, rounds.Round) {
	tp.c <- struct{}{}
}
func (tp *testProcessor) ProcessAdminMessage(
	[]byte, []string, [2]byte, receptionID.EphemeralIdentity, rounds.Round) {
	tp.c <- struct{}{}
}
func (tp *testProcessor) String() string { return "testProcessor" }

// testAdminProcessor is used as the broadcast.Processor in testing.
type testAdminProcessor struct {
	c            chan struct{}
	adminMsgChan chan []byte
}

func (tap *testAdminProcessor) Process(
	format.Message, []string, []byte, receptionID.EphemeralIdentity, rounds.Round) {
	tap.c <- struct{}{}
}
func (tap *testAdminProcessor) ProcessAdminMessage(
	innerCiphertext []byte, _ []string, _ [2]byte, _ receptionID.EphemeralIdentity, _ rounds.Round) {
	tap.adminMsgChan <- innerCiphertext
}
func (tap *testAdminProcessor) String() string { return "testProcessor" }
