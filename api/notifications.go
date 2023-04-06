///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"math"
	"time"
)

// RegisterForNotifications allows a client to register for push
// notifications.
// Note that clients are not required to register for push notifications
// especially as these rely on third parties (i.e., Firebase *cough*
// *cough* google's palantir *cough*) that may represent a security
// risk to the user.
func (c *Client) RegisterForNotifications(token string) error {
	jww.INFO.Printf("RegisterForNotifications(%s)", token)
	// Pull the host from the manage
	notificationBotHost, ok := c.comms.GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("RegisterForNotifications: Failed to retrieve host for notification bot")
	}
	intermediaryReceptionID, sig, err := c.getIidAndSig()
	if err != nil {
		return err
	}
	// Send the register message
	_, err = c.comms.RegisterForNotifications(notificationBotHost,
		&mixmessages.NotificationRegisterRequest{
			Token:                 token,
			IntermediaryId:        intermediaryReceptionID,
			TransmissionRsa:       rsa.CreatePublicKeyPem(c.GetStorage().User().GetCryptographicIdentity().GetTransmissionRSA().GetPublic()),
			TransmissionSalt:      c.GetUser().TransmissionSalt,
			TransmissionRsaSig:    c.GetStorage().User().GetTransmissionRegistrationValidationSignature(),
			IIDTransmissionRsaSig: sig,
			RegistrationTimestamp: c.GetUser().RegistrationTimestamp,
		})
	if err != nil {
		err := errors.Errorf(
			"RegisterForNotifications: Unable to register for notifications! %s", err)
		return err
	}

	return nil
}

// UnregisterForNotifications turns of notifications for this client
func (c *Client) UnregisterForNotifications() error {
	jww.INFO.Printf("UnregisterForNotifications()")
	// Pull the host from the manage
	notificationBotHost, ok := c.comms.GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("Failed to retrieve host for notification bot")
	}
	intermediaryReceptionID, sig, err := c.getIidAndSig()
	if err != nil {
		return err
	}
	// Send the unregister message
	_, err = c.comms.UnregisterForNotifications(notificationBotHost, &mixmessages.NotificationUnregisterRequest{
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

func (c *Client) getIidAndSig() ([]byte, []byte, error) {
	intermediaryReceptionID, err := ephemeral.GetIntermediaryId(c.GetUser().ReceptionID)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "RegisterForNotifications: Failed to form intermediary ID")
	}
	h, err := hash.NewCMixHash()
	if err != nil {
		return nil, nil, errors.WithMessage(err, "RegisterForNotifications: Failed to create cmix hash")
	}
	_, err = h.Write(intermediaryReceptionID)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "RegisterForNotifications: Failed to write intermediary ID to hash")
	}

	stream := c.rng.GetStream()
	c.GetUser()
	sig, err := rsa.Sign(stream, c.storage.User().GetCryptographicIdentity().GetTransmissionRSA(), hash.CMixHash, h.Sum(nil), nil)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "RegisterForNotifications: Failed to sign intermediary ID")
	}
	stream.Close()
	return intermediaryReceptionID, sig, nil
}

func (c *Client) SetAlternativeNotificationsHost(address string, tlsCertificate []byte) {
	c.comms.RemoveHost(&id.NotificationBot)
	hp := connect.GetDefaultHostParams()
	// Client will not send KeepAlive packets
	hp.KaClientOpts.Time = time.Duration(math.MaxInt64)
	hp.AuthEnabled = false
	hp.MaxRetries = 5
	_, err := c.comms.AddHost(&id.NotificationBot, address,
		tlsCertificate, hp)
	if err != nil {
		jww.WARN.Printf("Failed adding host for notifications: %+v", err)
	}
}
