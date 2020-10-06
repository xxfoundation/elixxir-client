package rounds

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/message"
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

func (m *Manager) processMessageRetrieval(comms messageRetrievalComms,
	quitCh <-chan struct{}) {

	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case ri := <-m.lookupRoundMessages:
			bundle, err := m.getMessagesFromGateway(ri, comms)
			if err != nil {
				jww.WARN.Printf("Failed to get messages for round %v: %s",
					ri.ID, err)
				break
			}
			if len(bundle.Messages) != 0 {
				m.messageBundles <- bundle
			}
		}
	}
}

func (m *Manager) getMessagesFromGateway(roundInfo *pb.RoundInfo,
	comms messageRetrievalComms) (message.Bundle, error) {

	rid := id.Round(roundInfo.ID)

	//Get the host object for the gateway to send to
	gwHost, err := gateway.GetLast(comms, roundInfo)
	if err != nil {
		return message.Bundle{}, errors.WithMessage(err, "Failed to get Gateway "+
			"to request from")
	}

	jww.INFO.Printf("Getting messages for RoundID %v via Gateway: %s", rid,
		gwHost.GetId())

	// send the request
	msgReq := &pb.GetMessages{
		ClientID: m.Uid.Marshal(),
		RoundID:  uint64(rid),
	}
	msgResp, err := comms.RequestMessages(gwHost, msgReq)
	// Fail the round if an error occurs so it can be tried again later
	if err != nil {
		m.p.Fail(id.Round(roundInfo.ID))
		return message.Bundle{}, errors.WithMessagef(err, "Failed to "+
			"request messages from %s for round %d", gwHost.GetId(), rid)
	}
	// if the gateway doesnt have the round, return an error
	if !msgResp.GetHasRound() {
		m.p.Done(rid)
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
			m.Uid, rid)
		return message.Bundle{}, nil
	}

	//build the bundle of messages to send to the message processor
	bundle := message.Bundle{
		Round:    rid,
		Messages: make([]format.Message, len(msgs)),
		Finish: func() {
			m.Session.GetCheckedRounds().Check(rid)
			m.p.Done(rid)
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
