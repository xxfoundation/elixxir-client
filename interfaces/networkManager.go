///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

type NetworkManager interface {
	// The stoppable can be nil.
	SendCMIX(message format.Message, recipient *id.ID, p params.CMIX) (id.Round, ephemeral.Id, error)
	SendManyCMIX(messages []message.TargetedCmixMessage, p params.CMIX) (id.Round, []ephemeral.Id, error)
	GetInstance() *network.Instance
	GetHealthTracker() HealthTracker
	GetEventManager() EventManager
	GetSender() *gateway.Sender
	Follow(report ClientErrorReport) (stoppable.Stoppable, error)
	CheckGarbledMessages()
	InProgressRegistrations() int

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
	SetPoolFilter(f gateway.Filter)

	/* Identities are all network identites which the client is currently
	 trying to pick up message on. Each identity has a defult edge
	 pickup that it will receive on, but this default is generally
	 low privacy and an alternative should be used in most cases
	 all identities imply a default edge. An identity must be added,
	 fake ones will be used to poll the network if none are present */

	// AddIdentity adds an identity to be tracked
	AddIdentity(identity Identity)
	// RemoveIdentity removes a currently tracked identity.
	RemoveIdentity(ephID ephemeral.Id)

	//fingerprints
	AddFingerprint(fingerprint format.Fingerprint, response FingerprintResponse)
	CheckFingerprint(fingerprint format.Fingerprint)bool
	RemoveFingerprint(fingerprint format.Fingerprint)bool


	/* trigger - predefined hash based tags appended to all cmix messages
	 which, though trial hashing, are used to determine if a message
	 is for a specific identity, can can contain metadata about
	 the sending party
	 These can be processed by the notifications system, or can be used to*/


	AddEdge(preimage EdgePreimage, identity *id.ID)
	RemoveEdge(preimage EdgePreimage, identity *id.ID) error


}

type EdgePreimage struct{
	Data   []byte
	Type   string
	Source []byte
}

type Identity struct {
	// Identity
	EphId       ephemeral.Id
	Source      *id.ID
	AddressSize uint8

	// Usage variables
	End         time.Time // Timestamp when active polling will stop
	ExtraChecks uint      // Number of extra checks executed as active after the
	// ID exits active

	// Polling parameters
	StartValid time.Time // Timestamp when the ephID begins being valid
	EndValid   time.Time // Timestamp when the ephID stops being valid

	// Makes the identity not store on disk
	Ephemeral bool
}

type FingerprintResponse func(format.Message)


type Ratchet interface {
	SendE2E(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
	SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error)
	AddPartner(partnerID *id.ID, partnerPubKey,
		myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
		mySIDHPrivKey *sidh.PrivateKey,
		sendParams, receiveParams params.E2ESessionParams)
	GetPartner(partnerID *id.ID) (*Manager, error)
	DeletePartner(partnerId *id.ID)
	GetAllPartnerIDs() []*id.ID


}

//for use in key exchange which needs to be callable inside of network
type SendE2E func(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
