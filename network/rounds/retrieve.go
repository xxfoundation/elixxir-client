///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage/reception"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
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

const noRoundError = "does not have round %d"

// processMessageRetrieval received a roundLookup request and pings the gateways
// of that round for messages for the requested identity in the roundLookup
func (m *Manager) processMessageRetrieval(comms messageRetrievalComms,
	quitCh <-chan struct{}) {
	forceLookupRetryTracker := make(map[uint64]struct{})
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case rl := <-m.lookupRoundMessages:
			ri := rl.roundInfo
			jww.INFO.Printf("Checking for messages in round %d", ri.ID)
			err := m.Session.UncheckedRounds().AddRound(rl.roundInfo,
				rl.identity.EphId, rl.identity.Source)
			if err != nil {
				jww.ERROR.Printf("Could not add round %d in unchecked rounds store: %v",
					rl.roundInfo.ID, err)
			}

			// Convert gateways in round to proper ID format
			gwIds := make([]*id.ID, len(ri.Topology))
			for i, idBytes := range ri.Topology {
				gwId, err := id.Unmarshal(idBytes)
				if err != nil {
					jww.FATAL.Panicf("processMessageRetrieval: Unable to unmarshal: %+v", err)
				}
				gwId.SetType(id.Gateway)
				gwIds[i] = gwId
			}
			// Target the last node in the team first because it has
			// messages first, randomize other members of the team
			var rndBytes [32]byte
			stream := m.Rng.GetStream()
			_, err = stream.Read(rndBytes[:])
			stream.Close()
			if err != nil {
				jww.FATAL.Panicf("Failed to randomize shuffle in round %d "+
					"from all gateways (%v): %s",
					id.Round(ri.ID), gwIds, err)
			}
			gwIds[0], gwIds[len(gwIds)-1] = gwIds[len(gwIds)-1], gwIds[0]
			shuffle.ShuffleSwap(rndBytes[:], len(gwIds)-1, func(i, j int) {
				gwIds[i+1], gwIds[j+1] = gwIds[j+1], gwIds[i+1]
			})

			// If ForceMessagePickupRetry, we are forcing processUncheckedRounds by
			// randomly not picking up messages (FOR INTEGRATION TEST). Only done if
			// round has not been ignored before
			var bundle message.Bundle
			_, ok := forceLookupRetryTracker[ri.ID]
			if !ok && m.params.ForceMessagePickupRetry {
				bundle, err = m.forceMessagePickupRetry(ri, rl, comms, gwIds)
				if err != nil {
					jww.ERROR.Printf("Failed to get pickup round %d "+
						"from all gateways (%v): %s",
						id.Round(ri.ID), gwIds, err)
				}
				forceLookupRetryTracker[ri.ID] = struct{}{}
				_, ok = forceLookupRetryTracker[ri.ID]
				jww.INFO.Printf("After adding round %d to tracker entry is %v", ri.ID, ok)
			} else {
				// Attempt to request for this gateway
				bundle, err = m.getMessagesFromGateway(id.Round(ri.ID), rl.identity, comms, gwIds)
				// After trying all gateways, if none returned we mark the round as a
				// failure and print out the last error
				if err != nil {
					jww.ERROR.Printf("Failed to get pickup round %d "+
						"from all gateways (%v): %s",
						id.Round(ri.ID), gwIds, err)
				}

			}

			if len(bundle.Messages) != 0 {
				jww.INFO.Printf("Removing round %d from unchecked store", ri.ID)
				err = m.Session.UncheckedRounds().Remove(id.Round(ri.ID))
				if err != nil {
					jww.ERROR.Printf("Could not remove round %d "+
						"from unchecked rounds store: %v", ri.ID, err)
				}

				// If successful and there are messages, we send them to another thread
				bundle.Identity = rl.identity
				m.messageBundles <- bundle
			}

		}
	}
}

// getMessagesFromGateway attempts to get messages from their assigned
// gateway host in the round specified. If successful
func (m *Manager) getMessagesFromGateway(roundID id.Round, identity reception.IdentityUse,
	comms messageRetrievalComms, gwIds []*id.ID) (message.Bundle, error) {
	start := time.Now()
	// Send to the gateways using backup proxies
	jww.INFO.Printf("Getting messages for round %d from %v", roundID, gwIds)
	result, err := m.sender.SendToPreferred(gwIds, func(host *connect.Host, target *id.ID) (interface{}, bool, error) {
		jww.DEBUG.Printf("Trying to get messages for round %v for ephemeralID %d (%v)  "+
			"via Gateway: %s", roundID, identity.EphId.Int64(), identity.Source.String(), host.GetId())

		// send the request
		msgReq := &pb.GetMessages{
			ClientID: identity.EphId[:],
			RoundID:  uint64(roundID),
			Target:   target.Marshal(),
		}

		// If the gateway doesnt have the round, return an error
		msgResp, err := comms.RequestMessages(host, msgReq)
		if err == nil && !msgResp.GetHasRound() {
			jww.INFO.Printf("No round error for round %d received from %s", roundID, target)
			return message.Bundle{}, false, errors.Errorf(noRoundError, roundID)
		}

		return msgResp, false, err
	})
	jww.INFO.Printf("Received message for round %d, processing...", roundID)
	// Fail the round if an error occurs so it can be tried again later
	if err != nil {
		jww.INFO.Printf("GetMessage error for round %d", roundID)
		return message.Bundle{}, errors.WithMessagef(err, "Failed to "+
			"request messages for round %d", roundID)
	}
	msgResp := result.(*pb.GetMessagesResponse)

	// If there are no messages print a warning. Due to the probabilistic nature
	// of the bloom filters, false positives will happen some times
	msgs := msgResp.GetMessages()
	if msgs == nil || len(msgs) == 0 {
		jww.WARN.Printf("no messages for client %s "+
			" in round %d. This happening every once in a while is normal,"+
			" but can be indicative of a problem if it is consistent",
			m.TransmissionID, roundID)
		return message.Bundle{}, nil
	}

	jww.INFO.Printf("Received %d messages in Round %v for %d (%s) in %s",
		len(msgs), roundID, identity.EphId.Int64(), identity.Source, time.Now().Sub(start))

	//build the bundle of messages to send to the message processor
	bundle := message.Bundle{
		Round:    roundID,
		Messages: make([]format.Message, len(msgs)),
		Finish: func() {
			return
		},
	}

	for i, slot := range msgs {
		msg := format.NewMessage(m.Session.Cmix().GetGroup().GetP().ByteLen())
		msg.SetPayloadA(slot.PayloadA)
		msg.SetPayloadB(slot.PayloadB)
		bundle.Messages[i] = msg
	}

	return bundle, nil

}

// Helper function which forces processUncheckedRounds by randomly
// not looking up messages
func (m *Manager) forceMessagePickupRetry(ri *pb.RoundInfo, rl roundLookup,
	comms messageRetrievalComms, gwIds []*id.ID) (bundle message.Bundle, err error) {
	// Flip a coin to determine whether to pick up message
	stream := m.Rng.GetStream()
	defer stream.Close()
	b := make([]byte, 8)
	_, err = stream.Read(b)
	if err != nil {
		jww.FATAL.Panic(err.Error())
	}
	result := binary.BigEndian.Uint64(b)
	jww.INFO.Printf("Random result: %d", result)
	if result%2 == 0 {
		jww.INFO.Printf("Forcing a message pickup retry for round %d", ri.ID)
		// Do not call get message, leaving the round to be picked up
		// in unchecked round scheduler process
		return
	}


	// Attempt to request for this gateway
	return m.getMessagesFromGateway(id.Round(ri.ID), rl.identity, comms, gwIds)
}
