package api

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
)

// RegisterForNotifications sends a message to notification bot indicating it
// is registering for notifications
func (cl *Client) RegisterForNotifications(notificationToken []byte) error {
	// Pull the host from the manage
	notificationBotHost, ok := cl.receptionManager.Comms.GetHost(id.NOTIFICATION_BOT)
	if !ok {
		return errors.New("Failed to retrieve host for notification bot")
	}

	// Send the register message
	_, err := cl.receptionManager.Comms.RegisterForNotifications(notificationBotHost,
		&mixmessages.NotificationToken{
			Token: notificationToken,
		})
	if err != nil {
		err := errors.Errorf(
			"RegisterForNotifications: Unable to register for notifications! %s", err)
		return err
	}

	return nil

}

// UnregisterForNotifications sends a message to notification bot indicating it
// no longer wants to be registered for notifications
func (cl *Client) UnregisterForNotifications() error {
	// Pull the host from the manage
	notificationBotHost, ok := cl.receptionManager.Comms.GetHost(id.NOTIFICATION_BOT)
	if !ok {
		return errors.New("Failed to retrieve host for notification bot")
	}

	// Send the unregister message
	_, err := cl.receptionManager.Comms.UnregisterForNotifications(notificationBotHost)
	if err != nil {
		err := errors.Errorf(
			"RegisterForNotifications: Unable to register for notifications! %s", err)
		return err
	}

	return nil

}
