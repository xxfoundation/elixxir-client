////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"crypto/ed25519"
	"encoding/json"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/channels"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
)

////////////////////////////////////////////////////////////////////////////////
// UI Callbacks                                                               //
////////////////////////////////////////////////////////////////////////////////

// ChannelUICallbacks is used for providing callbacks to the UI from the
// [EventModel].
type ChannelUICallbacks interface {
	EventUpdate(eventType int64, jsonData []byte)
}

// Event types used by ChannelUICallbacks.
const (
	// NickNameUpdate indicates the data is [NickNameUpdateJSON].
	NickNameUpdate int64 = 1000

	// NotificationUpdate indicates the data is [NotificationUpdateJSON].
	NotificationUpdate int64 = 2000

	// MessageReceived indicates the data is [MessageReceivedJSON].
	MessageReceived int64 = 3000

	// UserMuted indicates the data is [UserMutedJSON].
	UserMuted int64 = 4000

	// MessageDeleted indicates the data is [MessageDeletedJSON].
	MessageDeleted int64 = 5000

	// AdminKeyUpdate indicates the data is [AdminKeysUpdateJSON].
	AdminKeyUpdate int64 = 6000

	// DmTokenUpdate indicates the data is [DmTokenUpdateJSON].
	DmTokenUpdate int64 = 7000

	// ChannelUpdate indicates the data is [ChannelUpdateJSON].
	ChannelUpdate int64 = 8000
)

// channelUICallbacks is a simple wrapper for [channels.UiCallbacks].
type channelUICallbacks struct {
	eventUpdate func(eventType int64, jsonMarshallable any)
}

func wrapChannelUICallbacks(uiCB ChannelUICallbacks) *channelUICallbacks {
	if uiCB == nil {
		return nil
	}

	return &channelUICallbacks{func(eventType int64, jsonMarshallable any) {
		jsonData, err := json.Marshal(jsonMarshallable)
		if err != nil {
			jww.FATAL.Panicf(
				"[CH] Failed to JSON marshal %T: %+v", jsonMarshallable, err)
		}
		uiCB.EventUpdate(eventType, jsonData)
	}}
}

func (cuiCB *channelUICallbacks) NicknameUpdate(
	channelId *id.ID, nickname string, exists bool) {
	cuiCB.eventUpdate(NickNameUpdate, NickNameUpdateJSON{
		ChannelId: channelId,
		Nickname:  nickname,
		Exists:    exists,
	})
}

func (cuiCB *channelUICallbacks) NotificationUpdate(
	nfs []channels.NotificationFilter,
	changedNotificationStates []channels.NotificationState,
	deletedNotificationStates []*id.ID, maxState clientNotif.NotificationState) {
	cuiCB.eventUpdate(NotificationUpdate, NotificationUpdateJSON{
		NotificationFilters:       nfs,
		ChangedNotificationStates: changedNotificationStates,
		DeletedNotificationStates: deletedNotificationStates,
		MaxState:                  maxState,
	})
}

func (cuiCB *channelUICallbacks) AdminKeysUpdate(chID *id.ID, isAdmin bool) {
	cuiCB.eventUpdate(AdminKeyUpdate, AdminKeysUpdateJSON{
		ChannelId: chID,
		IsAdmin:   isAdmin,
	})
}

func (cuiCB *channelUICallbacks) DmTokenUpdate(chID *id.ID, sendToken bool) {
	cuiCB.eventUpdate(DmTokenUpdate, DmTokenUpdateJSON{
		ChannelId: chID,
		SendToken: sendToken,
	})
}

func (cuiCB *channelUICallbacks) ChannelUpdate(channelID *id.ID, deleted bool) {
	cuiCB.eventUpdate(ChannelUpdate, ChannelUpdateJSON{
		ChannelID: channelID,
		Deleted:   deleted,
	})
}

func (cuiCB *channelUICallbacks) MessageReceived(
	uuid int64, channelID *id.ID, update bool) {
	cuiCB.eventUpdate(MessageReceived, MessageReceivedJSON{
		UUID:      uuid,
		ChannelID: channelID,
		Update:    update,
	})
}

func (cuiCB *channelUICallbacks) UserMuted(
	channelID *id.ID, pubKey ed25519.PublicKey, unmute bool) {
	cuiCB.eventUpdate(UserMuted, UserMutedJSON{
		ChannelID: channelID,
		PubKey:    pubKey,
		Unmute:    unmute,
	})
}

func (cuiCB *channelUICallbacks) MessageDeleted(messageID message.ID) {
	cuiCB.eventUpdate(MessageDeleted, MessageDeletedJSON{
		MessageID: messageID,
	})
}

func unmarshalPingsJson(b []byte) ([]ed25519.PublicKey, error) {
	var pings []ed25519.PublicKey
	if b != nil && len(b) > 0 {
		return pings, json.Unmarshal(b, &pings)
	}
	return pings, nil
}

func unmarshalPingsMapJson(b []byte) (map[channels.PingType][]ed25519.PublicKey, error) {
	var pingsMap map[channels.PingType][]ed25519.PublicKey
	if b != nil && len(b) > 0 {
		return pingsMap, json.Unmarshal(b, &pingsMap)
	}
	return pingsMap, nil
}

// NickNameUpdateJSON is describes when your nickname changes due to a change on a
// remote.
//
// Example JSON:
//
//	{
//	  "channelID": "JsU7+QYpybOy/xgjYrJW675XRonGRoZj3YGFWzu/SoID",
//	  "nickname": "billNyeTheScienceGuy",
//	  "exists": true
//	}
type NickNameUpdateJSON struct {
	ChannelId *id.ID `json:"channelID"`
	Nickname  string `json:"nickname"`
	Exists    bool   `json:"exists"`
}

// NotificationUpdateJSON describes any time a notification
// level changes.
//
// Contains:
//   - notificationFilters - JSON of slice of [channels.NotificationFilter],
//     which is passed into [GetChannelNotificationReportsForMe] to filter
//     channel notifications for the user.
//   - changedNotificationStates - JSON of slice of [channels.NotificationState]
//     of added or changed channel notification statuses.
//   - deletedNotificationStates - JSON of slice of [id.ID] of deleted channel
//     notification statuses.
//   - maxState - The global notification state.
//
// Example JSON:
//
//	{
//	  "notificationFilters": [
//	    {
//	      "identifier": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4EDQXN5bW1Ub1B1YmxpY0JjYXN0",
//	      "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//	      "tags": ["af35cdae2159477d79f7ab33bf0bb73ccc1f212bfdc1b3ae78cf398c02878e01-usrping"],
//	      "allowLists": {
//	        "allowWithTags": {"2": {}},
//	        "allowWithoutTags": {"102": {}}
//	      }
//	    },
//	    {
//	      "identifier": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4EDU3ltbWV0cmljQnJvYWRjYXN0",
//	      "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//	      "tags": ["af35cdae2159477d79f7ab33bf0bb73ccc1f212bfdc1b3ae78cf398c02878e01-usrping"],
//	      "allowLists": {
//	        "allowWithTags": {"1": {}, "40000": {}},
//	        "allowWithoutTags": {}
//	      }
//	    },
//	    {
//	      "identifier": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoDQXN5bW1Ub1B1YmxpY0JjYXN0",
//	      "channelID": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoD",
//	      "tags": ["4f4b35a64a3bd7b06614c2f48d0cdda8b2220ca0fcba78cd2ed11ba38afc92f2-usrping"],
//	      "allowLists": {
//	        "allowWithTags": {"2": {}},
//	        "allowWithoutTags": {"102": {}}
//	      }
//	    }
//	  ],
//	  "changedNotificationStates": [
//	    {
//	      "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//	      "level": 20,
//	      "status": 2
//	    },
//	    {
//	      "channelID": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUD",
//	      "level": 40,
//	      "status": 2
//	    }
//	  ],
//	  "deletedNotificationStates": [
//	    "ZGVsZXRlZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"
//	  ],
//	  "maxState": 2
//	}
type NotificationUpdateJSON struct {
	NotificationFilters       []channels.NotificationFilter `json:"notificationFilters"`
	ChangedNotificationStates []channels.NotificationState  `json:"changedNotificationStates"`
	DeletedNotificationStates []*id.ID                      `json:"deletedNotificationStates"`
	MaxState                  clientNotif.NotificationState `json:"maxState"`
}

// AdminKeysUpdateJSON describes when you get or lose keys for a specific
// channel
//
// Example JSON:
//
//	{
//	  "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	  "IsAdmin":true
//	}
type AdminKeysUpdateJSON struct {
	ChannelId *id.ID `json:"channelID"`
	IsAdmin   bool   `json:"IsAdmin"`
}

// DmTokenUpdateJSON describes when the sending of dm tokens is enabled or
// disabled on a specific channel
//
// Example JSON:
//
//	{
//	  "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	  "sendToken":true
//	}
type DmTokenUpdateJSON struct {
	ChannelId *id.ID `json:"channelID"`
	SendToken bool   `json:"sendToken"`
}

// ChannelUpdateJSON is returned any time a Channel is joined/left.
//
// Example JSON:
//
//	{
//	  "channelID":"YSc2bDijXIVhmIsJk2OZQjU9ei2Dn6MS8tOpXlIaUpSV",
//	  "deleted":false"
//	}
type ChannelUpdateJSON struct {
	ChannelID *id.ID `json:"channelID"`
	Deleted   bool   `json:"deleted"`
}

// MessageReceivedJSON is returned any time a message is received or updated.
// Update is true if the row is old and was edited.
//
// Example JSON:
//
//	{
//	  "uuid":32,
//	  "channelID":"Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4HZ",
//	  "update":false
//	}
type MessageReceivedJSON struct {
	UUID      int64  `json:"uuid"`
	ChannelID *id.ID `json:"channelID"`
	Update    bool   `json:"update"`
}

// UserMutedJSON is returned for the MuteUser method of the impl.
//
// Example JSON:
//
//	{
//	  "channelID":"YSc2bDijXIVhmIsJk2OZQjU9ei2Dn6MS8tOpXlIaUpSV",
//	  "pubKey":"hClzdWkMI+LM7KDFxC/iuyIc0oiMzcBXBFgH0haZAjc=",
//	  "unmute":false
//	}
type UserMutedJSON struct {
	ChannelID *id.ID            `json:"channelID"`
	PubKey    ed25519.PublicKey `json:"pubKey"`
	Unmute    bool              `json:"unmute"`
}

// MessageDeletedJSON is returned any time a message is deleted.
//
// Example JSON:
//
//	{
//	  "messageID":"i9b7tL5sUmObxqW1LApC9H/yvnQzsRfq7yc8SCBtlK0="
//	}
type MessageDeletedJSON struct {
	MessageID message.ID `json:"messageID"`
}
