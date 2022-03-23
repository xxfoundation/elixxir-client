///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package rounds

import (
	"bytes"
	"gitlab.com/elixxir/client/network/gateway"
	ephemeral2 "gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
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
	stop := stoppable.NewSingle("singleStoppable")
	testNdf := getNDF()
	nodeId := id.NewIdFromString(ReturningGateway, id.Node, &testing.T{})
	gwId := nodeId.DeepCopy()
	gwId.SetType(id.Gateway)
	testNdf.Gateways = []ndf.Gateway{{ID: gwId.Marshal()}}
	testManager.Rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	var err error
	testManager.sender, err = gateway.NewSender(p, testManager.Rng,
		testNdf, mockComms, testManager.Session, nil)
	if err != nil {
		t.Errorf(err.Error())
	}

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, stop)

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)

		// Construct the round lookup
		iu := ephemeral2.IdentityUse{
			Identity: ephemeral2.Identity{
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
		err := stop.Close()
		if err != nil {
			t.Errorf("Failed to signal close to process: %+v", err)
		}
	}()

	// Ensure bundle received and has expected values
	time.Sleep(2 * time.Second)
	if reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Errorf("Did not receive a message bundle over the channel")
		t.FailNow()
	}

	if testBundle.Identity.EphId.Int64() != expectedEphID.Int64() {
		t.Errorf("Unexpected address ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedEphID, testBundle.Identity.EphId)
	}

	if !bytes.Equal(expectedPayload, testBundle.Messages[0].GetPayloadA()) {
		t.Errorf("Unexpected address ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedPayload, testBundle.Messages[0].GetPayloadA())

	}

}

// Utilize the mockComms to construct a gateway which does not have the round
func TestManager_ProcessMessageRetrieval_NoRound(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	testNdf := getNDF()
	nodeId := id.NewIdFromString(FalsePositive, id.Node, &testing.T{})
	gwId := nodeId.DeepCopy()
	gwId.SetType(id.Gateway)
	testNdf.Gateways = []ndf.Gateway{{ID: gwId.Marshal()}}
	testManager.Rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	testManager.sender, _ = gateway.NewSender(p,
		testManager.Rng,
		testNdf, mockComms, testManager.Session, nil)
	stop := stoppable.NewSingle("singleStoppable")

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, stop)

	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}

	// Construct a gateway without keyword ID in utils_test.go
	// ie mockComms does not return a round
	dummyGateway := id.NewIdFromString("Sauron", id.Gateway, t)

	go func() {
		// Construct the round lookup
		iu := ephemeral2.IdentityUse{
			Identity: ephemeral2.Identity{
				EphId:  expectedEphID,
				Source: dummyGateway,
			},
		}

		idList := [][]byte{dummyGateway.Marshal()}

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
		if err := stop.Close(); err != nil {
			t.Errorf("Failed to signal close to process: %+v", err)
		}
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
	stop := stoppable.NewSingle("singleStoppable")
	testNdf := getNDF()
	nodeId := id.NewIdFromString(FalsePositive, id.Node, &testing.T{})
	gwId := nodeId.DeepCopy()
	gwId.SetType(id.Gateway)
	testNdf.Gateways = []ndf.Gateway{{ID: gwId.Marshal()}}
	testManager.Rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	testManager.sender, _ = gateway.NewSender(p,
		testManager.Rng,
		testNdf, mockComms, testManager.Session, nil)

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, stop)

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		// Construct the round lookup
		iu := ephemeral2.IdentityUse{
			Identity: ephemeral2.Identity{
				EphId:  expectedEphID,
				Source: id.NewIdFromString("Source", id.User, t),
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
		if err := stop.Close(); err != nil {
			t.Errorf("Failed to signal close to process: %+v", err)
		}
	}()

	// Ensure no bundle was received due to false positive test
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
	stop := stoppable.NewSingle("singleStoppable")

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, stop)

	// Close the process early, before any logic below can be completed
	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	if err := stoppable.WaitForStopped(stop, 300*time.Millisecond); err != nil {
		t.Fatalf("Failed to stop stoppable: %+v", err)
	}

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		// Construct the round lookup
		iu := ephemeral2.IdentityUse{
			Identity: ephemeral2.Identity{
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
	// Ensure no bundle was received due to quiting process early
	if !reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Errorf("Received a message bundle over the channel, process should have quit before reception")
		t.FailNow()
	}

}

// Path in which multiple error comms are encountered before a happy path comms
func TestManager_ProcessMessageRetrieval_MultipleGateways(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	stop := stoppable.NewSingle("singleStoppable")
	testNdf := getNDF()
	nodeId := id.NewIdFromString(ReturningGateway, id.Node, &testing.T{})
	gwId := nodeId.DeepCopy()
	gwId.SetType(id.Gateway)
	testNdf.Gateways = []ndf.Gateway{{ID: gwId.Marshal()}}
	testManager.Rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	testManager.sender, _ = gateway.NewSender(p,
		testManager.Rng,
		testNdf, mockComms, testManager.Session, nil)

	// Create a local channel so reception is possible (testManager.messageBundles is
	// send only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, stop)

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)
		errorGateway := id.NewIdFromString(ErrorGateway, id.Gateway, t)
		// Construct the round lookup
		iu := ephemeral2.IdentityUse{
			Identity: ephemeral2.Identity{
				EphId:  expectedEphID,
				Source: requestGateway,
			},
		}

		// Create a list of ID's in which some error gateways must be contacted before the happy path
		idList := [][]byte{errorGateway.Bytes(), errorGateway.Bytes(), requestGateway.Bytes()}

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
		if err := stop.Close(); err != nil {
			t.Errorf("Failed to signal close to process: %+v", err)
		}
	}()

	// Ensure that expected bundle is still received from happy comm
	// despite initial errors
	time.Sleep(2 * time.Second)
	if reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Errorf("Did not receive a message bundle over the channel")
		t.FailNow()
	}

	if testBundle.Identity.EphId.Int64() != expectedEphID.Int64() {
		t.Errorf("Unexpected address ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedEphID, testBundle.Identity.EphId)
	}

	if !bytes.Equal(expectedPayload, testBundle.Messages[0].GetPayloadA()) {
		t.Errorf("Unexpected address ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedPayload, testBundle.Messages[0].GetPayloadA())

	}

}
