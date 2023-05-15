package notifications

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
)

type Manager interface {
	// Set can be used to turn on or off notifications for a given ID.
	// Will synchronize the state with all clients and register with the notifications
	// server if status == true and a Token is set
	// Group is used to segment the notifications lists so different users of the same
	// object do not interfere. Metadata will be synchronized, allowing more verbose
	// notifications settings. Max 1KB.
	Set(toBeNotifiedOn *id.ID, group string, metadata []byte, status NotificationState) error
	// Get returns the status of the notifications for the given ID, or
	// an error if not present
	Get(toBeNotifiedOn *id.ID) (status NotificationState, metadata []byte, group string, exists bool)
	// Delete deletes the given notification, unregistering it if it is registered
	// and removing the reference from the local store
	Delete(toBeNotifiedOn *id.ID) error
	// GetGroup the status of all registered notifications for
	// the given group. If the group isn't present, an empty map will be returned.
	GetGroup(group string) (Group, bool)
	// AddToken registers the Token with the remote server if this manager is
	// in set to register, otherwise it will return ErrRemoteRegistrationDisabled
	// This will add the token to the list of tokens which are forwarded the messages
	// for connected IDs.
	// the App will tell the server what App to forward the notifications to.
	AddToken(newToken, app string) error
	// RemoveToken removes the given Token from the server
	// It will remove all registered identities if it is the last Token
	RemoveToken() error
	// RegisterUpdateCallback registers a callback to be used to receive notifications
	// of changes in notifications. Because this is being called after initialization,
	// a poll of state via the get function will be necessary because notifications can be missed
	// You must rely on the data in the callback for the update, do not poll
	// the interface
	RegisterUpdateCallback(group string, nu Update)
}
type Update func(group Group, created, edits, deletions []*id.ID)

type Group map[id.ID]State

func (g Group) DeepCopy() Group {
	gCopy := make(Group, len(g))
	for key, value := range g {
		gCopy[key] = value
	}
	return gCopy
}

type State struct {
	Metadata []byte
	Status   NotificationState
}

type NotificationState uint8

const (
	// Mute - show no notifications for the id
	Mute NotificationState = iota
	// WhenOpen - show notifications only within the open app, no registration
	// or privacy leak will occur
	WhenOpen
	// Push - show notifications as push notification on applicable devices,
	// will have a minor privacy loss
	Push
)

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

type Comms interface {
	RegisterToken(host *connect.Host, message *pb.RegisterTokenRequest) (
		*messages.Ack, error)
	UnregisterToken(host *connect.Host, message *pb.UnregisterTokenRequest) (
		*messages.Ack, error)
	RegisterTrackedID(host *connect.Host,
		message *pb.TrackedIntermediaryIDRequest) (*messages.Ack, error)
	UnregisterTrackedID(host *connect.Host,
		message *pb.TrackedIntermediaryIDRequest) (*messages.Ack, error)
	GetHost(id *id.ID) (*connect.Host, bool)
}
