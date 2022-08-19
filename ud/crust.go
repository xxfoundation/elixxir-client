///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// GetUsernameValidationSignature will lazily load a username validation signature.
// If it is not already present within the Manager object, it will query the
// signature from the UD service.
// todo: there needs to be a way to retrieve the username from ud.store (in Master
//  this is valid)
func (m *Manager) GetUsernameValidationSignature(username string) ([]byte, error) {
	m.usernameValidationMux.Lock()
	defer m.usernameValidationMux.Unlock()
	var err error
	if m.usernameValidationSignature == nil {
		m.usernameValidationSignature, err = m.getUsernameValidationSignature(username)
		if err != nil {
			return nil, errors.Errorf("Failed to retrieve signature from UD: %v", err)
		}
	}

	return m.usernameValidationSignature, nil
}

// getUsernameValidationSignature is the helper function which queries the signature from
// the UD service.
func (m *Manager) getUsernameValidationSignature(username string) ([]byte, error) {
	rsaPrivKey, err := m.user.GetReceptionIdentity().GetRSAPrivateKey()
	if err != nil {
		return nil, err
	}

	publicKeyPem := rsa.CreatePublicKeyPem(rsaPrivKey.GetPublic())

	request := &pb.UsernameValidationRequest{
		Username:              username,
		ReceptionPublicKeyPem: publicKeyPem,
		UserId:                m.user.GetReceptionIdentity().ID.Bytes(),
	}

	response, err := m.comms.SendUsernameValidation(m.ud.host, request)
	if err != nil {
		return nil, err
	}

	return response.Signature, nil
}
