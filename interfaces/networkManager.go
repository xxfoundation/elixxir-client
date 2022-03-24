///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

/*===Sending==============================================================*/

package interfaces

import (
	"gitlab.com/elixxir/client/network/gateway"
	"time"

	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

type NetworkManager interface {
	// Follow starts the tracking of the network in a new thread.
	// Errors that occur are reported on the ClientErrorReport function if
	// passed. The returned stopable can be used to stop the follower.
	// Only one follower may run at a time.
	Follow(report ClientErrorReport) (stoppable.Stoppable, error)

	// SendCMIX sends a "raw" CMIX message payload to the provided recipient.
	// Returns the round ID of the round the payload was sent or an error
	// if it fails.
	SendCMIX(message format.Message, recipient *id.ID, p params.CMIX) (id.Round, ephemeral.Id, error)
	SendManyCMIX(messages []message.TargetedCmixMessage, p params.CMIX) (id.Round, []ephemeral.Id, error)

	/*===Accessors============================================================*/
	/* Accessors */
	// GetInstance returns the network instance object, which tracks the
	// state of the network
	GetInstance() *network.Instance

	// GetInstance returns the health tracker, which using a polling or
	// event api lets you determine if network following is functioning
	GetHealthTracker() HealthTracker

	// GetVerboseRounds returns stringification of verbose round info
	GetVerboseRounds() string

	// SetPoolFilter sets the filter used to filter gateway IDs.
	// allows you to disable proxying through certain gateways
	SetPoolFilter(f gateway.Filter)

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
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool) error
	// RemoveIdentity removes a currently tracked identity.
	RemoveIdentity(id *id.ID)

	/* Fingerprints are the primary mechanisim of identifying a picked up message over
	   cMix. They are a unique one time use 255 bit vector generally
	   assoceated with a specific encryption key, but can be used for an alternative proptocol.
	   When registering a fingeprprint, a MessageProcessorFP is registered to handle the message.
	   The */

	//AddFingerprint - Adds a fingerprint which will be handled by a specific processor
	AddFingerprint(fingerprint format.Fingerprint, processor MessageProcessor)
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

	Triggers are address to the session. When starting a new client, all triggers must be
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

	/*Address Space*/
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
}

type Preimage [32]byte

type EphemeralIdentity struct {
	// Identity
	EphId  ephemeral.Id
	Source *id.ID
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
	Process(message format.Message, receptionID EphemeralIdentity,
		round *mixmessages.RoundInfo)
}

//type Ratchet interface {
//	SendE2E(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
//	SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error)
//	AddPartner(partnerID *id.ID, partnerPubKey,
//		myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
//		mySIDHPrivKey *sidh.PrivateKey,
//		sendParams, receiveParams params.E2ESessionParams)
//	GetPartner(partnerID *id.ID) (*manager, error)
//	DeletePartner(partnerId *id.ID)
//	GetAllPartnerIDs() []*id.ID
//}

//for use in key exchange which needs to be callable inside of network
type SendE2E func(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
