///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package receive

import (
	"github.com/golang-collections/collections/set"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

type Switchboard struct {
	id          *byId
	messageType *byType

	mux sync.RWMutex
}

// New generates and returns a new switchboard object.
func New() *Switchboard {
	return &Switchboard{
		id:          newById(),
		messageType: newByType(),
	}
}

// Registers a new listener. Returns the ID of the new listener.
// Keep this around if you want to be able to delete the listener later.
//
// name is used for debug printing and not checked for uniqueness
//
// user: 0 for all, or any user ID to listen for messages from a particular
// user. 0 can be id.ZeroUser or id.ZeroID
// messageType: 0 for all, or any message type to listen for messages of that
// type. 0 can be Message.AnyType
// newListener: something implementing the Listener interface. Do not
// pass nil to this.
//
// If a message matches multiple listeners, all of them will hear the message.
func (sw *Switchboard) RegisterListener(user *id.ID,
	messageType catalog.MessageType, newListener Listener) ListenerID {

	// check the input data is valid
	if user == nil {
		jww.FATAL.Panicf("cannot register listener to nil user")
	}

	if newListener == nil {
		jww.FATAL.Panicf("cannot register nil listener")
	}

	//register the listener by both ID and messageType
	sw.mux.Lock()

	sw.id.Add(user, newListener)
	sw.messageType.Add(messageType, newListener)

	sw.mux.Unlock()

	//return a ListenerID so it can be unregistered in the future
	return ListenerID{
		userID:      user,
		messageType: messageType,
		listener:    newListener,
	}
}

// Registers a new listener built around the passed function.
// Returns the ID of the new listener.
// Keep this around if you want to be able to delete the listener later.
//
// name is used for debug printing and not checked for uniqueness
//
// user: 0 for all, or any user ID to listen for messages from a particular
// user. 0 can be id.ZeroUser or id.ZeroID
// messageType: 0 for all, or any message type to listen for messages of that
// type. 0 can be Message.AnyType
// newListener: a function implementing the ListenerFunc function type.
// Do not pass nil to this.
//
// If a message matches multiple listeners, all of them will hear the message.
func (sw *Switchboard) RegisterFunc(name string, user *id.ID,
	messageType catalog.MessageType, newListener ListenerFunc) ListenerID {
	// check that the input data is valid
	if newListener == nil {
		jww.FATAL.Panicf("cannot register function listener '%s' "+
			"with nil func", name)
	}

	// generate a funcListener object adhering to the listener interface
	fl := newFuncListener(newListener, name)

	//register the listener and return the result
	return sw.RegisterListener(user, messageType, fl)
}

// Registers a new listener built around the passed channel.
// Returns the ID of the new listener.
// Keep this around if you want to be able to delete the listener later.
//
// name is used for debug printing and not checked for uniqueness
//
// user: 0 for all, or any user ID to listen for messages from a particular
// user. 0 can be id.ZeroUser or id.ZeroID
// messageType: 0 for all, or any message type to listen for messages of that
// type. 0 can be Message.AnyType
// newListener: an item channel.
// Do not pass nil to this.
//
// If a message matches multiple listeners, all of them will hear the message.
func (sw *Switchboard) RegisterChannel(name string, user *id.ID,
	messageType catalog.MessageType, newListener chan Message) ListenerID {
	// check that the input data is valid
	if newListener == nil {
		jww.FATAL.Panicf("cannot register channel listener '%s' with"+
			" nil channel", name)
	}

	// generate a chanListener object adhering to the listener interface
	cl := newChanListener(newListener, name)

	//register the listener and return the result
	return sw.RegisterListener(user, messageType, cl)
}

// Speak broadcasts a message to the appropriate listeners.
// each is spoken to in their own goroutine
func (sw *Switchboard) Speak(item Message) {
	sw.mux.RLock()
	defer sw.mux.RUnlock()

	// Matching listeners: include those that match all criteria perfectly, as
	// well as those that do not care about certain criteria
	matches := sw.matchListeners(item)

	//Execute hear on all matched listeners in a new goroutine
	matches.Do(func(i interface{}) {
		r := i.(Listener)
		go r.Hear(item)
	})

	// print to log if nothing was heard
	if matches.Len() == 0 {
		jww.ERROR.Printf(
			"Message of type %v from user %q didn't match any listeners in"+
				" the map", item.MessageType, item.Sender)
	}
}

// Unregister removes the listener with the specified ID so it will no longer
// get called
func (sw *Switchboard) Unregister(listenerID ListenerID) {
	sw.mux.Lock()

	sw.id.Remove(listenerID.userID, listenerID.listener)
	sw.messageType.Remove(listenerID.messageType, listenerID.listener)

	sw.mux.Unlock()
}

// finds all listeners who match the items sender or ID, or have those fields
// as generic
func (sw *Switchboard) matchListeners(item Message) *set.Set {
	idSet := sw.id.Get(item.Sender)
	typeSet := sw.messageType.Get(item.MessageType)
	return idSet.Intersection(typeSet)
}
