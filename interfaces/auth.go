package interfaces

import (
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/xx_network/primitives/id"
)

type Auth interface {
	// Adds a general callback to be used on auth requests. This will be preempted
	// by any specific callback
	AddGeneralRequestCallback(cb auth.RequestCallback)
	// Adds a general callback to be used on auth requests. This will not be
	// preempted by any specific callback. It is recommended that the specific
	// callbacks are used, this is primarily for debugging.
	AddOverrideRequestCallback(cb auth.RequestCallback)
	// Adds a specific callback to be used on auth requests. This will preempt a
	// general callback, meaning the request will be heard on this callback and not
	// the general. Request will still be heard on override callbacks.
	AddSpecificRequestCallback(id *id.ID, cb auth.RequestCallback)
	// Removes a specific callback to be used on auth requests.
	RemoveSpecificRequestCallback(id *id.ID)
	// Adds a general callback to be used on auth confirms. This will be preempted
	// by any specific callback
	AddGeneralConfirmCallback(cb auth.ConfirmCallback)
	// Adds a general callback to be used on auth confirms. This will not be
	// preempted by any specific callback. It is recommended that the specific
	// callbacks are used, this is primarily for debugging.
	AddOverrideConfirmCallback(cb auth.ConfirmCallback)
	// Adds a specific callback to be used on auth confirms. This will preempt a
	// general callback, meaning the request will be heard on this callback and not
	// the general. Request will still be heard on override callbacks.
	AddSpecificConfirmCallback(id *id.ID, cb auth.ConfirmCallback)
	// Removes a specific callback to be used on auth confirm.
	RemoveSpecificConfirmCallback(id *id.ID)
}
