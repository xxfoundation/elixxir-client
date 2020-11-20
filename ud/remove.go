package ud

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
)

type removeFactComms interface {
	SendDeleteMessage(host *connect.Host, message *mixmessages.FactRemovalRequest) (*messages.Ack, error)
}

func (m *Manager) RemoveFact(fact fact.Fact) error {
	return m.removeFact(fact, nil)
}

func (m *Manager) removeFact(fact fact.Fact, rFC removeFactComms) error {
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
