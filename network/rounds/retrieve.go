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
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage/reception"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
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

			// Convert gateways in round to proper ID format
			gwIds := make([]*id.ID, len(ri.Topology))
			for i, idBytes := range ri.Topology {
				gwId, err := id.Unmarshal(idBytes)
				if err != nil {
					// TODO
				}
				gwIds[i] = gwId
			}

			// Attempt to request for this gateway
			bundle, err := m.getMessagesFromGateway(id.Round(ri.ID), rl.identity, comms, gwIds)

			// After trying all gateways, if none returned we mark the round as a
			// failure and print out the last error
			if err != nil {
				jww.ERROR.Printf("Failed to get pickup round %d "+
					"from all gateways (%v): %s",
					id.Round(ri.ID), gwIds, err)
			}

			if len(bundle.Messages) != 0 {
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

	// Send to the gateways using backup proxies
	result, err := m.sender.SendToPreferred(gwIds, func(host *connect.Host, target *id.ID) (interface{}, error) {
		jww.DEBUG.Printf("Trying to get messages for round %v for ephmeralID %d (%v)  "+
			"via Gateway: %s", roundID, identity.EphId.Int64(), identity.Source.String(), host.GetId())

		// send the request
		msgReq := &pb.GetMessages{
			ClientID: identity.EphId[:],
			RoundID:  uint64(roundID),
			Target:   target.Marshal(),
		}

		return comms.RequestMessages(host, msgReq)
	})

	// Fail the round if an error occurs so it can be tried again later
	if err != nil {
		return message.Bundle{}, errors.WithMessagef(err, "Failed to "+
			"request messages for round %d", roundID)
	}
	msgResp := result.(*pb.GetMessagesResponse)
	// if the gateway doesnt have the round, return an error
	if !msgResp.GetHasRound() {
		return message.Bundle{}, errors.Errorf(noRoundError)
	}

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

	jww.INFO.Printf("Received %d messages in Round %v for %d (%s)",
		len(msgs), roundID, identity.EphId.Int64(), identity.Source)

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
