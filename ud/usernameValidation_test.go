///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"crypto/rand"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// testUsernameValidation is a mock up of UD's response for a
// SendUsernameValidation comm.
type testUsernameValidation struct {
	publicKey *rsa.PublicKey
	username  string
}

func (tuv *testUsernameValidation) SendUsernameValidation(host *connect.Host,
	message *pb.UsernameValidationRequest) (*pb.UsernameValidation, error) {
	privKey, _ := rsa.LoadPrivateKeyFromPem([]byte(testKey))

	sig, _ := crust.SignVerification(rand.Reader, privKey,
		tuv.username, tuv.publicKey)

	return &pb.UsernameValidation{
		Signature:             sig,
		ReceptionPublicKeyPem: rsa.CreatePublicKeyPem(tuv.publicKey),
		Username:              tuv.username,
	}, nil
}

// Unit test of getUsernameValidationSignature.
func TestManager_GetUsernameValidationSignature(t *testing.T) {
	// Create our Manager object
	rng := csprng.NewSystemRNG()
	rsaPrivKey, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}

	h, err := comms.AddHost(id.NewIdFromBytes([]byte("testUD"), t), "",
		[]byte(testCert), connect.GetDefaultHostParams())
	if err != nil {
		t.Fatalf("Failed to load host: %v", err)
	}

	// Create our Manager object
	m := Manager{
		myID:          id.NewIdFromBytes([]byte("testda"), t),
		comms:         comms,
		net:           newTestNetworkManager(t),
		privKey:       rsaPrivKey,
		alternativeUd: &alternateUd{host: h},
		storage:       storage.InitTestingSession(t),
	}

	c := &testUsernameValidation{
		publicKey: rsaPrivKey.GetPublic(),
		username:  "admin",
	}

	_, err = m.queryUsernameValidationSignature(c)
	if err != nil {
		t.Fatalf("getUsernameValidationSignature error: %+v", err)
	}

}
