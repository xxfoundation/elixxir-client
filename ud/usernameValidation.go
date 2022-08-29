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
	"gitlab.com/xx_network/comms/connect"
)

// userValidationComms is a sub-interface of the Comms interface for
// username validation.
type userValidationComms interface {
	SendUsernameValidation(host *connect.Host,
		message *pb.UsernameValidationRequest) (*pb.UsernameValidation, error)
}

// GetUsernameValidationSignature will lazily load a username validation
// signature. If it is not already present within the Manager object, it
// will query the signature from the UD service.
func (m *Manager) GetUsernameValidationSignature() ([]byte, error) {
	m.usernameValidationMux.Lock()
	defer m.usernameValidationMux.Unlock()
	var err error

	// If validation signature is not present, request it from
	// UD
	if m.usernameValidationSignature == nil {
		m.usernameValidationSignature, err = m.getUsernameValidationSignature(m.comms)
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
	comms userValidationComms) ([]byte, error) {

	// Construct request for username validation
	request := &pb.UsernameValidationRequest{
		UserId: m.myID.Bytes(),
	}

	// Get UD host
	host, err := m.getHost()
	if err != nil {
		return nil, err
	}

	// Send request
	response, err := comms.SendUsernameValidation(host, request)
	if err != nil {
		return nil, err
	}

	// Verify response is valid
	err = crust.VerifyVerificationSignature(host.GetPubKey(), response.Username,
		response.ReceptionPublicKeyPem, response.Signature)
	if err != nil {
		return nil, err
	}

	// Return response
	return response.Signature, nil
}
