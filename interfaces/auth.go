///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

type RequestCallback func(requestor contact.Contact)
type ConfirmCallback func(partner contact.Contact)
type ResetNotificationCallback func(partner contact.Contact)

type Auth interface {
	// Adds a general callback to be used on auth requests. This will be preempted
	// by any specific callback
	AddGeneralRequestCallback(cb RequestCallback)
	// Adds a general callback to be used on auth requests. This will not be
	// preempted by any specific callback. It is recommended that the specific
	// callbacks are used, this is primarily for debugging.
	AddOverrideRequestCallback(cb RequestCallback)
	// Adds a specific callback to be used on auth requests. This will preempt a
	// general callback, meaning the request will be heard on this callback and not
	// the general. Request will still be heard on override callbacks.
	AddSpecificRequestCallback(id *id.ID, cb RequestCallback)
	// Removes a specific callback to be used on auth requests.
	RemoveSpecificRequestCallback(id *id.ID)
	// Adds a general callback to be used on auth confirms. This will be preempted
	// by any specific callback
	AddGeneralConfirmCallback(cb ConfirmCallback)
	// Adds a general callback to be used on auth confirms. This will not be
	// preempted by any specific callback. It is recommended that the specific
	// callbacks are used, this is primarily for debugging.
	AddOverrideConfirmCallback(cb ConfirmCallback)
	// Adds a specific callback to be used on auth confirms. This will preempt a
	// general callback, meaning the request will be heard on this callback and not
	// the general. Request will still be heard on override callbacks.
	AddSpecificConfirmCallback(id *id.ID, cb ConfirmCallback)
	// Removes a specific callback to be used on auth confirm.
	RemoveSpecificConfirmCallback(id *id.ID)
	// Add a callback to receive session renegotiation notifications
	AddResetNotificationCallback(cb ResetNotificationCallback)
	//Replays all pending received requests over tha callbacks
	ReplayRequests()
}
