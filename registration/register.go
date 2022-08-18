///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package registration

import (
	"encoding/base64"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

func (perm *Registration) Register(transmissionPublicKey, receptionPublicKey *rsa.PublicKey,
	registrationCode string) (transmissionSig []byte, receptionSig []byte, regTimestamp int64, err error) {
	return register(perm.comms, perm.host, transmissionPublicKey, receptionPublicKey, registrationCode)
}

// client.Comms should implement this interface
type registrationMessageSender interface {
	SendRegistrationMessage(host *connect.Host, message *pb.ClientRegistration) (*pb.SignedClientRegistrationConfirmations, error)
}

//register registers the user with optional registration code
// Returns an error if registration fails.
func register(comms registrationMessageSender, host *connect.Host,
	transmissionPublicKey, receptionPublicKey *rsa.PublicKey,
	registrationCode string) (
	transmissionSig []byte, receptionSig []byte, regTimestamp int64, err error) {

	// Send the message
	transmissionPem := string(rsa.CreatePublicKeyPem(transmissionPublicKey))
	receptionPem := string(rsa.CreatePublicKeyPem(receptionPublicKey))
	response, err := comms.
		SendRegistrationMessage(host,
			&pb.ClientRegistration{
				RegistrationCode:            registrationCode,
				ClientTransmissionRSAPubKey: transmissionPem,
				ClientReceptionRSAPubKey:    receptionPem,
			})
	if err != nil {
		err = errors.Wrap(err, "sendRegistrationMessage: Unable to "+
			"contact Identity Server!")
		return nil, nil, 0, err
	}
	if response.Error != "" {
		return nil, nil, 0, errors.Errorf("sendRegistrationMessage: "+
			"error handling message: %s", response.Error)
	}

	// Unmarshal reception confirmation
	receptionConfirmation := &pb.ClientRegistrationConfirmation{}
	err = proto.Unmarshal(response.GetClientReceptionConfirmation().
		ClientRegistrationConfirmation, receptionConfirmation)
	if err != nil {
		return nil, nil, 0, errors.WithMessage(err, "Failed to unmarshal "+
			"reception confirmation message")
	}

	// Verify reception signature
	receptionSignature := response.GetClientReceptionConfirmation().
		GetRegistrarSignature().Signature
	err = registration.VerifyWithTimestamp(host.GetPubKey(),
		receptionConfirmation.Timestamp, receptionPem,
		receptionSignature)
	if err != nil {
		return nil, nil, 0, errors.WithMessage(err, "Failed to verify reception signature")
	}

	// Unmarshal transmission confirmation
	transmissionConfirmation := &pb.ClientRegistrationConfirmation{}
	err = proto.Unmarshal(response.GetClientTransmissionConfirmation().
		ClientRegistrationConfirmation, transmissionConfirmation)
	if err != nil {
		return nil, nil, 0, errors.WithMessage(err, "Failed to unmarshal "+
			"transmission confirmation message")
	}

	jww.WARN.Printf("UD DEBUG: IN NETWORK REGISTER:"+
		"\ntimestamp: %d"+
		"\nrsa pub key PEM: %s"+
		"\nperm sig: %s",
		receptionConfirmation.Timestamp, receptionPem, base64.StdEncoding.EncodeToString(receptionSignature),
	)

	// Verify transmission signature
	transmissionSignature := response.GetClientTransmissionConfirmation().
		GetRegistrarSignature().Signature
	err = registration.VerifyWithTimestamp(host.GetPubKey(),
		transmissionConfirmation.Timestamp, transmissionPem,
		transmissionSignature)
	if err != nil {
		return nil, nil, 0, errors.WithMessage(err, "Failed to verify transmission signature")
	}

	return transmissionSignature,
		receptionSignature,
		receptionConfirmation.Timestamp, nil
}
