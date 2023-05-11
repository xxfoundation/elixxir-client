////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"strconv"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/sih"
	primNotif "gitlab.com/elixxir/primitives/notifications"
	"gitlab.com/xx_network/primitives/id"
)

// FilterCallback is a callback that returns a slice of [NotificationFilter] of
// all channels with notifications enabled any time a service is added or
// removed or the notification level changes. The [NotificationFilter] is used
// to determine which notifications from the notification server belong to the
// caller.
type FilterCallback func(nfs []NotificationFilter)

// notificationGroup is the name used for to denote channel notifications in the
// notification manager.
const notificationGroup = "channels"

// notifications manages the notification level for each channel.
type notifications struct {
	// User's public key
	pubKey ed25519.PublicKey

	// User supplied callback to return updated NotificationFilter objects to.
	cb FilterCallback

	channelGetter
	nm NotificationsManager
}

// channelGetter is an interface that retrieves a channel from the manager's
// channel list.
type channelGetter interface {
	getChannel(channelID *id.ID) (*joinedChannel, error)
}

// newNotifications initialises a new channels notifications manager.
func newNotifications(pubKey ed25519.PublicKey, cb FilterCallback,
	cg channelGetter, nm NotificationsManager) *notifications {
	n := &notifications{
		pubKey:        pubKey,
		cb:            cb,
		channelGetter: cg,
		nm:            nm,
	}
	nm.RegisterUpdateCallback(notificationGroup, n.notificationsUpdateCB)
	return n
}

// addChannel inserts the channel into the notification list with no the
// notification level set to none.
//
// Returns an error if the channel already exists.
func (n *notifications) addChannel(channelID *id.ID) error {
	return n.nm.Set(
		channelID, notificationGroup, NotifyNone.Marshal(), clientNotif.Mute)
}

// addChannel inserts the channel into the notification list with the given
// level.
func (n *notifications) removeChannel(channelID *id.ID) {
	n.nm.Delete(channelID)
}

// GetNotificationLevel returns the notification level for the given channel.
func (n *notifications) GetNotificationLevel(
	channelID *id.ID) (NotificationLevel, error) {

	_, metadata, _, exists := n.nm.Get(channelID)
	if !exists {
		return 0,
			errors.Errorf("channel %s not found in notifications", channelID)
	}

	return UnmarshalNotificationLevel(metadata), nil
}

// SetMobileNotificationsLevel sets the notification level for the given
// channel. The [NotificationLevel] dictates the type of notifications received.
// If push is set to true, then push notifications will be received when the app
// is closed. Otherwise, notifications will only appear when the app is open.
//
// To use push notifications, a token must be registered with the notification
// manager. Note, when enabling push notifications, information may be shared
// with third parties (i.e., Firebase and Google's Palantir) and may represent a
// security risk to the user.
func (n *notifications) SetMobileNotificationsLevel(channelID *id.ID,
	level NotificationLevel, push bool) error {
	jww.INFO.Printf("[CH] Set notification level for channel %s to %s",
		channelID, level)

	status := clientNotif.Mute
	if level != NotifyNone {
		if push {
			status = clientNotif.Push
		} else {
			status = clientNotif.WhenOpen
		}
	}
	return n.nm.Set(channelID, notificationGroup, level.Marshal(), status)
}

// notificationsUpdateCB gets the list of all services and assembles a list of
// NotificationFilter for each channel that exists in the compressed service
// list. The results are called on the user-registered FilterCallback.
func (n *notifications) notificationsUpdateCB(
	group clientNotif.Group, _, _, _ []*id.ID) {
	nfs := n.createFilterList(group)

	n.cb(nfs)
}

// createFilterList generates two NotificationFilter objects for each channel ID
// in the provided map; one for asymmetric messages and one for symmetric.
// The filter generated is based on its message type and NotificationLevel
// embedded in the Metadata.
func (n *notifications) createFilterList(group clientNotif.Group) []NotificationFilter {
	var nfs []NotificationFilter
	tags := makeUserPingTags(n.pubKey)
	for chanID, notif := range group {
		channelID := chanID.DeepCopy()

		ch, err := n.getChannel(channelID)
		if err != nil {
			jww.WARN.Printf("[CH] Cannot build notification filter for "+
				"channel %s: %+v", channelID, err)
			continue
		}

		level := UnmarshalNotificationLevel(notif.Metadata)
		if level == NotifyNone {
			continue
		}

		nfs = append(nfs,
			NotificationFilter{
				Identifier: ch.broadcast.AsymmetricIdentifier(),
				ChannelID:  channelID,
				Tags:       tags,
				AllowLists: notificationLevelAllowLists[asymmetric][level],
			},
			NotificationFilter{
				Identifier: ch.broadcast.SymmetricIdentifier(),
				ChannelID:  channelID,
				Tags:       tags,
				AllowLists: notificationLevelAllowLists[symmetric][level],
			})
	}

	return nfs
}

////////////////////////////////////////////////////////////////////////////////
// For Me / Notification Report                                               //
////////////////////////////////////////////////////////////////////////////////

// NotificationReport describes information about a single notification
// belonging to the user.
type NotificationReport struct {
	// Channel is the channel ID that the notification is for.
	Channel *id.ID `json:"channel"`

	// Type is the MessageType of the message that the notification belongs to.
	Type MessageType `json:"type"`
}

// GetNotificationReportsForMe checks the notification data against the filter
// list to determine which notifications belong to the user. A list of
// notifications reports is returned detailing all notifications for the user.
func GetNotificationReportsForMe(nfs []NotificationFilter,
	notificationData []*primNotif.Data) []NotificationReport {

	var nr []NotificationReport

	for _, data := range notificationData {
		for _, nf := range nfs {
			matchedTags, metadata, found, err := sih.EvaluateCompressedSIH(
				nf.ChannelID, data.MessageHash, nf.Identifier, nf.Tags,
				data.IdentityFP)
			if err != nil {
				jww.TRACE.Printf("[CH] Failed to evaluate compressed SIH for "+
					"channel %s and identifier %v", nf.ChannelID, nf.Identifier)
				continue
			}

			if found {
				messageType := UnmarshalMessageType(metadata)
				if nf.match(matchedTags, messageType) {
					nr = append(nr, NotificationReport{
						Channel: nf.ChannelID,
						Type:    messageType,
					})
				}
			}
		}
	}

	return nr
}

////////////////////////////////////////////////////////////////////////////////
// MessageTypeFilter                                                          //
////////////////////////////////////////////////////////////////////////////////

// NotificationFilter defines filtering properties for channel message
// notifications.
//
// These will be tested against every received notification. The notification,
// which is encrypted, will not be decrypted properly unless the identifier is
// correct. As a result, the identifier will be tested against a garbled message
// and the probability of false collisions is simply the random chance that all
// bloom bits are flipped.
//
// Given that K = 26 (see elixxir/crypto/sih/evaluatable.go), the collision
// chance is 1/2^26. The message types would also need to collide given that in
// the 16-bit space there are only expected to be a small number of messages
// that are valid notifications, the collision chance increases by ~20/2^16.
//
// Given this information, the number of evaluations where a user has a 50%
// chance of a false notification can be calculated as (x = ln(.5)/ln(1-p))
// where p = (1/2^26) * (1/2^10), x = 1.5 * 10^11, assuming a user is registered
// in 100 channels and received 100,000 total notifications a day, this number
// of events would occur after 15,243 days, which is the mean time to false
// notification through this system. This number is very acceptable.
type NotificationFilter struct {
	Identifier []byte `json:"identifier"`

	// ChannelID is the ID that the filter belongs to.
	ChannelID *id.ID `json:"channelID"`

	// Tags is a list of tags to match against.
	Tags []string `json:"tags"`

	AllowLists `json:"allowLists"`
}

// AllowLists contains the list of messages types allowed with or without tags.
type AllowLists struct {
	// AllowWithTags is a list of message types that are not filtered when
	// there are Tags.
	AllowWithTags map[MessageType]struct{} `json:"allowWithTags"`

	// AllowWithoutTags is a list of message types that are not filtered when
	// there are no Tags.
	AllowWithoutTags map[MessageType]struct{} `json:"allowWithoutTags"`
}

// match determines if the message with the given tags and message type are
// allowed through the filter.
func (nf NotificationFilter) match(
	matchedTags map[string]struct{}, mt MessageType) bool {
	// Check if any filter tags match the matched tags
	for _, tag := range nf.Tags {

		// If a tag matches, then check if the message type is in the allow with
		// tags list
		if _, exists := matchedTags[tag]; exists {
			if _, exists = nf.AllowWithTags[mt]; exists {
				return true
			}
			return false
		}
	}

	// If no tag matches, then check if the message type is in the allow without
	// tags list
	if _, exists := nf.AllowWithoutTags[mt]; exists {
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////
// NotificationLevel                                                          //
////////////////////////////////////////////////////////////////////////////////

// notificationSourceType is the type of broadcast message the notification will
// appear on.
type notificationSourceType uint8

const (
	symmetric  notificationSourceType = 0
	asymmetric notificationSourceType = 1
)

// notificationLevelAllowLists are the predefined message type allow lists for
// each notification level.
var notificationLevelAllowLists = map[notificationSourceType]map[NotificationLevel]AllowLists{
	symmetric: {
		NotifyPing: {
			map[MessageType]struct{}{Text: {}, FileTransfer: {}},
			map[MessageType]struct{}{},
		},
		NotifyAll: {
			map[MessageType]struct{}{},
			map[MessageType]struct{}{Text: {}, FileTransfer: {}},
		},
	},
	asymmetric: {
		NotifyPing: {
			map[MessageType]struct{}{AdminText: {}},
			map[MessageType]struct{}{Pinned: {}},
		},
		NotifyAll: {
			map[MessageType]struct{}{},
			map[MessageType]struct{}{AdminText: {}, Pinned: {}},
		},
	},
}

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

// Marshal returns the byte representation of the [NotificationLevel].
func (nl NotificationLevel) Marshal() []byte {
	return []byte{byte(nl)}
}

// UnmarshalNotificationLevel unmarshalls the byte slice into a
// [NotificationLevel].
func UnmarshalNotificationLevel(b []byte) NotificationLevel {
	return NotificationLevel(b[0])
}
