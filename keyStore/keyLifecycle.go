package keyStore

import (
	"errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

const (
	UNINITIALISED uint32 = iota
	KEYING
	READY
	USED
)

var IncorrectState = errors.New("operation could not occur on KeyLifecycle due to incorrect state")
var NoKeys = errors.New("no keys available")

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
	state uint32

	// List of Keys used for sending. When a key is used it is deleted.
	sendKeys LIFO

	// List of ReKey Keys that can be sent. When a key is used it is deleted.
	sendReKeys LIFO

	sync.Mutex
}

// Sets up a KeyLifecycle in KEYING mode to enable the process of a key
// negotiation.
func GenerateKeyLifecycle(privateKey *cyclic.Int, partner *id.User) *KeyLifecycle {
	kl := KeyLifecycle{
		privateKey: privateKey,
		partner:    partner,
		count:      0,
		state:      KEYING,
		sendKeys:   *NewLIFO(),
		sendReKeys: *NewLIFO(),
	}

	return &kl
}

// Post a key negotiation, initialises a key set to be used.
func (kl *KeyLifecycle) Initialise(ttl uint16, sendKeys, sendReKeys []E2EKey) error {
	kl.Lock()

	// Ensure that initialise has not been called previously
	if kl.state != KEYING {
		kl.Unlock()
		return IncorrectState
	}

	// Clear private key that is no longer needed
	kl.privateKey = nil

	// Set time to live (number of uses before rekey)
	kl.ttl = ttl

	// Load sendKeys slice into the sendKeys stack
	for i := 0; i < len(sendKeys); i++ {
		kl.sendKeys.Push(&sendKeys[i])
	}

	// Load sendReKeys slice into the sendReKeys stack
	for i := 0; i < len(sendReKeys); i++ {
		kl.sendReKeys.Push(&sendReKeys[i])
	}

	// Once initialisation is complete, set the state to READY
	kl.state = READY

	kl.Unlock()

	return nil
}

// Returns the first key on the stack. Returns true if it is time to rekey.
func (kl *KeyLifecycle) PopSendKey() (*E2EKey, bool, error) {
	// Check that the KeyLifecycle is in the READY state
	if kl.state != READY {
		return nil, false, IncorrectState
	}

	kl.Lock()
	var err error
	var key *E2EKey
	rekey := false

	// Get the key
	keyFace := kl.sendKeys.Pop()

	// Check if the key exists
	if keyFace == nil {
		err = NoKeys
	} else {
		key = keyFace.(*E2EKey)
	}

	// Increase key usage counter
	kl.count++

	// If the count reaches the ttl limit, then trigger a rekey
	if kl.count == uint32(kl.ttl) {
		rekey = true
		kl.state = USED
	}

	kl.Unlock()

	return key, rekey, err
}

// Returns the first key for rekeying on the stack.
func (kl *KeyLifecycle) PopSendReKey() (*E2EKey, error) {
	// Check that the KeyLifecycle is in the READY state
	if kl.state != READY {
		return nil, IncorrectState
	}

	kl.Lock()

	var err error
	var key *E2EKey

	// Get the key
	keyFace := kl.sendReKeys.Pop()

	// Check if the key exists
	if keyFace == nil {
		err = NoKeys
	} else {
		key = keyFace.(*E2EKey)
	}

	// Increase key usage counter
	kl.count++

	kl.Unlock()

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
	return kl.state
}

// Increments the counter by 1.
func (kl *KeyLifecycle) IncrCount() {
	kl.Lock()
	kl.count++
	kl.Unlock()
}
