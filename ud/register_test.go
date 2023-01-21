////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// NOTE: ud is not available in wasm
//go:build !js || !wasm

package ud

import (
	"bytes"
	"reflect"
	"testing"

	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
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
	m, _ := newTestManager(t)

	c := &testRegisterComm{}
	prng := NewPrng(42)

	mockSig := []byte("mock")

	err := m.register("testUser", mockSig, prng, c)
	if err != nil {
		t.Errorf("register() returned an error: %+v", err)
	}

	// Check if the UDBUserRegistration contents are correct
	isCorrect("testUser", mockSig, c.msg, m, t)

	// Verify the signed identity data
	pubKeyPem := m.user.GetReceptionIdentity().RSAPrivatePem
	privKey, err := rsa.LoadPrivateKeyFromPem(pubKeyPem)
	if err != nil {
		t.Fatalf("Failed to load public key: %+v", err)
	}

	err = rsa.Verify(privKey.GetPublic(), hash.CMixHash, c.msg.IdentityRegistration.Digest(),
		c.msg.IdentitySignature, nil)
	if err != nil {
		t.Errorf("Failed to verify signed identity data: %+v", err)
	}

	// Verify the signed fact
	usernameFact, _ := fact.NewFact(fact.Username, "testUser")
	err = rsa.Verify(privKey.GetPublic(), hash.CMixHash, factID.Fingerprint(usernameFact),
		c.msg.Frs.FactSig, nil)
	if err != nil {
		t.Errorf("Failed to verify signed fact data: %+v", err)
	}
}

// isCorrect checks if the UDBUserRegistration has all the expected fields minus
// any signatures.
func isCorrect(username string, mockSig []byte, msg *pb.UDBUserRegistration, m *Manager, t *testing.T) {
	if !bytes.Equal(mockSig, msg.PermissioningSignature) {
		t.Errorf("PermissioningSignature incorrect.\n\texpected: %v\n\treceived: %v",
			mockSig, msg.PermissioningSignature)
	}

	identity := m.user.GetReceptionIdentity()
	privKey, err := rsa.LoadPrivateKeyFromPem(identity.RSAPrivatePem)
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	pubKeyPem := rsa.CreatePublicKeyPem(privKey.GetPublic())

	if string(pubKeyPem) !=
		msg.RSAPublicPem {
		t.Errorf("RSAPublicPem incorrect.\n\texpected: %v\n\treceived: %v",
			string(pubKeyPem),
			msg.RSAPublicPem)
	}

	if username != msg.IdentityRegistration.Username {
		t.Errorf("IdentityRegistration Username incorrect.\n\texpected: %#v\n\treceived: %#v",
			username, msg.IdentityRegistration.Username)
	}

	dhKeyPriv, err := identity.GetDHKeyPrivate()
	if err != nil {
		t.Fatalf("%v", err)
	}

	grp := m.user.GetE2E().GetGroup()
	dhKeyPub := grp.ExpG(dhKeyPriv, grp.NewInt(1))

	if !bytes.Equal(dhKeyPub.Bytes(), msg.IdentityRegistration.DhPubKey) {
		t.Errorf("IdentityRegistration DhPubKey incorrect.\n\texpected: %#v\n\treceived: %#v",
			dhKeyPub.Bytes(), msg.IdentityRegistration.DhPubKey)
	}

	if !bytes.Equal(identity.Salt, msg.IdentityRegistration.Salt) {
		t.Errorf("IdentityRegistration Salt incorrect.\n\texpected: %#v\n\treceived: %#v",
			identity.Salt, msg.IdentityRegistration.Salt)
	}

	if !bytes.Equal(identity.ID.Marshal(), msg.Frs.UID) {
		t.Errorf("Frs UID incorrect.\n\texpected: %v\n\treceived: %v",
			identity.ID.Marshal(), msg.Frs.UID)
	}

	if !reflect.DeepEqual(&pb.Fact{Fact: username}, msg.Frs.Fact) {
		t.Errorf("Frs Fact incorrect.\n\texpected: %v\n\treceived: %v",
			&pb.Fact{Fact: username}, msg.Frs.Fact)
	}
}
