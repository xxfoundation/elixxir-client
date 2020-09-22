package permissioning

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
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
	sender.getHost, err = connect.NewHost(&id.Permissioning, "address", nil, false, false)
	if err != nil {
		t.Fatal(err)
	}

	regCode := "flooble doodle"
	sig, err := Register(&sender, key.GetPublic(), regCode)
	if err != nil {
		t.Error(err)
	}
	if string(sig) != "sig" {
		t.Error("expected signature to be 'sig'")
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

// Shows that returning an error from GetHost results in an error from
// Register
func TestRegisterWithPermissioning_GetHostErr(t *testing.T) {
	var sender MockRegistrationSender
	sender.succeedGetHost = false
	_, err := Register(&sender, nil, "")
	if err == nil {
		t.Error("no error if getHost fails")
	}
}

// Shows that returning an error from the permissioning server results in an
// error from Register
func TestRegisterWithPermissioning_ResponseErr(t *testing.T) {
	rng := csprng.NewSystemRNG()
	key, err := rsa.GenerateKey(rng, 256)
	if err != nil {
		t.Fatal(err)
	}
	var sender MockRegistrationSender
	sender.succeedGetHost = true
	sender.errInReply = "failure occurred on permissioning"
	_, err = Register(&sender, key.GetPublic(), "")
	if err == nil {
		t.Error("no error if registration fails on permissioning")
	}
}

// Shows that returning an error from the RPC (e.g. context deadline exceeded)
// results in an error from Register
func TestRegisterWithPermissioning_ConnectionErr(t *testing.T) {
	rng := csprng.NewSystemRNG()
	key, err := rsa.GenerateKey(rng, 256)
	if err != nil {
		t.Fatal(err)
	}
	var sender MockRegistrationSender
	sender.succeedGetHost = true
	sender.errSendRegistration = errors.New("connection problem")
	_, err = Register(&sender, key.GetPublic(), "")
	if err == nil {
		t.Error("no error if e.g. context deadline exceeded")
	}
}
