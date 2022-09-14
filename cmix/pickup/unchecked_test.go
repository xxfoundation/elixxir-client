////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package pickup

import (
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"testing"
	"time"
)

// Happy path.
func TestUncheckedRoundScheduler(t *testing.T) {
	// General initializations
	connect.TestingOnlyDisableTLS = true
	testManager := newManager(t)
	roundId := id.Round(5)
	mockComms := &mockMessageRetrievalComms{testingSignature: t}
	stop1 := stoppable.NewSingle("singleStoppable1")
	stop2 := stoppable.NewSingle("singleStoppable2")
	testNdf := getNDF()
	nodeId := id.NewIdFromString(ReturningGateway, id.Node, &testing.T{})
	gwId := nodeId.DeepCopy()
	gwId.SetType(id.Gateway)
	testNdf.Gateways = []ndf.Gateway{{ID: gwId.Marshal()}}
	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testManager.sender, _ = gateway.NewSender(
		p, rngGen, testNdf, mockComms, testManager.session, nil)

	// Create a local channel so reception is possible
	// (testManager.messageBundles is sent only via newManager call above)
	messageBundleChan := make(chan message.Bundle)
	testManager.messageBundles = messageBundleChan

	testBackoffTable := newTestBackoffTable(t)
	checkInterval := 250 * time.Millisecond
	// Initialize the message retrieval
	go testManager.processMessageRetrieval(mockComms, stop1)
	go testManager.processUncheckedRounds(checkInterval, testBackoffTable, stop2)

	requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)

	// Construct expected values for checking
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	idList := [][]byte{requestGateway.Bytes()}
	roundInfo := &pb.RoundInfo{
		ID:       uint64(roundId),
		Topology: idList,
	}

	// Add round to check
	err := testManager.unchecked.AddRound(
		roundId, roundInfo, requestGateway, expectedEphID)
	if err != nil {
		t.Fatalf("Could not add round to session: %v", err)
	}

	var testBundle message.Bundle
	select {
	case testBundle = <-messageBundleChan:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Did not receive a message bundle over the channel")
	}

	// Close the process
	if err = stop1.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}
	if err = stop2.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	if testBundle.Identity.EphId.Int64() != expectedEphID.Int64() {
		t.Errorf("Unexpected address ID in bundle.\nexpected: %v\nreceived: %v",
			expectedEphID, testBundle.Identity.EphId)
	}
}
