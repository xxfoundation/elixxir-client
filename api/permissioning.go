////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/storage"
)

// Returns an error if registration fails.
func (c *Client) RegisterWithPermissioning(registrationCode string) error {
	ctx := c.ctx
	netman := ctx.Manager

	//Check the regState is in proper state for registration
	regState := c.storage.GetRegistrationStatus()
	if regState != storage.KeyGenComplete {
		return errors.Errorf("Attempting to register before key generation!")
	}

	userData := ctx.Session.User()

	// Register with the permissioning server and generate user information
	regValidationSignature, err := netman.RegisterWithPermissioning(
		registrationCode)
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
