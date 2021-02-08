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
	"gitlab.com/xx_network/crypto/signature/rsa"
)

func (perm *Permissioning) Register(transmissionPublicKey, receptionPublicKey *rsa.PublicKey, registrationCode string) ([]byte, []byte, error) {
	return register(perm.comms, perm.host, transmissionPublicKey, receptionPublicKey, registrationCode)
}

// client.Comms should implement this interface
type registrationMessageSender interface {
	SendRegistrationMessage(host *connect.Host, message *pb.UserRegistration) (*pb.UserRegistrationConfirmation, error)
}

//register registers the user with optional registration code
// Returns an error if registration fails.
func register(comms registrationMessageSender, host *connect.Host,
	transmissionPublicKey, receptionPublicKey *rsa.PublicKey, registrationCode string) ([]byte, []byte, error) {

	response, err := comms.
		SendRegistrationMessage(host,
			&pb.UserRegistration{
				RegistrationCode:         registrationCode,
				ClientRSAPubKey:          string(rsa.CreatePublicKeyPem(transmissionPublicKey)),
				ClientReceptionRSAPubKey: string(rsa.CreatePublicKeyPem(receptionPublicKey)),
			})
	if err != nil {
		err = errors.Wrap(err, "sendRegistrationMessage: Unable to contact Identity Server!")
		return nil, nil, err
	}
	if response.Error != "" {
		return nil, nil, errors.Errorf("sendRegistrationMessage: error handling message: %s", response.Error)
	}
	return response.ClientSignedByServer.Signature, response.ClientReceptionSignedByServer.Signature, nil
}
