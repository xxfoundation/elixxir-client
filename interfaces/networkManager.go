///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

type NetworkManager interface {
	// The stoppable can be nil.
	SendE2E(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
	SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error)
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
}

//for use in key exchange which needs to be callable inside of network
type SendE2E func(m message.Send, p params.E2E, stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error)
