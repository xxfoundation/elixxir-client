///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package registration

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
	"time"
)

func NewMockRegSender(key, cert []byte) (*MockRegistrationSender, error) {
	privKey, err := rsa.LoadPrivateKeyFromPem(key)
	if err != nil {
		return nil, err
	}

	// Generate a pre-canned time for consistent testing
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		return nil, errors.Errorf("SignVerify error: "+
			"Could not parse precanned time: %v", err.Error())
	}

	h, err := connect.NewHost(&id.ClientRegistration, "address", cert,
		connect.GetDefaultHostParams())
	if err != nil {
		return nil, err
	}

	return &MockRegistrationSender{
		privKey: privKey,
		getHost: h,
		prng: &CountingReader{count: 0},
		mockTS: testTime,
	}, nil
}

type MockRegistrationSender struct {
	reg *pb.ClientRegistration
	// param passed to SendRegistrationMessage
	host *connect.Host
	privKey *rsa.PrivateKey
	prng *CountingReader
	mockTS time.Time
	// original host returned from GetHost
	getHost             *connect.Host
	errSendRegistration error
	errInReply          string
}

func (s *MockRegistrationSender) SendRegistrationMessage(host *connect.Host, message *pb.ClientRegistration) (*pb.SignedClientRegistrationConfirmations, error) {
	fmt.Printf("mockTs: %v\n", s.mockTS.UnixNano())
	fmt.Printf("transKet: %v\n", message.ClientTransmissionRSAPubKey)
	transSig, err := registration.SignWithTimestamp(s.prng, s.privKey,
		s.mockTS.UnixNano(), message.ClientTransmissionRSAPubKey)
	if err != nil {
		return nil, errors.Errorf("Failed to sign transmission: %v", err)
	}

	receptionSig, err := registration.SignWithTimestamp(s.prng, s.privKey,
		s.mockTS.UnixNano(), message.ClientReceptionRSAPubKey)
	if err != nil {
		return nil, errors.Errorf("Failed to sign reception: %v", err)
	}

	transConfirmation := &pb.ClientRegistrationConfirmation{
		Timestamp: s.mockTS.UnixNano(),
		RSAPubKey: message.ClientTransmissionRSAPubKey,
	}


	receptionConfirmation := &pb.ClientRegistrationConfirmation{
		Timestamp: s.mockTS.UnixNano(),
		RSAPubKey: message.ClientReceptionRSAPubKey,
	}

	transConfirmationData, err := proto.Marshal(transConfirmation)
	if err != nil {
		return nil, err
	}

	receptionConfirmationData, err := proto.Marshal(receptionConfirmation)
	if err != nil {
		return nil, err
	}

	return &pb.SignedClientRegistrationConfirmations{
		ClientTransmissionConfirmation: &pb.SignedRegistrationConfirmation{
			RegistrarSignature: &messages.RSASignature{
				Signature: transSig,
			},
			ClientRegistrationConfirmation: transConfirmationData,
		},
		ClientReceptionConfirmation: &pb.SignedRegistrationConfirmation{
			RegistrarSignature: &messages.RSASignature{
				Signature: receptionSig,
			},
			ClientRegistrationConfirmation: receptionConfirmationData,
		},
		Error: s.errInReply,
	}, s.errSendRegistration
}

func (s *MockRegistrationSender) GetHost(*id.ID) (*connect.Host, bool) {
	return s.getHost, true
}

// Shows that we get expected result from happy path
// Shows that registration gets RPCs with the correct parameters
func TestRegisterWithPermissioning(t *testing.T) {


	certData, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Fatalf("Could not load certificate: %v", err)
	}

	keyData, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Fatalf("Could not load private key: %v", err)
	}

	key, err := rsa.LoadPrivateKeyFromPem(keyData)
	if err != nil {
		t.Fatalf("Could not load public key")
	}

	sender, err := NewMockRegSender(keyData, certData)
	if err != nil {
		t.Fatalf("Failed to create mock sender: %v", err)
	}


	regCode := "flooble doodle"
	sig1, sig2, regTimestamp, err := register(sender, sender.getHost, key.GetPublic(), key.GetPublic(), regCode)
	if err != nil {
		t.Error(err)
	}

	if regTimestamp != sender.mockTS.UnixNano() {
		t.Fatalf("Unexpected timestamp returned from register: " +
			"\n\tExpected: %v" +
			"\n\tReceived: %v", sender.mockTS.UnixNano(), regTimestamp)
	}

	// todo compare sigs
	t.Logf("sig1: %v", sig1)
	t.Logf("sig2: %v", sig2)


}

// Shows that returning an error from the registration server results in an
// error from register
func TestRegisterWithPermissioning_ResponseErr(t *testing.T) {
	rng := csprng.NewSystemRNG()
	key, err := rsa.GenerateKey(rng, 256)
	if err != nil {
		t.Fatal(err)
	}
	var sender MockRegistrationSender
	sender.errInReply = "failure occurred on registration"
	_, _, _, err = register(&sender, nil, key.GetPublic(), key.GetPublic(), "")
	if err == nil {
		t.Error("no error if registration fails on registration")
	}
}

// Shows that returning an error from the RPC (e.g. context deadline exceeded)
// results in an error from register
func TestRegisterWithPermissioning_ConnectionErr(t *testing.T) {
	rng := csprng.NewSystemRNG()
	key, err := rsa.GenerateKey(rng, 256)
	if err != nil {
		t.Fatal(err)
	}
	var sender MockRegistrationSender
	sender.errSendRegistration = errors.New("connection problem")
	_, _, _, err = register(&sender, nil, key.GetPublic(), key.GetPublic(), "")
	if err == nil {
		t.Error("no error if e.g. context deadline exceeded")
	}
}

type CountingReader struct {
	count uint8
}

// Read just counts until 254 then starts over again
func (c *CountingReader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		c.count = (c.count + 1) % 255
		b[i] = c.count
	}
	return len(b), nil
}
