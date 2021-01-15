package ud

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	jww "github.com/spf13/jwalterweatherman"
)

type confirmFactComm interface {
	SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error)
}

// Confirms a fact first registered via AddFact. The confirmation ID comes from
// AddFact while the code will come over the associated communications system
func (m *Manager) SendConfirmFact(confirmationID, code string) error {
	jww.INFO.Printf("ud.SendConfirmFact(%s, %s)", confirmationID, code)
	if err := m.confirmFact(confirmationID, code, m.comms); err!=nil{
		return errors.WithMessage(err, "Failed to confirm fact")
	}
	return nil
}

func (m *Manager) confirmFact(confirmationID, code string, comm confirmFactComm) error {
	if !m.IsRegistered(){
		return errors.New("Failed to confirm fact: " +
			"client is not registered")
	}

	msg := &pb.FactConfirmRequest{
		ConfirmationID: confirmationID,
		Code:           code,
	}
	_, err := comm.SendConfirmFact(m.host, msg)
	return err
}
