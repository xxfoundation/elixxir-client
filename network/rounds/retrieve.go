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

func (m *Manager) processMessageRetrieval(comms messageRetrievalComms,
	quitCh <-chan struct{}) {

	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case rl := <-m.lookupRoundMessages:
			ri := rl.roundInfo
			bundle, err := m.getMessagesFromGateway(ri, comms, rl.identity)
			if err != nil {
				jww.WARN.Printf("Failed to get messages for round %v: %s",
					ri.ID, err)
				break
			}

			if len(bundle.Messages) != 0 {
				bundle.Identity = rl.identity
				m.messageBundles <- bundle
			}
		}
	}
}

func (m *Manager) getMessagesFromGateway(roundInfo *pb.RoundInfo,
	comms messageRetrievalComms, identity reception.IdentityUse) (message.Bundle, error) {

	rid := id.Round(roundInfo.ID)

	//Get the host object for the gateway to send to
	gwHost, err := gateway.GetLast(comms, roundInfo)
	if err != nil {
		return message.Bundle{}, errors.WithMessage(err, "Failed to get Gateway "+
			"to request from")
	}

	jww.INFO.Printf("Getting messages for RoundID %v for EphID %d "+
		"via Gateway: %s", rid, identity.EphId, gwHost.GetId())

	// send the request
	msgReq := &pb.GetMessages{
		ClientID: identity.EphId[:],
		RoundID:  uint64(rid),
	}
	msgResp, err := comms.RequestMessages(gwHost, msgReq)
	// Fail the round if an error occurs so it can be tried again later
	if err != nil {
		m.p.Fail(id.Round(roundInfo.ID), identity.EphId, identity.Source)
		return message.Bundle{}, errors.WithMessagef(err, "Failed to "+
			"request messages from %s for round %d", gwHost.GetId(), rid)
	}
	// if the gateway doesnt have the round, return an error
	if !msgResp.GetHasRound() {
		m.p.Done(id.Round(roundInfo.ID), identity.EphId, identity.Source)
		return message.Bundle{}, errors.Errorf("host %s does not have "+
			"roundID: %d", gwHost.String(), rid)
	}

	// If there are no messages print a warning. Due to the probabilistic nature
	// of the bloom filters, false positives will happen some times
	msgs := msgResp.GetMessages()
	if msgs == nil || len(msgs) == 0 {
		jww.WARN.Printf("host %s has no messages for client %s "+
			" in round %d. This happening every once in a while is normal,"+
			" but can be indicitive of a problem if it is consistant", gwHost,
			m.TransmissionID, rid)
		return message.Bundle{}, nil
	}

	jww.INFO.Printf("Received %d messages in Round %v via Gateway %s for %d (%s)",
		len(msgs), rid, gwHost.GetId(), identity.EphId.Int64(), identity.Source)

	//build the bundle of messages to send to the message processor
	bundle := message.Bundle{
		Round:    rid,
		Messages: make([]format.Message, len(msgs)),
		Finish: func() {
			m.p.Done(rid, identity.EphId, identity.Source)
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
