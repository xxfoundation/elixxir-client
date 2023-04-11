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
	"reflect"
	"testing"
	"time"
)

// Happy path.
func Test_manager_processBatchMessageRetrieval(t *testing.T) {
	// General initializations
	testManager := newManager(t)
	testManager.receivedResponses = make(chan *responsePart, 3)

	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	stop := stoppable.NewMulti("multiStoppable")
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
	retStop := stoppable.NewSingle("singleStoppableRet")
	go testManager.processBatchMessageRetrieval(mockComms, retStop)
	stop.Add(retStop)
	respStop := stoppable.NewSingle("singleStoppableResp")
	go testManager.processBatchMessageResponse(respStop)
	stop.Add(respStop)

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	payloadMsg := []byte(PayloadMessage)
	expectedPayload := make([]byte, 256)
	copy(expectedPayload, payloadMsg)

	go func() {
		requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)

		// Construct the round lookup
		ephIdentity := ephemeral2.EphemeralIdentity{
			EphId:  expectedEphID,
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

	// Receive the bundle over the channel
	var testBundle message.Bundle
	select {
	case testBundle = <-messageBundleChan:
	case <-time.After(300 * time.Millisecond):
		t.Errorf("Timed out waiting for messageBundleChan.")
	}

	// Close the process
	if err = stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	// Ensure bundle received and has expected values
	if reflect.DeepEqual(testBundle, message.Bundle{}) {
		t.Fatal("Did not receive a message bundle over the channel")
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
