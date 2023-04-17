////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"io"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// RegisterForNotifications allows a client to register for push notifications.
// Note that clients are not required to register for push notifications,
// especially as these rely on third parties (i.e., Firebase *cough* *cough*
// Google's Palantir *cough*) that may represent a security risk to the user.
// A client can register to receive push notifications on many IDs.
func (c *Cmix) RegisterForNotifications(toBeNotifiedOn *id.ID, token string) error {
	jww.INFO.Printf("RegisterForNotifications(%s, %s)", toBeNotifiedOn, token)

	// Pull the host from the manage
	notificationBotHost, ok := c.GetComms().GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("RegisterForNotifications: " +
			"Failed to retrieve host for notification bot")
	}
	stream := c.GetRng().GetStream()
	intermediaryReceptionID, sig, err := getIidAndSig(
		c.GetStorage().GetTransmissionRSA(), toBeNotifiedOn, stream)
	stream.Close()
	if err != nil {
		return errors.Wrap(err, "RegisterForNotifications")
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
		return errors.Wrap(err, "RegisterForNotifications: Unable to "+
			"register for notifications!")
	}

	return nil
}

// UnregisterForNotifications turns off notifications for this client.
func (c *Cmix) UnregisterForNotifications(toBeNotifiedOn *id.ID) error {
	jww.INFO.Printf("UnregisterForNotifications(%s)", toBeNotifiedOn)

	// Pull the host from the manage
	notificationBotHost, ok := c.GetComms().GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("UnregisterForNotifications: " +
			"Failed to retrieve host for notification bot")
	}

	stream := c.GetRng().GetStream()
	intermediaryReceptionID, sig, err := getIidAndSig(
		c.GetStorage().GetTransmissionRSA(), toBeNotifiedOn, stream)
	stream.Close()
	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications")
	}

	// Sends the unregister message
	_, err = c.GetComms().UnregisterForNotifications(notificationBotHost,
		&mixmessages.NotificationUnregisterRequest{
			IntermediaryId:        intermediaryReceptionID,
			IIDTransmissionRsaSig: sig,
		})
	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications: Unable to "+
			"unregister for notifications!")
	}

	return nil
}

func getIidAndSig(signer rsa.PrivateKey, toBeNotified *id.ID, rng io.Reader) (
	intermediaryReceptionID, sig []byte, err error) {
	intermediaryReceptionID, err = ephemeral.GetIntermediaryId(toBeNotified)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to form intermediary ID")
	}
	h, err := hash.NewCMixHash()
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to create cMix hash")
	}
	_, err = h.Write(intermediaryReceptionID)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to write intermediary ID to hash")
	}

	sig, err = signer.SignPSS(rng, hash.CMixHash, h.Sum(nil), nil)
	if err != nil {
		return nil, nil,
			errors.WithMessage(err, "Failed to sign intermediary ID")
	}
	return intermediaryReceptionID, sig, nil
}
