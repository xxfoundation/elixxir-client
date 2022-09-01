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
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/stoppable"
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
}

type roundLookup struct {
	Round    rounds.Round
	Identity receptionID.EphemeralIdentity
}

const noRoundError = "does not have round %d"

// processMessageRetrieval received a roundLookup request and pings the gateways
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
			jww.DEBUG.Printf("Checking for messages in round %d", ri.ID)

			if !m.params.RealtimeOnly {
				err := m.unchecked.AddRound(id.Round(ri.ID), ri.Raw,
					rl.Identity.Source, rl.Identity.EphId)
				if err != nil {
					jww.FATAL.Panicf(
						"Failed to denote Unchecked Round for round %d",
						id.Round(ri.ID))
				}
			}

			// Convert gateways in round to proper ID format
			gwIds := make([]*id.ID, ri.Topology.Len())
			for i := 0; i < ri.Topology.Len(); i++ {
				gwId := ri.Topology.GetNodeAtIndex(i).DeepCopy()
				gwId.SetType(id.Gateway)
				gwIds[i] = gwId
			}

			if len(gwIds) == 0 {
				jww.WARN.Printf("Empty gateway ID List")
				continue
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
					jww.ERROR.Printf("Failed to get pickup round %d from all "+
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
					jww.ERROR.Printf("Failed to get pickup round %d "+
						"from all gateways (%v): %s", ri.ID, gwIds, err)
				}

			}

			jww.TRACE.Printf("messages: %v\n", bundle.Messages)

			if len(bundle.Messages) != 0 {
				// If successful and there are messages, we send them to another
				// thread
				bundle.Identity = receptionID.EphemeralIdentity{
					EphId:  rl.Identity.EphId,
					Source: rl.Identity.Source,
				}
				bundle.RoundInfo = rl.Round
				m.messageBundles <- bundle

				jww.DEBUG.Printf("Removing round %d from unchecked store", ri.ID)
				if !m.params.RealtimeOnly {
					err = m.unchecked.Remove(
						id.Round(ri.ID), rl.Identity.Source, rl.Identity.EphId)
					if err != nil {
						jww.ERROR.Printf("Could not remove round %d from "+
							"unchecked rounds store: %v", ri.ID, err)
					}
				}

			}

		}
	}
}

// getMessagesFromGateway attempts to get messages from their assigned gateway
// host in the round specified. If successful
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

	jww.INFO.Printf("Received messages for round %d, processing...", roundID)

	// Fail the round if an error occurs so that it can be tried again later
	if err != nil {
		return message.Bundle{}, errors.WithMessagef(
			err, "Failed to request messages for round %d", roundID)
	}
	msgResp := result.(*pb.GetMessagesResponse)

	// If there are no messages, print a warning. Due to the probabilistic
	// nature of the bloom filters, false positives will happen sometimes
	msgs := msgResp.GetMessages()
	if len(msgs) == 0 {
		jww.WARN.Printf("no messages for client %s "+
			" in round %d. This happening every once in a while is normal,"+
			" but can be indicative of a problem if it is consistent",
			identity.Source, roundID)
		if m.params.RealtimeOnly {
			err = m.unchecked.Remove(roundID, identity.Source, identity.EphId)
			if err != nil {
				jww.ERROR.Printf("Failed to remove round %d: %+v", roundID, err)
			}
		}

		return message.Bundle{}, nil
	}

	jww.INFO.Printf("Received %d messages in Round %d for %d (%s) in %s",
		len(msgs), roundID, identity.EphId.Int64(), identity.Source,
		netTime.Now().Sub(start))

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

// Helper function which forces processUncheckedRounds by randomly not looking
// up messages.
func (m *pickup) forceMessagePickupRetry(ri rounds.Round, rl roundLookup,
	comms MessageRetrievalComms, gwIds []*id.ID,
	stop *stoppable.Single) (bundle message.Bundle, err error) {
	rnd, _ := m.unchecked.GetRound(
		ri.ID, rl.Identity.Source, rl.Identity.EphId)
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
			jww.INFO.Printf("Forcing a message pickup retry for round %d", ri.ID)
			// Do not call get message, leaving the round to be picked up in
			// unchecked round scheduler process
			return
		}

	}

	// Attempt to request for this gateway
	return m.getMessagesFromGateway(
		ri.ID, rl.Identity, comms, gwIds, stop)
}
