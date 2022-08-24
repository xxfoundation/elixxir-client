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
	"gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// GetUsernameValidationSignature will lazily load a username validation
// signature. If it is not already present within the Manager object, it
// will query the signature from the UD service.
func (m *Manager) GetUsernameValidationSignature() ([]byte, error) {
	m.usernameValidationMux.Lock()
	defer m.usernameValidationMux.Unlock()
	var err error

	// Retrieve username
	username, err := m.store.GetUsername()
	if err != nil {
		return nil, errors.Errorf("Failed to retrieve username "+
			"within store: %+v", err)
	}

	// If validation signature is not present, request it from
	// UD
	if m.usernameValidationSignature == nil {
		m.usernameValidationSignature, err = m.getUsernameValidationSignature(
			username, m.comms)
		if err != nil {
			return nil, errors.Errorf("Failed to retrieve signature from "+
				"UD: %v", err)
		}
	}

	return m.usernameValidationSignature, nil
}

// getUsernameValidationSignature is the helper function which queries
// the signature from the UD service.
func (m *Manager) getUsernameValidationSignature(
	username string, comms userValidationComms) (
	[]byte, error) {

	// Retrieve the public key and serialize it to a PEM file
	rsaPrivKey, err := m.user.GetReceptionIdentity().GetRSAPrivateKey()
	if err != nil {
		return nil, err
	}
	publicKeyPem := rsa.CreatePublicKeyPem(rsaPrivKey.GetPublic())

	// Construct request for username validation
	request := &pb.UsernameValidationRequest{
		Username:              username,
		ReceptionPublicKeyPem: publicKeyPem,
		UserId:                m.user.GetReceptionIdentity().ID.Bytes(),
	}

	// Send request
	response, err := comms.SendUsernameValidation(m.ud.host, request)
	if err != nil {
		return nil, err
	}

	// Verify response is valid
	err = crust.VerifyVerificationSignature(m.ud.host.GetPubKey(), username,
		publicKeyPem, response.Signature)
	if err != nil {
		return nil, err
	}

	// Return response
	return response.Signature, nil
}
