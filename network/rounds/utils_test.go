///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package rounds

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func newManager(face interface{}) *Manager {
	sess1 := storage.InitTestingSession(face)

	testManager := &Manager{
		lookupRoundMessages: make(chan roundLookup),
		messageBundles:      make(chan message.Bundle),
		p:                   newProcessingRounds(),
		Internal: internal.Internal{
			Session:        sess1,
			TransmissionID: sess1.GetUser().TransmissionID,
		},
	}

	return testManager
}

// Build ID off of this string for expected gateway
// which will return on over mock comm
const ReturningGateway = "GetMessageRequest"
const FalsePositive = "FalsePositive"
const PayloadMessage = "Payload"
const ErrorGateway = "Error"

type mockMessageRetrievalComms struct {
	testingSignature *testing.T
}

func (mmrc *mockMessageRetrievalComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	h, _ := connect.NewHost(hostId, "0.0.0.0", []byte(""), connect.HostParams{
		MaxRetries:  0,
		AuthEnabled: false,
	})
	return h, true
}

// Mock comm which returns differently based on the host ID
// ReturningGateway returns a happy path response, in which there is a message
// FalsePositive returns a response in which there were no messages in the round
// ErrorGateway returns an error on the mock comm
// Any other ID returns default no round errors
func (mmrc *mockMessageRetrievalComms) RequestMessages(host *connect.Host,
	message *pb.GetMessages) (*pb.GetMessagesResponse, error) {
	payloadMsg := []byte(PayloadMessage)
	payload := make([]byte, 256)
	copy(payload, payloadMsg)
	testSlot := &pb.Slot{
		PayloadA: payload,
		PayloadB: payload,
	}

	// If we are the requesting on the returning gateway, return a mock response
	returningGateway := id.NewIdFromString(ReturningGateway, id.Gateway, mmrc.testingSignature)
	if host.GetId().Cmp(returningGateway) {
		return &pb.GetMessagesResponse{
			Messages: []*pb.Slot{testSlot},
			HasRound: true,
		}, nil
	}

	// Return an empty message structure (ie a false positive in the bloom filter)
	falsePositive := id.NewIdFromString(FalsePositive, id.Gateway, mmrc.testingSignature)
	if host.GetId().Cmp(falsePositive) {
		return &pb.GetMessagesResponse{
			Messages: []*pb.Slot{},
			HasRound: true,
		}, nil
	}

	// Return a mock error
	errorGateway := id.NewIdFromString(ErrorGateway, id.Gateway, mmrc.testingSignature)
	if host.GetId().Cmp(errorGateway) {
		return &pb.GetMessagesResponse{}, errors.Errorf("Connection error")
	}

	return nil, nil
}
