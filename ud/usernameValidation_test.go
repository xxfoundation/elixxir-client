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
	pubKey   *rsa.PublicKey
	username string
}

func (tuv *testUsernameValidation) SendUsernameValidation(host *connect.Host,
	message *pb.UsernameValidationRequest) (*pb.UsernameValidation, error) {
	privKey, _ := rsa.LoadPrivateKeyFromPem([]byte(testKey))
	sig, _ := crust.SignVerification(rand.Reader, privKey,
		tuv.username, tuv.pubKey)

	return &pb.UsernameValidation{
		Signature:             sig,
		ReceptionPublicKeyPem: rsa.CreatePublicKeyPem(tuv.pubKey),
		Username:              tuv.username,
	}, nil
}

// Unit test of queryUsernameValidationSignature.
func TestManager_queryUsernameValidationSignature(t *testing.T) {
	// Create our Manager object

	m, _ := newTestManager(t)
	rsaPrivKey, err := m.user.GetReceptionIdentity().GetRSAPrivateKey()
	if err != nil {
		t.Fatalf("Failed to retrieve private key: %v", err)
	}

	c := &testUsernameValidation{
		pubKey:   rsaPrivKey.GetPublic(),
		username: "admin",
	}

	_, err = m.queryUsernameValidationSignature(c)
	if err != nil {
		t.Fatalf("queryUsernameValidationSignature error: %+v", err)
	}

}
