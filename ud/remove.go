package ud

import (
	"crypto/rand"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

type removeFactComms interface {
	SendRemoveFact(host *connect.Host, message *mixmessages.FactRemovalRequest) (*messages.Ack, error)
}

// RemoveFact removes a previously confirmed fact. Will fail if the fact is not
// associated with this client.
func (m *Manager) RemoveFact(fact fact.Fact) error {
	jww.INFO.Printf("ud.RemoveFact(%s)", fact.Stringify())
	return m.removeFact(fact, m.comms)
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

	// Create a hash of our fact
	fHash := factID.Fingerprint(fact)

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

	// get UD host
	host, err := m.getHost()
	if err != nil {
		return err
	}

	// Send the message
	_, err = rFC.SendRemoveFact(host, &remFactMsg)
	if err != nil {
		return err
	}

	// Remove from storage
	return m.storage.GetUd().DeleteFact(fact)
}

type removeUserComms interface {
	SendRemoveUser(host *connect.Host, message *mixmessages.FactRemovalRequest) (*messages.Ack, error)
}

// RemoveUser removes a previously confirmed fact. Will fail if the fact is not
// associated with this client.
func (m *Manager) RemoveUser(fact fact.Fact) error {
	jww.INFO.Printf("ud.RemoveUser(%s)", fact.Stringify())
	return m.removeUser(fact, m.comms)
}

func (m *Manager) removeUser(fact fact.Fact, rFC removeUserComms) error {
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

	// Create a hash of our fact
	fHash := factID.Fingerprint(fact)

	// Sign our inFact for putting into the request
	fsig, err := rsa.Sign(rand.Reader, m.privKey, hash.CMixHash, fHash, nil)
	if err != nil {
		return err
	}

	// Create our Fact Removal Request message data
	remFactMsg := mixmessages.FactRemovalRequest{
		UID:         m.myID.Marshal(),
		RemovalData: &mmFact,
		FactSig:     fsig,
	}

	// get UD host
	host, err := m.getHost()
	if err != nil {
		return err
	}

	// Send the message
	_, err = rFC.SendRemoveUser(host, &remFactMsg)

	// Return the error
	return err
}
