package keyStore

import (
	user2 "gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

// Local types in order to implement functions that return real types
// instead of interfaces
type outKeyMap sync.Map
type inKeyMap  sync.Map

// Transmission Keys map
var TransmissionKeys = new(outKeyMap)

// Transmission ReKeys map
var TransmissionReKeys = new(outKeyMap)

// Receiption Keys map
var ReceptionKeys = new(inKeyMap)

// Stores a KeyStack entry for given user
func (m *outKeyMap) Store(user *id.User, keys *KeyStack) {
	(*sync.Map)(m).Store(user, keys)
}

// Returns KeyStack entry for given user
// Nil if not found
func (m *outKeyMap) Load(user *id.User) *KeyStack {
	val, ok := (*sync.Map)(m).Load(user)

	if ok {
		return val.(*KeyStack)
	} else {
		return nil
	}
}

// Atomically Pops first key from KeyStack for given user
// Updates Key Manager state
// Returns E2EKey and KeyAction
func (m *outKeyMap) Pop(user *id.User) (*E2EKey, KeyAction) {
	val, ok := (*sync.Map)(m).Load(user)

	if !ok {
		return nil, None
	}

	keyStack := val.(*KeyStack)

	// Lock stack
	keyStack.Lock()
	// Pop key
	e2eKey := keyStack.Pop()
	// Update Key Manager State
	action := e2eKey.GetManager().UpdateState(e2eKey.GetOuterType() == format.Rekey)
	// Unlock stack
	keyStack.Unlock()
	return e2eKey, action
}

// Deletes a KeyStack entry for given user
func (m *outKeyMap) Delete(user *id.User) {
	(*sync.Map)(m).Delete(user)
}

// Stores a key for given fingerprint
func (m *inKeyMap) Store(fingerprint format.Fingerprint, key *E2EKey) {
	(*sync.Map)(m).Store(fingerprint, key)
}

// Pops key for given fingerprint, i.e, returns it and
// deletes it from the map
// Returns nil if not found
func (m *inKeyMap) Pop(fingerprint format.Fingerprint) *E2EKey {
	val, ok := (*sync.Map)(m).Load(fingerprint)

	if !ok {
		return nil
	}
	// Delete key from map
	m.Delete(fingerprint)
	return val.(*E2EKey)
}

// Deletes a key for given fingerprint
func (m *inKeyMap) Delete(fingerprint format.Fingerprint) {
	(*sync.Map)(m).Delete(fingerprint)
}

// Deletes all keys from fingerprint list
func (m *inKeyMap) DeleteList(fingerprints []format.Fingerprint) {
	for _, fp := range fingerprints {
		(*sync.Map)(m).Delete(fp)
	}
}

// For now, generate a lot of keys
const (
	minKeys uint16 = 1000
	maxKeys uint16 = 2000
	ttlScalar float64 = 1.2 // generate 20% extra keys
	threshold uint16 = 1500 // min 1500 keys
)

func RegisterPartner(partner *id.User, pubKey signature.DSAPublicKey) {
	// Get needed variables from session
	grp := user2.TheSession.GetGroup()
	user := user2.TheSession.GetCurrentUser().User
	privKey := user2.TheSession.GetPrivateKey()

	// Generate baseKey
	pubKeyVal := grp.NewIntFromLargeInt(pubKey.GetKey())
	baseKey, _ := diffieHellman.CreateDHSessionKey(pubKeyVal, privKey, grp)

	// Generate key TTL and number of keys
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		minKeys, maxKeys,
		e2e.TTLParams{ttlScalar, threshold})

	// Generate all keys
	// Generate numKeys send keys
	sendKeys := e2e.DeriveKeys(grp, baseKey, user, uint(numKeys))
	// Generate keysTTL send reKeys
	sendReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, user, uint(keysTTL))
	// Generate numKeys recv keys
	recvKeys := e2e.DeriveKeys(grp, baseKey, partner, uint(numKeys))
	// Generate keysTTL recv reKeys
	recvReKeys := e2e.DeriveEmergencyKeys(grp, baseKey, partner, uint(keysTTL))

	// Create KeyManager
	keyMan := NewKeyManager(baseKey, partner, numKeys, keysTTL)
	// Lock key manager here for safety
	keyMan.Lock()
	// Unlock only when done
	defer keyMan.Unlock()

	// Create Send Keys Stack and set it on keyManager and
	// TransmissionKeys map
	sendKeysStack := new(KeyStack)
	keyMan.sendKeys = sendKeysStack
	TransmissionKeys.Store(partner, sendKeysStack)

	// Create send E2E Keys and add to stack
	for _, key := range sendKeys {
		e2ekey := new(E2EKey)
		e2ekey.key = key
		e2ekey.manager = keyMan
		e2ekey.outer = format.E2E
		sendKeysStack.Push(e2ekey)
	}

	// Create Send ReKeys Stack and set it on keyManager and
	// TransmissionReKeys map
	sendReKeysStack := new(KeyStack)
	keyMan.sendReKeys = sendReKeysStack
	TransmissionReKeys.Store(partner, sendReKeysStack)

	// Create send E2E ReKeys and add to stack
	for _, key := range sendReKeys {
		e2ekey := new(E2EKey)
		e2ekey.key = key
		e2ekey.manager = keyMan
		e2ekey.outer = format.Rekey
		sendReKeysStack.Push(e2ekey)
	}

	// Create Receive E2E Keys and add them to ReceptionKeys map
	// while keeping a list of the fingerprints
	fingerprintList := make([]format.Fingerprint, numKeys)
	for i, key := range recvKeys {
		e2ekey := new(E2EKey)
		e2ekey.key = key
		e2ekey.manager = keyMan
		e2ekey.outer = format.E2E
		sendKeysStack.Push(e2ekey)
		fingerprintList[i] = e2ekey.KeyFingerprint()
		ReceptionKeys.Store(fingerprintList[i], e2ekey)
	}

	// Set fingerprintList on KeyManager
	keyMan.receiveKeysFP = fingerprintList

	// Create Receive E2E Keys and add them to ReceptionKeys map
	// while keeping a list of the fingerprints
	fingerprintListRe := make([]format.Fingerprint, numKeys)
	for i, key := range recvReKeys {
		e2ekey := new(E2EKey)
		e2ekey.key = key
		e2ekey.manager = keyMan
		e2ekey.outer = format.Rekey
		sendKeysStack.Push(e2ekey)
		fingerprintListRe[i] = e2ekey.KeyFingerprint()
		ReceptionKeys.Store(fingerprintListRe[i], e2ekey)
	}

	// Set fingerprintList on KeyManager
	keyMan.receiveReKeysFP = fingerprintListRe
}
