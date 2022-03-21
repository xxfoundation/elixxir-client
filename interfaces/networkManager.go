///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	"time"

	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

type NetworkManager interface {
	/* Process contol */
	Follow(report ClientErrorReport) (stoppable.Stoppable, error)
	// GetSender - Object that all gateway I/O goes though. It handles
	// selecting gateways and proxying messages through them
	GetSender() *gateway.Sender

	/* Parameters and events */
	GetInstance() *network.Instance
	GetHealthTracker() HealthTracker
	GetEventManager() EventManager
	InProgressRegistrations() int

	CheckGarbledMessages()

	// GetAddressSize returns the current address size of IDs. Blocks until an
	// address size is known.
	GetAddressSize() uint8

	// GetVerboseRounds returns stringification of verbose round info
	GetVerboseRounds() string

	// RegisterAddressSizeNotification returns a channel that will trigger for
	// every address space size update. The provided tag is the unique ID for
	// the channel. Returns an error if the tag is already used.
	RegisterAddressSizeNotification(tag string) (chan uint8, error)

	// UnregisterAddressSizeNotification stops broadcasting address space size
	// updates on the channel with the specified tag.
	UnregisterAddressSizeNotification(tag string)

	// SetPoolFilter sets the filter used to filter gateway IDs.
	// allows you to disable proxying through certain gateways
	SetPoolFilter(f gateway.Filter)

	/* Message Sending */
	SendCMIX(message format.Message, recipient *id.ID, p params.CMIX) (id.Round, ephemeral.Id, error)
	SendManyCMIX(messages []message.TargetedCmixMessage, p params.CMIX) (id.Round, []ephemeral.Id, error)

	/* Message Receiving */
	/* Identities are all network identites which the client is currently
	trying to pick up message on. Each identity has a default trigger
	pickup that it will receive on, but this default is generally
	low privacy and an alternative should be used in most cases. An identity must be added
	to receive messages, fake ones will be used to poll the network
	if none are present.  */

	// AddIdentity adds an identity to be tracked
	// and Identity is Defined by a source ID and a current EphemeralID
	// In its IdentityParams, paremeters describing the properties
	// of the identity as well as how long it will last are described
	AddIdentity(Identity, IdentityParams)
	// RemoveIdentity removes a currently tracked identity.
	RemoveIdentity(Identity)

	/* Fingerprints are the primary mechanisim of identifying a picked up message over
	   cMix. They are a unique one time use 255 bit vector generally
	   assoceated with a specific encryption key, but can be used for an alternative proptocol.
	   When registering a fingeprprint, a MessageProcessorFP is registered to handle the message.
	   The */

	//AddFingerprint - Adds a fingerprint which will be handled by a specific processor
	AddFingerprint(fingerprint format.Fingerprint, processor MessageProcessorFP)
	AddFingerprints(fingerprints map[format.Fingerprint]MessageProcessorFP)
	RemoveFingerprint(fingerprint format.Fingerprint)
	RemoveFingerprints(fingerprints []format.Fingerprint)
	CheckFingerprint(fingerprint format.Fingerprint) bool

	/* trigger - predefined hash based tags appended to all cmix messages
	which, though trial hashing, are used to determine if a message applies
	to this client

	Triggers are used for 2 purposes -  can be processed by the notifications system,
	or can be used to implement custom non fingerprint processing of payloads.
	I.E. key negotiation, broadcast negotiation

	A tag is appended to the message of the format tag = H(H(messageContents),preimage)
	and trial hashing is used to determine if a message adheres to a tag.
	WARNING: If a preiamge is known by an adversary, they can determine which messages
	are for the client.

	Due to the extra overhead of trial hashing, triggers are processed after fingerprints.
	If a fingerprint match occurs on the message, triggers will not be handled.

	Triggers are ephemeral to the session. When starting a new client, all triggers must be
	re-added before StartNetworkFollower is called.
	*/

	// AddTrigger - Adds a trigger which can call a message
	// handing function or be used for notifications.
	// Multiple triggers can be registered for the same preimage.
	//   preimage - the preimage which is triggered on
	//   type - a descriptive string of the trigger. Generally used in notifications
	//   source - a byte buffer of related data. Generally used in notifications.
	//     Example: Sender ID
	AddTrigger(trigger Trigger, response MessageProcessor) error

	// RemoveTrigger - If only a single response is associated with the preimage, the entire
	// preimage is removed. If there is more than one response, only the given response is removed
	// if nil is passed in for response, all triggers for the preimage will be removed
	RemoveTrigger(preimage Preimage, response MessageProcessor) error

	// TrackTriggers - Registers a callback which will get called every time triggers change.
	// It will receive the triggers list every time it is modified.
	// Will only get callbacks while the Network Follower is running.
	// Multiple trackTriggers can be registered
	TrackTriggers(func(triggers []Trigger))
}

type Preimage [32]byte

type Identity struct {
	// Identity
	EphId  ephemeral.Id
	Source *id.ID
}
type IdentityParams struct {
	AddressSize uint8

	// Usage variables
	End         time.Time // Timestamp when active polling will stop
	ExtraChecks uint      // Number of extra checks executed as active after the
	// ID exits active

	// Polling parameters
	StartValid time.Time // Timestamp when the ephID begins being valid
	EndValid   time.Time // Timestamp when the ephID stops being valid

	// Makes the identity not store on disk
	// When an ephemeral identity is deleted, all fingerprints & triggers
	// associated with it also delete.
	// TODO: This should not be confused with EphID for checking
	// when messages are for the the user. That's a different type
	// of Ephemeral in this context.
	Ephemeral bool
}

type Trigger struct {
	Preimage
	Type   string
	Source []byte
}

type MessageProcessor interface {
	// Process decrypts and hands off the message to its internal down
	// stream message processing system.
	// CRITICAL: Fingerprints should never be used twice. Process must
	// denote, in long term storage, usage of a fingerprint and that
	// fingerprint must not be added again during application load.
	// It is a security vulnerability to reuse a fingerprint. It leaks
	// privacy and can lead to compromise of message contents and integrity.
	Process(message format.Message, receptionID Identity,
		round *mixmessages.RoundInfo)
}

//type Ratchet interface {
//	SendE2E(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
//	SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error)
//	AddPartner(partnerID *id.ID, partnerPubKey,
//		myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
//		mySIDHPrivKey *sidh.PrivateKey,
//		sendParams, receiveParams params.E2ESessionParams)
//	GetPartner(partnerID *id.ID) (*Manager, error)
//	DeletePartner(partnerId *id.ID)
//	GetAllPartnerIDs() []*id.ID
//}

//for use in key exchange which needs to be callable inside of network
type SendE2E func(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
