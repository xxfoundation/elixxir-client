////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package pickup

import (
	"encoding/binary"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/gateway"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

type MessageRetrievalComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	RequestMessages(host *connect.Host, message *pb.GetMessages) (
		*pb.GetMessagesResponse, error)
	RequestBatchMessages(host *connect.Host,
		message *pb.GetMessagesBatch) (*pb.GetMessagesResponseBatch, error)
}

type roundLookup struct {
	Round    rounds.Round
	Identity receptionID.EphemeralIdentity
}

const noRoundError = "does not have round %d"

// processMessageRetrieval receives a roundLookup request and pings the gateways
// of that round for messages for the requested Identity in the roundLookup.
func (m *pickup) processMessageRetrieval(comms MessageRetrievalComms,
	stop *stoppable.Single) {

	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case rl := <-m.lookupRoundMessages:
			ri := rl.Round
			jww.DEBUG.Printf("[processMessageRetrieval] Checking for messages in round %d", ri.ID)
			err := m.unchecked.AddRound(id.Round(ri.ID), ri.Raw,
				rl.Identity.Source, rl.Identity.EphId)
			if err != nil {
				jww.FATAL.Panicf(
					"Failed to denote Unchecked Round for round %d",
					id.Round(ri.ID))
			}

			gwIds := m.getGatewayList(rl)
			// If ForceMessagePickupRetry, we are forcing processUncheckedRounds
			// by randomly not picking up messages (FOR INTEGRATION TEST). Only
			// done if round has not been ignored before.
			var bundle message.Bundle
			if m.params.ForceMessagePickupRetry {
				bundle, err = m.forceMessagePickupRetry(
					ri, rl, comms, gwIds, stop)

				// Exit if the thread has been stopped
				if stoppable.CheckErr(err) {
					jww.ERROR.Print(err)
					continue
				}
				if err != nil {
					jww.ERROR.Printf("[processMessageRetrieval] Failed to get pickup round %d from all "+
						"gateways (%v): %s", ri.ID, gwIds, err)
				}
			} else {
				// Attempt to request for this gateway
				bundle, err = m.getMessagesFromGateway(
					id.Round(ri.ID), rl.Identity, comms, gwIds, stop)

				// Exit if the thread has been stopped
				if stoppable.CheckErr(err) {
					jww.ERROR.Print(err)
					continue
				}

				// After trying all gateways, if none returned we mark the round
				// as a failure and print out the last error
				if err != nil {
					jww.ERROR.Printf("[processMessageRetrieval] Failed to get pickup round %d "+
						"from all gateways (%v): %s", rl.Round.ID, gwIds, err)
				}
			}

			m.processBundle(bundle, rl.Identity, rl.Round)
		}
	}
}

// getGatewayList returns a shuffled list of gateways for a roundLookup request.
func (m *pickup) getGatewayList(rl roundLookup) []*id.ID {
	ri := rl.Round

	// Convert gateways in round to proper ID format
	gwIds := make([]*id.ID, ri.Topology.Len())
	for i := 0; i < ri.Topology.Len(); i++ {
		gwId := ri.Topology.GetNodeAtIndex(i).DeepCopy()
		gwId.SetType(id.Gateway)
		gwIds[i] = gwId
	}

	if len(gwIds) == 0 {
		jww.WARN.Printf("Empty gateway ID List")
		return nil
	}

	// Target the last nodes in the team first because it has messages
	// first, randomize other members of the team
	var rndBytes [32]byte
	stream := m.rng.GetStream()
	_, err := stream.Read(rndBytes[:])
	stream.Close()
	if err != nil {
		jww.FATAL.Panicf("Failed to randomize shuffle in round %d "+
			"from all gateways (%v): %s", ri.ID, gwIds, err)
	}

	gwIds[0], gwIds[len(gwIds)-1] = gwIds[len(gwIds)-1], gwIds[0]
	shuffle.ShuffleSwap(rndBytes[:], len(gwIds)-1, func(i, j int) {
		gwIds[i+1], gwIds[j+1] = gwIds[j+1], gwIds[i+1]
	})

	return gwIds
}

// getMessagesFromGateway attempts to pick up messages from their assigned
// gateway in the round specified. If successful, it returns a message.Bundle.
func (m *pickup) getMessagesFromGateway(roundID id.Round,
	identity receptionID.EphemeralIdentity, comms MessageRetrievalComms,
	gwIds []*id.ID, stop *stoppable.Single) (message.Bundle, error) {
	start := netTime.Now()
	// Send to the gateways using backup proxies
	result, err := m.sender.SendToPreferred(gwIds,
		func(host *connect.Host, target *id.ID, _ time.Duration) (interface{}, error) {
			jww.DEBUG.Printf("Trying to get messages for round %d for "+
				"ephemeralID %d (%s) via Gateway: %s", roundID,
				identity.EphId.Int64(), identity.Source, host.GetId())

			// send the request
			msgReq := &pb.GetMessages{
				ClientID: identity.EphId[:],
				RoundID:  uint64(roundID),
				Target:   target.Marshal(),
			}

			// If the gateway doesn't have the round, return an error
			msgResp, err := comms.RequestMessages(host, msgReq)

			if err != nil {
				// You need to default to a retryable errors because otherwise
				// we cannot enumerate all errors
				return nil, errors.WithMessage(err, gateway.RetryableError)
			}

			if !msgResp.GetHasRound() {
				errRtn := errors.Errorf(noRoundError, roundID)
				return message.Bundle{},
					errors.WithMessage(errRtn, gateway.RetryableError)
			}

			return msgResp, nil
		}, stop, m.params.SendTimeout)

	// Fail the round if an error occurs so that it can be tried again later
	if err != nil {
		return message.Bundle{}, errors.WithMessagef(
			err, "Failed to request messages for round %d", roundID)
	}
	msgResp := result.(*pb.GetMessagesResponse)

	bundle, err := m.buildMessageBundle(msgResp, identity, roundID)
	if err != nil {
		return message.Bundle{}, errors.WithMessagef(err, "Failed to process pickup response for round %d", roundID)
	}

	jww.INFO.Printf("Received %d messages in Round %d for %d (%s) in %s",
		len(bundle.Messages), roundID, identity.EphId.Int64(), identity.Source,
		netTime.Now().Sub(start))

	return bundle, nil
}

// processBundle accepts a message.Bundle, EphemeralIdentity and round ID.
// If the bundle contains any messages, it iterates through them, sending
// them to the bundle channel for handling, and removing the associated
// rounds from m.unchecked.
func (m *pickup) processBundle(bundle message.Bundle, rid receptionID.EphemeralIdentity, ri rounds.Round) {
	jww.TRACE.Printf("messages: %v\n", bundle.Messages)

	if len(bundle.Messages) != 0 {
		// If successful and there are messages, we send them to another
		// thread
		bundle.Identity = receptionID.EphemeralIdentity{
			EphId:  rid.EphId,
			Source: rid.Source,
		}
		bundle.RoundInfo = ri
		m.messageBundles <- bundle

		jww.DEBUG.Printf("Removing round %d from unchecked store", ri.ID)
		err := m.unchecked.Remove(
			id.Round(ri.ID), rid.Source, rid.EphId)
		if err != nil {
			jww.ERROR.Printf("Could not remove round %d from "+
				"unchecked rounds store: %v", ri.ID, err)
		}
	}
}

// buildMessageBundle builds a message.Bundle from a passed in
// pb.GetMessagesResponse, EphemeralIdentity and round ID.
func (m *pickup) buildMessageBundle(msgResp *pb.GetMessagesResponse, identity receptionID.EphemeralIdentity, roundID id.Round) (message.Bundle, error) {
	// If there are no messages, print a warning. Due to the probabilistic
	// nature of the bloom filters, false positives will happen sometimes
	msgs := msgResp.GetMessages()
	if len(msgs) == 0 {
		jww.WARN.Printf("no messages for client %s "+
			" in round %d. This happening every once in a while is normal,"+
			" but can be indicative of a problem if it is consistent",
			identity.Source, roundID)

		err := m.unchecked.EndCheck(roundID, identity.Source, identity.EphId)
		if err != nil {
			jww.ERROR.Printf("Failed to end the check for the round round %d: %+v", roundID, err)
		}

		return message.Bundle{}, nil
	}

	// Build the bundle of messages to send to the message processor
	bundle := message.Bundle{
		Round:    roundID,
		Messages: make([]format.Message, len(msgs)),
		Finish:   func() {},
	}

	mSize := m.session.GetCmixGroup().GetP().ByteLen()
	for i, slot := range msgs {
		msg := format.NewMessage(mSize)
		msg.SetPayloadA(slot.PayloadA)
		msg.SetPayloadB(slot.PayloadB)
		jww.INFO.Printf("Received message of msgDigest: %s, round %d",
			msg.Digest(), roundID)
		bundle.Messages[i] = msg
	}
	return bundle, nil
}

// forceMessagePickupRetry is a helper function which forces
// processUncheckedRounds by randomly not looking up messages.
func (m *pickup) forceMessagePickupRetry(ri rounds.Round, rl roundLookup,
	comms MessageRetrievalComms, gwIds []*id.ID,
	stop *stoppable.Single) (bundle message.Bundle, err error) {
	if m.shouldForceMessagePickupRetry(rl) {
		// Do not call get message, leaving the round to be picked up in
		// unchecked round scheduler process
		return
	}

	// Attempt to request for this gateway
	return m.getMessagesFromGateway(
		ri.ID, rl.Identity, comms, gwIds, stop)
}

// shouldForceMessagePickupRetry randomly determines if a roundLookup
// should be skipped to force a retry.
func (m *pickup) shouldForceMessagePickupRetry(rl roundLookup) bool {
	rnd, _ := m.unchecked.GetRound(
		rl.Round.ID, rl.Identity.Source, rl.Identity.EphId)
	var err error
	if rnd.NumChecks == 0 {
		// Flip a coin to determine whether to pick up message
		b := make([]byte, 8)
		stream := m.rng.GetStream()
		_, err = stream.Read(b)
		if err != nil {
			jww.FATAL.Panic(err)
		}
		stream.Close()

		result := binary.BigEndian.Uint64(b)
		if result%2 == 0 {
			jww.INFO.Printf("Forcing a message pickup retry for round %d", rl.Round.ID)
			return true
		}
	}
	return false
}
