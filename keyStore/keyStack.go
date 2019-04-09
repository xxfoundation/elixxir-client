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

func NewKeyStack() *KeyStack {
	ks := new(KeyStack)
	ks.keys = stack.New()
	return ks
}

func (ks *KeyStack) Push(key *E2EKey) {
	ks.keys.Push(key)
}

// Returns the first key on the stack, and the key action from the Key Manager
func (ks *KeyStack) Pop() *E2EKey {
	var key *E2EKey

	// Get the key
	ks.Lock()
	keyFace := ks.keys.Pop()
	ks.Unlock()

	// Check if the key exists and panic otherwise
	if keyFace == nil {
		globals.Log.FATAL.Panicf("E2E key stack is empty!")
	} else {
		key = keyFace.(*E2EKey)
	}

	return key
}

// Deletes all keys from stack, i.e., pops all
func (ks *KeyStack) Delete() {
	ks.Lock()
	defer ks.Unlock()
	size := ks.keys.Len()
	for i := 0; i < size; i++ {
		ks.keys.Pop()
	}
}
