////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/network/permissioning"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
)

// Returns an error if registration fails.
func RegisterWithPermissioning(ctx context.Context, comms client.Comms, registrationCode string) error {
	instance := ctx.Manager.GetInstance()
	instance.GetPartialNdf()

	//Check the regState is in proper state for registration
	regState := ctx.Session.GetRegistrationStatus()
	if regState != storage.KeyGenComplete {
		return errors.Errorf("Attempting to register before key generation!")
	}

	userData := ctx.Session.User()

	// Register with the permissioning server and generate user information
	regValidationSignature, err := permissioning.RegisterWithPermissioning(&comms,
		userData.GetCryptographicIdentity().GetRSA().GetPublic(), registrationCode)
	if err != nil {
		globals.Log.INFO.Printf(err.Error())
		return err
	}

	// update the session with the registration response
	userData.SetRegistrationValidationSignature(regValidationSignature)

	err = ctx.Session.ForwardRegistrationStatus(storage.PermissioningComplete)
	if err != nil {
		return err
	}

	return nil
}
