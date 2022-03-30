///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/xx_network/primitives/id"
)

// public switchboard interface which only allows registration and does not
// allow speaking messages
type Switchboard interface {
	// Registers a new listener. Returns the ID of the new listener.
	// Keep this around if you want to be able to delete the listener later.
	//
	// name is used for debug printing and not checked for uniqueness
	//
	// user: 0 for all, or any user ID to listen for messages from a particular
	// user. 0 can be id.ZeroUser or id.ZeroID
	// messageType: 0 for all, or any message type to listen for messages of
	// that type. 0 can be Receive.AnyType
	// newListener: something implementing the Listener interface. Do not
	// pass nil to this.
	//
	// If a message matches multiple listeners, all of them will hear the
	// message.
	RegisterListener(user *id.ID, messageType message.Type,
		newListener switchboard.Listener) switchboard.ListenerID

	// Registers a new listener built around the passed function.
	// Returns the ID of the new listener.
	// Keep this around if you want to be able to delete the listener later.
	//
	// name is used for debug printing and not checked for uniqueness
	//
	// user: 0 for all, or any user ID to listen for messages from a particular
	// user. 0 can be id.ZeroUser or id.ZeroID
	// messageType: 0 for all, or any message type to listen for messages of
	// that type. 0 can be Receive.AnyType
	// newListener: a function implementing the ListenerFunc function type.
	// Do not pass nil to this.
	//
	// If a message matches multiple listeners, all of them will hear the
	// message.
	RegisterFunc(name string, user *id.ID, messageType message.Type,
		newListener switchboard.ListenerFunc) switchboard.ListenerID

	// Registers a new listener built around the passed channel.
	// Returns the ID of the new listener.
	// Keep this around if you want to be able to delete the listener later.
	//
	// name is used for debug printing and not checked for uniqueness
	//
	// user: 0 for all, or any user ID to listen for messages from a particular
	// user. 0 can be id.ZeroUser or id.ZeroID
	// messageType: 0 for all, or any message type to listen for messages of
	// that type. 0 can be Receive.AnyType
	// newListener: an item channel.
	// Do not pass nil to this.
	//
	// If a message matches multiple listeners, all of them will hear the
	// message.
	RegisterChannel(name string, user *id.ID, messageType message.Type,
		newListener chan message.Receive) switchboard.ListenerID

	// Unregister removes the listener with the specified ID so it will no
	// longer get called
	Unregister(listenerID switchboard.ListenerID)
}
