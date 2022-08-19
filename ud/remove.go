package ud

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// RemoveFact removes a previously confirmed fact. This will fail
// if the fact passed in is not UD service does not associate this
// fact with this user.
func (m *Manager) RemoveFact(f fact.Fact) error {
	jww.INFO.Printf("ud.RemoveFact(%s)", f.Stringify())
	m.factMux.Lock()
	defer m.factMux.Unlock()
	return m.removeFact(f, m.comms)
}

// removeFact is a helper function which contacts the UD service
// to remove the association of a fact with a user.
func (m *Manager) removeFact(f fact.Fact,
	rFC removeFactComms) error {

	// Construct the message to send
	// Convert our Fact to a mixmessages Fact for sending
	mmFact := mixmessages.Fact{
		Fact:     f.Fact,
		FactType: uint32(f.T),
	}

	// Create a hash of our fact
	fHash := factID.Fingerprint(f)

	// Sign our inFact for putting into the request
	identity := m.user.GetReceptionIdentity()
	privKey, err := identity.GetRSAPrivateKey()
	if err != nil {
		return err
	}
	stream := m.getRng().GetStream()
	defer stream.Close()
	fSig, err := rsa.Sign(stream, privKey, hash.CMixHash, fHash, nil)
	if err != nil {
		return err
	}

	// Create our Fact Removal Request message data
	remFactMsg := mixmessages.FactRemovalRequest{
		UID:         identity.ID.Marshal(),
		RemovalData: &mmFact,
		FactSig:     fSig,
	}

	// Send the message
	_, err = rFC.SendRemoveFact(m.ud.host, &remFactMsg)
	if err != nil {
		return err
	}

	// Remove from storage
	return m.store.DeleteFact(f)
}

// PermanentDeleteAccount removes the username associated with this user
// from the UD service. This will only take a username type fact,
// and the fact must be associated with this user.
func (m *Manager) PermanentDeleteAccount(f fact.Fact) error {
	jww.INFO.Printf("ud.PermanentDeleteAccount(%s)", f.Stringify())
	if f.T != fact.Username {
		return errors.New(fmt.Sprintf("PermanentDeleteAccount must only remove "+
			"a username. Cannot remove fact %q", f.Fact))
	}
	identity := m.user.GetReceptionIdentity()
	privKey, err := identity.GetRSAPrivateKey()
	if err != nil {
		return err
	}

	return m.permanentDeleteAccount(f, identity.ID, privKey, m.comms, m.ud.host)
}

// permanentDeleteAccount is a helper function for PermanentDeleteAccount.
func (m *Manager) permanentDeleteAccount(f fact.Fact, myId *id.ID, privateKey *rsa.PrivateKey,
	rFC removeUserComms, udHost *connect.Host) error {

	// Construct the message to send
	// Convert our Fact to a mixmessages Fact for sending
	mmFact := mixmessages.Fact{
		Fact:     f.Fact,
		FactType: uint32(f.T),
	}

	// Create a hash of our fact
	fHash := factID.Fingerprint(f)

	// Sign our inFact for putting into the request
	stream := m.getRng().GetStream()
	defer stream.Close()
	fsig, err := rsa.Sign(stream, privateKey, hash.CMixHash, fHash, nil)
	if err != nil {
		return err
	}

	// Create our Fact Removal Request message data
	remFactMsg := mixmessages.FactRemovalRequest{
		UID:         myId.Marshal(),
		RemovalData: &mmFact,
		FactSig:     fsig,
	}

	// Send the message
	_, err = rFC.SendRemoveUser(udHost, &remFactMsg)

	// Return the error
	return err
}
