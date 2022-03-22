package ud

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
)

type confirmFactComm interface {
	SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error)
}

// SendConfirmFact confirms a fact first registered via AddFact. The
// confirmation ID comes from AddFact while the code will come over the
// associated communications system.
func (m *Manager) SendConfirmFact(confirmationID, code string) error {
	jww.INFO.Printf("ud.SendConfirmFact(%s, %s)", confirmationID, code)
	if err := m.confirmFact(confirmationID, code, m.comms); err != nil {
		return errors.WithMessage(err, "Failed to confirm fact")
	}
	return nil
}

func (m *Manager) confirmFact(confirmationID, code string, comm confirmFactComm) error {
	if !m.IsRegistered() {
		return errors.New("Failed to confirm fact: " +
			"client is not registered")
	}

	// get UD host
	host, err := m.getHost()
	if err != nil {
		return err
	}

	msg := &pb.FactConfirmRequest{
		ConfirmationID: confirmationID,
		Code:           code,
	}
	_, err = comm.SendConfirmFact(host, msg)
	if err != nil {
		return err
	}

	err = m.storage.GetUd().ConfirmFact(confirmationID)
	if err != nil {
		return errors.WithMessagef(err, "Failed to confirm fact in storage with confirmation ID: %q", confirmationID)
	}

	return nil
}
