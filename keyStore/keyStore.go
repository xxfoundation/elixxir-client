package keyStore

import (
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

// Local types in order to implement functions that
// return real types instead of interfaces
type keyManMap sync.Map
type inKeyMap sync.Map

// Stores a KeyManager entry for given user
func (m *keyManMap) Store(user *id.User, km *KeyManager) {
	(*sync.Map)(m).Store(*user, km)
}

// Loads a KeyManager entry for given user
func (m *keyManMap) Load(user *id.User) *KeyManager {
	val, ok := (*sync.Map)(m).Load(*user)
	if !ok {
		return nil
	} else {
		return val.(*KeyManager)
	}
}

// Deletes a KeyManager entry for given user
func (m *keyManMap) Delete(user *id.User) {
	(*sync.Map)(m).Delete(*user)
}

// Stores an *E2EKey for given fingerprint
func (m *inKeyMap) Store(fingerprint format.Fingerprint, key *E2EKey) {
	(*sync.Map)(m).Store(fingerprint, key)
}

// Pops key for given fingerprint, i.e,
// returns and deletes it from the map
// Atomically updates Key Manager Receiving state
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
	key.GetManager().updateRecvState(
		key.GetOuterType() == format.Rekey,
		key.keyNum)
	return key
}

// Deletes a key for given fingerprint
func (m *inKeyMap) Delete(fingerprint format.Fingerprint) {
	(*sync.Map)(m).Delete(fingerprint)
}

// Deletes keys from a given list of fingerprints
func (m *inKeyMap) DeleteList(fingerprints []format.Fingerprint) {
	for _, fp := range fingerprints {
		m.Delete(fp)
	}
}

// KeyStore contains the E2E key
// and Key Managers maps
// Send keys are obtained directly from the Key Manager
// which is looked up in the sendKeyManagers map
// Receiving keys are lookup up by fingerprint on
// receptionKeys map
// RecvKeyManagers map is needed in order to maintain
// active Key Managers when the session is stored/loaded
// It is not a sync.map since it won't ever be accessed
// other than for storage purposes, i.e., there is no
// Get function for this map
type KeyStore struct {
	// Transmission Keys map
	// Maps id.User to *KeyManager
	sendKeyManagers *keyManMap

	// Reception Keys map
	// Maps format.Fingerprint to *E2EKey
	receptionKeys *inKeyMap

	// Reception Key Managers map
	recvKeyManagers map[id.User]*KeyManager
}

func NewStore() *KeyStore {
	ks := new(KeyStore)
	ks.sendKeyManagers = new(keyManMap)
	ks.receptionKeys = new(inKeyMap)
	ks.recvKeyManagers = make(map[id.User]*KeyManager)
	return ks
}

// Add a Send KeyManager to respective map in KeyStore
func (ks *KeyStore) AddSendManager(km *KeyManager) {
	ks.sendKeyManagers.Store(km.GetPartner(), km)
}

// Get a Send KeyManager from respective map in KeyStore
// based on partner ID
func (ks *KeyStore) GetSendManager(partner *id.User) *KeyManager {
	return ks.sendKeyManagers.Load(partner)
}

// Delete a Send KeyManager from respective map in KeyStore
// based on partner ID
func (ks *KeyStore) DeleteSendManager(partner *id.User) {
	ks.sendKeyManagers.Delete(partner)
}

// Add a Receiving E2EKey to the correct KeyStore map
// based on its fingerprint
func (ks *KeyStore) AddRecvKey(fingerprint format.Fingerprint,
	key *E2EKey) {
	ks.receptionKeys.Store(fingerprint, key)
}

// Get the Receiving Key stored in correct KeyStore map
// based on the given fingerprint
func (ks *KeyStore) GetRecvKey(fingerprint format.Fingerprint) *E2EKey {
	return ks.receptionKeys.Pop(fingerprint)
}

// Delete multiple Receiving E2EKeys from the correct KeyStore map
// based on a list of fingerprints
func (ks *KeyStore) DeleteRecvKeyList(fingerprints []format.Fingerprint) {
	ks.receptionKeys.DeleteList(fingerprints)
}

// Add a Receive KeyManager to respective map in KeyStore
// NOTE: This function operates on a normal map
// be sure to not cause multi threading issues when calling
func (ks *KeyStore) AddRecvManager(km *KeyManager) {
	ks.recvKeyManagers[*km.GetPartner()] = km
}

// Delete a Receive KeyManager based on partner ID from respective map in KeyStore
// NOTE: This function operates on a normal map
// be sure to not cause multi threading issues when calling
func (ks *KeyStore) DeleteRecvManager(partner *id.User) {
	delete(ks.recvKeyManagers, *partner)
}
