package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
)

type confirmFactComm interface {
	SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error)
}

func (m *Manager) SendConfirmFact(confirmationID, code string) (*messages.Ack, error) {
	return m.confirmFact(confirmationID, code, m.comms)
}

func (m *Manager) confirmFact(confirmationID, code string, comm confirmFactComm) (*messages.Ack, error) {
	msg := &pb.FactConfirmRequest{
		ConfirmationID: confirmationID,
		Code:           code,
	}

	return comm.SendConfirmFact(m.host, msg)
}
