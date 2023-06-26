////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
)

// ChannelUICallbacks is used for providing callbacks to the UI from the EventModel.
type ChannelUICallbacks interface {
	EventUpdate(eventType int64, jsonData []byte)
}

// Event Type
const (
	NickNameUpdate int64 = 1000
	// NotificationUpdate int64 = 2000
	MessageReceived int64 = 3000
	UserMuted       int64 = 4000
	MessageDeleted  int64 = 5000
	AdminKeyUpdate  int64 = 6000
	ChannelUpdate   int64 = 7000
)

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
//	{
//	   "channelID":"YSc2bDijXIVhmIsJk2OZQjU9ei2Dn6MS8tOpXlIaUpSV"
//	}
type ChannelUpdateJson struct {
	ChannelID *id.ID `json:"channelID"`
}
