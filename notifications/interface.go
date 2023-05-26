////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package notifications

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"strconv"

	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
)

var ErrInvalidNotificationsState = errors.New("invalid notifications state")

type Manager interface {
	// Set turns notifications on or off for a given ID. It synchronizes the
	// state with all clients and registers with the notification server if
	// status != Mute and a token is set.
	//
	// Group is used to segment the notification lists so that different users
	// of the same object do not interfere. Metadata will be synchronized,
	// allowing more verbose notifications settings. Max 1KB.
	//
	// It returns [ErrInvalidNotificationsState] if the passed in state is
	// invalid.
	//
	// This function, in general, will not be called over the bindings. It will
	// be used by intermediary structures like channels and DMs to provide
	// notification access on a per-case basis.
	//
	// Parameters:
	//   - toBeNotifiedOn - ID that you are tracking. You will receive
	//     notifications that need to be filtered every time a message is
	//     received on this ID.
	//   - group - The group this is categorized in. Used for callbacks and the
	//     GetGroup function to allow for automatic filtering of registered
	//     notifications for a specific submodule or use case. An error is
	//     returned if Set is called on an ID that is already registered at a
	//     different ID.
	//   - metadata - An extra field allowing storage and synchronization of
	//     specific use-case notification data.
	//   - status - The notifications state the ID should be in. These are
	//        Mute - show no notifications for the id
	//        WhenOpen - show notifications only within the open app, no
	//        registration or privacy leak will occur
	//        Push - show notifications as push notification on applicable
	//        devices, will have a minor privacy loss
	Set(toBeNotifiedOn *id.ID, group string, metadata []byte,
		status NotificationState) error

	// Get returns the status of the notifications for the given ID. Returns
	// false if the ID is not registered.
	Get(toBeNotifiedOn *id.ID) (
		status NotificationState, metadata []byte, group string, exists bool)

	// Delete deletes the given ID, unregisters it if it is registered, and
	// removes the reference from the local store.
	Delete(toBeNotifiedOn *id.ID) error

	// SetMaxState sets the maximum functional state of any identity downstream.
	// Modules will be told to clamp any state greater than maxState to
	// maxState.
	//
	// Depending on UX requirements, they may still show the state in an altered
	// manner. For example, greying out a description. This is designed so that
	// when the state is raised, the old configs are maintained.
	//
	// This will unregister/re-register with the push server when leaving or
	// entering the Push maxState. The default maxState Push will return an
	// error if the maxState is not a valid state.
	SetMaxState(maxState NotificationState) error

	// GetMaxState returns the current MaxState.
	GetMaxState() NotificationState

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
// Functionally clamps any state greater than the maxState to maxState.
type Update func(group Group, created, edits, deletions []*id.ID,
	maxState NotificationState)

type Group map[id.ID]State

func (g Group) DeepCopy() Group {
	gCopy := make(Group, len(g))
	for key, value := range g {
		gCopy[key] = value
	}
	return gCopy
}

type State struct {
	Metadata []byte            `json:"metadata"`
	Status   NotificationState `json:"status"`
}

// NotificationState indicates the status of notifications for an ID.
type NotificationState int64

const (
	// Mute shows no notifications for the ID.
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
		return "Mute"
	case WhenOpen:
		return "WhenOpen"
	case Push:
		return "Push"
	default:
		return "Unknown notifications state: " + strconv.Itoa(int(ns))
	}
}

// IsValid returns an error if the notification state is not valid.
func (ns NotificationState) IsValid() error {
	if ns >= Mute && ns <= Push {
		return nil
	}
	return ErrInvalidNotificationsState
}

type Comms interface {
	RegisterToken(host *connect.Host, message *pb.RegisterTokenRequest) (
		*messages.Ack, error)
	UnregisterToken(host *connect.Host, message *pb.UnregisterTokenRequest) (
		*messages.Ack, error)
	RegisterTrackedID(host *connect.Host,
		message *pb.RegisterTrackedIdRequest) (*messages.Ack, error)
	UnregisterTrackedID(host *connect.Host,
		message *pb.UnregisterTrackedIdRequest) (*messages.Ack, error)
	GetHost(id *id.ID) (*connect.Host, bool)
}

// MockComms are used when instantiating notification manager without
// a remote bot. They return success on operations when they do not occur
type MockComms struct{}

func (mc *MockComms) RegisterToken(host *connect.Host,
	message *pb.RegisterTokenRequest) (*messages.Ack, error) {
	jww.DEBUG.Printf("Notifications dummy RegisterToken comm called, " +
		"success returned")
	return &messages.Ack{}, nil
}

func (mc *MockComms) UnregisterToken(host *connect.Host,
	message *pb.UnregisterTokenRequest) (*messages.Ack, error) {
	jww.DEBUG.Printf("Notifications dummy UnregisterToken comm called, " +
		"success returned")
	return &messages.Ack{}, nil
}

func (mc *MockComms) RegisterTrackedID(host *connect.Host,
	message *pb.RegisterTrackedIdRequest) (*messages.Ack, error) {
	jww.DEBUG.Printf("Notifications dummy RegisterTrackedID comm called, " +
		"success returned")
	return &messages.Ack{}, nil
}

func (mc *MockComms) UnregisterTrackedID(host *connect.Host,
	message *pb.UnregisterTrackedIdRequest) (*messages.Ack, error) {
	jww.DEBUG.Printf("Notifications dummy UnregisterTrackedID comm called, " +
		"success returned")
	return &messages.Ack{}, nil
}

func (mc *MockComms) GetHost(id *id.ID) (*connect.Host, bool) {
	jww.DEBUG.Printf("Notifications dummy GetHost comm called, " +
		"success returned")
	return &connect.Host{}, true
}
