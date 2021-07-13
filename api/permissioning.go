///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage"
)

// Returns an error if registration fails.
func (c *Client) registerWithPermissioning() error {
	userData := c.storage.User()
	//get the users public key
	transmissionPubKey := userData.GetCryptographicIdentity().GetTransmissionRSA().GetPublic()
	receptionPubKey := userData.GetCryptographicIdentity().GetReceptionRSA().GetPublic()

	//load the registration code
	regCode, err := c.storage.GetRegCode()
	if err != nil {
		return errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	//register with registration
	transmissionRegValidationSignature, receptionRegValidationSignature,
		registrationTimestamp, err := c.permissioning.Register(transmissionPubKey, receptionPubKey, regCode)
	if err != nil {
		return errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	//store the signature
	userData.SetTransmissionRegistrationValidationSignature(transmissionRegValidationSignature)
	userData.SetReceptionRegistrationValidationSignature(receptionRegValidationSignature)
	userData.SetRegistrationTimestamp(registrationTimestamp)

	//update the registration status
	err = c.storage.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return errors.WithMessage(err, "failed to update local state "+
			"after registration with permissioning")
	}
	return nil
}
