package keyStore

import (
	"github.com/golang-collections/collections/stack"
	"gitlab.com/elixxir/client/globals"
	"sync"
)

// KeyStack contains a stack of E2E keys (or rekeys)
// Also has a mutex for access control
type KeyStack struct {
	// List of Keys used for sending
	// When a key is used it is deleted (pop'ed)
	keys *stack.Stack
	// Lock
	sync.Mutex
}

// Create a new KeyStack
// It creates the internal stack.Stack object
func NewKeyStack() *KeyStack {
	ks := new(KeyStack)
	ks.keys = stack.New()
	return ks
}

// Push an E2EKey into the stack
func (ks *KeyStack) Push(key *E2EKey) {
	ks.keys.Push(key)
}

// Returns the top key on the stack
// Internally holds the lock when
// running Pop on the internal stack.Stack object
func (ks *KeyStack) Pop() *E2EKey {
	var key *E2EKey

	// Get the key
	ks.Lock()
	keyFace := ks.keys.Pop()
	ks.Unlock()

	// Check if the key exists and panic otherwise
	if keyFace == nil {
		globals.Log.WARN.Printf("E2E key stack is empty!")
		key = nil
	} else {
		key = keyFace.(*E2EKey)
	}

	return key
}

// Deletes all keys from stack, i.e., pops all
// Internally holds the lock
func (ks *KeyStack) Delete() {
	ks.Lock()
	defer ks.Unlock()
	length := ks.keys.Len()
	for i := 0; i < length; i++ {
		ks.keys.Pop()
	}
}
