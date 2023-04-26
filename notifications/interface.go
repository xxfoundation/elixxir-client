package notifications

import "gitlab.com/xx_network/primitives/id"

type Manager interface {
	// RegisterForNotifications Registers for notifications, registering
	// the token with the notifications if it isnt present and then
	// linking the passed ID
	RegisterForNotifications(toBeNotifiedOn *id.ID) error
	// UnregisterNotificationIdentity will unregister a specific
	// Identity with the notification's server. Will return
	// an error if that identity isnt registered or no registrations
	// have happened.
	UnregisterNotificationIdentity(toBeNotifiedOn *id.ID) error
	// AddToken adds the given token to the list of tokens which
	// will be forwarded the message
	AddToken(newToken string) error
	// RemoveToken removes the given token from the server
	// It will remove all registered identities if it is the
	// last token
	RemoveToken() error
}
