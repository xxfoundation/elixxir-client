package ud

import (
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
)


type removeFactComms interface{
	SendDeleteMessage(host *connect.Host, message *messages.AuthenticatedMessage)(*messages.Ack, error)
}

func (m *Manager)RemoveFact(fact contact.Fact)error{
	return nil//m.removeFact(fact,m.comms)
}

func (m *Manager)removeFact(fact contact.Fact, SendDeleteMessage removeFactComms)error {
	//digest the fact
	fact.Stringify()
	//sign the fact
	//rsa.Sign()

	//constuct the message


	//send the message

	//return the error



}
