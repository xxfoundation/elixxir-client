////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receive

import (
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// verify func listener adheres to the listener interface
var _ Listener = &funcListener{}

// verify chan listener adheres to the listener interface
var _ Listener = &chanListener{}

// test listenerID returns the userID
func TestListenerID_GetUserID(t *testing.T) {
	lid := ListenerID{
		userID:      id.NewIdFromUInt(42, id.User, t),
		messageType: 42,
		listener:    nil,
	}

	if !lid.GetUserID().Cmp(lid.userID) {
		t.Errorf("Returned userID does not match")
	}
}

// test listenerID returns the messageType
func TestListenerID_GetMessageType(t *testing.T) {
	lid := ListenerID{
		userID:      id.NewIdFromUInt(42, id.User, t),
		messageType: 42,
		listener:    nil,
	}

	if lid.GetMessageType() != lid.messageType {
		t.Errorf("Returned message type does not match")
	}
}

// test listenerID returns the name
func TestListenerID_GetName(t *testing.T) {
	name := "test"

	lid := ListenerID{
		userID:      id.NewIdFromUInt(42, id.User, t),
		messageType: 42,
		listener:    newFuncListener(nil, name),
	}

	if lid.GetName() != name {
		t.Errorf("Returned name type does not match")
	}
}

// tests new function listener creates the funcListener properly
func TestNewFuncListener(t *testing.T) {
	f := func(item Message) {}
	name := "test"
	listener := newFuncListener(f, name)

	if listener.listener == nil {
		t.Errorf("function is wrong")
	}

	if listener.name != name {
		t.Errorf("name is wrong")
	}
}

// tests FuncListener Hear works
func TestFuncListener_Hear(t *testing.T) {
	m := Message{
		Payload:     []byte{0, 1, 2, 3},
		Sender:      id.NewIdFromUInt(42, id.User, t),
		MessageType: 69,
	}

	heard := make(chan Message, 1)

	f := func(item Message) {
		heard <- item
	}

	listener := newFuncListener(f, "test")

	listener.Hear(m)

	select {
	case item := <-heard:
		if !reflect.DeepEqual(item, m) {
			t.Errorf("Heard message did not match")
		}
	case <-time.After(25 * time.Millisecond):
		t.Errorf("Did not hear")
	}
}

// Test FuncListener returns the correct name
func TestFuncListener_Name(t *testing.T) {
	name := "test"
	listener := newFuncListener(nil, name)

	if listener.Name() != name {
		t.Errorf("Name did not match")
	}
}

// tests new chan listener creates the chanListener properly
func TestNewChanListener(t *testing.T) {
	c := make(chan Message)
	name := "test"
	listener := newChanListener(c, name)

	if listener.listener == nil {
		t.Errorf("function is wrong")
	}

	if listener.name != name {
		t.Errorf("name is wrong")
	}
}

// tests ChanListener Hear works
func TestChanListener_Hear(t *testing.T) {
	m := Message{
		Payload:     []byte{0, 1, 2, 3},
		Sender:      id.NewIdFromUInt(42, id.User, t),
		MessageType: 69,
	}

	heard := make(chan Message, 1)

	listener := newChanListener(heard, "test")

	listener.Hear(m)

	select {
	case item := <-heard:
		if !reflect.DeepEqual(item, m) {
			t.Errorf("Heard message did not match")
		}
	case <-time.After(25 * time.Millisecond):
		t.Errorf("Did not hear")
	}
}

// Test FuncListener returns the correct name
func TestChanListener_Name(t *testing.T) {
	name := "test"
	listener := newChanListener(nil, name)

	if listener.Name() != name {
		t.Errorf("Name did not match")
	}
}
