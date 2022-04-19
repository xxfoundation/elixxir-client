package ud

import (
	"bytes"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
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
	m := newTestManager(t)

	udHost, err := m.getOrAddUdHost()
	if err != nil {
		t.Fatalf("Failed to get/add ud host: %+v", err)
	}

	c := &testRegisterComm{}
	prng := NewPrng(42)

	err = m.register("testUser", prng, c, udHost)
	if err != nil {
		t.Errorf("register() returned an error: %+v", err)
	}

	// Check if the UDBUserRegistration contents are correct
	isCorrect("testUser", c.msg, m, t)

	// Verify the signed identity data
	pubKey := m.user.PortableUserInfo().ReceptionRSA.GetPublic()
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
func isCorrect(username string, msg *pb.UDBUserRegistration, m *Manager, t *testing.T) {
	userInfo := m.user.PortableUserInfo()

	if !bytes.Equal(m.user.GetReceptionRegistrationValidationSignature(), msg.PermissioningSignature) {
		t.Errorf("PermissioningSignature incorrect.\n\texpected: %v\n\treceived: %v",
			m.user.GetReceptionRegistrationValidationSignature(), msg.PermissioningSignature)
	}

	if string(rsa.CreatePublicKeyPem(userInfo.TransmissionRSA.GetPublic())) !=
		msg.RSAPublicPem {
		t.Errorf("RSAPublicPem incorrect.\n\texpected: %v\n\treceived: %v",
			string(rsa.CreatePublicKeyPem(userInfo.TransmissionRSA.GetPublic())),
			msg.RSAPublicPem)
	}

	if username != msg.IdentityRegistration.Username {
		t.Errorf("IdentityRegistration Username incorrect.\n\texpected: %#v\n\treceived: %#v",
			username, msg.IdentityRegistration.Username)
	}

	if !bytes.Equal(userInfo.E2eDhPublicKey.Bytes(), msg.IdentityRegistration.DhPubKey) {
		t.Errorf("IdentityRegistration DhPubKey incorrect.\n\texpected: %#v\n\treceived: %#v",
			userInfo.E2eDhPublicKey.Bytes(), msg.IdentityRegistration.DhPubKey)
	}

	if !bytes.Equal(userInfo.TransmissionSalt, msg.IdentityRegistration.Salt) {
		t.Errorf("IdentityRegistration Salt incorrect.\n\texpected: %#v\n\treceived: %#v",
			userInfo.TransmissionSalt, msg.IdentityRegistration.Salt)
	}

	if !bytes.Equal(userInfo.TransmissionID.Marshal(), msg.Frs.UID) {
		t.Errorf("Frs UID incorrect.\n\texpected: %v\n\treceived: %v",
			userInfo.TransmissionID.Marshal(), msg.Frs.UID)
	}

	if !reflect.DeepEqual(&pb.Fact{Fact: username}, msg.Frs.Fact) {
		t.Errorf("Frs Fact incorrect.\n\texpected: %v\n\treceived: %v",
			&pb.Fact{Fact: username}, msg.Frs.Fact)
	}
}
