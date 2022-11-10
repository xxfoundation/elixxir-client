////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receive

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/catalog"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"strings"
)

//Listener interface for a listener adhere to
type Listener interface {
	// the Hear function is called to exercise the listener, passing in the
	// data as an item
	Hear(item Message)
	// Returns a name, used for debugging
	Name() string
}

// ListenerFunc This function type defines callbacks that get passed
// when the listener is listened to. It will always be called in its
// own goroutine. It may be called multiple times simultaneously
type ListenerFunc func(item Message)

// ListenerID id object returned when a listener is created and is
// used to delete it from the system
type ListenerID struct {
	userID      *id.ID
	messageType catalog.MessageType
	listener    Listener
}

// GetUserID getter for userID
func (lid ListenerID) GetUserID() *id.ID {
	return lid.userID
}

// GetMessageType getter for message type
func (lid ListenerID) GetMessageType() catalog.MessageType {
	return lid.messageType
}

// GetName getter for name
func (lid ListenerID) GetName() string {
	return lid.listener.Name()
}

// String returns the values in the ListenerID in a human-readable format. This
// functions adheres to the fmt.Stringer interface.
func (lid ListenerID) String() string {
	str := []string{
		"userID:" + lid.userID.String(),
		"messageType:" + lid.messageType.String() +
			"(" + strconv.FormatUint(uint64(lid.messageType), 10) + ")",
		"listener:" + lid.listener.Name(),
	}

	return "{" + strings.Join(str, " ") + "}"
}

/*internal listener implementations*/

//listener based off of a function
type funcListener struct {
	listener ListenerFunc
	name     string
}

//  newFuncListener creates a new FuncListener Adhereing to the listener interface out of the
// passed function and name, returns a pointer to the result
func newFuncListener(listener ListenerFunc, name string) *funcListener {
	return &funcListener{
		listener: listener,
		name:     name,
	}
}

// Adheres to the Hear function of the listener interface, calls the internal
// function with the passed item
func (fl *funcListener) Hear(item Message) {
	fl.listener(item)
}

// Adheres to the Name function of the listener interface, returns a name.
// used for debugging
func (fl *funcListener) Name() string {
	return fl.name
}

//listener based off of a channel
type chanListener struct {
	listener chan Message
	name     string
}

// creates a new ChanListener Adhereing to the listener interface out of the
// passed channel and name, returns a pointer to the result
func newChanListener(listener chan Message, name string) *chanListener {
	return &chanListener{
		listener: listener,
		name:     name,
	}
}

// Adheres to the Hear function of the listener interface, calls the passed the
// heard item across the channel.  Drops the item if it cannot put it into the
// channel immediately
func (cl *chanListener) Hear(item Message) {
	select {
	case cl.listener <- item:
	default:
		jww.WARN.Printf("Switchboard failed to speak on channel "+
			"listener %s", cl.name)
	}
}

// Adheres to the Name function of the listener interface, returns a name.
// used for debugging
func (cl *chanListener) Name() string {
	return cl.name
}
