package ud

import (
	"gitlab.com/elixxir/client/interfaces/contact"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"io"
)

type addFactComms interface {
	SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error)
}

func (m *Manager) SendRegisterFact(fact contact.Fact) (*pb.FactRegisterResponse, error) {
	return m.addFact(fact, m.comms)
}

func (m *Manager) addFact(fact fact.Fact, aFC addFactComms) (*pb.FactRegisterResponse, error) {
	// Construct the message to send
	// Convert our Fact to a mixmessages Fact for sending
	mmFact := pb.Fact{
		Fact:     fact.Fact,
		FactType: uint32(fact.T),
	}

	rsa.Sign(io.Reader, m.privKey, )
	//signature.Sign(mmFact, m.privKey)

	// Create our Fact Removal Request message data
	remFactMsg := pb.FactRegisterRequest{
		UID: m.host.GetId().Marshal(),
		Fact: &mmFact,
		FactSig: []byte("B"),
	}

	// Send the message
	response, err := aFC.SendRegisterFact(m.host, &remFactMsg)

	// Return the error
	return response, err
}
