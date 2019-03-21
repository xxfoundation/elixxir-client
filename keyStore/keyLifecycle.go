package keyStore

import (
	"errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"sync/atomic"
)

const (
	UNINITIALISED uint32 = iota
	KEYING
	READY
	USED
)

var IncorrectState = errors.New("operation could not occur on KeyLifecycle due to incorrect state")
var NoKeys = errors.New("no keys available")

// The keylifecycle is part of a larger system which keeps track of e2e keys.  Within e2e a negotiation occurs which
// creates a set of keys for sending and receiving which are held in separate storage maps, the first keyed on userID
// and the second based upon a key fingerprint. All usages will connect back to this structure to ensure thread safety
// and denote usage.
type KeyLifecycle struct {
	// Underlying key
	privateKey *cyclic.Int

	// Designates end-to-end partner
	partner *id.User

	// Usage counter that is incremented every time a key from the set is used
	count uint32

	// Value of the counter at which a rekey is triggered
	ttl uint16

	// Designates if the partner has successfully used a key from this set,
	// indicating it has been accepted and can be used for emergency keying.
	// This confirmation will either come over the return path during a rekey or
	// in the form of a message successfully decrypted which was sent using a
	// key from the set.
	// 0 = uninitialised
	// 1 = keying
	// 2 = ready
	// 3 = used
	state *uint32

	sync.Mutex
}

// Sets up a KeyLifecycle in KEYING mode to enable the process of a key
// negotiation.
func GenerateKeyLifecycle(privateKey *cyclic.Int, partner *id.User) *KeyLifecycle {
	state := new(uint32)
	*state = KEYING
	kl := KeyLifecycle{
		privateKey: privateKey,
		partner:    partner,
		count:      0,
		state:      state,
	}

	return &kl
}

// Post a key negotiation, initialises a key set to be used.
func (kl *KeyLifecycle) Initialise(ttl uint16, sendKeys, sendReKeys []E2EKey) (*SendKeyset, error) {
	kl.Lock()

	// Ensure that initialise has not been called previously
	if atomic.LoadUint32(kl.state) != KEYING {
		kl.Unlock()
		return nil, IncorrectState
	}

	// Clear private key that is no longer needed
	kl.privateKey = nil

	// Set time to live (number of uses before rekey)
	kl.ttl = ttl

	sks := SendKeyset{
		NewLIFO(),
		NewLIFO(),
		kl,
	}

	// Load sendKeys slice into the sendKeys stack
	for i := 0; i < len(sendKeys); i++ {
		sks.sendKeys.Push(&sendKeys[i])
	}

	// Load sendReKeys slice into the sendReKeys stack
	for i := 0; i < len(sendReKeys); i++ {
		sks.sendReKeys.Push(&sendReKeys[i])
	}

	// Once initialisation is complete, set the state to READY
	success := atomic.CompareAndSwapUint32(kl.state, KEYING, READY)

	kl.Unlock()

	if !success {
		panic("unsafe access to key lifecycle occurred")
	}

	return &sks, nil
}

func (kl *KeyLifecycle) incrementCount() bool {
	rekey := false

	// Increase key usage counter
	kl.count++

	// If the count reaches the ttl limit, then trigger a rekey
	if kl.count == uint32(kl.ttl) {
		rekey = true
		success := atomic.CompareAndSwapUint32(kl.state, READY, USED)
		if !success {
			panic("unsafe access to key lifecycle occurred")
		}
	}
	return rekey
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

// Returns a copy of the private key.
func (kl *KeyLifecycle) CopyPrivateKey() *cyclic.Int {
	return cyclic.NewIntFromBytes(kl.privateKey.Bytes())
}

// Returns the count of the KeyLifecycle.
func (kl *KeyLifecycle) GetCount() uint32 {
	return kl.count
}

// Returns the state of the KeyLifecycle.
func (kl *KeyLifecycle) GetState() uint32 {
	return *kl.state
}

// Increments the counter by 1.
func (kl *KeyLifecycle) IncrCount() {
	kl.Lock()
	kl.count++
	kl.Unlock()
}
