///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
)

// Returns an error if registration fails.
func (c *Cmix) registerWithPermissioning() error {
	//get the users public key
	transmissionPubKey := c.storage.GetTransmissionRSA().GetPublic()
	receptionPubKey := c.storage.GetReceptionRSA().GetPublic()

	//load the registration code
	regCode, err := c.storage.GetRegCode()
	if err != nil {
		return errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	//register with registration
	transmissionRegValidationSignature, receptionRegValidationSignature,
		registrationTimestamp, err := c.permissioning.Register(
		transmissionPubKey, receptionPubKey, regCode)
	if err != nil {
		return errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	//store the signature
	c.storage.SetTransmissionRegistrationValidationSignature(
		transmissionRegValidationSignature)
	c.storage.SetReceptionRegistrationValidationSignature(
		receptionRegValidationSignature)
	c.storage.SetRegistrationTimestamp(registrationTimestamp)

	//update the registration state
	err = c.storage.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return errors.WithMessage(err, "failed to update local state "+
			"after registration with permissioning")
	}
	return nil
}

// ConstructProtoUserFile is a helper function which is used for proto
// client testing.  This is used for development testing.
func (c *Cmix) ConstructProtoUserFile() ([]byte, error) {

	//load the registration code
	regCode, err := c.storage.GetRegCode()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	Usr := user.Proto{
		TransmissionID:               c.GetUser().TransmissionID,
		TransmissionSalt:             c.GetUser().TransmissionSalt,
		TransmissionRSA:              c.GetUser().TransmissionRSA,
		ReceptionID:                  c.GetUser().ReceptionID,
		ReceptionSalt:                c.GetUser().ReceptionSalt,
		ReceptionRSA:                 c.GetUser().ReceptionRSA,
		Precanned:                    c.GetUser().Precanned,
		RegistrationTimestamp:        c.GetUser().RegistrationTimestamp,
		RegCode:                      regCode,
		TransmissionRegValidationSig: c.storage.GetTransmissionRegistrationValidationSignature(),
		ReceptionRegValidationSig:    c.storage.GetReceptionRegistrationValidationSignature(),
		E2eDhPrivateKey:              nil,
		E2eDhPublicKey:               nil,
	}

	jsonBytes, err := json.Marshal(Usr)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	return jsonBytes, nil
}
