package ud

import (
	"crypto/rand"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// SendRegisterFact adds a fact for the user to user discovery. Will only
// succeed if the user is already registered and the system does not have the
// fact currently registered for any user.
// This does not complete the fact registration process, it returns a
// confirmation id instead. Over the communications system the fact is
// associated with, a code will be sent. This confirmation ID needs to be
// called along with the code to finalize the fact.
func (m *Manager) SendRegisterFact(f fact.Fact) (string, error) {
	jww.INFO.Printf("ud.SendRegisterFact(%s)", f.Stringify())
	return m.addFact(f, m.e2e.GetReceptionID(), m.comms)
}

func (m *Manager) addFact(inFact fact.Fact, myId *id.ID, aFC addFactComms) (string, error) {

	// get UD host
	udHost, err := m.getOrAddUdHost()
	if err != nil {
		return "", err
	}

	// Create a primitives Fact so we can hash it
	f, err := fact.NewFact(inFact.T, inFact.Fact)
	if err != nil {
		return "", err
	}

	// Create a hash of our fact
	fHash := factID.Fingerprint(f)

	// Sign our inFact for putting into the request
	privKey := m.user.PortableUserInfo().ReceptionRSA
	fSig, err := rsa.Sign(rand.Reader, privKey, hash.CMixHash, fHash, nil)
	if err != nil {
		return "", err
	}

	// Create our Fact Removal Request message data
	remFactMsg := pb.FactRegisterRequest{
		UID: myId.Marshal(),
		Fact: &pb.Fact{
			Fact:     f.Fact,
			FactType: uint32(f.T),
		},
		FactSig: fSig,
	}

	// Send the message
	response, err := aFC.SendRegisterFact(udHost, &remFactMsg)

	confirmationID := ""
	if response != nil {
		confirmationID = response.ConfirmationID
	}

	err = m.store.StoreUnconfirmedFact(confirmationID, f)
	if err != nil {
		return "", errors.WithMessagef(err,
			"Failed to store unconfirmed fact %v", f.Fact)
	}
	// Return the error
	return confirmationID, err
}
