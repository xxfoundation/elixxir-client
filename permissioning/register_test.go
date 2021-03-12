///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package permissioning

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

type MockRegistrationSender struct {
	reg *pb.UserRegistration
	// param passed to SendRegistrationMessage
	host *connect.Host
	// original host returned from GetHost
	getHost             *connect.Host
	succeedGetHost      bool
	errSendRegistration error
	errInReply          string
}

func (s *MockRegistrationSender) SendRegistrationMessage(host *connect.Host, message *pb.UserRegistration) (*pb.UserRegistrationConfirmation, error) {
	s.reg = message
	s.host = host
	return &pb.UserRegistrationConfirmation{
		ClientSignedByServer: &messages.RSASignature{
			Nonce:     []byte("nonce"),
			Signature: []byte("sig"),
		},
		ClientReceptionSignedByServer: &messages.RSASignature{
			Nonce:     []byte("receptionnonce"),
			Signature: []byte("receptionsig"),
		},
		Error: s.errInReply,
	}, s.errSendRegistration
}

func (s *MockRegistrationSender) GetHost(*id.ID) (*connect.Host, bool) {
	return s.getHost, s.succeedGetHost
}

// Shows that we get expected result from happy path
// Shows that permissioning gets RPCs with the correct parameters
func TestRegisterWithPermissioning(t *testing.T) {
	rng := csprng.NewSystemRNG()
	key, err := rsa.GenerateKey(rng, 256)
	if err != nil {
		t.Fatal(err)
	}

	var sender MockRegistrationSender
	sender.succeedGetHost = true
	sender.getHost, err = connect.NewHost(&id.Permissioning, "address", nil,
		connect.GetDefaultHostParams())
	if err != nil {
		t.Fatal(err)
	}

	regCode := "flooble doodle"
	sig1, sig2, err := register(&sender, sender.getHost, key.GetPublic(), key.GetPublic(), regCode)
	if err != nil {
		t.Error(err)
	}
	if string(sig1) != "sig" {
		t.Error("expected signature to be 'sig'")
	}
	if string(sig2) != "receptionsig" {
		t.Error("expected signature to be 'receptionsig'")
	}
	if sender.host.String() != sender.getHost.String() {
		t.Errorf("hosts differed. expected %v, got %v", sender.host, sender.getHost)
	}
	passedPub, err := rsa.LoadPublicKeyFromPem([]byte(sender.reg.ClientRSAPubKey))
	if err != nil {
		t.Error("failed to decode passed public key")
		t.Error(err)
	}
	if !reflect.DeepEqual(passedPub, key.GetPublic()) {
		t.Error("public keys different from expected")
	}
	if sender.reg.RegistrationCode != regCode {
		t.Error("passed regcode different from expected")
	}
}

// Shows that returning an error from the permissioning server results in an
// error from register
func TestRegisterWithPermissioning_ResponseErr(t *testing.T) {
	rng := csprng.NewSystemRNG()
	key, err := rsa.GenerateKey(rng, 256)
	if err != nil {
		t.Fatal(err)
	}
	var sender MockRegistrationSender
	sender.succeedGetHost = true
	sender.errInReply = "failure occurred on permissioning"
	_, _, err = register(&sender, nil, key.GetPublic(), key.GetPublic(), "")
	if err == nil {
		t.Error("no error if registration fails on permissioning")
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
	sender.succeedGetHost = true
	sender.errSendRegistration = errors.New("connection problem")
	_, _, err = register(&sender, nil, key.GetPublic(), key.GetPublic(), "")
	if err == nil {
		t.Error("no error if e.g. context deadline exceeded")
	}
}
