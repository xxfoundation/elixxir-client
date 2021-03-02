///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage/reception"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strings"
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

const noRoundError = "does not have round"

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
			var bundle message.Bundle

			// Get a shuffled list of gateways in the round
			gwHosts, err := gateway.GetAllShuffled(comms, ri)
			if err != nil {
				jww.WARN.Printf("Failed to get gateway hosts from "+
					"round %v, not requesting from them",
					ri.ID)
				break
			}

			// Attempt to request messages for every gateway in the list.
			// If we retrieve without error, then we exit. If we error, then
			// we retry with the next gateway in the list until we exhaust the list
			for i, gwHost := range gwHosts {
				// Attempt to request for this gateway
				bundle, err = m.getMessagesFromGateway(id.Round(ri.ID), rl.identity, comms, gwHost)
				if err != nil {

					// If the round is not in the gateway, this is an error
					// in which there are no retries
					if strings.Contains(err.Error(), noRoundError) {
						jww.WARN.Printf("Failed to get messages for round %v: %s",
							ri.ID, err)
						break
					}

					jww.WARN.Printf("Failed on gateway [%d/%d] to get messages for round %v",
						i, len(gwHosts), ri.ID)

					// Retry for the next gateway in the list
					continue
				}

				// If a non-error request, no longer retry
				break

			}
			if err != nil {
				m.p.Fail(id.Round(ri.ID), rl.identity.EphId, rl.identity.Source)

			}

			if err == nil && len(bundle.Messages) != 0 {
				bundle.Identity = rl.identity
				m.messageBundles <- bundle
			}
		}
	}
}

// getMessagesFromGateway attempts to get messages from their assigned
// gateway host in the round specified. If successful
func (m *Manager) getMessagesFromGateway(roundID id.Round, identity reception.IdentityUse,
	comms messageRetrievalComms, gwHost *connect.Host) (message.Bundle, error) {

	jww.DEBUG.Printf("Trying to get messages for RoundID %v for EphID %d "+
		"via Gateway: %s", roundID, identity.EphId, gwHost.GetId())

	// send the request
	msgReq := &pb.GetMessages{
		ClientID: identity.EphId[:],
		RoundID:  uint64(roundID),
	}
	msgResp, err := comms.RequestMessages(gwHost, msgReq)
	// Fail the round if an error occurs so it can be tried again later
	if err != nil {
		return message.Bundle{}, errors.WithMessagef(err, "Failed to "+
			"request messages from %s for round %d", gwHost.GetId(), roundID)
	}
	// if the gateway doesnt have the round, return an error
	if !msgResp.GetHasRound() {
		m.p.Done(roundID, identity.EphId, identity.Source)
		return message.Bundle{}, errors.Errorf(noRoundError)
	}

	// If there are no messages print a warning. Due to the probabilistic nature
	// of the bloom filters, false positives will happen some times
	msgs := msgResp.GetMessages()
	if msgs == nil || len(msgs) == 0 {
		jww.WARN.Printf("host %s has no messages for client %s "+
			" in round %d. This happening every once in a while is normal,"+
			" but can be indicitive of a problem if it is consistant", gwHost,
			m.TransmissionID, roundID)
		return message.Bundle{}, nil
	}

	jww.INFO.Printf("Received %d messages in Round %v via Gateway: %s",
		len(msgs), roundID, gwHost.GetId())

	//build the bundle of messages to send to the message processor
	bundle := message.Bundle{
		Round:    roundID,
		Messages: make([]format.Message, len(msgs)),
		Finish: func() {
			m.p.Done(roundID, identity.EphId, identity.Source)
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
