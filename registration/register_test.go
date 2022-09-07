////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package registration

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
	"time"
)

var expectedSignatureOne = []byte{7, 15, 58, 201, 193, 112, 205, 247, 7, 200, 21, 185, 22, 82, 81, 114, 245, 179, 56, 157, 67, 209, 153, 59, 232, 119, 40, 84, 70, 246, 63, 211, 175, 190, 184, 152, 218, 74, 190, 232, 234, 106, 44, 249, 6, 86, 133, 191, 252, 74, 162, 114, 85, 211, 145, 41, 182, 33, 101, 86, 214, 106, 192, 8, 137, 153, 4, 17, 81, 202, 163, 117, 185, 75, 41, 5, 174, 50, 111, 234, 0, 94, 234, 105, 222, 74, 70, 225, 71, 81, 66, 203, 160, 128, 217, 93, 47, 132, 50, 40, 86, 115, 223, 200, 207, 103, 197, 35, 49, 82, 144, 142, 161, 104, 209, 163, 59, 19, 30, 132, 38, 91, 96, 21, 116, 200, 71, 108, 193, 68, 12, 33, 143, 146, 21, 6, 208, 222, 58, 91, 178, 217, 224, 168, 18, 222, 149, 165, 195, 1, 220, 63, 109, 153, 51, 151, 229, 158, 82, 172, 26, 67, 60, 128, 157, 64, 104, 131, 255, 88, 16, 208, 175, 211, 2, 221, 140, 200, 120, 169, 70, 142, 95, 183, 3, 213, 23, 125, 37, 157, 167, 88, 80, 25, 209, 184, 156, 91, 21, 242, 140, 250, 116, 227, 114, 214, 49, 98, 196, 58, 194, 9, 177, 223, 62, 88, 123, 14, 196, 224, 118, 247, 245, 103, 42, 239, 16, 170, 62, 255, 246, 244, 228, 1, 149, 146, 205, 47, 169, 21, 105, 0, 148, 137, 158, 170, 45, 16, 239, 179, 180, 120, 90, 131, 105, 16}
var expectedSignatureTwo = []byte{97, 206, 133, 26, 212, 226, 126, 58, 99, 225, 29, 219, 143, 47, 86, 153, 2, 43, 151, 157, 37, 150, 30, 81, 206, 141, 255, 164, 203, 254, 173, 35, 77, 150, 7, 208, 79, 82, 39, 163, 81, 230, 188, 149, 161, 54, 113, 241, 80, 97, 198, 225, 93, 130, 169, 46, 76, 115, 202, 101, 219, 201, 233, 60, 85, 181, 153, 153, 192, 56, 41, 119, 7, 211, 202, 245, 95, 150, 186, 162, 48, 77, 15, 192, 15, 196, 29, 68, 169, 212, 47, 46, 115, 242, 171, 86, 57, 170, 127, 23, 166, 36, 42, 174, 70, 73, 65, 255, 254, 199, 16, 165, 57, 77, 91, 145, 132, 180, 211, 123, 210, 161, 7, 114, 180, 130, 242, 52, 27, 211, 138, 163, 38, 233, 122, 102, 172, 217, 40, 99, 203, 255, 239, 147, 20, 249, 52, 109, 45, 106, 16, 41, 221, 45, 29, 125, 197, 42, 80, 167, 165, 82, 10, 54, 19, 114, 240, 127, 212, 126, 86, 125, 35, 142, 130, 172, 144, 7, 238, 215, 29, 105, 70, 171, 217, 161, 214, 26, 30, 201, 119, 191, 77, 81, 86, 118, 15, 180, 185, 20, 220, 236, 183, 67, 242, 255, 93, 16, 1, 31, 177, 211, 189, 231, 125, 83, 213, 65, 3, 209, 186, 70, 76, 51, 109, 153, 24, 81, 200, 57, 43, 8, 91, 24, 64, 118, 108, 237, 8, 204, 206, 95, 215, 72, 160, 42, 214, 133, 140, 86, 206, 0, 152, 139, 67, 234}

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
		prngOne: &CountingReader{count: 0},
		prngTwo: &CountingReader{count: 5},

		mockTS: testTime,
	}, nil
}

type MockRegistrationSender struct {
	reg *pb.ClientRegistration
	// param passed to SendRegistrationMessage
	host    *connect.Host
	privKey *rsa.PrivateKey
	prngOne *CountingReader
	prngTwo *CountingReader

	mockTS time.Time
	// original host returned from GetHost
	getHost             *connect.Host
	errSendRegistration error
	errInReply          string
}

func (s *MockRegistrationSender) SendRegistrationMessage(host *connect.Host, message *pb.ClientRegistration) (*pb.SignedClientRegistrationConfirmations, error) {
	transSig, err := registration.SignWithTimestamp(s.prngOne, s.privKey,
		s.mockTS.UnixNano(), message.ClientTransmissionRSAPubKey)
	if err != nil {
		return nil, errors.Errorf("Failed to sign transmission: %v", err)
	}

	receptionSig, err := registration.SignWithTimestamp(s.prngTwo, s.privKey,
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
		t.Fatalf("Unexpected timestamp returned from register: "+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", sender.mockTS.UnixNano(), regTimestamp)
	}

	if !bytes.Equal(expectedSignatureOne, sig1) {
		t.Fatalf("Unexpected signature one."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedSignatureOne, sig1)
	}

	if !bytes.Equal(expectedSignatureTwo, sig2) {
		t.Fatalf("Unexpected signature two."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedSignatureTwo, sig2)
	}

}

// Shows that returning an error from the registration server results in an
// error from register
func TestRegisterWithPermissioning_ResponseErr(t *testing.T) {
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

	sender.errInReply = "failure occurred on registration"
	_, _, _, err = register(sender, nil, key.GetPublic(), key.GetPublic(), "")
	if err == nil {
		t.Error("no error if registration fails on registration")
	}
}

// Shows that returning an error from the RPC (e.g. context deadline exceeded)
// results in an error from register
func TestRegisterWithPermissioning_ConnectionErr(t *testing.T) {
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
	sender.errSendRegistration = errors.New("connection problem")
	_, _, _, err = register(sender, nil, key.GetPublic(), key.GetPublic(), "")
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
