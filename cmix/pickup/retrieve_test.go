////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package pickup

import (
	"bytes"
	"gitlab.com/elixxir/client/v4/cmix/gateway"
	ephemeral2 "gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

// Happy path.
func Test_manager_processMessageRetrieval(t *testing.T) {
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
	testManager.rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	var err error
	addChan := make(chan network.NodeGateway, 1)
	testManager.sender, err = gateway.NewTestingSender(p, testManager.rng,
		testNdf, mockComms, testManager.session, addChan, t)
	if err != nil {
		t.Errorf(err.Error())
	}

	// Create a local channel so reception is possible
	// (testManager.messageBundles is sent only via newManager call above)
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
		ephIdentity := ephemeral2.EphemeralIdentity{
			EphID:  expectedEphID,
			Source: requestGateway,
		}

		round := rounds.Round{
			ID:       roundId,
			Topology: connect.NewCircuit([]*id.ID{requestGateway}),
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			Round:    round,
			Identity: ephIdentity,
		}

	}()

	var testBundle message.Bundle

	select {
	case testBundle = <-messageBundleChan:
	case <-time.After(300 * time.Millisecond):
		t.Errorf("Timed out waiting for messageBundleChan.")
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	// Ensure bundle received and has expected values
	if reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Fatal("Did not receive a message bundle over the channel")
	}

	if testBundle.Identity.EphID.Int64() != expectedEphID.Int64() {
		t.Errorf("Unexpected address ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedEphID, testBundle.Identity.EphID)
	}

	if !bytes.Equal(expectedPayload, testBundle.Messages[0].GetPayloadA()) {
		t.Errorf("Unexpected address ID in bundle."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedPayload, testBundle.Messages[0].GetPayloadA())

	}

}

// Utilize the mockComms to construct a gateway which does not have the round.
func Test_manager_processMessageRetrieval_NoRound(t *testing.T) {
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
	testManager.rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	addChan := make(chan network.NodeGateway, 1)
	mccc := &mockCertCheckerComm{}

	testManager.sender, _ = gateway.NewSender(p,
		testManager.rng,
		testNdf, mockComms, testManager.session, mccc, addChan)
	stop := stoppable.NewSingle("singleStoppable")

	// Create a local channel so reception is possible
	// (testManager.messageBundles is sent only via newManager call above)
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
		identity := ephemeral2.EphemeralIdentity{
			EphID:  expectedEphID,
			Source: dummyGateway,
		}

		round := rounds.Round{
			ID:       roundId,
			Topology: connect.NewCircuit([]*id.ID{dummyGateway}),
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			Round:    round,
			Identity: identity,
		}

	}()

	// Ensure the bundle was not received
	var testBundle message.Bundle
	select {
	case testBundle = <-messageBundleChan:
		t.Errorf("Should not receive a message bundle, mock gateway should "+
			"not return round.\nreceived: %+v", testBundle)
	case <-time.After(30 * time.Millisecond):
	}

	// Close the process
	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}
}

// Test the path where there are no messages. Simulating a false positive in a
// bloom filter.
func Test_manager_processMessageRetrieval_FalsePositive(t *testing.T) {
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
	testManager.rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	addChan := make(chan network.NodeGateway, 1)
	mccc := &mockCertCheckerComm{}

	testManager.sender, _ = gateway.NewSender(p,
		testManager.rng,
		testNdf, mockComms, testManager.session, mccc, addChan)

	// Create a local channel so reception is possible
	// (testManager.messageBundles is sent only via newManager call above)
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
		identity := ephemeral2.EphemeralIdentity{
			EphID:  expectedEphID,
			Source: id.NewIdFromString("Source", id.User, t),
		}

		requestGateway := id.NewIdFromString(FalsePositive, id.Gateway, t)

		round := rounds.Round{
			ID:       roundId,
			Topology: connect.NewCircuit([]*id.ID{requestGateway}),
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			Round:    round,
			Identity: identity,
		}

	}()

	// Ensure the bundle was not received
	var testBundle message.Bundle
	select {
	case testBundle = <-messageBundleChan:
		t.Fatalf("Received a message bundle over the channel, should receive "+
			"empty message list: %+v", testBundle)
	case <-time.After(30 * time.Millisecond):
	}

	// Close the process
	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}
}

// Ensure that the quit chan closes the program, on an otherwise happy path.
func Test_manager_processMessageRetrieval_Quit(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	stop := stoppable.NewSingle("singleStoppable")

	// Create a local channel so reception is possible
	// (testManager.messageBundles is sent only via newManager call above)
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
		identity := ephemeral2.EphemeralIdentity{
			EphID: expectedEphID,
		}

		requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)

		round := rounds.Round{
			ID:       roundId,
			Topology: connect.NewCircuit([]*id.ID{requestGateway}),
		}
		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			Round:    round,
			Identity: identity,
		}

	}()

	// Ensure the bundle was not received
	var testBundle message.Bundle
	select {
	case testBundle = <-messageBundleChan:
		t.Fatalf("Received a message bundle over the channel, process should "+
			"have quit before reception: %+v", testBundle)
	case <-time.After(30 * time.Millisecond):
	}

	// Close the process
	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}
}

// Path in which multiple error comms are encountered before a happy path comms.
func Test_manager_processMessageRetrieval_MultipleGateways(t *testing.T) {
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
	testManager.rng = fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	addChan := make(chan network.NodeGateway, 1)

	testManager.sender, _ = gateway.NewTestingSender(
		p, testManager.rng, testNdf, mockComms, testManager.session, addChan, t)

	// Create a local channel so reception is possible
	// (testManager.messageBundles is sent only via newManager call above)
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
		identity := ephemeral2.EphemeralIdentity{
			EphID:  expectedEphID,
			Source: requestGateway,
		}

		round := rounds.Round{
			ID: roundId,
			// Create a list of IDs in which some error gateways must be
			// contacted before the happy path
			Topology: connect.NewCircuit(
				[]*id.ID{errorGateway, requestGateway}),
		}

		// Send a round look up request
		testManager.lookupRoundMessages <- roundLookup{
			Round:    round,
			Identity: identity,
		}

	}()

	// Receive the bundle over the channel
	var testBundle message.Bundle
	select {
	case testBundle = <-messageBundleChan:
	case <-time.After(300 * time.Millisecond):
		t.Errorf("Timed out waiting for messageBundleChan.")
	}

	// Close the process
	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	// Ensure that expected bundle is still received from happy comm despite
	// initial errors
	if reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Fatal("Did not receive a message bundle over the channel.")
	}

	if testBundle.Identity.EphID.Int64() != expectedEphID.Int64() {
		t.Errorf("Unexpected address ID in bundle.\nexpected: %v\nreceived: %v",
			expectedEphID, testBundle.Identity.EphID)
	}

	if !bytes.Equal(expectedPayload, testBundle.Messages[0].GetPayloadA()) {
		t.Errorf("Unexpected address ID in bundle.\nexpected: %v\nreceived: %v",
			expectedPayload, testBundle.Messages[0].GetPayloadA())
	}
}
