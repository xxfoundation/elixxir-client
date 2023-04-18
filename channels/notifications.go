package channels

import (
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/xx_network/primitives/id"
)

type NotificationLevel uint8

const (
	None NotificationLevel = 10 // Get no notifications. This is the default
	Ping NotificationLevel = 20 // Get notification
	Self NotificationLevel = 30 // Get notifications of replies
	All  NotificationLevel = 40
)

func (m *manager) SetMobileNotificationsLevel(token []byte, channelID *id.ID, level NotificationLevel) {

}

// TrackedServicesCallback is a callback that returns the [message.ServiceList]
// and [message.CompressedServiceList] of all channels with notifications enables
// any time a service is added or removed.
type TrackedServicesCallback func(
	sl message.ServiceList, csl message.CompressedServiceList)

// EnableNotifications will allow the user to receive notifications for channels.
// All new channels will have notifications enabled; any already existing
// channels need to have them enabled using [EnableChannelNotifications].
//
// The callback returns the service lists that can be passed into
// [bindings.GetNotificationsReport].
func (m *manager) EnableNotifications(
	cb TrackedServicesCallback, token string, user E2e) {
	jww.INFO.Printf("[CH] Enabling channel notifications with token %s", token)
	m.mux.Lock()
	defer m.mux.Unlock()
	m.notificationsManager = newNotificationsManager(token, user)
	m.registerServicesCallback(cb)
}

// notificationsManager manages which channels have notifications enabled.
type notificationsManager struct {
	// List of channels with notifications enabled
	notifyChannels map[id.ID]struct{}

	// Third-party token used for notifications
	token string

	user E2e
}

// newNotificationsManager initialises a new notificationsManager.
func newNotificationsManager(token string, user E2e) *notificationsManager {
	return &notificationsManager{
		notifyChannels: make(map[id.ID]struct{}),
		token:          token,
		user:           user,
	}
}

// registerServicesCallback registers the provided callback that returns the
// list of services registered for any channel with notifications enabled. It is
// called every time a channel service is added or removed.
func (m *manager) registerServicesCallback(cb TrackedServicesCallback) {
	m.net.TrackServices(func(
		sl message.ServiceList, csl message.CompressedServiceList) {
		channelsSl := make(message.ServiceList)
		channelsCsl := make(message.CompressedServiceList)

		m.mux.Lock()
		for chanID := range m.notifyChannels {
			if s, exists := sl[chanID]; exists {
				channelsSl[chanID] = s
			}
			if s, exists := csl[chanID]; exists {
				channelsCsl[chanID] = s
			}
		}
		m.mux.Unlock()

		cb(channelsSl, channelsCsl)
	})
}

// EnableChannelNotifications enables notifications for the given channel and
// includes services for this channel in the [message.ServiceList] and
// [message.CompressedServiceList] returned by the [TrackedServicesCallback].
//
// Returns an error if the channel does not exist.
func (m *manager) EnableChannelNotifications(channelID *id.ID) error {
	jww.INFO.Printf("[CH] Enable notifications for channel %s", channelID)
	m.mux.Lock()
	defer m.mux.Unlock()
	if _, exists := m.channels[*channelID]; !exists {
		return ChannelDoesNotExistsErr
	}
	m.notifyChannels[*channelID] = struct{}{}
	return m.enableChannelNotifications(channelID)
}
func (m *manager) enableChannelNotifications(channelID *id.ID) error {
	m.notifyChannels[*channelID] = struct{}{}
	return m.user.RegisterForNotifications(channelID, m.notificationsManager.token)
}

// DisableChannelNotifications disables notifications for the given channel and
// stops including services for this channel in the [message.ServiceList] and
// [message.CompressedServiceList] returned by the [TrackedServicesCallback].
func (m *manager) DisableChannelNotifications(channelID *id.ID) error {
	jww.INFO.Printf("[CH] Disable notifications for channel %s", channelID)
	m.mux.Lock()
	defer m.mux.Unlock()
	if _, exists := m.channels[*channelID]; !exists {
		return nil
	} else if _, exists = m.notifyChannels[*channelID]; !exists {
		return nil
	}
	return m.disableChannelNotifications(channelID)
}
func (m *manager) disableChannelNotifications(channelID *id.ID) error {
	delete(m.notifyChannels, *channelID)
	return m.user.UnregisterForNotifications(channelID)
}
