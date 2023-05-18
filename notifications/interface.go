////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package notifications

import (
	"gitlab.com/xx_network/primitives/id"
	"strconv"
)

type Manger interface {
	// Set turns notifications on or off for a given ID. It synchronizes the
	// state with all clients and register with the notifications server if
	// status != Mute and a token is set.
	//
	// Group is used to segment the notifications lists so that different users
	// of the same object do not interfere. Metadata will be synchronized,
	// allowing more verbose notifications settings. Max 1KB.
	Set(toBeNotifiedOn *id.ID, group string, metadata []byte,
		status NotificationState) error

	// Get returns the status of the notifications for the given ID. Returns
	// false if the ID is not registered.
	Get(toBeNotifiedOn *id.ID) (
		status NotificationState, metadata []byte, group string, exists bool)

	// Delete deletes the given ID, unregisters it if it is registered, and
	// removes the reference from the local store.
	Delete(toBeNotifiedOn *id.ID)

	// GetGroup returns the state of all registered notifications for the given
	// group. If the group is not present, then it returns false.
	GetGroup(group string) (Group, bool)

	// AddToken registers the token with the remote server if this manager is
	// in set to register, otherwise it will return ErrRemoteRegistrationDisabled.
	//
	// This will add the token to the list of tokens that are forwarded the
	// messages for connected IDs. The App will tell the server what app to
	// forward the notifications to.
	AddToken(newToken, app string) error

	// RemoveToken removes the given token from the notification server.
	// It will remove all registered identities if it is the last Token.
	RemoveToken() error

	// RegisterUpdateCallback registers a callback to be used to receive updates
	// to changes in notifications. Because this is being called after
	// initialization, a poll of state via the get function will be necessary
	// because notifications can be missed. You must rely on the data in the
	// callback for the update and not poll the interface.
	RegisterUpdateCallback(group string, nu Update)
}

// Update is called every time there is a change to notifications.
type Update func(group Group, created, edits, deletions []*id.ID)

type Group map[id.ID]State

type State struct {
	Metadata []byte            `json:"metadata"`
	Status   NotificationState `json:"status"`
}

// NotificationStatus indicates the status of notifications for an ID.
type NotificationState uint8

const (
	// Mute shows no notifications for the ID
	Mute NotificationState = iota

	// WhenOpen shows notifications for this ID only when the app is running and
	// open. No registration or privacy leaks occur in this state.
	WhenOpen

	// Push shows notifications for this ID as push notification on applicable
	// devices. This state has a minor privacy loss.
	Push
)

// String prints a human-readable version of the [NotificationStatus] for
// logging and debugging. This function adheres to the [fmt.Stringer] interface.
func (ns NotificationState) String() string {
	switch ns {
	case Mute:
		return "mute"
	case WhenOpen:
		return "when open"
	case Push:
		return "push"
	default:
		return "invalid NotificationStatus: " + strconv.Itoa(int(ns))
	}
}
