package notifications

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
)

// RegisterForNotifications allows a client to register for push notifications.
// Note that clients are not required to register for push notifications,
// especially as these rely on third parties (i.e., Firebase *cough* *cough*
// Google's Palantir *cough*) that may represent a security risk to the user.
// A client can register to receive push notifications on many IDs.
func (m *manager) RegisterForNotifications(toBeNotifiedOn *id.ID) error {
	jww.INFO.Printf("RegisterForNotifications(%s, %s)", toBeNotifiedOn, m.token)
	m.mux.Lock()
	defer m.mux.Lock()

	intermediaryReceptionID, sig, err := m.getIidAndSig(toBeNotifiedOn)
	if err != nil {
		return errors.Wrap(err, "RegisterForNotifications")
	}

	// Send the register message
	_, err = m.comms.RegisterForNotifications(m.notificationHost,
		&mixmessages.NotificationRegisterRequest{
			Token:                 token,
			IntermediaryId:        intermediaryReceptionID,
			TransmissionRsa:       m.transmissionRSA.Public().MarshalPem(),
			TransmissionSalt:      m.registrationSalt,
			TransmissionRsaSig:    m.transmissionRegistrationValidationSignature,
			IIDTransmissionRsaSig: sig,
			RegistrationTimestamp: m.registrationTimestamp.UnixNano(),
		})
	if err != nil {
		return errors.Wrap(err, "RegisterForNotifications: Unable to "+
			"register for notifications!")
	}

	return nil
}
