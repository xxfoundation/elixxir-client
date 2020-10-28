package api

import jww "github.com/spf13/jwalterweatherman"

// RegisterForNotifications allows a client to register for push
// notifications.
// Note that clients are not required to register for push notifications
// especially as these rely on third parties (i.e., Firebase *cough*
// *cough* google's palantir *cough*) that may represent a security
// risk to the user.
func (c *Client) RegisterForNotifications(token []byte) error {
	jww.INFO.Printf("RegisterForNotifications(%s)", token)
	// // Pull the host from the manage
	// notificationBotHost, ok := cl.receptionManager.Comms.GetHost(&id.NotificationBot)
	// if !ok {
	// 	return errors.New("Failed to retrieve host for notification bot")
	// }

	// // Send the register message
	// _, err := cl.receptionManager.Comms.RegisterForNotifications(notificationBotHost,
	// 	&mixmessages.NotificationToken{
	// 		Token: notificationToken,
	// 	})
	// if err != nil {
	// 	err := errors.Errorf(
	// 		"RegisterForNotifications: Unable to register for notifications! %s", err)
	// 	return err
	// }

	return nil
}

// UnregisterForNotifications turns of notifications for this client
func (c *Client) UnregisterForNotifications() error {
	jww.INFO.Printf("UnregisterForNotifications()")
	// // Pull the host from the manage
	// notificationBotHost, ok := cl.receptionManager.Comms.GetHost(&id.NotificationBot)
	// if !ok {
	// 	return errors.New("Failed to retrieve host for notification bot")
	// }

	// // Send the unregister message
	// _, err := cl.receptionManager.Comms.UnregisterForNotifications(notificationBotHost)
	// if err != nil {
	// 	err := errors.Errorf(
	// 		"RegisterForNotifications: Unable to register for notifications! %s", err)
	// 	return err
	// }

	return nil
}
