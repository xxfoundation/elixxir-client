package keyStore

import (
	"github.com/golang-collections/collections/stack"
	"sync/atomic"
)

//Stores all the keys from a single key negotiation.  Will be stored an a map keyed on user id.  To send a message,
//the user to send to will be looked up and then a send key will be popped from the negotiation.
type SendKeyset struct {
	// List of Keys used for sending. When a key is used it is deleted.
	sendKeys *stack.Stack

	// List of ReKey Keys that can be sent. When a key is used it is deleted.
	sendReKeys *stack.Stack

	// pointer to controling lifecycle
	lifecycle *KeyLifecycle
}

// Returns the first key on the stack. Returns true if it is time to rekey.
func (sks *SendKeyset) PopSendKey() (*E2EKey, bool, error) {
	sks.lifecycle.Lock()

	// Check that the KeyLifecycle is in the READY state
	if atomic.LoadUint32(sks.lifecycle.state) != READY {
		sks.lifecycle.Unlock()
		return nil, false, IncorrectState
	}

	var err error
	var key *E2EKey

	// Get the key
	keyFace := sks.sendKeys.Pop()

	// Check if the key exists
	if keyFace == nil {
		err = NoKeys
	} else {
		key = keyFace.(*E2EKey)
	}

	rekey := sks.lifecycle.incrementCount()

	sks.lifecycle.Unlock()

	return key, rekey, err
}

// Returns the first key for rekeying on the stack.
// if it returns a NoKeys error then it is time to delete the lifecycle
func (sks *SendKeyset) PopSendReKey() (*E2EKey, error) {
	sks.lifecycle.Lock()
	state := atomic.LoadUint32(sks.lifecycle.state)
	//FIXME: this was only checking ready before, the test need to validate this
	// Check that the KeyLifecycle is in the READY state or the USED state
	if state != READY && state != USED {
		sks.lifecycle.Unlock()
		return nil, IncorrectState
	}

	var err error
	var key *E2EKey

	// Get the key
	keyFace := sks.sendReKeys.Pop()

	// Check if the key exists
	if keyFace == nil {
		err = NoKeys
	} else {
		key = keyFace.(*E2EKey)
	}

	sks.lifecycle.Unlock()

	return key, err
}
