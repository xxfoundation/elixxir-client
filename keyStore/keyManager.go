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
// With that limit, and setting maxKeys to 800,
// we need a Threshold of 224, and a scalar
// smaller than 1.28 to ensure we never generate
// more than 1024 keys
// With 2 receiving states for ReKeys we can hold
// 128 Rekeys
const (
	keyStates   uint16  = 16
	reKeyStates uint16  = 2
	MinKeys     uint16  = 500
	MaxKeys     uint16  = 800
	TTLScalar   float64 = 1.2 // generate 20% extra keys
	Threshold   uint16  = 224
	NumReKeys   uint16  = 128
)

// The KeyManager keeps track of all keys used in a single E2E
// relationship between the user and a partner
// It tracks usage of send Keys and ReKeys in an atomic state
// It tracks usage of receiving Keys and ReKeys in lists of
// atomic "dirty bit" states
// It also owns the send Keys and ReKeys stacks of keys,
// and lists of receiving Keys and ReKeys fingerprints
// All Key Managers can be stored in the session object, and
// can be GOB encoded/decoded, preserving the state
// When the GOB Decode is successful, GenerateKeys can be called
// on the KeyManager to generate all keys that have not been used
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

	// Received Keys dirty bits
	recvKeysState [keyStates]*uint64
	// Received ReKeys dirty bits
	recvReKeysState [reKeyStates]*uint64

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
// Receives the baseKey, partner userID, numKeys, ttl and numReKeys
// All internal states are forced to 0 for safety purposes
func NewKeyManager(baseKey *cyclic.Int, partner *id.User,
	numKeys uint32, ttl uint16, numReKeys uint16) *KeyManager {

	km := new(KeyManager)
	km.baseKey = baseKey
	km.partner = partner
	km.state = new(uint64)
	*km.state = 0
	km.ttl = ttl
	km.numKeys = numKeys
	km.numReKeys = numReKeys
	for i, _ := range km.recvKeysState {
		km.recvKeysState[i] = new(uint64)
		*km.recvKeysState[i] = 0
	}
	for i, _ := range km.recvReKeysState {
		km.recvReKeysState[i] = new(uint64)
		*km.recvReKeysState[i] = 0
	}
	return km
}

// Constants needed for access to send Keys state
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
	return keyCounter >= uint32(ttl)
}

// Check if state ReKey Counter >= nKeys
// Return true if so, which should trigger a purge
func stateReKeyCmp(state uint64, nKeys uint16) bool {
	reKeyCounter := uint16((state & stateReKeyMask) >> stateReKeyShift)
	return reKeyCounter >= nKeys
}

// UpdateState atomically updates internal state
// of key manager for send Keys or ReKeys
func (km *KeyManager) updateState(rekey bool) KeyAction {
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

// UpdateRecvState atomically updates internal
// receiving state of key manager
// It sets the correct bit of state index based on keyNum
// and rekey
func (km *KeyManager) updateRecvState(rekey bool, keyNum uint32) {
	stateIdx := keyNum / 64
	stateBit := uint64(1 << (keyNum % 64))

	if rekey {
		atomic.AddUint64(km.recvReKeysState[stateIdx], stateBit)
	} else {
		atomic.AddUint64(km.recvKeysState[stateIdx], stateBit)
	}
}

// Return true if bit specified by keyNum is set, meaning
// that a particular key or reKey has been used
func (km *KeyManager) checkRecvStateBit(rekey bool, keyNum uint32) bool {
	stateIdx := keyNum / 64
	stateBit := uint64(1 << (keyNum % 64))

	var state uint64
	if rekey {
		state = atomic.LoadUint64(km.recvReKeysState[stateIdx])
	} else {
		state = atomic.LoadUint64(km.recvKeysState[stateIdx])
	}

	return (state & stateBit) != 0
}

// GenerateKeys will generate all previously unused keys based on
// KeyManager states
// Sending Keys and ReKeys are generated and then pushed to a stack,
// meaning that they are used in a LIFO manner.
// This makes it easier to generate all send keys from a pre-existing state
// as the number of unused keys will be simply numKeys - usedKeys
// where usedKeys is extracted from the KeyManager state
// Receiving Keys and ReKeys are generated in order, but there is no
// guarantee that they will be used in order, this is why KeyManager
// keeps a list of fingerprint for all receiving keys
// When generating receiving keys from pre-existing state, all bits
// from receiving states are checked, and if the bit is set ("dirty")
// the key is not added to the Reception Keys map and fingerprint list
// This way, this function can be used to generate all keys when a new
// E2E relationship is established, and also to generate all previously
// unused keys based on KeyManager state, when reloading an user session
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
		if !km.checkRecvStateBit(false, uint32(i)) {
			e2ekey := new(E2EKey)
			e2ekey.key = key
			e2ekey.manager = km
			e2ekey.outer = format.E2E
			e2ekey.keyNum = uint32(i)
			keyFP := e2ekey.KeyFingerprint()
			km.receiveKeysFP = append(km.receiveKeysFP, keyFP)
			ReceptionKeys.Store(keyFP, e2ekey)
		}
	}

	// Create Receive E2E Keys and add them to ReceptionKeys map
	// while keeping a list of the fingerprints
	km.receiveReKeysFP = make([]format.Fingerprint, 0)
	for i, key := range recvReKeys {
		if !km.checkRecvStateBit(true, uint32(i)) {
			e2ekey := new(E2EKey)
			e2ekey.key = key
			e2ekey.manager = km
			e2ekey.outer = format.Rekey
			e2ekey.keyNum = uint32(i)
			keyFP := e2ekey.KeyFingerprint()
			km.receiveReKeysFP = append(km.receiveReKeysFP, keyFP)
			ReceptionKeys.Store(keyFP, e2ekey)
		}
	}
}

// Destroy will remove all keys managed by the KeyManager
// from the key maps, and then it will delete the sending
// keys stacks
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

// GobEncode the KeyManager so that it can be saved in
// the session file
func (km *KeyManager) GobEncode() ([]byte, error) {
	// Anonymous structure that flattens nested structures
	s := struct {
		Partner        []byte
		State          []byte
		TTL            []byte
		NumKeys        []byte
		NumReKeys      []byte
		RecvKeyState   []byte
		RecvReKeyState []byte
		BaseKey        []byte
	}{
		km.partner.Bytes(),
		make([]byte, 8),
		make([]byte, 2),
		make([]byte, 4),
		make([]byte, 2),
		make([]byte, 8*keyStates),
		make([]byte, 8*reKeyStates),
		make([]byte, 0),
	}

	// Convert all internal uints to bytes
	binary.BigEndian.PutUint64(s.State, *km.state)
	binary.BigEndian.PutUint16(s.TTL, km.ttl)
	binary.BigEndian.PutUint32(s.NumKeys, km.numKeys)
	binary.BigEndian.PutUint16(s.NumReKeys, km.numReKeys)
	for i := 0; i < int(keyStates); i++ {
		binary.BigEndian.PutUint64(
			s.RecvKeyState[i*8:(i+1)*8],
			*km.recvKeysState[i])
	}
	for i := 0; i < int(reKeyStates); i++ {
		binary.BigEndian.PutUint64(
			s.RecvReKeyState[i*8:(i+1)*8],
			*km.recvReKeysState[i])
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

// GobDecode bytes into a new Key Manager
// It can be used to get Key Managers from the
// store session file
// GenerateKeys should then be run so that all
// key maps are restored properly
func (km *KeyManager) GobDecode(in []byte) error {
	// Anonymous structure that flattens nested structures
	s := struct {
		Partner        []byte
		State          []byte
		TTL            []byte
		NumKeys        []byte
		NumReKeys      []byte
		RecvKeyState   []byte
		RecvReKeyState []byte
		BaseKey        []byte
	}{
		make([]byte, 32),
		make([]byte, 8),
		make([]byte, 2),
		make([]byte, 4),
		make([]byte, 2),
		make([]byte, 8*keyStates),
		make([]byte, 8*reKeyStates),
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
	for i := 0; i < int(keyStates); i++ {
		km.recvKeysState[i] = new(uint64)
		*km.recvKeysState[i] = binary.BigEndian.Uint64(
			s.RecvKeyState[i*8 : (i+1)*8])
	}
	for i := 0; i < int(reKeyStates); i++ {
		km.recvReKeysState[i] = new(uint64)
		*km.recvReKeysState[i] = binary.BigEndian.Uint64(
			s.RecvReKeyState[i*8 : (i+1)*8])
	}

	return nil
}
