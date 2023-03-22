////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"io"
)

// RegisterForNotifications allows a client to register for push notifications.
// Note that clients are not required to register for push notifications,
// especially as these rely on third parties (i.e., Firebase *cough* *cough*
// Google's palantir *cough*) that may represent a security risk to the user.
// A client can register to receive push notifications on many IDs.
func (c *Cmix) RegisterForNotifications(toBeNotifiedOn *id.ID, token string) error {
	jww.INFO.Printf("RegisterForNotifications(%s)", token)
	// Pull the host from the manage
	notificationBotHost, ok := c.GetComms().GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("RegisterForNotifications: " +
			"Failed to retrieve host for notification bot")
	}
	stream := c.GetRng().GetStream()
	defer stream.Close()
	intermediaryReceptionID, sig, err := getIidAndSig(c.GetStorage().GetTransmissionRSA(),
		toBeNotifiedOn, stream)
	if err != nil {
		return err
	}

	privKey := c.GetStorage().GetTransmissionRSA()
	pubPEM := privKey.Public().MarshalPem()
	regSig := c.GetStorage().GetTransmissionRegistrationValidationSignature()
	regTS := c.GetStorage().GetRegistrationTimestamp()

	// Send the register message
	_, err = c.GetComms().RegisterForNotifications(notificationBotHost,
		&mixmessages.NotificationRegisterRequest{
			Token:                 token,
			IntermediaryId:        intermediaryReceptionID,
			TransmissionRsa:       pubPEM,
			TransmissionSalt:      c.GetStorage().GetTransmissionSalt(),
			TransmissionRsaSig:    regSig,
			IIDTransmissionRsaSig: sig,
			RegistrationTimestamp: regTS.UnixNano(),
		})
	if err != nil {
		err := errors.Errorf("RegisterForNotifications: Unable to "+
			"register for notifications! %s", err)
		return err
	}

	return nil
}

// UnregisterForNotifications turns off notifications for this client.
func (c *Cmix) UnregisterForNotifications(toBeNotifiedOn *id.ID) error {
	jww.INFO.Printf("UnregisterForNotifications()")
	// Pull the host from the manage
	notificationBotHost, ok := c.GetComms().GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("Failed to retrieve host for notification bot")
	}

	stream := c.GetRng().GetStream()
	defer stream.Close()
	intermediaryReceptionID, sig, err := getIidAndSig(c.GetStorage().GetTransmissionRSA(),
		toBeNotifiedOn, stream)
	if err != nil {
		return err
	}
	// Sends the unregister message
	_, err = c.GetComms().UnregisterForNotifications(notificationBotHost,
		&mixmessages.NotificationUnregisterRequest{
			IntermediaryId:        intermediaryReceptionID,
			IIDTransmissionRsaSig: sig,
		})
	if err != nil {
		err := errors.Errorf(
			"RegisterForNotifications: Unable to register for notifications! %s", err)
		return err
	}

	return nil
}

func getIidAndSig(signer rsa.PrivateKey, toBeNotified *id.ID, rng io.Reader) ([]byte, []byte, error) {
	intermediaryReceptionID, err := ephemeral.GetIntermediaryId(
		toBeNotified)
	if err != nil {
		return nil, nil, errors.WithMessage(err,
			"RegisterForNotifications: Failed to form intermediary ID")
	}
	h, err := hash.NewCMixHash()
	if err != nil {
		return nil, nil, errors.WithMessage(err,
			"RegisterForNotifications: Failed to create cMix hash")
	}
	_, err = h.Write(intermediaryReceptionID)
	if err != nil {
		return nil, nil, errors.WithMessage(err,
			"RegisterForNotifications: Failed to write intermediary ID to hash")
	}

	sig, err := signer.SignPSS(rng,
		hash.CMixHash, h.Sum(nil), nil)
	if err != nil {
		return nil, nil, errors.WithMessage(err,
			"RegisterForNotifications: Failed to sign intermediary ID")
	}
	return intermediaryReceptionID, sig, nil
}
