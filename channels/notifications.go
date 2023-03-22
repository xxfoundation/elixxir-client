package channels

import "gitlab.com/xx_network/primitives/id"

type NotificationLevel uint8

const (
	None NotificationLevel = 1000 // Get no notifications. This is the default
	Ping NotificationLevel = 2000 // Get notification
	Self NotificationLevel = 3000 // Get notifications of replies
	All  NotificationLevel = 4000
)

func (m *manager) SetMobileNotificationsLevel(token []byte, channelID *id.ID, level NotificationLevel) {

}
