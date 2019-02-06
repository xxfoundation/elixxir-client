////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package switchboard

import (
	"gitlab.com/elixxir/client/parse"
	"sync"
	"testing"
	"time"
	"gitlab.com/elixxir/client/cmixproto"
	"bytes"
	"gitlab.com/elixxir/primitives/userid"
)

type MockListener struct {
	NumHeard        int
	IsFallback      bool
	LastMessage     []byte
	LastMessageType cmixproto.Type
	mux             sync.Mutex
}

func (ml *MockListener) Hear(msg *parse.Message, isHeardElsewhere bool) {
	ml.mux.Lock()
	defer ml.mux.Unlock()

	if !isHeardElsewhere || !ml.IsFallback {
		ml.NumHeard++
		ml.LastMessage = msg.Body
		ml.LastMessageType = msg.Type
	}
}

var specificUserID = new(id.UserID).SetUints(&[4]uint64{0, 0, 0, 5})
var specificMessageType cmixproto.Type = 8
var delay = 10 * time.Millisecond

func OneListenerSetup() (*Switchboard, *MockListener) {
	var listeners *Switchboard
	listeners = NewSwitchboard()
	// add one listener to the map
	fullyMatchedListener := &MockListener{}
	// TODO different type for message types?
	listeners.Register(specificUserID, specificMessageType,
		fullyMatchedListener)
	return listeners, fullyMatchedListener
}

func TestListenerMap_SpeakOne(t *testing.T) {
	// set up
	listeners, fullyMatchedListener := OneListenerSetup()

	// speak
	listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: specificMessageType,
			Body: make([]byte, 0),
		},
		Sender:   specificUserID,
		Receiver: zeroUserID,
	})

	// determine whether the listener heard the message
	time.Sleep(delay)
	expected := 1
	if fullyMatchedListener.NumHeard != 1 {
		t.Errorf("The listener heard %v messages instead of %v",
			fullyMatchedListener.NumHeard, expected)
	}
}

func TestListenerMap_SpeakManyToOneListener(t *testing.T) {
	// set up
	listeners, fullyMatchedListener := OneListenerSetup()

	// speak
	for i := 0; i < 20; i++ {
		go listeners.Speak(&parse.Message{TypedBody: parse.TypedBody{
			Type: specificMessageType,
			Body: make([]byte, 0),
		},
			Sender: specificUserID,
			Receiver: zeroUserID})
	}

	// determine whether the listener heard the message
	time.Sleep(delay)
	expected := 20
	if fullyMatchedListener.NumHeard != expected {
		t.Errorf("The listener heard %v messages instead of %v",
			fullyMatchedListener.NumHeard, expected)
	}
}

func TestListenerMap_SpeakToAnother(t *testing.T) {
	// set up
	listeners, fullyMatchedListener := OneListenerSetup()

	// speak
	listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: specificMessageType,
			Body: make([]byte, 0),
		},
		Sender:   nonzeroUserID,
		Receiver: zeroUserID,
	})

	// determine whether the listener heard the message
	time.Sleep(delay)
	expected := 0
	if fullyMatchedListener.NumHeard != expected {
		t.Errorf("The listener heard %v messages instead of %v",
			fullyMatchedListener.NumHeard, expected)
	}
}

func TestListenerMap_SpeakDifferentType(t *testing.T) {
	// set up
	listeners, fullyMatchedListener := OneListenerSetup()

	// speak
	listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: specificMessageType + 1,
			Body: make([]byte, 0),
		},
		Sender:   specificUserID,
		Receiver: zeroUserID,
	})

	// determine whether the listener heard the message
	time.Sleep(delay)
	expected := 0
	if fullyMatchedListener.NumHeard != expected {
		t.Errorf("The listener heard %v messages instead of %v",
			fullyMatchedListener.NumHeard, expected)
	}
}

var zeroUserID = id.ZeroID
var nonzeroUserID = new(id.UserID).SetUints(&[4]uint64{0, 0, 0, 786})
var zeroType cmixproto.Type

func WildcardListenerSetup() (*Switchboard, *MockListener) {
	var listeners *Switchboard
	listeners = NewSwitchboard()
	// add one listener to the map
	wildcardListener := &MockListener{}
	// TODO different type for message types?
	listeners.Register(zeroUserID, zeroType,
		wildcardListener)
	return listeners, wildcardListener
}

func TestListenerMap_SpeakWildcard(t *testing.T) {
	// set up
	listeners, wildcardListener := WildcardListenerSetup()

	// speak
	listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: specificMessageType + 1,
			Body: make([]byte, 0),
		},
		Sender:   specificUserID,
		Receiver: zeroUserID,
	})

	// determine whether the listener heard the message
	time.Sleep(delay)
	expected := 1
	if wildcardListener.NumHeard != expected {
		t.Errorf("The listener heard %v messages instead of %v",
			wildcardListener.NumHeard, expected)
	}
}

func TestListenerMap_SpeakManyToMany(t *testing.T) {
	listeners := NewSwitchboard()

	individualListeners := make([]*MockListener, 0)

	// one user, many types
	for messageType := cmixproto.Type(1); messageType <= cmixproto.
		Type(20); messageType++ {
		newListener := MockListener{}
		listeners.Register(specificUserID, messageType,
			&newListener)
		individualListeners = append(individualListeners, &newListener)
	}
	// wildcard listener for the user
	userListener := &MockListener{}
	listeners.Register(specificUserID, zeroType, userListener)
	// wildcard listener for all messages
	wildcardListener := &MockListener{}
	listeners.Register(zeroUserID, zeroType, wildcardListener)

	// send to all types for our user
	for messageType := cmixproto.Type(1); messageType <= cmixproto.Type(20);
	messageType++ {
		go listeners.Speak(&parse.Message{
			TypedBody: parse.TypedBody{
				Type: messageType,
				Body: make([]byte, 0),
			},
			Sender:   specificUserID,
			Receiver: zeroUserID,
		})
	}
	// send to all types for a different user
	otherUser := id.NewUserIDFromUint(98, t)
	for messageType := cmixproto.Type(1); messageType <= cmixproto.Type(
		20); messageType++ {
		go listeners.Speak(&parse.Message{
			TypedBody: parse.TypedBody{
				Type: messageType,
				Body: make([]byte, 0),
			},
			Sender:   otherUser,
			Receiver: nonzeroUserID,
		})
	}

	time.Sleep(delay)

	expectedIndividuals := 1
	expectedUserWildcard := 20
	expectedAllWildcard := 40
	for i := 0; i < len(individualListeners); i++ {
		if individualListeners[i].NumHeard != expectedIndividuals {
			t.Errorf("Individual listener got %v messages, "+
				"expected %v messages", individualListeners[i].NumHeard, expectedIndividuals)
		}
	}
	if userListener.NumHeard != expectedUserWildcard {
		t.Errorf("User wildcard got %v messages, expected %v message",
			userListener.NumHeard, expectedUserWildcard)
	}
	if wildcardListener.NumHeard != expectedAllWildcard {
		t.Errorf("User wildcard got %v messages, expected %v message",
			wildcardListener.NumHeard, expectedAllWildcard)
	}
}

func TestListenerMap_SpeakFallback(t *testing.T) {
	var listeners *Switchboard
	listeners = NewSwitchboard()
	// add one normal and one fallback listener to the map
	fallbackListener := &MockListener{}
	fallbackListener.IsFallback = true
	listeners.Register(zeroUserID, zeroType,
		fallbackListener)
	specificListener := &MockListener{}
	listeners.Register(specificUserID, specificMessageType, specificListener)

	// send exactly one message to each of them
	listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: specificMessageType,
			Body: make([]byte, 0),
		},
		Sender:   specificUserID,
		Receiver: nonzeroUserID,
	})
	listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: specificMessageType + 1,
			Body: make([]byte, 0),
		},
		Sender:   specificUserID,
		Receiver: nonzeroUserID,
	})

	time.Sleep(delay)

	expected := 1

	if specificListener.NumHeard != expected {
		t.Errorf("Specific listener: Expected %v, got %v messages", expected,
			specificListener.NumHeard)
	}
	if fallbackListener.NumHeard != expected {
		t.Errorf("Fallback listener: Expected %v, got %v messages", expected,
			specificListener.NumHeard)
	}
}

func TestListenerMap_SpeakBody(t *testing.T) {
	listeners, listener := OneListenerSetup()
	expected := []byte{0x01, 0x02, 0x03, 0x04}
	listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: specificMessageType,
			Body: expected,
		},
		Sender:   specificUserID,
		Receiver: nonzeroUserID,
	})
	time.Sleep(delay)
	if !bytes.Equal(listener.LastMessage, expected) {
		t.Errorf("Received message was %v, expected %v",
			listener.LastMessage, expected)
	}
	if listener.LastMessageType != specificMessageType {
		t.Errorf("Received message type was %v, expected %v",
			listener.LastMessageType, specificMessageType)
	}
}

func TestListenerMap_Unregister(t *testing.T) {
	listeners := NewSwitchboard()
	listenerID := listeners.Register(specificUserID, specificMessageType,
		&MockListener{})
	listeners.Unregister(listenerID)
	if len(listeners.listeners[*specificUserID][specificMessageType]) != 0 {
		t.Error("The listener was still in the map after we stopped" +
			" listening on it")
	}
}
