package listener

import (
	"testing"
	"sync"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/parse"
	"time"
	"bytes"
)

type MockListener struct {
	NumHeard    int
	LastMessage []byte
	LastMessageType int64
	mux         sync.Mutex
}

func (ml *MockListener) Hear(message []byte, messageType int64) {
	ml.mux.Lock()
	defer ml.mux.Unlock()

	ml.NumHeard++
	ml.LastMessage = message
	ml.LastMessageType = messageType
}

var specificUserID globals.UserID = "5"
var specificMessageType int64 = 8
var delay = 10 * time.Millisecond

func OneListenerSetup() (*ListenerMap, *MockListener) {
	var listeners *ListenerMap
	listeners = NewListenerMap()
	// add one listener to the map
	fullyMatchedListener := &MockListener{}
	// TODO different type for message types?
	listeners.Listen(specificUserID, specificMessageType,
		fullyMatchedListener, false)
	return listeners, fullyMatchedListener
}

func TestListenerMap_SpeakOne(t *testing.T) {
	// set up
	listeners, fullyMatchedListener := OneListenerSetup()

	// speak
	listeners.Speak(specificUserID, parse.TypedBody{specificMessageType,
		make([]byte, 0)})

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
		go listeners.Speak(specificUserID, parse.TypedBody{specificMessageType,
			make([]byte, 0)})
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
	otherUserID := globals.UserID(specificUserID[0] + 1)
	listeners.Speak(otherUserID,
		parse.TypedBody{specificMessageType,
			make([]byte, 0)})

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
	listeners.Speak(specificUserID,
		parse.TypedBody{specificMessageType + 1,
			make([]byte, 0)})

	// determine whether the listener heard the message
	time.Sleep(delay)
	expected := 0
	if fullyMatchedListener.NumHeard != expected {
		t.Errorf("The listener heard %v messages instead of %v",
			fullyMatchedListener.NumHeard, expected)
	}
}

var zeroUserID globals.UserID
var zeroType int64

func WildcardListenerSetup() (*ListenerMap, *MockListener) {
	var listeners *ListenerMap
	listeners = NewListenerMap()
	// add one listener to the map
	wildcardListener := &MockListener{}
	// TODO different type for message types?
	listeners.Listen(zeroUserID, zeroType,
		wildcardListener, false)
	return listeners, wildcardListener
}

func TestListenerMap_SpeakWildcard(t *testing.T) {
	// set up
	listeners, wildcardListener := WildcardListenerSetup()

	// speak
	listeners.Speak(specificUserID,
		parse.TypedBody{specificMessageType + 1,
			make([]byte, 0)})

	// determine whether the listener heard the message
	time.Sleep(delay)
	expected := 1
	if wildcardListener.NumHeard != expected {
		t.Errorf("The listener heard %v messages instead of %v",
			wildcardListener.NumHeard, expected)
	}
}

func TestListenerMap_SpeakManyToMany(t *testing.T) {
	listeners := NewListenerMap()

	individualListeners := make([]*MockListener, 0)

	// one user, many types
	for messageType := 1; messageType <= 20; messageType++ {
		newListener := MockListener{}
		listeners.Listen(specificUserID, int64(messageType),
			&newListener,
			false)
		individualListeners = append(individualListeners, &newListener)
	}
	// wildcard listener for the user
	userListener := &MockListener{}
	listeners.Listen(specificUserID, zeroType, userListener, false)
	// wildcard listener for all messages
	wildcardListener := &MockListener{}
	listeners.Listen(zeroUserID, zeroType, wildcardListener, false)

	// send to all types for our user
	for messageType := 1; messageType <= 20; messageType++ {
		go listeners.Speak(specificUserID, parse.TypedBody{
			BodyType: int64(messageType),
			Body:     make([]byte, 0),
		})
	}
	// send to all types for a different user
	otherUser := globals.UserID(specificUserID[0] + 1)
	for messageType := 1; messageType <= 20; messageType++ {
		go listeners.Speak(otherUser, parse.TypedBody{
			BodyType: int64(messageType),
			Body:     make([]byte, 0),
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
	var listeners *ListenerMap
	listeners = NewListenerMap()
	// add one normal and one fallback listener to the map
	fallbackListener := &MockListener{}
	listeners.Listen(zeroUserID, zeroType,
		fallbackListener, true)
	specificListener := &MockListener{}
	listeners.Listen(specificUserID, specificMessageType, specificListener,
		false)

	// send exactly one message to each of them
	listeners.Speak(specificUserID, parse.TypedBody{specificMessageType,
		make([]byte, 0)})
	listeners.Speak(specificUserID, parse.TypedBody{specificMessageType + 1,
		make([]byte, 0)})

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
	listeners.Speak(specificUserID, parse.TypedBody{specificMessageType,
		expected})
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
