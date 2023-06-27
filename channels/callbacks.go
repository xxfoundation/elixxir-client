////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
)

// Event types used by ChannelUICallbacks.
const (
	NickNameUpdate     int64 = 1000
	NotificationUpdate int64 = 2000
	MessageReceived    int64 = 3000
	UserMuted          int64 = 4000
	MessageDeleted     int64 = 5000
	AdminKeyUpdate     int64 = 6000
	DmTokenUpdate      int64 = 7000
	ChannelUpdate      int64 = 8000
)

// ChannelUICallbacks is used for providing callbacks to the UI from the EventModel.
type ChannelUICallbacks interface {
	EventUpdate(eventType int64, jsonData []byte)
}

// MessageReceivedJson is returned any time a message is received or updated.
// Update is true if the row is old and was edited.
//
//	{
//	  "uuid":32,
//	  "channelID":"Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4HZ",
//	  "update":false
//	}
type MessageReceivedJson struct {
	Uuid      int64  `json:"uuid"`
	ChannelID *id.ID `json:"channelID"`
	Update    bool   `json:"update"`
}

// UserMutedJson is returned for the MuteUser method of the impl.
//
//	{
//	  "channelID":"YSc2bDijXIVhmIsJk2OZQjU9ei2Dn6MS8tOpXlIaUpSV",
//	  "pubKey":"hClzdWkMI+LM7KDFxC/iuyIc0oiMzcBXBFgH0haZAjc=",
//	  "unmute":false
//	}
type UserMutedJson struct {
	ChannelID *id.ID            `json:"channelID"`
	PubKey    ed25519.PublicKey `json:"pubKey"`
	Unmute    bool              `json:"unmute"`
}

// MessageDeletedJson is returned any time a message is deleted.
//
//	{
//	  "messageID":"i9b7tL5sUmObxqW1LApC9H/yvnQzsRfq7yc8SCBtlK0="
//	}
type MessageDeletedJson struct {
	MessageID message.ID `json:"messageID"`
}

// ChannelUpdateJson is returned any time a Channel is joined/left.
//
//		{
//		   "channelID":"YSc2bDijXIVhmIsJk2OZQjU9ei2Dn6MS8tOpXlIaUpSV",
//	    "deleted":false"
//		}
type ChannelUpdateJson struct {
	ChannelID *id.ID `json:"channelID"`
	Deleted   bool   `json:"deleted"`
}

// NickNameUpdateJson is describes when your nickname changes due to a change on a
// remote.
//
//	{
//	 "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	 "nickname":"billNyeTheScienceGuy",
//	 "exists":true
//	}
type NickNameUpdateJson struct {
	ChannelId *id.ID `json:"channelID"`
	Nickname  string `json:"nickname"`
	Exists    bool   `json:"exists"`
}

// NotificationUpdateJson describes any time a notification
// level changes.
//
// It contains  a slice of [NotificationFilter] for all channels with
// notifications enabled. The [NotificationFilter] is used to determine
// which notifications from the notification server belong to the caller.
//
// It also contains  a map of all channel notification states that have
// changed and all that have been deleted. The maxState is the global state
// set for notifications.
//
// Contains:
//
//   - notificationFilters - JSON of a slice of
//     [channels.NotificationFilter].
//
//   - changedNotificationStates - JSON of a slice of
//     [channels.NotificationState] of added or changed channel notification
//     statuses.
//
//   - deletedNotificationStates - JSON of a slice of [id.ID] of deleted
//     channel notification statuses.
//
//   - maxState - The global notification state.
//
//     {
//     "notificationFilters": [
//     {
//     "identifier": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4EDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//     "tags": [
//     "af35cdae2159477d79f7ab33bf0bb73ccc1f212bfdc1b3ae78cf398c02878e01-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4EDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//     "tags": [
//     "af35cdae2159477d79f7ab33bf0bb73ccc1f212bfdc1b3ae78cf398c02878e01-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoD",
//     "tags": [
//     "4f4b35a64a3bd7b06614c2f48d0cdda8b2220ca0fcba78cd2ed11ba38afc92f2-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoD",
//     "tags": [
//     "4f4b35a64a3bd7b06614c2f48d0cdda8b2220ca0fcba78cd2ed11ba38afc92f2-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQD",
//     "tags": [
//     "72c73b78133739042a5b15c635853d2617324345f611fb272d1fa894a2adf96a-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQD",
//     "tags": [
//     "72c73b78133739042a5b15c635853d2617324345f611fb272d1fa894a2adf96a-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     },
//     {
//     "identifier": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUD",
//     "tags": [
//     "0e2aaeacd3cf1d1738cc09f94405b6d1a841af575211cb6f4d39b7d4914d5341-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUD",
//     "tags": [
//     "0e2aaeacd3cf1d1738cc09f94405b6d1a841af575211cb6f4d39b7d4914d5341-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     },
//     {
//     "identifier": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wD",
//     "tags": [
//     "cbce1940f0102e791d67412328acfa53f9881494e249eb345efac084224187b6-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wD",
//     "tags": [
//     "cbce1940f0102e791d67412328acfa53f9881494e249eb345efac084224187b6-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQD",
//     "tags": [
//     "ae893892e2b1253bb39fdac5f81b08c5ab69cb06ee79345715972c1af6c5125f-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQD",
//     "tags": [
//     "ae893892e2b1253bb39fdac5f81b08c5ab69cb06ee79345715972c1af6c5125f-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJADQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJAD",
//     "tags": [
//     "a164b0f40c21a86e925869e5a9d7c16886891e03f96809f68b8ade57160c7028-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJADU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJAD",
//     "tags": [
//     "a164b0f40c21a86e925869e5a9d7c16886891e03f96809f68b8ade57160c7028-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     },
//     {
//     "identifier": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4DQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4D",
//     "tags": [
//     "4ca23f9d385589fc530680c9e406099c6a34fd74d4002d19608ce6c971cabfa0-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4DU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4D",
//     "tags": [
//     "4ca23f9d385589fc530680c9e406099c6a34fd74d4002d19608ce6c971cabfa0-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     }
//     ],
//     "changedNotificationStates": [
//     {
//     "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//     "level": 20,
//     "status": 2
//     },
//     {
//     "channelID": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUD",
//     "level": 40,
//     "status": 2
//     }
//     ],
//     "deletedNotificationStates": [
//     "ZGVsZXRlZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"
//     ],
//     "maxState": 2
//     }
type NotificationUpdateJson struct {
	NotificationFilters       []NotificationFilter          `json:"notificationFilters"`
	ChangedNotificationStates []NotificationState           `json:"changedNotificationStates"`
	DeletedNotificationStates []*id.ID                      `json:"deletedNotificationStates"`
	MaxState                  clientNotif.NotificationState `json:"maxState"`
}

// AdminKeysUpdateJson describes when you get or lose keys for a specific
// channel
//
//	{
//	 "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	 "IsAdmin":true
//	}
type AdminKeysUpdateJson struct {
	ChannelId *id.ID `json:"channelID"`
	IsAdmin   bool   `json:"IsAdmin"`
}

// DmTokenUpdateJson describes when the sending of dm tokens is enabled or
// disabled on a specific channel
//
//	{
//	 "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	 "sendToken":true
//	}
type DmTokenUpdateJson struct {
	ChannelId *id.ID `json:"channelID"`
	SendToken bool   `json:"sendToken"`
}
