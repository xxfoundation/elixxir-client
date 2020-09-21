package rounds

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/xx_network/primitives/id"
	pb "gitlab.com/elixxir/comms/mixmessages"
)
}




func (m *Manager) getMessagesFromGateway(roundInfo *pb.RoundInfo) ([]*pb.Slot, error) {

	gwHost, err := gateway.GetLast(m.comms, roundInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get Gateway "+
			"to request from")
	}

	user := m.session.User().GetCryptographicIdentity()
	userID := user.GetUserID().Bytes()

	// First get message id list
	msgReq := &pb.GetMessages{
		ClientID: userID,
		RoundID:  roundInfo.ID,
	}
	msgResp, err := m.comms.RequestMessages(gwHost, msgReq)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to request "+
			"messages from %s for round %s", gwHost.GetId(), roundInfo.ID)
	}

	// If no error, then we have checked the round and finished processing
	ctx.Session.GetCheckedRounds.Check(roundInfo.ID)
	network.Processing.Done(roundInfo.ID)

	if !msgResp.GetHasRound() {
		jww.ERROR.Printf("host %s does not have roundID: %d",
			gwHost, roundInfo.ID)
		return nil
	}

	msgs := msgResp.GetMessages()

	if msgs == nil || len(msgs) == 0 {
		jww.ERROR.Printf("host %s has no messages for client %s "+
			" in round %d", gwHost, user, roundInfo.ID)
		return nil
	}

	return msgs

}
