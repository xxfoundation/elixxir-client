package keyStore

import (
	"github.com/golang-collections/collections/stack"
	"gitlab.com/elixxir/client/globals"
	"sync"
)

// KeyStack contains the stack of E2E keys (or rekeys)
// Has a mutex for access control
type KeyStack struct {
	// List of Keys used for sending. When a key is used it is deleted.
	keys *stack.Stack
	// Lock
	sync.Mutex
}

func (ks *KeyStack) Push(key *E2EKey) {
	ks.keys.Push(key)
}

// Returns the first key on the stack, and the key action from the Key Manager
// NOTE: Caller is responsible for locking the stack
func (ks *KeyStack) Pop() (*E2EKey) {
	var key *E2EKey

	// Get the key
	keyFace := ks.keys.Pop()

	// Check if the key exists and panic otherwise
	if keyFace == nil {
		globals.Log.FATAL.Panicf("E2E sendKey stack is empty!")
	} else {
		key = keyFace.(*E2EKey)
	}

	return key
}

// Deletes all keys from stack, i.e., pops all
// Internally holds lock
func (ks *KeyStack) Delete() {
	ks.Lock()
	defer ks.Unlock()
	for i := 0; i < ks.keys.Len(); i++ {
		ks.keys.Pop()
	}
}
