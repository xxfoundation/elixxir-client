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

// UnregisterNotificationIdentity turns off notifications for a specific identity
func (c *Cmix) UnregisterNotificationIdentity(toBeNotifiedOn *id.ID) error {
	jww.INFO.Printf("UnregisterNotificationIdentity(%s)", toBeNotifiedOn)

	// Pull the host from the manage

	stream := c.GetRng().GetStream()
	intermediaryReceptionID, sig, err := getIidAndSig(
		c.GetStorage().GetTransmissionRSA(), toBeNotifiedOn, stream)
	stream.Close()
	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications")
	}

	return c.unregisterForNotifications(&mixmessages.NotificationUnregisterRequest{
		TransmissionRSA:       c.GetStorage().GetTransmissionRSA().Public().MarshalPem(),
		IntermediaryId:        intermediaryReceptionID,
		IIDTransmissionRsaSig: sig,
	})
}

// UnregisterNotificationDevice turns off notifications for a specific device
// token.
func (c *Cmix) UnregisterNotificationDevice(token string) error {
	jww.INFO.Printf("UnregisterNotificationDevice(%s)", token)

	// Pull the host from the manage

	stream := c.GetRng().GetStream()
	receptionID := c.GetStorage().GetReceptionID()
	_, sig, err := getIidAndSig(
		c.GetStorage().GetTransmissionRSA(), receptionID, stream)
	stream.Close()
	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications")
	}

	return c.unregisterForNotifications(&mixmessages.NotificationUnregisterRequest{
		Token:                 token,
		IIDTransmissionRsaSig: sig,
		TransmissionRSA:       c.GetStorage().GetTransmissionRSA().Public().MarshalPem(),
	})
}

// UnregisterNotificationLegacy unregisters a user using only the intermediary
// of their reception ID, hitting the legacy endpoint.  This only works for xxm
// clients where users have a 1:! relationship with both identities and tokens.
func (c *Cmix) UnregisterNotificationLegacy() error {
	toBeNotifiedOn := c.GetStorage().GetReceptionID()
	jww.INFO.Printf("UnregisterNotificationLegacy(%s)", toBeNotifiedOn)

	// Pull the host from the manage

	stream := c.GetRng().GetStream()
	intermediaryReceptionID, sig, err := getIidAndSig(
		c.GetStorage().GetTransmissionRSA(), toBeNotifiedOn, stream)
	stream.Close()
	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications")
	}

	return c.unregisterForNotifications(&mixmessages.NotificationUnregisterRequest{
		IntermediaryId:        intermediaryReceptionID,
		IIDTransmissionRsaSig: sig,
	})
}

func (c *Cmix) unregisterForNotifications(request *mixmessages.NotificationUnregisterRequest) error {
	notificationBotHost, ok := c.GetComms().GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("UnregisterForNotifications: " +
			"Failed to retrieve host for notification bot")
	}

	// Sends the unregister message
	_, err := c.GetComms().UnregisterForNotifications(notificationBotHost,
		request)
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
