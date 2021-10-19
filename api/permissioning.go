///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/user"
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



	err = c.printProtoUser(regCode, transmissionRegValidationSignature, receptionRegValidationSignature)
	if err != nil {
		return errors.WithMessage(err, "failed to print proto user")
	}

	//update the registration state
	err = c.storage.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return errors.WithMessage(err, "failed to update local state "+
			"after registration with permissioning")
	}
	return nil
}

// todo: remove once deploy has been tested
func (c *Client) printProtoUser(regCode string,
	transmissionRegValidationSignature, receptionRegValidationSignature []byte) error {
	// todo: remove this once proto has been generated
	username, err := c.GetStorage().User().GetUsername()
	if err != nil {
		return errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	Usr := user.Proto{
		TransmissionID:               c.GetUser().TransmissionID,
		TransmissionSalt:               c.GetUser().TransmissionSalt,
		TransmissionRSA:                c.GetUser().TransmissionRSA,
		ReceptionID:                    c.GetUser().ReceptionID,
		ReceptionSalt:                c.GetUser().ReceptionSalt,
		ReceptionRSA:                 c.GetUser().ReceptionRSA,
		Precanned:                    c.GetUser().Precanned,
		RegistrationTimestamp:       c.GetUser().RegistrationTimestamp,
		Username:                    username,
		RegCode:                     regCode,
		TransmissionRegValidationSig: transmissionRegValidationSignature,
		ReceptionRegValidationSig:    receptionRegValidationSignature,
		CmixDhPrivateKey:             c.GetStorage().Cmix().GetDHPrivateKey(),
		CmixDhPublicKey:              c.GetStorage().Cmix().GetDHPublicKey(),
		E2eDhPrivateKey:              c.GetStorage().E2e().GetDHPrivateKey(),
		E2eDhPublicKey:               c.GetStorage().E2e().GetDHPublicKey(),
	}

	jsonBytes, err := json.Marshal(Usr)
	if err != nil {
		return errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	jww.INFO.Printf("PROTO USER JSON: \n%s", string(jsonBytes))

	return nil
}