///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/xx_network/primitives/id"
)

// Listener provides a callback to hear a message
// An object implementing this interface can be called back when the client
// gets a message of the type that the registerer specified at registration
// time.
type Listener interface {
	// Hear is called to receive a message in the UI
	Hear(message *Message)
	// Returns a name, used for debugging
	Name() string
}

// A callback when which is used to receive notification if network health
// changes
type NetworkHealthCallback interface {
	Callback(bool)
}

// RoundEventCallback handles waiting on the exact state of a round on
// the cMix network.
type RoundEventCallback interface {
	EventCallback(rid, state int, timedOut bool)
}

// RoundCompletionCallback is returned when the completion of a round is known.
type RoundCompletionCallback interface {
	EventCallback(rid int, success, timedOut bool)
}

// MessageDeliveryCallback gets called on the determination if all events
// related to a message send were successful.
type MessageDeliveryCallback interface {
	EventCallback(msgID []byte, delivered, timedOut bool, roundResults []byte)
}

// AuthRequestCallback notifies the register whenever they receive an auth
// request
type AuthRequestCallback interface {
	Callback(requestor *Contact)
}

// AuthConfirmCallback notifies the register whenever they receive an auth
// request confirmation
type AuthConfirmCallback interface {
	Callback(partner *Contact)
}

// AuthRequestCallback notifies the register whenever they receive an auth
// request
type AuthResetNotificationCallback interface {
	Callback(requestor *Contact)
}

// Generic Unregister - a generic return used for all callbacks which can be
// unregistered
// Interface which allows the un-registration of a listener
type Unregister struct {
	f func()
}

//Call unregisters a callback
func (u *Unregister) Unregister() {
	u.f()
}

//creates an unregister interface for listeners
func newListenerUnregister(lid switchboard.ListenerID, sw interfaces.Switchboard) *Unregister {
	f := func() {
		sw.Unregister(lid)
	}
	return &Unregister{f: f}
}

//creates an unregister interface for round events
func newRoundUnregister(rid id.Round, ec *dataStructures.EventCallback,
	re interfaces.RoundEvents) *Unregister {
	f := func() {
		re.Remove(rid, ec)
	}
	return &Unregister{f: f}
}

//creates an unregister interface for round events
func newRoundListUnregister(rounds []id.Round, ec []*dataStructures.EventCallback,
	re interfaces.RoundEvents) *Unregister {
	f := func() {
		for i, r := range rounds {
			re.Remove(r, ec[i])
		}
	}
	return &Unregister{f: f}
}

type ClientError interface {
	Report(source, message, trace string)
}

type LogWriter interface {
	Log(string)
}

type writerAdapter struct {
	lw LogWriter
}

func (wa *writerAdapter) Write(p []byte) (n int, err error) {
	wa.lw.Log(string(p))
	return len(p), nil
}
