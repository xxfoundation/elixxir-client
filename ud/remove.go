package ud

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
)

type removeFactComms interface {
	SendDeleteMessage(host *connect.Host, message *mixmessages.FactRemovalRequest) (*messages.Ack, error)
}

// Removes a previously confirmed fact.  Will fail if the fact is not
// associated with this client.
func (m *Manager) RemoveFact(fact fact.Fact) error {
	jww.INFO.Printf("ud.RemoveFact(%s)", fact.Stringify())
	return m.removeFact(fact, nil)
}

func (m *Manager) removeFact(fact fact.Fact, rFC removeFactComms) error {
	if !m.IsRegistered() {
		return errors.New("Failed to remove fact: " +
			"client is not registered")
	}

	// Construct the message to send
	// Convert our Fact to a mixmessages Fact for sending
	mmFact := mixmessages.Fact{
		Fact:     fact.Fact,
		FactType: uint32(fact.T),
	}

	// Create our Fact Removal Request message data
	remFactMsg := mixmessages.FactRemovalRequest{
		UID:         m.host.GetId().Marshal(),
		RemovalData: &mmFact,
	}

	// Send the message
	_, err := rFC.SendDeleteMessage(m.host, &remFactMsg)

	// Return the error
	return err
}
