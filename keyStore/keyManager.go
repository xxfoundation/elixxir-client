package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

type KeyAction uint8

const (
	None KeyAction = iota
	Rekey
	Purge
	Deleted
)

// The keylifecycle is part of a larger system which keeps track of e2e keys.  Within e2e a negotiation occurs which
// creates a set of keys for sending and receiving which are held in separate storage maps, the first keyed on userID
// and the second based upon a key fingerprint. All usages will connect back to this structure to ensure thread safety
// and denote usage.
type KeyManager struct {
	// Underlying key
	baseKey *cyclic.Int

	// Designates end-to-end partner
	partner *id.User

	// State and usage counter, formatted as follows:
	//                      Bits
	// |    63   | 62 - 56 |   55 - 40   | 39 - 32 |  31 - 0   |
	// | deleted |  empty  | rekey count |  empty  | key count |
	// |  1 bit  |  7 bits |   16 bits   |  8 bits |   32 bits |
	state uint64

	// Value of the counter at which a rekey is triggered
	ttl uint16

	// Total number of keys
	numKeys   uint32
	// Total number of rekey keys
	numReKeys uint16

	// Deletion Lock
	sync.Mutex

	// SendKeys Stack
	sendKeys   *KeyStack
	// SendReKeys Stack
	sendReKeys *KeyStack
	// Receive keys list
	receiveKeysFP []format.Fingerprint
	// Receive reKeys list
	receiveReKeysFP []format.Fingerprint
}

// Creates a new KeyManager to manage E2E Keys between user and partner
// Receives the group, baseKey, partner ID, TTL Parameters for Key and numReKeys
// min, max and params are used to determine the number of keys to generate, and
// the TTL value which triggers a rekey
// The numReKey will be used to generate reKeys that can be used to send reKey messages
func NewKeyManager(baseKey *cyclic.Int, partner *id.User,
	numKeys uint32, numReKeys uint16) *KeyManager {

	return &KeyManager{
		baseKey: baseKey,
		partner: partner,
		ttl: numReKeys,
		numKeys: numKeys,
		numReKeys: numReKeys,
	}
}

const (
	stateDeleteMask uint64 = 0x8000000000000000
	stateKeyMask    uint64 = 0x00000000FFFFFFFF
	stateReKeyMask  uint64 = 0x00FFFF0000000000
	stateReKeyShift uint64 = 40
	stateDeleteIncr uint64 = 1 << 63
	stateKeyIncr    uint64 = 1
	stateReKeyIncr  uint64 = 1 << stateReKeyShift
)

// Check if state Key Counter >= ttl
// Return true if so, which should trigger a rekey
func stateKeyCmp(state uint64, ttl uint16) bool {
	keyCounter := uint32(state & stateKeyMask)
	if keyCounter >= uint32(ttl) {
		return true
	}
	return false
}

// Check if state ReKey Counter >= ttl
// Return true if so, which should trigger a purge
func stateReKeyCmp(state uint64, ttl uint16) bool {
	reKeyCounter := uint16((state & stateReKeyMask) >> stateReKeyShift)
	if reKeyCounter >= ttl {
		return true
	}
	return false
}

// The UpdateState function will hold the lock for entire
// duration, instead of having an atomic state
func (km *KeyManager) UpdateState(rekey bool) KeyAction {
	km.Lock()
	defer km.Unlock()

	if km.state & stateDeleteMask != 0 {
		return Deleted
	}

	if rekey {
		km.state += stateReKeyIncr
		if stateReKeyCmp(km.state, km.numReKeys) {
			// set delete bit
			km.state += stateDeleteIncr
			return Purge
		}
	} else {
		km.state += stateKeyIncr
		if stateKeyCmp(km.state, km.ttl) {
			return Rekey
		}
	}
	return None
}

// The destroy function will hold the lock
func (km *KeyManager) Destroy() {
	km.Lock()
	defer km.Unlock()
	// Eliminate receiving keys
	ReceptionKeys.DeleteList(km.receiveKeysFP)
	ReceptionKeys.DeleteList(km.receiveReKeysFP)

	// Empty send keys and reKeys stacks
	// and remove them from maps
	TransmissionKeys.Delete(km.partner)
	TransmissionReKeys.Delete(km.partner)
	km.sendKeys.Delete()
	km.sendReKeys.Delete()

	// Hopefully when the function returns there
	// will be no keys referencing this Manager left,
	// so it will be garbage collected
}
