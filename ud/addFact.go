package ud

import (
	"crypto/rand"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
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

func (m *Manager) addFact(inFact fact.Fact, aFC addFactComms) (*pb.FactRegisterResponse, error) {
	if !m.IsRegistered() {
		return nil, errors.New("Failed to add fact: " +
			"client is not registered")
	}

	// Create a primitives Fact so we can hash it
	f, err := fact.NewFact(inFact.T, inFact.Fact)
	if err != nil {
		return &pb.FactRegisterResponse{}, err
	}

	// Create a hash of our fact
	fhash := factID.Fingerprint(f)

	// Sign our inFact for putting into the request
	fsig, err := rsa.Sign(rand.Reader, m.privKey, hash.CMixHash, fhash, nil)
	if err != nil {
		return &pb.FactRegisterResponse{}, err
	}

	uid := m.storage.User().GetCryptographicIdentity().GetUserID()

	// Create our Fact Removal Request message data
	remFactMsg := pb.FactRegisterRequest{
		UID: uid.Marshal(),
		Fact: &pb.Fact{
			Fact:     inFact.Fact,
			FactType: uint32(inFact.T),
		},
		FactSig: fsig,
	}

	// Send the message
	response, err := aFC.SendRegisterFact(m.host, &remFactMsg)

	// Return the error
	return response, err
}
