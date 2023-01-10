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
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// RegisterForNotifications allows a client to register for push notifications.
// Note that clients are not required to register for push notifications,
// especially as these rely on third parties (i.e., Firebase *cough* *cough*
// Google's palantir *cough*) that may represent a security risk to the user.
func (m *E2e) RegisterForNotifications(token string) error {
	jww.INFO.Printf("RegisterForNotifications(%s)", token)
	// Pull the host from the manage
	notificationBotHost, ok := m.GetComms().GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("RegisterForNotifications: " +
			"Failed to retrieve host for notification bot")
	}
	intermediaryReceptionID, sig, err := m.getIidAndSig()
	if err != nil {
		return err
	}

	privKey := m.GetStorage().GetTransmissionRSA()
	pubPEM := privKey.Public().MarshalPem()
	regSig := m.GetStorage().GetTransmissionRegistrationValidationSignature()
	regTS := m.GetStorage().GetRegistrationTimestamp()

	// Send the register message
	_, err = m.GetComms().RegisterForNotifications(notificationBotHost,
		&mixmessages.NotificationRegisterRequest{
			Token:                 token,
			IntermediaryId:        intermediaryReceptionID,
			TransmissionRsa:       pubPEM,
			TransmissionSalt:      m.GetStorage().GetTransmissionSalt(),
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
func (m *E2e) UnregisterForNotifications() error {
	jww.INFO.Printf("UnregisterForNotifications()")
	// Pull the host from the manage
	notificationBotHost, ok := m.GetComms().GetHost(&id.NotificationBot)
	if !ok {
		return errors.New("Failed to retrieve host for notification bot")
	}
	intermediaryReceptionID, sig, err := m.getIidAndSig()
	if err != nil {
		return err
	}
	// Sends the unregister message
	_, err = m.GetComms().UnregisterForNotifications(notificationBotHost,
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

func (m *E2e) getIidAndSig() ([]byte, []byte, error) {
	intermediaryReceptionID, err := ephemeral.GetIntermediaryId(
		m.GetStorage().GetReceptionID())
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

	stream := m.GetRng().GetStream()
	sig, err := m.GetStorage().GetTransmissionRSA().SignPSS(stream,
		hash.CMixHash, h.Sum(nil), nil)
	if err != nil {
		return nil, nil, errors.WithMessage(err,
			"RegisterForNotifications: Failed to sign intermediary ID")
	}
	stream.Close()
	return intermediaryReceptionID, sig, nil
}
