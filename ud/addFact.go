package ud

import (
	"crypto/rand"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type addFactComms interface {
	SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error)
}

// SendRegisterFact adds a fact for the user to user discovery. Will only
// succeed if the user is already registered and the system does not have the
// fact currently registered for any user.
// This does not complete the fact registration process, it returns a
// confirmation id instead. Over the communications system the fact is
// associated with, a code will be sent. This confirmation ID needs to be
// called along with the code to finalize the fact.
func (m *Manager) SendRegisterFact(fact fact.Fact) (string, error) {
	jww.INFO.Printf("ud.SendRegisterFact(%s)", fact.Stringify())
	return m.addFact(fact, m.myID, m.comms)
}

func (m *Manager) addFact(inFact fact.Fact, uid *id.ID, aFC addFactComms) (string, error) {

	if !m.IsRegistered() {
		return "", errors.New("Failed to add fact: " +
			"client is not registered")
	}

	// Create a primitives Fact so we can hash it
	f, err := fact.NewFact(inFact.T, inFact.Fact)
	if err != nil {
		return "", err
	}

	// Create a hash of our fact
	fHash := factID.Fingerprint(f)

	// Sign our inFact for putting into the request
	fSig, err := rsa.Sign(rand.Reader, m.privKey, hash.CMixHash, fHash, nil)
	if err != nil {
		return "", err
	}

	// Create our Fact Removal Request message data
	remFactMsg := pb.FactRegisterRequest{
		UID: uid.Marshal(),
		Fact: &pb.Fact{
			Fact:     inFact.Fact,
			FactType: uint32(inFact.T),
		},
		FactSig: fSig,
	}

	// get UD host
	host, err := m.getHost()
	if err != nil {
		return "", err
	}

	// Send the message
	response, err := aFC.SendRegisterFact(host, &remFactMsg)

	confirmationID := ""
	if response != nil {
		confirmationID = response.ConfirmationID
	}

	err = m.storage.GetUd().StoreUnconfirmedFact(confirmationID, f)
	if err != nil {
		return "", errors.WithMessagef(err, "Failed to store unconfirmed fact %v", f.Fact)
	}
	// Return the error
	return confirmationID, err
}
