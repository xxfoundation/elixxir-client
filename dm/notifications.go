////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/dm"
	"gitlab.com/elixxir/crypto/sih"
	primNotif "gitlab.com/elixxir/primitives/notifications"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"sync"
)

// NotificationUpdate is a callback that is called any time a notification
// level changes.
//
// It returns a slice of [NotificationFilter] for all DM conversations with
// notifications enabled. The [NotificationFilter] is used to determine which
// notifications from the notification server belong to the caller.
//
// It also returns a map of all DM conversation notification states that have
// changed and all that have been deleted. The maxState is the global state set
// for notifications.
type NotificationUpdate func(nf NotificationFilter,
	changedNotificationStates []NotificationState,
	deletedNotificationStates []ed25519.PublicKey)

// NotificationState contains information about the notifications for a DM
// conversation.
type NotificationState struct {
	PartnerPubKey ed25519.PublicKey `json:"partnerPubKey"`
	Level         NotificationLevel `json:"level"`
}

// notificationGroup is the name used for to denote DM notifications in the
// notification manager.
const notificationGroup = "channelsDM"

// notifications manages the notification level for each channel.
type notifications struct {
	// User's ID, public key, and private key
	id      *id.ID
	pubKey  ed25519.PublicKey
	privKey ed25519.PrivateKey

	// partnerTagMap is a map of DM partner's [ed25519.PublicKey] to their SIH
	// tag. The key of the map is the [ed25519.PublicKey] cast to a string.
	partnerTagMap map[string]string

	// User supplied callback to return updated NotificationFilter and channel
	// notification statuses to.
	cb NotificationUpdate

	// Remotely-synced storage that contains a list of all DM partner's and
	// their notification level.
	us *partnerStore

	nm NotificationsManager

	mux sync.Mutex
}

// newNotifications initializes a new channels notifications manager.
func newNotifications(myID *id.ID, pubKey ed25519.PublicKey,
	privKey ed25519.PrivateKey, cb NotificationUpdate,
	us *partnerStore, nm NotificationsManager) (*notifications, error) {
	n := &notifications{
		id:            myID,
		pubKey:        pubKey,
		privKey:       privKey,
		partnerTagMap: make(map[string]string),
		cb:            cb,
		us:            us,
		nm:            nm,
		mux:           sync.Mutex{},
	}

	err := n.us.listen(n.updateSihTagsCB)
	if err != nil {
		return nil, err
	}

	return n, n.nm.Set(myID, notificationGroup, nil, clientNotif.Push)
}

func statusToLevel(status partnerStatus) NotificationLevel {
	switch status {
	case statusMute, statusBlocked:
		return NotifyNone
	case statusNotifyAll:
		return NotifyAll
	}
	return 0
}

// updateSihTagsCB is a callback registered on the partnerStore to receive updates
func (n *notifications) updateSihTagsCB(edits []elementEdit) {
	n.mux.Lock()
	defer n.mux.Unlock()

	var changedNotificationStates []NotificationState
	var deletedNotificationStates []ed25519.PublicKey
	for _, edit := range edits {
		switch edit.operation {
		case versioned.Created, versioned.Loaded:
			if edit.new.Status == statusNotifyAll {
				n.partnerTagMap[string(edit.new.PublicKey)] =
					dm.MakeReceiverSihTag(edit.new.PublicKey, n.privKey)
			}

			changedNotificationStates = append(changedNotificationStates, NotificationState{
				PartnerPubKey: edit.new.PublicKey,
				Level:         statusToLevel(edit.new.Status),
			})
		case versioned.Updated:
			if edit.old.Status == statusNotifyAll && edit.new.Status != statusNotifyAll {
				delete(n.partnerTagMap, string(edit.new.PublicKey))
			} else if edit.old.Status != statusNotifyAll && edit.new.Status == statusNotifyAll {
				n.partnerTagMap[string(edit.new.PublicKey)] =
					dm.MakeReceiverSihTag(edit.new.PublicKey, n.privKey)
			}

			changedNotificationStates = append(changedNotificationStates, NotificationState{
				PartnerPubKey: edit.new.PublicKey,
				Level:         statusToLevel(edit.new.Status),
			})
		case versioned.Deleted:
			delete(n.partnerTagMap, string(edit.new.PublicKey))

			deletedNotificationStates = append(
				deletedNotificationStates, edit.new.PublicKey)
		}
	}

	tags := make([]string, 0, len(n.partnerTagMap))
	publicKeys := make(map[string]ed25519.PublicKey, len(n.partnerTagMap))
	for pubKey, tag := range n.partnerTagMap {
		tags = append(tags, tag)
		publicKeys[tag] = []byte(pubKey)
	}

	nf := NotificationFilter{
		Identifier:   n.pubKey,
		MyID:         n.id,
		Tags:         tags,
		PublicKeys:   publicKeys,
		AllowedTypes: allowList[NotifyAll],
	}

	go n.cb(nf, changedNotificationStates, deletedNotificationStates)
}

// EnableNotifications enables DM notifications.
//
// To use push notifications, a token must be registered with the notification
// manager. Note, when enabling push notifications, information may be shared
// with third parties (i.e., Firebase and Google's Palantir) and may represent a
// security risk to the user.
func (n *notifications) EnableNotifications(myID *id.ID,
	status clientNotif.NotificationState) error {

	return n.nm.Set(myID, notificationGroup, nil, status)
}

// GetNotificationLevel returns the notification level for the given channel.
func (n *notifications) GetNotificationLevel(
	partnerPubKey ed25519.PublicKey) (NotificationLevel, error) {

	user, exists := n.us.get(partnerPubKey)
	if !exists {
		return 0,
			errors.Errorf("not DM conversation found with %X", partnerPubKey)
	}

	switch user.Status {
	case statusMute, statusBlocked:
		return NotifyNone, nil
	case statusNotifyAll:
		return NotifyAll, nil
	}

	return 0, errors.Errorf("invalid status found: %d", user.Status)
}

// SetMobileNotificationsLevel sets the notification level for the given DM
// conversation partner. The [NotificationLevel] dictates the type of
// notifications received and the status controls weather the notification is
// push or in-app. If muted, both the level and status must be set to mute.
func (n *notifications) SetMobileNotificationsLevel(partnerPubKey ed25519.PublicKey,
	level NotificationLevel) error {
	jww.INFO.Printf("[CH] Set notification level for DM partner %X to %s",
		partnerPubKey, level)

	var status partnerStatus
	switch level {
	case NotifyNone:
		status = statusMute
	case NotifyAll:
		status = statusNotifyAll
	default:
		return errors.Errorf("invalid notification level: %d", level)
	}

	n.us.set(partnerPubKey, status)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// For Me / Notification Report                                               //
////////////////////////////////////////////////////////////////////////////////

// NotificationReport describes information about a single notification
// belonging to the user.
type NotificationReport struct {
	// Public key of DM partner.
	Partner ed25519.PublicKey `json:"partner"`

	// Type is the MessageType of the message that the notification belongs to.
	Type MessageType `json:"type"`
}

// GetNotificationReportsForMe checks the notification data against the filter
// list to determine which notifications belong to the user. A list of
// notification reports is returned detailing all notifications for the user.
func GetNotificationReportsForMe(nf NotificationFilter,
	notificationData []*primNotif.Data) []NotificationReport {

	var nrs []NotificationReport

	for _, data := range notificationData {
		matchedTags, metadata, found, err := sih.EvaluateCompressedSIH(
			nf.MyID, data.MessageHash, nf.Identifier, nf.Tags, data.IdentityFP)
		if err != nil {
			jww.TRACE.Printf("[CH] Failed to evaluate compressed SIH for "+
				"DM partner identifier %v", nf.Identifier)
			continue
		}

		if found {
			var b [2]byte
			copy(b[:], metadata)
			messageType := UnmarshalMessageType(b)

			nr, allowed := nf.match(matchedTags, messageType)
			if allowed {
				nrs = append(nrs, nr)
			}
		}
	}

	return nrs
}

////////////////////////////////////////////////////////////////////////////////
// MessageTypeFilter                                                          //
////////////////////////////////////////////////////////////////////////////////

// NotificationFilter defines filtering properties for DM message notifications.
//
// These will be tested against every received notification. The notification,
// which is encrypted, will not be decrypted properly unless the identifier is
// correct. As a result, the identifier will be tested against a garbled message,
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
	// Identifier is this user's public key. It is set by a partner on the
	// message.CompressedService when sending a DM message.
	Identifier []byte `json:"identifier"`

	// MyID is this user's reception ID.
	MyID *id.ID `json:"myID"`

	// Tags is a list of all SIH tags for each DM partner. It is used to ensure
	// a received notification is from a valid DM partner.
	Tags []string `json:"tags"`

	// PublicKeys is a map of tags to their public keys. Used to identify the
	// owner of the message triggering a notification.
	PublicKeys map[string]ed25519.PublicKey

	// List of MessageType to notify on.
	AllowedTypes map[MessageType]struct{} `json:"allowedTypes"`
}

// match determines if the message with the given tags and message type are
// allowed through the filter.
func (nf NotificationFilter) match(
	matchedTags map[string]struct{}, mt MessageType) (NotificationReport, bool) {

	// Check if the message type is allowed
	if _, exists := nf.AllowedTypes[mt]; exists {

		// Verify that the public key exists (it always should at this point)
		for tag := range matchedTags {
			pubKey, exists := nf.PublicKeys[tag]
			if exists {
				return NotificationReport{
					Partner: pubKey,
					Type:    mt,
				}, true
			}
		}
	}

	return NotificationReport{}, false
}

////////////////////////////////////////////////////////////////////////////////
// NotificationLevel                                                          //
////////////////////////////////////////////////////////////////////////////////

// NotificationLevel specifies what level of notifications should be received
// for a channel.
type NotificationLevel uint8

var allowList = map[NotificationLevel]map[MessageType]struct{}{
	NotifyNone: {},
	NotifyAll:  {TextType: {}, ReplyType: {}},
}

const (
	// NotifyNone results in no notifications.
	NotifyNone NotificationLevel = 10

	// NotifyAll results in notifications from all messages except silent ones.
	NotifyAll NotificationLevel = 40
)

// String prints a human-readable form of the [NotificationLevel] for logging
// and debugging. This function adheres to the [fmt.Stringer] interface.
func (nl NotificationLevel) String() string {
	switch nl {
	case NotifyNone:
		return "none"
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
