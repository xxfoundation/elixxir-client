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
	numStates   uint16  = 16
	numReStates uint16  = 2
	MinKeys     uint16  = 500
	MaxKeys     uint16  = 800
	TTLScalar   float64 = 1.2 // generate 20% extra keys
	Threshold   uint16  = 224
	NumReKeys   uint16  = 128
)

// The KeyManager keeps track of all keys used in a single E2E
// uni-directional relationship between the user and a partner
// It tracks usage of send Keys and ReKeys in an atomic sendState
// OR
// It tracks usage of receiving Keys and ReKeys in lists of
// atomic "dirty bit" states
// It also owns the send Keys and ReKeys stacks of keys
// OR lists of receiving Keys and ReKeys fingerprints
// All Key Managers can be stored in the session object, and
// can be GOB encoded/decoded, preserving the state
// When the GOB Decode is successful, GenerateKeys can be called
// on the KeyManager to generate all keys that have not been used
type KeyManager struct {
	// Underlying key
	baseKey *cyclic.Int
	// Own Private Key
	privKey *cyclic.Int
	// Partner Public Key
	pubKey  *cyclic.Int

	// Designates end-to-end partner
	partner *id.User

	// True if key manager tracks send keys, false if receive keys
	sendOrRecv bool

	// State of Sending Keys and Rekeys, formatted as follows:
	//                      Bits
	// |    63   | 62 - 56 |   55 - 40   | 39 - 32 |  31 - 0   |
	// | deleted |  empty  | rekey count |  empty  | key count |
	// |  1 bit  |  7 bits |   16 bits   |  8 bits |   32 bits |
	sendState *uint64

	// Value of the counter at which a rekey is triggered
	ttl uint16

	// Total number of Keys
	numKeys uint32
	// Total number of Rekey keys
	numReKeys uint16

	// Received Keys dirty bits
	// Each bit represents a single Receiving Key
	recvKeysState [numStates]*uint64
	// Received ReKeys dirty bits
	// Each bit represents a single Receiving ReKey
	recvReKeysState [numReStates]*uint64

	// Send Keys Stack
	sendKeys *KeyStack
	// Send ReKeys Stack
	sendReKeys *KeyStack
	// Receive Keys fingerprint list
	recvKeysFingerprint []format.Fingerprint
	// Receive ReKeys fingerprint list
	recvReKeysFingerprint []format.Fingerprint
}

// Creates a new KeyManager to manage E2E Keys between user and partner
// Receives the baseKey, privKey, pubKey, partner userID, numKeys, ttl and numReKeys
// All internal states are forced to 0 for safety purposes
func NewManager(baseKey *cyclic.Int,
	privKey *cyclic.Int, pubKey *cyclic.Int,
	partner *id.User, sendOrRecv bool,
	numKeys uint32, ttl uint16, numReKeys uint16) *KeyManager {

	km := new(KeyManager)
	km.baseKey = baseKey
	km.privKey = privKey
	km.pubKey = pubKey
	km.partner = partner
	km.sendOrRecv = sendOrRecv
	km.sendState = new(uint64)
	*km.sendState = 0
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

// Get the private key from the Key Manager
func (km *KeyManager) GetPrivKey() *cyclic.Int {
	return km.privKey
}

// Get the public key from the Key Manager
func (km *KeyManager) GetPubKey() *cyclic.Int {
	return km.pubKey
}

// Get the partner ID from the Key Manager
func (km *KeyManager) GetPartner() *id.User {
	return km.partner
}

// Constants needed for access to sendState
//                      Bits
// |    63   | 62 - 56 |   55 - 40   | 39 - 32 |  31 - 0   |
// | deleted |  empty  | rekey count |  empty  | key count |
// |  1 bit  |  7 bits |   16 bits   |  8 bits |   32 bits |
const (
	// Delete is most significant bit
	stateDeleteMask uint64 = 0x8000000000000000
	// Key Counter is lowest 32 bits
	stateKeyMask    uint64 = 0x00000000FFFFFFFF
	// ReKey Counter is bits 55 to 40 (0 indexed)
	stateReKeyMask  uint64 = 0x00FFFF0000000000
	// ReKey Counter shift value is 40
	stateReKeyShift uint64 = 40
	// Delete Increment is 1 shifted by 63 bits
	stateDeleteIncr uint64 = 1 << 63
	// Key Counter increment is 1
	stateKeyIncr    uint64 = 1
	// ReKey Counter increment is 1 << 40
	stateReKeyIncr  uint64 = 1 << stateReKeyShift
)

// Check if a Rekey should be triggered
// Extract the Key counter from state and then
// compare to passed val
func checkRekey(state uint64, val uint16) bool {
	keyCounter := uint32(state & stateKeyMask)
	return keyCounter >= uint32(val)
}

// Check if a Purge should be triggered
// Extract the ReKey counter from state and then
// compare to passed val
func checkPurge(state uint64, val uint16) bool {
	reKeyCounter := uint16((state & stateReKeyMask) >> stateReKeyShift)
	return reKeyCounter >= val
}

// UpdateState atomically updates internal state
// of key manager for send Keys or ReKeys
// Once the number of used keys reaches the TTL value
// a Rekey Action is returned
// Once the number of used ReKeys reaches the the NumReKeys
// value, a Purge Action is returned, and the Key Manager
// can be destroyed
// When a Purge is returned, the state topmost bit is set,
// indicating that the KeyManager is now Deleted
// This means that if the caller doesn't destroy it
// right away, any further send Keys obtained from the
// global key map will have the action set to Deleted
// which can be used to trigger an error
func (km *KeyManager) updateState(rekey bool) Action {
	var stateIncr uint64
	// Choose the correct increment according to key type
	if rekey {
		stateIncr = stateReKeyIncr
	} else {
		stateIncr = stateKeyIncr
	}

	// Atomically increment the state and save result
	result := atomic.AddUint64(km.sendState, stateIncr)

	// Check if KeyManager is in Deleted state
	if result&stateDeleteMask != 0 {
		return Deleted
	}

	// Check if result should trigger a Purge
	if rekey && checkPurge(result, km.numReKeys) {
		// set delete bit
		atomic.AddUint64(km.sendState, stateDeleteIncr)
		return Purge
	// Check if result should trigger a Rekey
	} else if !rekey && checkRekey(result, km.ttl) {
		return Rekey
	}
	return None
}

// UpdateRecvState atomically updates internal
// receiving state of key manager
// It sets the correct bit of state index based on keyNum
// and rekey
// The keyNum is used to select the correct state from the array
// Since each state is an uint64, keyNum / 64 determines the index
// and keyNum % 64 determines the bit that needs to be set
// Rekey is used to select which state array to update:
// recvReKeysState or recvKeysState
// The state is atomically updated by adding a value of 1 shifted
// to the determined bit
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
// The keyNum is used to select the correct state from the array
// Since each state is an uint64, keyNum / 64 determines the index
// and keyNum % 64 determines the bit that needs to be read
// Rekey is used to select which state array to update:
// recvReKeysState or recvKeysState
// The state is atomically loaded and then the bit mask is applied
// to check if the value is 0 or different
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
func (km *KeyManager) GenerateKeys(grp *cyclic.Group, userID *id.User,
	ks *KeyStore) {
	if km.sendOrRecv {
		// Calculate how many unused send keys are needed
		usedSendKeys := uint32(*km.sendState & stateKeyMask)
		numGenSendKeys := uint(km.numKeys - usedSendKeys)
		usedSendReKeys := uint16((*km.sendState & stateReKeyMask) >> stateReKeyShift)
		numGenSendReKeys := uint(km.numReKeys - usedSendReKeys)

		// Generate numGenSendKeys send keys
		sendKeys := e2e.DeriveKeys(grp, km.baseKey, userID, numGenSendKeys)
		// Generate numGenSendReKeys send reKeys
		sendReKeys := e2e.DeriveEmergencyKeys(grp, km.baseKey, userID, numGenSendReKeys)

		// Create Send Keys Stack on keyManager and
		// set on TransmissionKeys map
		km.sendKeys = NewKeyStack()
		ks.TransmissionKeys.Store(km.partner, km.sendKeys)

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
		ks.TransmissionReKeys.Store(km.partner, km.sendReKeys)

		// Create send E2E ReKeys and add to stack
		for _, key := range sendReKeys {
			e2ekey := new(E2EKey)
			e2ekey.key = key
			e2ekey.manager = km
			e2ekey.outer = format.Rekey
			km.sendReKeys.Push(e2ekey)
		}
	} else {
		// For receiving keys, generate all, and then only add to the map
		// the unused ones based on recvStates
		// Generate numKeys recv keys
		recvKeys := e2e.DeriveKeys(grp, km.baseKey, km.partner, uint(km.numKeys))
		// Generate numReKeys recv reKeys
		recvReKeys := e2e.DeriveEmergencyKeys(grp, km.baseKey, km.partner, uint(km.numReKeys))

		// Create Receive E2E Keys and add them to ReceptionKeys map
		// while keeping a list of the fingerprints
		// Skip keys that were already used as per recvStates
		km.recvKeysFingerprint = make([]format.Fingerprint, 0)
		for i, key := range recvKeys {
			if !km.checkRecvStateBit(false, uint32(i)) {
				e2ekey := new(E2EKey)
				e2ekey.key = key
				e2ekey.manager = km
				e2ekey.outer = format.E2E
				e2ekey.keyNum = uint32(i)
				keyFP := e2ekey.KeyFingerprint()
				km.recvKeysFingerprint = append(km.recvKeysFingerprint, keyFP)
				ks.ReceptionKeys.Store(keyFP, e2ekey)
			}
		}

		// Create Receive E2E Keys and add them to ReceptionKeys map
		// while keeping a list of the fingerprints
		km.recvReKeysFingerprint = make([]format.Fingerprint, 0)
		for i, key := range recvReKeys {
			if !km.checkRecvStateBit(true, uint32(i)) {
				e2ekey := new(E2EKey)
				e2ekey.key = key
				e2ekey.manager = km
				e2ekey.outer = format.Rekey
				e2ekey.keyNum = uint32(i)
				keyFP := e2ekey.KeyFingerprint()
				km.recvReKeysFingerprint = append(km.recvReKeysFingerprint, keyFP)
				ks.ReceptionKeys.Store(keyFP, e2ekey)
			}
		}
	}
}

// Destroy will remove all keys managed by the KeyManager
// from the key maps
func (km *KeyManager) Destroy(ks *KeyStore) {
	if km.sendOrRecv {
		// Empty send keys and reKeys stacks
		// and remove them from maps
		ks.TransmissionKeys.Delete(km.partner)
		ks.TransmissionReKeys.Delete(km.partner)
		km.sendKeys.Delete()
		km.sendReKeys.Delete()
	} else {
		// Eliminate receiving keys
		ks.ReceptionKeys.DeleteList(km.recvKeysFingerprint)
		ks.ReceptionKeys.DeleteList(km.recvReKeysFingerprint)
	}

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
		SendOrRecv     []byte
		State          []byte
		TTL            []byte
		NumKeys        []byte
		NumReKeys      []byte
		RecvKeyState   []byte
		RecvReKeyState []byte
		BaseKey        []byte
		PrivKey        []byte
		PubKey         []byte
	}{
		km.partner.Bytes(),
		make([]byte, 1),
		make([]byte, 8),
		make([]byte, 2),
		make([]byte, 4),
		make([]byte, 2),
		make([]byte, 8*numStates),
		make([]byte, 8*numReStates),
		make([]byte, 0),
		make([]byte, 0),
		make([]byte, 0),
	}

	// Set send or receive
	if km.sendOrRecv {
		s.SendOrRecv[0] = 0xFF
	} else {
		s.SendOrRecv[0] = 0x00
	}

	// Convert all internal uints to bytes
	binary.BigEndian.PutUint64(s.State, *km.sendState)
	binary.BigEndian.PutUint16(s.TTL, km.ttl)
	binary.BigEndian.PutUint32(s.NumKeys, km.numKeys)
	binary.BigEndian.PutUint16(s.NumReKeys, km.numReKeys)
	for i := 0; i < int(numStates); i++ {
		binary.BigEndian.PutUint64(
			s.RecvKeyState[i*8:(i+1)*8],
			*km.recvKeysState[i])
	}
	for i := 0; i < int(numReStates); i++ {
		binary.BigEndian.PutUint64(
			s.RecvReKeyState[i*8:(i+1)*8],
			*km.recvReKeysState[i])
	}

	// GobEncode baseKey
	keyBytes, err := km.baseKey.GobEncode()

	if err != nil {
		return nil, err
	}

	// Add baseKey to struct
	s.BaseKey = append(s.BaseKey, keyBytes...)

	// GobEncode privKey
	keyBytes, err = km.privKey.GobEncode()

	if err != nil {
		return nil, err
	}

	// Add privKey to struct
	s.PrivKey = append(s.BaseKey, keyBytes...)

	// GobEncode pubKey
	keyBytes, err = km.pubKey.GobEncode()

	if err != nil {
		return nil, err
	}

	// Add pubKey to struct
	s.PubKey = append(s.BaseKey, keyBytes...)

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
		SendOrRecv     []byte
		State          []byte
		TTL            []byte
		NumKeys        []byte
		NumReKeys      []byte
		RecvKeyState   []byte
		RecvReKeyState []byte
		BaseKey        []byte
		PrivKey        []byte
		PubKey         []byte
	}{
		make([]byte, 32),
		make([]byte, 1),
		make([]byte, 8),
		make([]byte, 2),
		make([]byte, 4),
		make([]byte, 2),
		make([]byte, 8*numStates),
		make([]byte, 8*numReStates),
		[]byte{},
		[]byte{},
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

	km.privKey = new(cyclic.Int)
	err = km.privKey.GobDecode(s.PrivKey)

	if err != nil {
		return err
	}

	km.pubKey = new(cyclic.Int)
	err = km.pubKey.GobDecode(s.PubKey)

	if err != nil {
		return err
	}

	if s.SendOrRecv[0] == 0xFF {
		km.sendOrRecv = true
	} else {
		km.sendOrRecv = false
	}

	km.partner = new(id.User).SetBytes(s.Partner)
	km.sendState = new(uint64)
	*km.sendState = binary.BigEndian.Uint64(s.State)
	km.ttl = binary.BigEndian.Uint16(s.TTL)
	km.numKeys = binary.BigEndian.Uint32(s.NumKeys)
	km.numReKeys = binary.BigEndian.Uint16(s.NumReKeys)
	for i := 0; i < int(numStates); i++ {
		km.recvKeysState[i] = new(uint64)
		*km.recvKeysState[i] = binary.BigEndian.Uint64(
			s.RecvKeyState[i*8 : (i+1)*8])
	}
	for i := 0; i < int(numReStates); i++ {
		km.recvReKeysState[i] = new(uint64)
		*km.recvReKeysState[i] = binary.BigEndian.Uint64(
			s.RecvReKeyState[i*8 : (i+1)*8])
	}

	return nil
}
