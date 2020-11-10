package ud

import (
	"crypto/rand"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

type removeFactComms interface {
	SendDeleteMessage(host *connect.Host, message *messages.AuthenticatedMessage) (*messages.Ack, error)
}

func (m *Manager) RemoveFact(fact contact.Fact) error {
	return m.removeFact(fact, m.comms)
}

func (m *Manager) removeFact(fact contact.Fact, rFC removeFactComms) error {
	// Construct the message to send
	// Convert our Fact to a mixmessages Fact for sending
	mmFact := mixmessages.Fact{
		Fact:     fact.Fact,
		FactType: uint32(fact.T),
	}

	// Sign the fact
	signedFact, err := rsa.Sign(rand.Reader, m.privKey, hash.CMixHash, mmFact.Digest(), rsa.NewDefaultOptions())
	if err != nil {
		return err
	}

	// Create our Fact Removal Request message data
	remFactMsg := mixmessages.FactRemovalRequest{
		UID:         m.host.GetId().Marshal(),
		RemovalData: &mmFact,
	}

	// Marshal it to bytes for sending over the wire
	remFactMsgMarshalled, err := proto.Marshal(&remFactMsg)
	if err != nil {
		return err
	}

	// Convert our marshalled Fact Removal Request to an Any
	// object for sending in an authed message
	remFactMsgAny := any.Any{
		TypeUrl: "gitlab.com/elixxir/client/interfaces/contact.Fact",
		Value:   remFactMsgMarshalled,
	}

	// Create our AuthenticatedMessage so we can send the data over
	msg := messages.AuthenticatedMessage{
		ID:        nil,
		Signature: signedFact,
		Token:     nil,
		Client:    nil,
		Message:   &remFactMsgAny,
	}

	// Send the message
	_, err = rFC.SendDeleteMessage(m.host, &msg)

	// Return the error
	return err
}
