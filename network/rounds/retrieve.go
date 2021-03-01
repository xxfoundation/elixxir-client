///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage/reception"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

type messageRetrievalComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	RequestMessages(host *connect.Host,
		message *pb.GetMessages) (*pb.GetMessagesResponse, error)
}

type roundLookup struct {
	roundInfo *pb.RoundInfo
	identity  reception.IdentityUse
}

// processMessageRetrieval received a roundLookup request and pings the gateways
// of that round for messages for the requested identity in the roundLookup
func (m *Manager) processMessageRetrieval(comms messageRetrievalComms,
	quitCh <-chan struct{}) {

	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case rl := <-m.lookupRoundMessages:
			ri := rl.roundInfo
			// Data channel for first message poller thread to
			// which gets a successful comm from it's gateway
			// to send to the main thread
			bundleChan := make(chan message.Bundle)

			// Acts as an additional signal channel
			// which will close the other message poller threads once
			// the main thread receives it's message bundle
			stopChan := make(chan struct{})
			for index := 0; index < len(ri.Topology); index++ {
				// Initiate message pollers that each repeatedly request one gateway in
				// the round for a message bundle, until a successful communication completes
				// from any of the message pollers
				go func(localIndex int) {
					gwHost, err := gateway.GetFromIndex(comms, ri, localIndex)
					if err != nil {
						jww.WARN.Printf("Failed to get %d/%d gateway from round %v, not requesting from them",
							localIndex, len(ri.Topology), ri.ID)
						return
					}

					m.getMessagesFromGateway(id.Round(ri.ID), rl.identity.EphId,
						comms, gwHost, bundleChan, stopChan)

				}(index)
			}

			// Block until we receive the bundle from the
			// first successful gateway message poller.
			// Signals to all other message pollers that they
			// can return by closing the stop channel
			bundle := <-bundleChan
			close(stopChan)
			if len(bundle.Messages) != 0 {
				bundle.Identity = rl.identity
				m.messageBundles <- bundle
			}
		}
	}
}

// getMessagesFromGateway repeatedly attempts to get messages from their assigned
// gateway host in the round specified. If this running thread is successful,
// it sends through the message bundle to the receiver. If the receiver has received
// a bundle from another thread, this thread instead closes without sending a bundle.
func (m *Manager) getMessagesFromGateway(roundID id.Round, ephid ephemeral.Id,
	comms messageRetrievalComms, gwHost *connect.Host,
	bundleChan chan<- message.Bundle, stopSignal <-chan struct{}) {

	var messageResponse *pb.GetMessagesResponse
	var bundle message.Bundle
	var err error
	jww.DEBUG.Printf("Trying to get messages for RoundID %v for EphID %d "+
		"via Gateway: %s", roundID, ephid, gwHost.GetId())

	for messageResponse == nil {
		// The try-receive operation is to try to exit the goroutine as
		// early as possible, without continuously polling it's gateway
		select {
		case <-stopSignal:
			return
		default:
		}

		// send the request
		msgReq := &pb.GetMessages{
			ClientID: ephid[:],
			RoundID:  uint64(roundID),
		}
		messageResponse, err = comms.RequestMessages(gwHost, msgReq)
		// Retry the request on an error on the comm
		if err != nil {
			continue
		}

		// if the gateway doesnt have the round, break out of the request loop
		// so that this and all other threads have closed
		if !messageResponse.GetHasRound() {
			m.p.Done(roundID)
			m.Session.GetCheckedRounds().Check(roundID)
			jww.WARN.Printf("Failed to get messages for round %v: host %s does not have "+
				"roundID: %d",
				roundID, gwHost, roundID)
			break
		}

		// If there are no messages print a warning. Due to the probabilistic nature
		// of the bloom filters, false positives will happen some times
		msgs := messageResponse.GetMessages()
		if msgs == nil || len(msgs) == 0 {
			jww.WARN.Printf("Failed to get messages for round: "+
				"host %s has no messages for client %s "+
				" in round %d. This happening every once in a while is normal,"+
				" but can be indicitive of a problem if it is consistant", gwHost,
				m.TransmissionID, roundID)
			// In case of a false positive, exit loop so that this
			// and all other threads may close
			break
		}

		//build the bundle of messages to send to the message processor
		bundle = message.Bundle{
			Round:    roundID,
			Messages: make([]format.Message, len(msgs)),
			Finish: func() {
				m.Session.GetCheckedRounds().Check(roundID)
				m.p.Done(roundID)
			},
		}

		for i, slot := range msgs {
			msg := format.NewMessage(m.Session.Cmix().GetGroup().GetP().ByteLen())
			msg.SetPayloadA(slot.PayloadA)
			msg.SetPayloadB(slot.PayloadB)
			bundle.Messages[i] = msg
		}

	}

	// The try-receive operation stops from returning if another
	// poller has already sent their message bundle through the line.
	// Otherwise, it sends the bundle across to the receiver
	select {
	case <-stopSignal:
		return
	case bundleChan <- bundle:
		jww.INFO.Printf("Received %d messages in Round %v via Gateway: %s",
			len(bundle.Messages), roundID, gwHost.GetId())
		return
	}
}
