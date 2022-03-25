package network

import (
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

type Manager interface {
	// Follow starts the tracking of the network in a new thread.
	// Errors that occur are reported on the ClientErrorReport function if
	// passed. The returned stopable can be used to stop the follower.
	// Only one follower may run at a time.
	Follow(report ClientErrorReport) (stoppable.Stoppable, error)

	/*===Sending==============================================================*/

	// SendCMIX sends a "raw" CMIX message payload to the provided recipient.
	// Returns the round ID of the round the payload was sent or an error
	// if it fails.
	SendCMIX(message format.Message, recipient *id.ID, p CMIXParams) (
		id.Round, ephemeral.Id, error)

	// SendManyCMIX sends many "raw" cMix message payloads to each of the provided
	// recipients. Used to send messages in group chats. Metadata is NOT as well
	// protected with this call and can leak data about yourself. Should be
	// replaced with multiple uses of SendCmix in most cases. Returns the round
	// ID of the round the payload was sent or an error if it fails.
	// WARNING: Potentially Unsafe
	SendManyCMIX(messages []TargetedCmixMessage, p CMIXParams) (
		id.Round, []ephemeral.Id, error)

	/*===Message Reception====================================================*/
	/* Identities are all network identites which the client is currently
	trying to pick up message on. An identity must be added
	to receive messages, fake ones will be used to poll the network
	if none are present. On creation of the network handler, the identity in
	session storage will be automatically added*/

	// AddIdentity adds an identity to be tracked
	// If persistent is false, the identity will not be stored to disk and
	// will be dropped on reload.
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool) error
	// RemoveIdentity removes a currently tracked identity.
	RemoveIdentity(id *id.ID)

	/* Fingerprints are the primary mechanism of identifying a picked up
	message over cMix. They are a unique one time use 255 bit vector generally
	associated with a specific encryption key, but can be used for an
	alternative protocol.When registering a fingerprint, a MessageProcessor
	is registered to handle the message.*/

	// AddFingerprint - Adds a fingerprint which will be handled by a
	// specific processor for messages received by the given identity
	AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
		mp MessageProcessor) error

	// DeleteFingerprint deletes a single fingerprint associated with the given
	// identity if it exists
	DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint)

	// DeleteClientFingerprints deletes al fingerprint associated with the given
	// identity if it exists
	DeleteClientFingerprints(identity *id.ID)

	/* trigger - predefined hash based tags appended to all cMix messages
	which, though trial hashing, are used to determine if a message applies
	to this client

	Triggers are used for 2 purposes - They can be processed by the
	notifications system, or can be used to implement custom non fingerprint
	processing of payloads. I.E. key negotiation, broadcast negotiation

	A tag is appended to the message of the format tag = H(H(messageContents),
	preimage) and trial hashing is used to determine if a message adheres to a
	tag.
	WARNING: If a preimage is known by an adversary, they can determine which
	messages are for the client on reception (which is normally hidden due to
	collision between ephemeral IDs.

	Due to the extra overhead of trial hashing, triggers are processed after fingerprints.
	If a fingerprint match occurs on the message, triggers will not be handled.

	Triggers are address to the session. When starting a new client, all triggers must be
	re-added before StartNetworkFollower is called.
	*/

	// AddTrigger - Adds a trigger which can call a message handing function or
	// be used for notifications. Multiple triggers can be registered for the
	// same preimage.
	//   preimage - the preimage which is triggered on
	//   type - a descriptive string of the trigger. Generally used in notifications
	//   source - a byte buffer of related data. Generally used in notifications.
	//     Example: Sender ID
	AddTrigger(identity *id.ID, newTrigger Trigger, response MessageProcessor)

	// DeleteTrigger - If only a single response is associated with the
	// preimage, the entire preimage is removed. If there is more than one
	// response, only the given response is removed if nil is passed in for
	// response, all triggers for the preimage will be removed
	DeleteTrigger(identity *id.ID, preimage Preimage, response MessageProcessor) error

	// DeleteClientTriggers - deletes all triggers assoseated with the given identity
	DeleteClientTriggers(identity *id.ID)

	// TrackTriggers - Registers a callback which will get called every time triggers change.
	// It will receive the triggers list every time it is modified.
	// Will only get callbacks while the Network Follower is running.
	// Multiple trackTriggers can be registered
	TrackTriggers(TriggerTracker)


	//Dropped Messages Pickup
	RegisterDroppedMessagesPickup(response MessageProcessor)
	DenoteReception(msgId uint)

	/* In inProcess */
	// it is possible to receive a message over cMix before the fingerprints or
	// triggers are registered. As a result, when handling fails, messages are
	// put in the inProcess que for a set number of retries.

	// CheckInProgressMessages - retry processing all messages in check in
	// progress messages. Call this after adding fingerprints or triggers
	//while the follower is running.
	CheckInProgressMessages()

	/*===Health Monitor=======================================================*/
	// The health monitor is a system which tracks if the client sees a live
	// network. It can either be polled or set up with events

	// IsHealthy Returns true if currently healthy
	IsHealthy() bool

	// WasHealthy returns true if the network has ever been healthy in this run
	WasHealthy() bool

	// AddHealthCallback - adds a callback which gets called whenever the heal
	// changes. Returns a registration ID which can be used to unregister
	AddHealthCallback(f func(bool)) uint64

	// RemoveHealthCallback - Removes a health callback using its
	// registration ID
	RemoveHealthCallback(uint64)

	/*===Nodes================================================================*/
	/* Keys must be registed with nodes in order to send messages throug them.
	this process is in general automatically handled by the Network Manager*/

	// HasNode can be used to determine if a keying relationship exists with a
	// node.
	HasNode(nid *id.ID) bool

	// NumRegisteredNodes Returns the total number of nodes we have a keying
	// relationship with
	NumRegisteredNodes() int

	// TriggerNodeRegistration Triggers the negotiation of a keying
	// relationship with a given node
	TriggerNodeRegistration(nid *id.ID)

	/*===Historical Rounds====================================================*/
	/* A complete set of round info is not kept on the client, and sometimes
	the network will need to be queried to get round info. Historical rounds
	is the system internal to the Network Manager to do this.
	It can be used externally as well.*/

	// LookupHistoricalRound - looks up the passed historical round on the
	// network
	LookupHistoricalRound(rid id.Round, callback func(info *mixmessages.RoundInfo,
		success bool)) error

	/*===Sender===============================================================*/
	/* The sender handles sending comms to the network. It tracks connections to
	gateways and handles proxying to gateways for targeted comms. It can be
	used externally to contact gateway directly, bypassing the majority of
	the network package*/

	// SendToAny can be used to send the comm to any gateway in the network.
	SendToAny(sendFunc func(host *connect.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error)

	// SendToPreferred sends to a specific gateway, doing so through another
	// gateway as a proxy if not directly connected.
	SendToPreferred(targets []*id.ID, sendFunc func(host *connect.Host,
		target *id.ID, timeout time.Duration) (interface{}, error),
		stop *stoppable.Single, timeout time.Duration) (interface{}, error)

	// SetGatewayFilter sets a function which will be used to filter gateways
	// before connecting.
	SetGatewayFilter(f func(map[id.ID]int,
		*ndf.NetworkDefinition) map[id.ID]int)

	// GetHostParams - returns the host params used when connectign to gateways
	GetHostParams() connect.HostParams

	/*===Address Space========================================================*/
	// The network compasses identities into a smaller address space to cause
	// collisions and hide the actual recipient of messages. These functions
	// allow for the tracking of this addresses space. In general, address space
	// issues are completely handled by the network package

	// GetAddressSpace GetAddressSize returns the current address size of IDs. Blocks until an
	// address size is known.
	GetAddressSpace() uint8

	// RegisterAddressSpaceNotification returns a channel that will trigger for
	// every address space size update. The provided tag is the unique ID for
	// the channel. Returns an error if the tag is already used.
	RegisterAddressSpaceNotification(tag string) (chan uint8, error)

	// UnregisterAddressSpaceNotification stops broadcasting address space size
	// updates on the channel with the specified tag.
	UnregisterAddressSpaceNotification(tag string)

	/*===Accessors============================================================*/

	// GetInstance returns the network instance object, which tracks the
	// state of the network
	GetInstance() *network.Instance

	// GetVerboseRounds returns stringification of verbose round info
	GetVerboseRounds() string
}

type Preimage [32]byte

type Trigger struct {
	Preimage
	Type   string
	Source []byte
}

type TriggerTracker func(triggers []Trigger)

type MessageProcessor interface {
	// Process decrypts and hands off the message to its internal down
	// stream message processing system.
	// CRITICAL: Fingerprints should never be used twice. Process must
	// denote, in long term storage, usage of a fingerprint and that
	// fingerprint must not be added again during application load.
	// It is a security vulnerability to reuse a fingerprint. It leaks
	// privacy and can lead to compromise of message contents and integrity.
	Process(message format.Message, receptionID receptionID.EphemeralIdentity,
		round *mixmessages.RoundInfo)
}

type ClientErrorReport func(source, message, trace string)
