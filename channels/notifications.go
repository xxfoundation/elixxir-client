////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// TODO: handle token changes

// TrackedServicesCallback is a callback that returns the
// [message.CompressedServiceList] of all channels with notifications enabled
// any time a service is added or removed or the notification level changes.
type TrackedServicesCallback func(csl message.CompressedServiceList)

// Storage values.
const (
	notificationsKvVersion = 0
	notificationsKvKey     = "channelsNotifications"
)

// notifications manages the notification level for each channel.
type notifications struct {
	// List of channels and their notification levels
	channels map[id.ID]NotificationLevel

	// Third-party token used for third-party notification service
	token string

	// Interface for xxdk.E2e that contains functions for registering and
	// unregistering notifications
	user E2e
	net  Client
	kv   *versioned.KV

	mux sync.RWMutex
}

// newNotifications initialises a new channels notifications manager.
func newNotifications(
	token string, user E2e, kv *versioned.KV, net Client) *notifications {
	return &notifications{
		channels: make(map[id.ID]NotificationLevel),
		token:    token,
		user:     user,
		net:      net,
		kv:       kv,
	}
}

// addChannel inserts the channel into the notification list with no the
// notification level set to none.
//
// Returns an error if the channel already exists.
func (n *notifications) addChannel(channelID *id.ID) error {
	n.mux.Lock()
	defer n.mux.Unlock()
	if _, exists := n.channels[*channelID]; exists {
		return ChannelAlreadyExistsErr
	}
	n.channels[*channelID] = NotifyNone
	return nil
}

// addChannel inserts the channel into the notification list with the given
// level.
//
// Returns an error if the channel already exists.
func (n *notifications) removeChannel(channelID *id.ID) error {
	n.mux.Lock()
	defer n.mux.Unlock()
	level, exists := n.channels[*channelID]
	if exists {
		return ChannelAlreadyExistsErr
	}
	if level != NotifyNone {
		err := n.user.UnregisterForNotifications(channelID)
		if err != nil {
			return err
		}
	}

	delete(n.channels, *channelID)

	return nil
}

// func (m *manager) SetMobileNotificationsLevel(token string, channelID *id.ID, level NotificationLevel) {
// 	jww.INFO.Printf("[CH] Enabling channel notifications with token %s", token)
// 	m.mux.Lock()
// 	defer m.mux.Unlock()
// 	m.notificationsManager = newNotificationsManager(token, user)
// 	m.registerServicesCallback(cb)
//
// }

// SetMobileNotificationsLevel sets the notification level for the given
// channel.
//
// RegisterForNotifications allows a client to register for push notifications.
// Note that clients are not required to register for push notifications,
// especially as these rely on third parties (i.e., Firebase *cough* *cough*
// Google's Palantir *cough*) that may represent a security risk to the user.
// A client can register to receive push notifications on many IDs.
func (n *notifications) SetMobileNotificationsLevel(
	channelID *id.ID, level NotificationLevel) error {
	jww.INFO.Printf("[CH] Set notification level for channel %s to %s",
		channelID, level)
	n.mux.Lock()
	defer n.mux.Unlock()

	chanLevel, exists := n.channels[*channelID]
	if !exists {
		return ChannelDoesNotExistsErr
	}

	// Determine if the channel needs to be registered or unregistered
	if chanLevel == NotifyNone && level != NotifyNone {
		if n.token == "" {
			return errors.New("cannot enable notifications when no token is registered")
		}
		return n.user.RegisterForNotifications(channelID, n.token)
	} else if chanLevel != NotifyNone && level == NotifyNone {
		return n.user.UnregisterForNotifications(channelID)
	}

	n.channels[*channelID] = level
	return nil
}

// registerServicesCallback registers the provided callback that returns the
// list of services registered for any channel with notifications enabled. It is
// called every time a channel service is added or removed.
func (n *notifications) registerServicesCallback(cb TrackedServicesCallback) {
	n.net.TrackServices(func(
		_ message.ServiceList, csl message.CompressedServiceList) {

		n.mux.Lock()
		for chanID, level := range n.channels {
			if sList, exists := csl[chanID]; exists {
				for i, s := range sList {
					var tags []string
					var filter MessageTypeFilter
					switch level {
					case NotifyNone:
					case NotifyPing:
						filter = MessageTypeFilter{
							FilterType: false,
							Allow:      []MessageType{Pinned},
						}

					case NotifyAll:
						filter = MessageTypeFilter{
							FilterType: true,
							Disallow:   []MessageType{AdminReplay},
						}
					}

					filterData, err := json.Marshal(filter)
					if err != nil {
						jww.FATAL.Panicf(
							"Failed to JSON marshal %T: %+v", filter, err)
					}

					csl[chanID][i] = message.CompressedService{
						Identifier: s.Identifier,
						Tags:       tags,
						Metadata:   filterData,
					}
				}
			}
		}
		n.mux.Unlock()

		cb(csl)
	})
}

// save marshals and saves the channels and token to storage.
func (n *notifications) save() error {
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}

	return n.kv.Set(notificationsKvKey, &versioned.Object{
		Version:   notificationsKvVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	})
}

// load loads and unmarshalls the channels and token from storage into
// notifications.
func (n *notifications) load() error {
	obj, err := n.kv.Get(notificationsKvKey, notificationsKvVersion)
	if err != nil {
		return err
	}

	return json.Unmarshal(obj.Data, n)
}

// notificationsDisk contains the fields of notifications in a structure that
// can be JSON marshalled and unmarshalled to be saved/loaded from storage.
type notificationsDisk struct {
	Channels map[string]NotificationLevel `json:"channels"`
	Token    string                       `json:"token"`
}

// MarshalJSON marshals the notifications into valid JSON. This function adheres
// to the json.Marshaler interface.
func (n *notifications) MarshalJSON() ([]byte, error) {
	nd := notificationsDisk{
		Channels: make(map[string]NotificationLevel, len(n.channels)),
		Token:    n.token,
	}

	for uid, level := range n.channels {
		nd.Channels[base64.StdEncoding.EncodeToString(uid.Marshal())] = level
	}

	return json.Marshal(nd)
}

// UnmarshalJSON unmarshalls JSON into the notifications. This function adheres
// to the json.Unmarshaler interface.
func (n *notifications) UnmarshalJSON(data []byte) error {
	var nd notificationsDisk
	if err := json.Unmarshal(data, &nd); err != nil {
		return err
	}

	channels := make(map[id.ID]NotificationLevel, len(nd.Channels))
	for uidBase64, level := range nd.Channels {
		uidBytes, err := base64.StdEncoding.DecodeString(uidBase64)
		if err != nil {
			return err
		}
		uid, err := id.Unmarshal(uidBytes)
		if err != nil {
			return err
		}
		channels[*uid] = level
	}

	n.channels = channels
	n.token = nd.Token

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// MessageTypeFilter                                                          //
////////////////////////////////////////////////////////////////////////////////

// MessageTypeFilter defines filtering properties for channel message types.
type MessageTypeFilter struct {
	// FilterType determines the type of filter. When set to true, the filter is
	// in "open" mode where only messages on the Disallow list are filter. When
	// set to true, the filter is in "closed" mode where all messages except
	// those on the Allow list are filter.
	FilterType bool `json:"filterType"`

	// Allow is a list of messages types that are not filtered when in "closed"
	// (false) mode.
	Allow []MessageType `json:"allow,omitempty"`

	// Disallow is a list of messages types that are filtered when in "open"
	// (true) mode.
	Disallow []MessageType `json:"disallow,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////
// NotificationLevel                                                          //
////////////////////////////////////////////////////////////////////////////////

// NotificationLevel specifies what level of notifications should be received
// for a channel.
type NotificationLevel uint8

const (
	// NotifyNone results in no notifications.
	NotifyNone NotificationLevel = 10

	// NotifyPing results in notifications from tags, replies, and pins.
	NotifyPing NotificationLevel = 20

	// NotifyAll results in notifications from all messages except silent ones
	// and replays.
	NotifyAll NotificationLevel = 40
)

// String prints a human-readable form of the [NotificationLevel] for logging
// and debugging. This function adheres to the [fmt.Stringer] interface.
func (nl NotificationLevel) String() string {
	switch nl {
	case NotifyNone:
		return "none"
	case NotifyPing:
		return "ping"
	case NotifyAll:
		return "all"
	default:
		return "INVALID NOTIFICATION LEVEL: " + strconv.Itoa(int(nl))
	}
}
