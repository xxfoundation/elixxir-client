package bindings

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/xx_network/primitives/id"
)

// Listener provides a callback to hear a message
// An object implementing this interface can be called back when the client
// gets a message of the type that the regi    sterer specified at registration
// time.
type Listener interface {
	// Hear is called to receive a message in the UI
	Hear(message Message)
	// Returns a name, used for debugging
	Name() string
}

// A callback when which is used to receive notification if network health
// changes
type NetworkHealthCallback interface {
	Callback(bool)
}

/*
// RoundEventHandler handles round events happening on the cMix network.
type RoundEventCallback interface {
	EventCallback(rid int, state byte, timedOut bool)
}*/

// Generic Unregister - a generic return used for all callbacks which can be
// unregistered
// Interface which allows the un-registration of a listener
type Unregister struct {
	f func()
}

//Call unregisters a callback
func (u Unregister) Unregister() {
	u.f()
}

//creates an unregister interface for listeners
func newListenerUnregister(lid switchboard.ListenerID, sw interfaces.Switchboard) Unregister {
	f := func() {
		sw.Unregister(lid)
	}
	return Unregister{f: f}
}

//creates an unregister interface for round events
func newRoundUnregister(rid id.Round, ec *dataStructures.EventCallback,
	re interfaces.RoundEvents) Unregister {
	f := func() {
		re.Remove(rid, ec)
	}
	return Unregister{f: f}
}
