package ud

import (
	"crypto/rand"
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

// RemoveFact removes a previously confirmed fact. Will fail if the fact is not
// associated with this client.
func (m *Manager) RemoveFact(f fact.Fact) error {
	jww.INFO.Printf("ud.RemoveFact(%s)", f.Stringify())

	return m.removeFact(f, m.comms)
}

func (m *Manager) removeFact(f fact.Fact,
	rFC removeFactComms) error {

	// Get UD host
	udHost, err := m.getOrAddUdHost()
	if err != nil {
		return err
	}

	// Construct the message to send
	// Convert our Fact to a mixmessages Fact for sending
	mmFact := mixmessages.Fact{
		Fact:     f.Fact,
		FactType: uint32(f.T),
	}

	// Create a hash of our fact
	fHash := factID.Fingerprint(f)

	// Sign our inFact for putting into the request
	fSig, err := rsa.Sign(rand.Reader, m.privKey, hash.CMixHash, fHash, nil)
	if err != nil {
		return err
	}

	// Create our Fact Removal Request message data
	remFactMsg := mixmessages.FactRemovalRequest{
		UID:         m.myID.Marshal(),
		RemovalData: &mmFact,
		FactSig:     fSig,
	}

	// Send the message
	_, err = rFC.SendRemoveFact(udHost, &remFactMsg)
	if err != nil {
		return err
	}

	// Remove from storage
	return m.store.DeleteFact(f)
}

// RemoveUser removes a previously confirmed fact.
// This call will fail if the fact is not associated with this client.
func (m *Manager) RemoveUser(f fact.Fact) error {
	jww.INFO.Printf("ud.RemoveUser(%s)", f.Stringify())
	if f.T != fact.Username {
		return errors.New(fmt.Sprintf("RemoveUser must only remove "+
			"a username. Cannot remove fact %q", f.Fact))
	}

	udHost, err := m.getOrAddUdHost()
	if err != nil {
		return err
	}

	return removeUser(f, m.myID, m.privKey, m.comms, udHost)
}

func removeUser(f fact.Fact, myId *id.ID, privateKey *rsa.PrivateKey,
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
	fsig, err := rsa.Sign(rand.Reader, privateKey, hash.CMixHash, fHash, nil)
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
