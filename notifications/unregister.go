package notifications

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// UnregisterNotificationIdentity turns off notifications for a specific identity
func (m *manager) UnregisterNotificationIdentity(toBeNotifiedOn *id.ID) error {
	jww.INFO.Printf("UnregisterNotificationIdentity(%s)", toBeNotifiedOn)
	m.mux.Lock()
	defer m.mux.Lock()

	ts := netTime.Now()

	intermediaryReceptionID, sig, err := m.getIidAndSig(toBeNotifiedOn, ts, "NotificationUnregisterRequest")

	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications")
	}

	return m.unregisterForNotifications(&mixmessages.NotificationUnregisterRequest{
		TransmissionRSA:       m.transmissionRSA.Public().MarshalPem(),
		IntermediaryId:        intermediaryReceptionID,
		IIDTransmissionRsaSig: sig,
		Timestamp:             ts,
	})
}

// UnregisterNotificationIdentity turns off notifications for a specific device token
func (m *manager) UnregisterNotificationDevice() error {
	jww.INFO.Printf("UnregisterNotificationDevice(%s)", m.token)
	m.mux.Lock()
	defer m.mux.Lock()

	// Pull the host from the manage

	ts := netTime.Now()

	intermediaryReceptionID, sig, err := m.getIidAndSig(&id.ID{}, ts, "UnregisterNotificationDevice")

	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications")
	}

	return c.unregisterForNotifications(&mixmessages.NotificationUnregisterRequest{
		Token:                 token,
		IIDTransmissionRsaSig: sig,
		TransmissionRSA:       c.GetStorage().GetTransmissionRSA().Public().MarshalPem(),
	})
}

func (m *manager) unregisterForNotifications(request *mixmessages.NotificationUnregisterRequest) error {
	// Sends the unregister message
	_, err := m.comms.UnregisterForNotifications(m.notificationHost,
		request)
	if err != nil {
		return errors.Wrap(err, "UnregisterForNotifications: Unable to "+
			"unregister for notifications!")
	}
	return nil
}
