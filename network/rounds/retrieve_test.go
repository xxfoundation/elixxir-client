///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package rounds

import (
	"bytes"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage/reception"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"reflect"
	"testing"
	"time"
)

// Happy path
func TestManager_ProcessMessageRetrieval(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	quitChan := make(chan struct{})

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, quitChan)

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)

		// Construct the round lookup
		iu := reception.IdentityUse{
			Identity: reception.Identity{
				EphId:  expectedEphID,
				Source: requestGateway,
			},
		}

		idList := [][]byte{requestGateway.Bytes()}

		roundInfo := &pb.RoundInfo{
			ID:       uint64(roundId),
			Topology: idList,
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			roundInfo: roundInfo,
			identity:  iu,
		}

	}()

	var testBundle message.Bundle
	go func() {
		// Receive the bundle over the channel
		time.Sleep(1 * time.Second)
		testBundle = <-messageBundleChan

		// Close the process
		quitChan <- struct{}{}

	}()

	time.Sleep(2 * time.Second)
	if reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Errorf("Did not receive a message bundle over the channel")
		t.FailNow()
	}

	if testBundle.Identity.EphId.Int64() != expectedEphID.Int64() {
		t.Errorf("Unexpected ephemeral ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedEphID, testBundle.Identity.EphId)
	}

	if !bytes.Equal(expectedPayload, testBundle.Messages[0].GetPayloadA()) {
		t.Errorf("Unexpected ephemeral ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedPayload, testBundle.Messages[0].GetPayloadA())

	}

}

// Utilize the mockComms to construct a gateway which does not have the round
func TestManager_ProcessMessageRetrieval_NoRound(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	quitChan := make(chan struct{})

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, quitChan)

	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}

	// Construct a gateway without keyword ID in utils_test.go
	// ie mockComms does not return a round
	dummyGateway := id.NewIdFromString("Sauron", id.Gateway, t)

	go func() {
		// Construct the round lookup
		iu := reception.IdentityUse{
			Identity: reception.Identity{
				EphId:  expectedEphID,
				Source: dummyGateway,
			},
		}

		idList := [][]byte{dummyGateway.Bytes()}

		roundInfo := &pb.RoundInfo{
			ID:       uint64(roundId),
			Topology: idList,
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			roundInfo: roundInfo,
			identity:  iu,
		}

	}()

	var testBundle message.Bundle
	go func() {
		// Receive the bundle over the channel
		time.Sleep(1 * time.Second)
		testBundle = <-messageBundleChan

		// Close the process
		quitChan <- struct{}{}

	}()

	time.Sleep(2 * time.Second)
	if !reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Errorf("Should not receive a message bundle, mock gateway should not return round."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", message.Bundle{}, testBundle)
	}
}

// Test the path where there are no messages,
// simulating a false positive in a bloom filter
func TestManager_ProcessMessageRetrieval_FalsePositive(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	quitChan := make(chan struct{})

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, quitChan)

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		// Construct the round lookup
		iu := reception.IdentityUse{
			Identity: reception.Identity{
				EphId: expectedEphID,
			},
		}

		requestGateway := id.NewIdFromString(FalsePositive, id.Gateway, t)

		idList := [][]byte{requestGateway.Bytes()}

		roundInfo := &pb.RoundInfo{
			ID:       uint64(roundId),
			Topology: idList,
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			roundInfo: roundInfo,
			identity:  iu,
		}

	}()

	var testBundle message.Bundle
	go func() {
		// Receive the bundle over the channel
		time.Sleep(1 * time.Second)
		testBundle = <-messageBundleChan

		// Close the process
		quitChan <- struct{}{}

	}()

	time.Sleep(2 * time.Second)
	if !reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Errorf("Received a message bundle over the channel, should receive empty message list")
		t.FailNow()
	}

}

// Ensure that the quit chan closes the program, on an otherwise happy path
func TestManager_ProcessMessageRetrieval_Quit(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	quitChan := make(chan struct{})

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, quitChan)

	// Close the process early, before any logic below can be completed
	quitChan <- struct{}{}

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		// Construct the round lookup
		iu := reception.IdentityUse{
			Identity: reception.Identity{
				EphId: expectedEphID,
			},
		}

		requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)

		idList := [][]byte{requestGateway.Bytes()}

		roundInfo := &pb.RoundInfo{
			ID:       uint64(roundId),
			Topology: idList,
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			roundInfo: roundInfo,
			identity:  iu,
		}

	}()

	var testBundle message.Bundle
	go func() {
		// Receive the bundle over the channel
		testBundle = <-messageBundleChan

	}()

	time.Sleep(1 * time.Second)
	if !reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Errorf("Received a message bundle over the channel, process should have quit before reception")
		t.FailNow()
	}

}
