///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"crypto/rand"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"testing"
)

// testUsernameValidation is a mock up of UD's response for a
// SendUsernameValidation comm.
type testUsernameValidation struct {
	pubKeyPem []byte
}

func (tuv *testUsernameValidation) SendUsernameValidation(host *connect.Host,
	message *pb.UsernameValidationRequest) (*pb.UsernameValidation, error) {
	privKey, _ := rsa.LoadPrivateKeyFromPem([]byte(testKey))
	sig, _ := crust.SignVerification(rand.Reader, privKey,
		message.Username, tuv.pubKeyPem)

	return &pb.UsernameValidation{
		Signature: sig,
	}, nil
}

// Unit test of getUsernameValidationSignature.
func TestManager_GetUsernameValidationSignature(t *testing.T) {
	// Create our Manager object

	m, _ := newTestManager(t)
	rsaPrivKey, err := m.user.GetReceptionIdentity().GetRSAPrivateKey()
	if err != nil {
		t.Fatalf("Failed to retrieve private key: %v", err)
	}
	publicKeyPem := rsa.CreatePublicKeyPem(rsaPrivKey.GetPublic())

	c := &testUsernameValidation{
		pubKeyPem: publicKeyPem,
	}

	_, err = m.getUsernameValidationSignature("admin", c)
	if err != nil {
		t.Fatalf("getUsernameValidationSignature error: %+v", err)
	}

}
