package keyStore

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"sync/atomic"
)

type KeyAction uint8

const (
	None KeyAction = iota
	Rekey
	Purge
	Deleted
)

// Hardcoded limits for keys
// With 16 receiving states we can hold
// 16*64=1024 dirty bits for receiving keys
// This includes keys+rekeys
// With that limit, setting maxKeys to 800
// and ttlScalar to 1.2, we can generate
// a maximum amount of 960 receiving keys
// This leaves space for 1024-960 = 64 rekeys
const (
	maxStates uint16  = 16
	minKeys   uint16  = 500
	maxKeys   uint16  = 800
	ttlScalar float64 = 1.1 // generate 10% extra keys
	threshold uint16  = 160
	numReKeys uint16  = 64
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
	state *uint64

	// Value of the counter at which a rekey is triggered
	ttl uint16

	// Total number of keys
	numKeys uint32
	// Total number of rekey keys
	numReKeys uint16

	// Received keys dirty bits
	recvState [maxStates]*uint64

	// SendKeys Stack
	sendKeys *KeyStack
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
	numKeys uint32, ttl uint16, numReKeys uint16) *KeyManager {

	km := new(KeyManager)
	km.baseKey = baseKey
	km.partner = partner
	km.state = new(uint64)
	km.ttl = ttl
	km.numKeys = numKeys
	km.numReKeys = numReKeys
	for i, _ := range km.recvState {
		km.recvState[i] = new(uint64)
	}
	return km
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

// Check if state ReKey Counter >= NumReKeys
// Return true if so, which should trigger a purge
func stateReKeyCmp(state uint64, nKeys uint16) bool {
	reKeyCounter := uint16((state & stateReKeyMask) >> stateReKeyShift)
	if reKeyCounter >= nKeys {
		return true
	}
	return false
}

// The UpdateState atomically updates internal state
// of key manager
func (km *KeyManager) UpdateState(rekey bool) KeyAction {
	var stateIncr uint64
	if rekey {
		stateIncr = stateReKeyIncr
	} else {
		stateIncr = stateKeyIncr
	}

	result := atomic.AddUint64(km.state, stateIncr)

	if result&stateDeleteMask != 0 {
		return Deleted
	}

	if rekey && stateReKeyCmp(result, km.numReKeys) {
		// set delete bit
		atomic.AddUint64(km.state, stateDeleteIncr)
		return Purge
	} else if !rekey && stateKeyCmp(result, km.ttl) {
		return Rekey
	}
	return None
}

// The UpdateRecvState atomically updates internal
// receiving state of key manager
func (km *KeyManager) UpdateRecvState(keyID uint32) {
	stateIdx := keyID / 64
	stateBit := uint64(1 << (keyID % 64))

	atomic.AddUint64(km.recvState[stateIdx], stateBit)
}

// The destroy function will hold the lock
func (km *KeyManager) Destroy() {
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
