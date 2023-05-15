package channels

import "gitlab.com/xx_network/primitives/id"

type NotificationLevel uint8

const (
	None NotificationLevel = 10 // Get no notifications. This is the default
	Ping NotificationLevel = 20 // Get notification
	Self NotificationLevel = 30 // Get notifications of replies
	All  NotificationLevel = 40
)

func (m *manager) SetMobileNotificationsLevel(token []byte, channelID *id.ID, level NotificationLevel) {

}
