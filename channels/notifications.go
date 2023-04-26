////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"sync"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/sih"
	primNotif "gitlab.com/elixxir/primitives/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// FilterCallback is a callback that returns a slice of [NotificationFilter] of
// all channels with notifications enabled any time a service is added or
// removed or the notification level changes. The [NotificationFilter] is used
// to determine which notifications from the notification server belong to the
// caller.
type FilterCallback func(nfs []NotificationFilter)

// Storage values.
const (
	notificationsKvVersion = 0
	notificationsKvKey     = "channelsNotifications"
)

// notifications manages the notification level for each channel.
type notifications struct {
	// User's public key
	pubKey ed25519.PublicKey

	// User supplied callback to return updated NotificationFilter objects to.
	cb FilterCallback

	// List of channels and their notification levels
	channels map[id.ID]NotificationLevel

	kv  *versioned.KV
	nm  NotificationsManager
	net Client

	mux sync.Mutex
}

// newOrLoadNotifications initialises a new channels notifications manager if
// none exists. If one has previously been made, it is loaded.
func newOrLoadNotifications(pubKey ed25519.PublicKey, cb FilterCallback,
	nm NotificationsManager, kv *versioned.KV, net Client) *notifications {
	n := newNotifications(pubKey, cb, nm, kv, net)

	err := n.load()
	if err != nil && kv.Exists(err) {
		jww.FATAL.Panicf("Failed to load notification manager: %+v", err)
	}

	return n
}

// newNotifications initialises a new channels notifications manager.
func newNotifications(pubKey ed25519.PublicKey, cb FilterCallback,
	nm NotificationsManager, kv *versioned.KV, net Client) *notifications {
	return &notifications{
		pubKey:   pubKey,
		cb:       cb,
		channels: make(map[id.ID]NotificationLevel),
		kv:       kv,
		nm:       nm,
		net:      net,
		mux:      sync.Mutex{},
	}
}

// addChannel inserts the channel into the notification list with no the
// notification level set to none.
//
// Returns an error if the channel already exists.
func (n *notifications) addChannel(channelID *id.ID) error {
	n.mux.Lock()
	defer n.mux.Unlock()
	if level, exists := n.channels[*channelID]; exists {
		jww.FATAL.Panicf("[CH] Cannot add channel %s to notification list "+
			"when it already exists with level %s", channelID, level)
	}

	n.channels[*channelID] = NotifyNone
	return n.save()
}

// addChannel inserts the channel into the notification list with the given
// level.
func (n *notifications) removeChannel(channelID *id.ID) {
	n.mux.Lock()
	defer n.mux.Unlock()
	level, exists := n.channels[*channelID]
	if !exists {
		jww.WARN.Printf("[CH] Cannot remove channel %s from notification "+
			"list when it does not exist.", channelID)
		return
	}

	if level != NotifyNone {
		err := n.nm.UnregisterNotificationIdentity(channelID)
		if err != nil {
			// TODO: Instead of returning an error here, make failed unregisters
			//  be added to a list to be retried.
			jww.ERROR.Printf("[CH] Failed to unregister channel %s for "+
				"notifications.", channelID)
		}
	}

	delete(n.channels, *channelID)

	// Print an error to the log instead of returning or panicking on save error
	// because the worst thing that happens is a storage leak
	if err := n.save(); err != nil {
		jww.ERROR.Printf("[CH] Failed to update channel notification storage "+
			"after removing channel %s: %+v", channelID, err)
	}
}

// SetMobileNotificationsLevel sets the notification level for the given
// channel. If the notification leve lis changed from [NotifyNone] to another
// level, then the channel is registered with the external notification server.
// If a channel level is set to [NotifyNone], then it is unregistered.
//
// Note, when enabling notifications, information may be shared with third
// parties (i.e., Firebase and Google's Palantir) that may represent a security
// risk to the user.
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
		if err := n.nm.RegisterForNotifications(channelID); err != nil {
			return err
		}
	} else if chanLevel != NotifyNone && level == NotifyNone {
		if err := n.nm.UnregisterNotificationIdentity(channelID); err != nil {
			return err
		}
	}

	n.channels[*channelID] = level

	if err := n.save(); err != nil {
		return err
	}

	// Call the callback with the updated filter list
	go n.serviceTracker(n.net.GetServices())

	return nil
}

// serviceTracker gets the list of all services and assembles a list of
// NotificationFilter for each channel that exists in the compressed service
// list. The results are called on the user-registered FilterCallback.
func (n *notifications) serviceTracker(
	_ message.ServiceList, csl message.CompressedServiceList) {
	n.mux.Lock()
	nfs := n.createFilterList(csl)
	n.mux.Unlock()

	n.cb(nfs)
}

func (n *notifications) createFilterList(
	csl message.CompressedServiceList) []NotificationFilter {
	var nfs []NotificationFilter
	for chanID, level := range n.channels {
		channelID := &chanID
		if sList, exists := csl[chanID]; exists {
			for _, s := range sList {
				if level == NotifyNone {
					continue
				}

				nfs = append(nfs, NotificationFilter{
					Identifier: s.Identifier,
					ChannelID:  channelID,
					Tags:       makeUserPingTags(n.pubKey),
					AllowLists: notificationLevelAllowLists[level],
				})
			}
		}
	}

	return nfs
}

////////////////////////////////////////////////////////////////////////////////
// Storage                                                                    //
////////////////////////////////////////////////////////////////////////////////

// save marshals and saves the channels to storage.
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

// load loads and unmarshalls the channels from storage into notifications.
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
}

// MarshalJSON marshals the notifications into valid JSON. This function adheres
// to the json.Marshaler interface.
func (n *notifications) MarshalJSON() ([]byte, error) {
	nd := notificationsDisk{
		Channels: make(map[string]NotificationLevel, len(n.channels)),
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

	return nil
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

// GetNotificationReportForMe checks the notification data against the filter
// list to determine which notifications belong to the user. A list of
// notifications reports is returned detailing all notifications for the user.
func GetNotificationReportForMe(nfs []NotificationFilter,
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

// notificationLevelAllowLists are the predefined message type allow lists for
// each notification level.
var notificationLevelAllowLists = map[NotificationLevel]AllowLists{
	NotifyPing: {
		map[MessageType]struct{}{Text: {}, AdminText: {}, FileTransfer: {}},
		map[MessageType]struct{}{Pinned: {}},
	},
	NotifyAll: {
		map[MessageType]struct{}{},
		map[MessageType]struct{}{Text: {}, AdminText: {}, FileTransfer: {}},
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
