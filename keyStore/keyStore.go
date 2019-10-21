package keyStore

import (
	"bytes"
	"encoding/gob"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
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

// Internal helper function to get a list of all values
// contained in a KeyManMap
func (m *keyManMap) values() []*KeyManager {
	valueList := make([]*KeyManager, 0)
	(*sync.Map)(m).Range(func(key, value interface{}) bool {
		valueList = append(valueList, value.(*KeyManager))
		return true
	})
	return valueList
}

// Internal helper function to get a list of all keys
// contained in a KeyManMap
func (m *keyManMap) keys() []id.User {
	keyList := make([]id.User, 0)
	(*sync.Map)(m).Range(func(key, value interface{}) bool {
		keyList = append(keyList, key.(id.User))
		return true
	})
	return keyList
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
		key.GetOuterType() == parse.Rekey,
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
// It is not a sync.map since it won't be accessed
// very often
// It still contains a lock for multithreaded access
type KeyStore struct {
	// Key generation parameters
	params *KeyParams

	// Transmission Keys map
	// Maps id.User to *KeyManager
	sendKeyManagers *keyManMap

	// Reception Keys map
	// Maps format.Fingerprint to *E2EKey
	receptionKeys *inKeyMap

	// Reception Key Managers map
	recvKeyManagers map[id.User]*ReceptionKeyManagerBuffer

	lock sync.Mutex
}

func NewStore() *KeyStore {
	ks := new(KeyStore)
	ks.params = &KeyParams{
		MinKeys:   minKeys,
		MaxKeys:   maxKeys,
		NumRekeys: numReKeys,
		TTLParams: e2e.TTLParams{
			TTLScalar:  ttlScalar,
			MinNumKeys: threshold,
		},
	}
	ks.sendKeyManagers = new(keyManMap)
	ks.receptionKeys = new(inKeyMap)
	ks.recvKeyManagers = make(map[id.User]*ReceptionKeyManagerBuffer)
	return ks
}

// Get Key generation parameters from KeyStore
func (ks *KeyStore) GetKeyParams() *KeyParams {
	return ks.params
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

// GetPartners returns the list of partners we have keys for
func (ks *KeyStore) GetPartners() []id.User {
	return ks.sendKeyManagers.keys()
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

// Add a Receive KeyManager to respective map in KeyStore
func (ks *KeyStore) AddRecvManager(km *KeyManager) {
	ks.lock.Lock()
	defer ks.lock.Unlock()

	//ks.recvKeyManagers = km
	keys, ok := ks.recvKeyManagers[*km.partner]

	if ok {
		toBeDeleted := keys.push(km)
		ks.DeleteReceiveKeysByFingerprint(toBeDeleted)
	} else {
		newBuffer := NewReceptionKeyManagerBuffer()
		newBuffer.push(km)
		ks.recvKeyManagers[*km.partner] = newBuffer
	}
}

// Gets the Key manager at the current location on the ReceptionKeyManagerBuffer
// based on partner ID
func (ks *KeyStore) GetRecvManager(partner *id.User) *KeyManager {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	return ks.recvKeyManagers[*partner].getCurrentReceptionKeyManager()
}

// Delete a Receive KeyManager based on partner ID from respective map in KeyStore
func (ks *KeyStore) DeleteRecvManager(partner *id.User) {
	ks.lock.Lock()
	defer ks.lock.Unlock()
	delete(ks.recvKeyManagers, *partner)
}

// GobEncode the KeyStore
func (ks *KeyStore) GobEncode() ([]byte, error) {
	var buf bytes.Buffer

	// Create new encoder that will transmit the buffer
	enc := gob.NewEncoder(&buf)

	// Transmit the Key Parameters
	err := enc.Encode(ks.params)

	if err != nil {
		return nil, err
	}

	// Transmit the Send Key Managers
	kmList := ks.sendKeyManagers.values()
	err = enc.Encode(kmList)

	if err != nil {
		return nil, err
	}

	// Transmit the Receive Key Managers
	err = enc.Encode(ks.recvKeyManagers)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GobDecode the KeyStore from bytes
// NOTE: ReconstructKeys must be called after GobDecoding a KeyStore
func (ks *KeyStore) GobDecode(in []byte) error {
	var buf bytes.Buffer

	// Write bytes to the buffer
	buf.Write(in)

	// Create new decoder that reads from the buffer
	dec := gob.NewDecoder(&buf)

	// Decode Key Parameters
	err := dec.Decode(&ks.params)

	if err != nil {
		return err
	}

	// Decode Key Managers List
	var kmList []*KeyManager
	err = dec.Decode(&kmList)

	if err != nil {
		return err
	}

	// Decode Recv Key Managers map
	err = dec.Decode(&ks.recvKeyManagers)

	if err != nil {
		return err
	}

	// Reconstruct Send Key Manager map
	ks.sendKeyManagers = new(keyManMap)
	ks.receptionKeys = new(inKeyMap)
	for _, km := range kmList {
		ks.AddSendManager(km)
	}

	return nil
}

// ReconstructKeys loops through all key managers and
// calls GenerateKeys on each of them, in order to rebuild
// the key maps
func (ks *KeyStore) ReconstructKeys(grp *cyclic.Group, userID *id.User) {

	kmList := ks.sendKeyManagers.values()
	for _, km := range kmList {
		km.GenerateKeys(grp, userID)
		ks.AddSendManager(km)
	}

	for _, kmb := range ks.recvKeyManagers {
		for _, km := range kmb.managers {
			if km != nil {
				e2eKeys := km.GenerateKeys(grp, userID)
				ks.AddReceiveKeysByFingerprint(e2eKeys)
			}
		}
	}
}

func (ks *KeyStore) DeleteReceiveKeysByFingerprint(toBeDeleted []format.Fingerprint) {
	if len(toBeDeleted) != 0 {
		ks.receptionKeys.DeleteList(toBeDeleted)
	}
}

func (ks *KeyStore) AddReceiveKeysByFingerprint(newKeys []*E2EKey) {
	for _, key := range newKeys {
		ks.AddRecvKey(key.KeyFingerprint(), key)
	}
}

// Delete multiple Receiving E2EKeys from the correct KeyStore map
// based on a list of fingerprints
func (ks *KeyStore) DeleteRecvKeyList(fingerprints []format.Fingerprint) {
	ks.receptionKeys.DeleteList(fingerprints)
}
