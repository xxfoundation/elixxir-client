///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	usernameValidationStore   = "usernameValidationStore"
	usernameValidationVersion = 0
)

// GetUsernameValidationSignature will lazily load a username validation
// signature. If it is not already present within the Manager object, it will
// retrieve it from storage. If the signature is not present in either
// structure, it will query the signature from the UD service.
func (m *Manager) GetUsernameValidationSignature() ([]byte, error) {
	m.usernameValidationMux.Lock()
	defer m.usernameValidationMux.Unlock()
	var err error

	// If validation signature is not present, request it from
	// UD
	if m.usernameValidationSignature == nil {
		// Check if the data is in storage
		m.usernameValidationSignature, err = m.loadOrGetUsernameValidation()
		if err != nil {
			return nil, err
		}
	}

	return m.usernameValidationSignature, nil
}

// loadOrGetUsernameValidation is a helper function which will lazily load the
// username validation signature from storage. If it does not exist in storage,
// then it will query the signature from the UD service.
func (m *Manager) loadOrGetUsernameValidation() ([]byte, error) {
	var validationSignature []byte

	// Attempt storage retrieval
	obj, err := m.getKv().Get(usernameValidationStore, usernameValidationVersion)
	if err != nil {
		// If we failed to retrieve from storage,
		// request the username from the network, and set on the object
		validationSignature,
			err = m.queryUsernameValidationSignature(m.comms)
		if err != nil {
			return nil, errors.Errorf("Failed to retrieve signature from "+
				"UD: %v", err)
		}
	} else {
		// Put stored data in the object data
		validationSignature = obj.Data
	}

	return validationSignature, nil
}

// queryUsernameValidationSignature is the helper function which queries
// the signature from the UD service.
func (m *Manager) queryUsernameValidationSignature(
	comms userValidationComms) ([]byte, error) {

	// Construct request for username validation
	request := &pb.UsernameValidationRequest{
		UserId: m.user.GetReceptionIdentity().ID.Bytes(),
	}

	// Send request
	response, err := comms.SendUsernameValidation(m.ud.host, request)
	if err != nil {
		return nil, err
	}

	publicKey, err := rsa.LoadPublicKeyFromPem(response.ReceptionPublicKeyPem)
	if err != nil {
		return nil, err
	}

	// Verify response is valid
	err = crust.VerifyVerificationSignature(m.ud.host.GetPubKey(),
		crust.HashUsername(response.Username),
		publicKey, response.Signature)
	if err != nil {
		return nil, err
	}

	// Store request
	// fixme: need to pull release for the KV API update
	err = m.getKv().Set(usernameValidationStore, 0,
		&versioned.Object{
			Version:   usernameValidationVersion,
			Timestamp: netTime.Now(),
			Data:      response.Signature,
		})
	if err != nil {
		return nil, err
	}

	// Return response
	return response.Signature, nil
}
