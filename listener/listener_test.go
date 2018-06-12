package listener

import (
	"testing"
	"sync"
	"gitlab.com/privategrity/client/globals"
)

type MockListener struct {
	NumHeard int
	mux sync.Mutex
}

func (ml *MockListener) Hear(message []byte) {
	ml.mux.Lock()
	ml.NumHeard++
	defer ml.mux.Unlock()
}

func TestListenerMap_Speak(t *testing.T) {
	var listeners *ListenerMap
	// initialize listener map
	listeners = NewListenerMap()
	// add all three tiers of listeners to map
	fullyMatchedListener := &MockListener{}
	var specificUserID globals.UserID
	specificUserID = "5" // can be arbitrary bytes in golang
	// TODO different type for message types?
	var specificMessageType int
	specificMessageType = 8
	listeners.Listen(specificUserID, specificMessageType,
		fullyMatchedListener, false);
	// speak
}
