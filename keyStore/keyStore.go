package keyStore

import (
	"gitlab.com/elixxir/client/user"
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
type inKeyMap sync.Map

// Transmission Keys map
var TransmissionKeys = new(outKeyMap)

// Transmission ReKeys map
var TransmissionReKeys = new(outKeyMap)

// Receiption Keys map
var ReceptionKeys = new(inKeyMap)

// Stores a KeyStack entry for given user
func (m *outKeyMap) Store(user *id.User, keys *KeyStack) {
	(*sync.Map)(m).Store(*user, keys)
}

// Atomically Pops first key from KeyStack for given user
// Updates Key Manager state
// Returns E2EKey and KeyAction
func (m *outKeyMap) Pop(user *id.User) (*E2EKey, KeyAction) {
	val, ok := (*sync.Map)(m).Load(*user)

	if !ok {
		return nil, None
	}

	keyStack := val.(*KeyStack)

	// Pop key
	e2eKey := keyStack.Pop()
	// Update Key Manager State
	action := e2eKey.GetManager().UpdateState(e2eKey.GetOuterType() == format.Rekey)
	return e2eKey, action
}

// Deletes a KeyStack entry for given user
func (m *outKeyMap) Delete(user *id.User) {
	(*sync.Map)(m).Delete(*user)
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

	var key *E2EKey
	if !ok {
		return nil
	} else {
		key = val.(*E2EKey)
	}
	// Delete key from map
	m.Delete(fingerprint)
	// Update Key Manager Receiving State
	key.GetManager().UpdateRecvState(key.GetKeyID())
	return key
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

func RegisterPartner(partnerID *id.User, pubKey *signature.DSAPublicKey) {
	// Get needed variables from session
	grp := user.TheSession.GetGroup()
	userID := user.TheSession.GetCurrentUser().User
	privKey := user.TheSession.GetPrivateKey()

	// Generate baseKey
	pubKeyVal := grp.NewIntFromLargeInt(pubKey.GetKey())
	baseKey, _ := diffieHellman.CreateDHSessionKey(pubKeyVal, privKey, grp)

	// Generate key TTL and number of keys
	keysTTL, numKeys := e2e.GenerateKeyTTL(baseKey.GetLargeInt(),
		minKeys, maxKeys,
		e2e.TTLParams{ttlScalar, threshold})

	// Create KeyManager
	keyMan := NewKeyManager(baseKey, partnerID, numKeys, keysTTL, numReKeys)

	// Generate Keys
	keyMan.GenerateKeys(grp, userID)
}
