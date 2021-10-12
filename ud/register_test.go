package ud

import (
	"bytes"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"reflect"
	"testing"
)

type testRegisterComm struct {
	msg *pb.UDBUserRegistration
}

func (t *testRegisterComm) SendRegisterUser(_ *connect.Host, msg *pb.UDBUserRegistration) (*messages.Ack, error) {
	t.msg = msg
	return &messages.Ack{}, nil
}

// Happy path.
func TestManager_register(t *testing.T) {
	isReg := uint32(0)

	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}

	// Set up manager
	m := &Manager{
		comms:      comms,
		net:        newTestNetworkManager(t),
		rng:        fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG),
		storage:    storage.InitTestingSession(t),
		registered: &isReg,
	}

	c := &testRegisterComm{}

	err = m.register("testUser", c)
	if err != nil {
		t.Errorf("register() returned an error: %+v", err)
	}

	// Check if the UDBUserRegistration contents are correct
	m.isCorrect("testUser", c.msg, t)

	// Verify the signed identity data
	pubKey := m.storage.User().GetCryptographicIdentity().GetTransmissionRSA().GetPublic()
	err = rsa.Verify(pubKey, hash.CMixHash, c.msg.IdentityRegistration.Digest(),
		c.msg.IdentitySignature, nil)
	if err != nil {
		t.Errorf("Failed to verify signed identity data: %+v", err)
	}

	// Verify the signed fact
	usernameFact, _ := fact.NewFact(fact.Username, "testUser")
	err = rsa.Verify(pubKey, hash.CMixHash, factID.Fingerprint(usernameFact),
		c.msg.Frs.FactSig, nil)
	if err != nil {
		t.Errorf("Failed to verify signed fact data: %+v", err)
	}
}

// isCorrect checks if the UDBUserRegistration has all the expected fields minus
// any signatures.
func (m *Manager) isCorrect(username string, msg *pb.UDBUserRegistration, t *testing.T) {
	user := m.storage.User()
	cryptoUser := m.storage.User().GetCryptographicIdentity()

	if !bytes.Equal(user.GetTransmissionRegistrationValidationSignature(), msg.PermissioningSignature) {
		t.Errorf("PermissioningSignature incorrect.\n\texpected: %v\n\treceived: %v",
			user.GetTransmissionRegistrationValidationSignature(), msg.PermissioningSignature)
	}

	if string(rsa.CreatePublicKeyPem(cryptoUser.GetTransmissionRSA().GetPublic())) != msg.RSAPublicPem {
		t.Errorf("RSAPublicPem incorrect.\n\texpected: %v\n\treceived: %v",
			string(rsa.CreatePublicKeyPem(cryptoUser.GetTransmissionRSA().GetPublic())), msg.RSAPublicPem)
	}

	if username != msg.IdentityRegistration.Username {
		t.Errorf("IdentityRegistration Username incorrect.\n\texpected: %#v\n\treceived: %#v",
			username, msg.IdentityRegistration.Username)
	}

	if !bytes.Equal(m.storage.E2e().GetDHPublicKey().Bytes(), msg.IdentityRegistration.DhPubKey) {
		t.Errorf("IdentityRegistration DhPubKey incorrect.\n\texpected: %#v\n\treceived: %#v",
			m.storage.E2e().GetDHPublicKey().Bytes(), msg.IdentityRegistration.DhPubKey)
	}

	if !bytes.Equal(cryptoUser.GetTransmissionSalt(), msg.IdentityRegistration.Salt) {
		t.Errorf("IdentityRegistration Salt incorrect.\n\texpected: %#v\n\treceived: %#v",
			cryptoUser.GetTransmissionSalt(), msg.IdentityRegistration.Salt)
	}

	if !bytes.Equal(cryptoUser.GetTransmissionID().Marshal(), msg.Frs.UID) {
		t.Errorf("Frs UID incorrect.\n\texpected: %v\n\treceived: %v",
			cryptoUser.GetTransmissionID().Marshal(), msg.Frs.UID)
	}

	if !reflect.DeepEqual(&pb.Fact{Fact: username}, msg.Frs.Fact) {
		t.Errorf("Frs Fact incorrect.\n\texpected: %v\n\treceived: %v",
			&pb.Fact{Fact: username}, msg.Frs.Fact)
	}
}
