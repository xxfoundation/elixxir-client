package keyStore

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
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
	MinKeys   uint16  = 500
	MaxKeys   uint16  = 800
	TTLScalar float64 = 1.1 // generate 10% extra keys
	Threshold uint16  = 160
	NumReKeys uint16  = 64
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

// Return true if bit specified by keyID is set, meaning
// that particular key has been used
func (km *KeyManager) checkRecvStateBit(keyID uint32) bool {
	stateIdx := keyID / 64
	stateBit := uint64(1 << (keyID % 64))
	if (*km.recvState[stateIdx] & stateBit) != 0 {
		return true
	}
	return false
}

func (km *KeyManager) GenerateKeys(grp *cyclic.Group, userID *id.User) {
	// Calculate how many unused send keys are needed
	usedSendKeys := uint32(*km.state & stateKeyMask)
	numGenSendKeys := uint(km.numKeys - usedSendKeys)
	usedSendReKeys := uint16((*km.state & stateReKeyMask) >> stateReKeyShift)
	numGenSendReKeys := uint(km.numReKeys - usedSendReKeys)

	// Generate numGenSendKeys send keys
	sendKeys := e2e.DeriveKeys(grp, km.baseKey, userID, numGenSendKeys)
	// Generate numGenSendReKeys send reKeys
	sendReKeys := e2e.DeriveEmergencyKeys(grp, km.baseKey, userID, numGenSendReKeys)

	// For receiving keys, generate all, and then only add to the map
	// the unused ones based on recvStates
	// Generate numKeys recv keys
	recvKeys := e2e.DeriveKeys(grp, km.baseKey, km.partner, uint(km.numKeys))
	// Generate numReKeys recv reKeys
	recvReKeys := e2e.DeriveEmergencyKeys(grp, km.baseKey, km.partner, uint(km.numReKeys))

	// Create Send Keys Stack on keyManager and
	// set on TransmissionKeys map
	km.sendKeys = NewKeyStack()
	TransmissionKeys.Store(km.partner, km.sendKeys)

	// Create send E2E Keys and add to stack
	for _, key := range sendKeys {
		e2ekey := new(E2EKey)
		e2ekey.key = key
		e2ekey.manager = km
		e2ekey.outer = format.E2E
		km.sendKeys.Push(e2ekey)
	}

	// Create Send ReKeys Stack on keyManager and
	// set on TransmissionReKeys map
	km.sendReKeys = NewKeyStack()
	TransmissionReKeys.Store(km.partner, km.sendReKeys)

	// Create send E2E ReKeys and add to stack
	for _, key := range sendReKeys {
		e2ekey := new(E2EKey)
		e2ekey.key = key
		e2ekey.manager = km
		e2ekey.outer = format.Rekey
		km.sendReKeys.Push(e2ekey)
	}

	// Create Receive E2E Keys and add them to ReceptionKeys map
	// while keeping a list of the fingerprints
	// Skip keys that were already used as per recvStates
	km.receiveKeysFP = make([]format.Fingerprint, 0)
	for i, key := range recvKeys {
		if !km.checkRecvStateBit(uint32(i)) {
			e2ekey := new(E2EKey)
			e2ekey.key = key
			e2ekey.manager = km
			e2ekey.outer = format.E2E
			e2ekey.keyID = uint32(i)
			keyFP := e2ekey.KeyFingerprint()
			km.receiveKeysFP = append(km.receiveKeysFP, keyFP)
			ReceptionKeys.Store(keyFP, e2ekey)
		}
	}

	// Create Receive E2E Keys and add them to ReceptionKeys map
	// while keeping a list of the fingerprints
	km.receiveReKeysFP = make([]format.Fingerprint, 0)
	for i, key := range recvReKeys {
		if !km.checkRecvStateBit(km.numKeys + uint32(i)) {
			e2ekey := new(E2EKey)
			e2ekey.key = key
			e2ekey.manager = km
			e2ekey.outer = format.Rekey
			e2ekey.keyID = km.numKeys + uint32(i)
			keyFP := e2ekey.KeyFingerprint()
			km.receiveReKeysFP = append(km.receiveReKeysFP, keyFP)
			ReceptionKeys.Store(keyFP, e2ekey)
		}
	}
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

func (km *KeyManager) GobEncode() ([]byte, error) {
	// Anonymous structure that flattens nested structures
	s := struct {
		Partner   []byte
		State     []byte
		TTL       []byte
		NumKeys   []byte
		NumReKeys []byte
		RecvState []byte
		BaseKey   []byte
	}{
		km.partner.Bytes(),
		make([]byte, 8),
		make([]byte, 2),
		make([]byte, 4),
		make([]byte, 2),
		make([]byte, 8*maxStates),
		make([]byte, 0),
	}

	// Convert all internal uints to bytes
	binary.BigEndian.PutUint64(s.State, *km.state)
	binary.BigEndian.PutUint16(s.TTL, km.ttl)
	binary.BigEndian.PutUint32(s.NumKeys, km.numKeys)
	binary.BigEndian.PutUint16(s.NumReKeys, km.numReKeys)
	for i := 0; i < int(maxStates); i++ {
		binary.BigEndian.PutUint64(s.RecvState[i*8:(i+1)*8], *km.recvState[i])
	}

	// GobEncode baseKey
	baseKeyBytes, err := km.baseKey.GobEncode()

	if err != nil {
		return nil, err
	}

	// Add baseKey to struct
	s.BaseKey = append(s.BaseKey, baseKeyBytes...)

	var buf bytes.Buffer

	// Create new encoder that will transmit the buffer
	enc := gob.NewEncoder(&buf)

	// Transmit the data
	err = enc.Encode(s)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (km *KeyManager) GobDecode(in []byte) error {
	// Anonymous structure that flattens nested structures
	s := struct {
		Partner   []byte
		State     []byte
		TTL       []byte
		NumKeys   []byte
		NumReKeys []byte
		RecvState []byte
		BaseKey   []byte
	}{
		make([]byte, 32),
		make([]byte, 8),
		make([]byte, 2),
		make([]byte, 4),
		make([]byte, 2),
		make([]byte, 8*maxStates),
		[]byte{},
	}

	var buf bytes.Buffer

	// Write bytes to the buffer
	buf.Write(in)

	// Create new decoder that reads from the buffer
	dec := gob.NewDecoder(&buf)

	// Receive and decode data
	err := dec.Decode(&s)

	if err != nil {
		return err
	}

	// Convert decoded bytes and put into key manager structure
	km.baseKey = new(cyclic.Int)
	err = km.baseKey.GobDecode(s.BaseKey)

	if err != nil {
		return err
	}

	km.partner = new(id.User).SetBytes(s.Partner)
	km.state = new(uint64)
	*km.state = binary.BigEndian.Uint64(s.State)
	km.ttl = binary.BigEndian.Uint16(s.TTL)
	km.numKeys = binary.BigEndian.Uint32(s.NumKeys)
	km.numReKeys = binary.BigEndian.Uint16(s.NumReKeys)
	for i := 0; i < int(maxStates); i++ {
		km.recvState[i] = new(uint64)
		*km.recvState[i] = binary.BigEndian.Uint64(s.RecvState[i*8:(i+1)*8])
	}

	return nil
}
