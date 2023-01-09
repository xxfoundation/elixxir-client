////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/storage/user"
)

// registerWithPermissioning returns an error if registration fails.
func (c *Cmix) registerWithPermissioning() error {
	// Get the users public key
	transmissionPubKey := c.storage.GetTransmissionRSA().Public()
	receptionPubKey := c.storage.GetReceptionRSA().Public()

	// Load the registration code
	regCode, err := c.storage.GetRegCode()
	if err != nil {
		return errors.WithMessage(err, "failed to register with permissioning")
	}

	// Register with registration
	transmissionRegValidationSignature, receptionRegValidationSignature,
		registrationTimestamp, err := c.permissioning.Register(
		transmissionPubKey, receptionPubKey, regCode)
	if err != nil {
		return errors.WithMessage(err, "failed to register with permissioning")
	}

	// store the signature
	c.storage.SetTransmissionRegistrationValidationSignature(
		transmissionRegValidationSignature)
	c.storage.SetReceptionRegistrationValidationSignature(
		receptionRegValidationSignature)
	c.storage.SetRegistrationTimestamp(registrationTimestamp)

	// Update the registration state
	err = c.storage.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return errors.WithMessage(err, "failed to update local state "+
			"after registration with permissioning")
	}
	return nil
}

// ConstructProtoUserFile is a helper function that is used for proto client
// testing. This is used for development testing.
func (c *Cmix) ConstructProtoUserFile() ([]byte, error) {

	// Load the registration code
	regCode, err := c.storage.GetRegCode()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get registration code")
	}

	userInfo := c.GetStorage().PortableUserInfo()
	Usr := user.Proto{
		TransmissionID:               userInfo.TransmissionID,
		TransmissionSalt:             userInfo.TransmissionSalt,
		TransmissionRSA:              userInfo.TransmissionRSA,
		ReceptionID:                  userInfo.ReceptionID,
		ReceptionSalt:                userInfo.ReceptionSalt,
		ReceptionRSA:                 userInfo.ReceptionRSA,
		Precanned:                    userInfo.Precanned,
		RegistrationTimestamp:        userInfo.RegistrationTimestamp,
		RegCode:                      regCode,
		TransmissionRegValidationSig: c.storage.GetTransmissionRegistrationValidationSignature(),
		ReceptionRegValidationSig:    c.storage.GetReceptionRegistrationValidationSignature(),
		E2eDhPrivateKey:              nil,
		E2eDhPublicKey:               nil,
	}

	jsonBytes, err := json.Marshal(Usr)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to JSON marshal user.Proto")
	}

	return jsonBytes, nil
}
