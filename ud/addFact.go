package ud

import (
	"crypto/rand"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

type addFactComms interface {
	SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error)
}

func (m *Manager) SendRegisterFact(fact fact.Fact) (*pb.FactRegisterResponse, error) {
	return m.addFact(fact, m.comms)
}

func (m *Manager) addFact(fact fact.Fact, aFC addFactComms) (*pb.FactRegisterResponse, error) {
	// Construct the message to send
	// Convert our Fact to a mixmessages Fact for sending
	mmFact := pb.Fact{
		Fact:     fact.Fact,
		FactType: uint32(fact.T),
	}

	// Sign our fact for putting into the request
	fsig, err := rsa.Sign(rand.Reader, m.privKey, hash.CMixHash, mmFact.Digest(), nil)
	if err != nil {
		return &pb.FactRegisterResponse{}, err
	}

	// Create our Fact Removal Request message data
	remFactMsg := pb.FactRegisterRequest{
		UID:     m.host.GetId().Marshal(),
		Fact:    &mmFact,
		FactSig: fsig,
	}

	// Send the message
	response, err := aFC.SendRegisterFact(m.host, &remFactMsg)

	// Return the error
	return response, err
}
