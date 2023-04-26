package notifications

import "gitlab.com/xx_network/primitives/id"

type Manger interface{
	// Set can be used to turn on or off notifications for a given ID.
	// Will synchronize the state with all clients and register with the notifications
	// server if status == true and a token is set
	// Group is used to segment the notifications lists so different users of the same
	// object do not interfere. Metadata will be synchronized, allowing more verbose
	// notifications settings. Max 1KB.
	Set(toBeNotifiedOn *id.ID, group string, metadata []byte, status bool) error
	// Get returns the status of the notifications for the given ID, or
	// an error if not present
	Get(toBeNotifiedOn *id.ID) (status bool, metadata []byte, group string, err error)
	// GetGroup the status of all registered notifications for
	// the given group. If the group isn't present, an empty map will be returned.
	GetGroup(group string) map[*id.ID]struct{status bool, metadata []byte}
	// AddToken adds the given token to the list of tokens which
	// will be forwarded the message
	// the app will tell the server what app to forward the notifications to. There will
	// be separate designations for iOS and Android.
	AddToken(newToken, app string) error
	// RemoveToken removes the given token from the server
	// It will remove all registered identities if it is the
	// last token
	RemoveToken() error
	// RegisterUpdateCallback registers a callback to be used to receive notifications
	// of changes in notifications. Because this is being called after initialization,
	// a poll of state via the get function will be necessary because notifications can be missed
	RegisterUpdateCallback(group string, nu Update)
}
type Update func(id *id.ID, metadata []byte, status bool)

