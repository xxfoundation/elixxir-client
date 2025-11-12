////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/ekv/portable"
)

// init sets the log level to INFO.
func init() {
	jww.SetLogThreshold(jww.LevelInfo)
	jww.SetStdoutThreshold(jww.LevelInfo)
}

// cmixTrackerSingleton is used to track Cmix objects so that they can be
// referenced by ID back over the bindings.
var cmixTrackerSingleton = &cmixTracker{
	tracked: make(map[int]*Cmix),
	count:   0,
}

// Cmix wraps the xxdk.Cmix struct, implementing additional functions to support
// the bindings Cmix interface.
type Cmix struct {
	Api *xxdk.Cmix
	id  int
}

// GenericKeyValue is a simple key-value storage interface that can be used to
// back the ekv Storage interface. This allows ekv to work with any key-value
// store including browser localStorage, IndexedDB, etc.
//
// Bindings clients must implement this interface to use NewCmixWithKV and
// LoadCmixWithKV functions.
type GenericKeyValue interface {
	// Get retrieves the value for the given key.
	// Returns an error if the key does not exist.
	Get(key string) ([]byte, error)

	// Set stores the value for the given key.
	Set(key string, value []byte) error

	// Delete removes the key and its value.
	Delete(key string) error

	// Keys returns all keys in the store as a JSON array of strings.
	Keys() ([]byte, error)
}

// NewCmix creates user storage, generates keys, connects, and registers with
// the network. Note that this does not register a username/identity, but merely
// creates a new cryptographic identity for adding such information at a later
// date.
//
// Users of this function should delete the storage directory on error.
func NewCmix(ndfJSON, storageDir string, password []byte,
	registrationCode string) error {
	secret := copyAndClear(password)
	err := xxdk.NewCmix(ndfJSON, storageDir, secret, registrationCode)
	if err != nil {
		return errors.Errorf("Failed to create new cmix: %+v", err)
	}
	return nil
}

// NewCmixWithKV creates user storage backed by a key-value store, generates
// keys, connects, and registers with the network. Note that this does not
// register a username/identity, but merely creates a new cryptographic identity
// for adding such information at a later date.
//
// Users of this function should delete the storage directory on error.
func NewCmixWithKV(kv GenericKeyValue, ndfJSON, storageDir string,
	password []byte, registrationCode string) error {
	secret := copyAndClear(password)
	wrappedKV := newGenericKeyValueWrapper(kv)
	err := xxdk.NewCmixWithKV(wrappedKV, ndfJSON, storageDir, secret, registrationCode)
	if err != nil {
		return errors.Errorf("Failed to create new cmix with KV: %+v", err)
	}
	return nil
}

// NewSynchronizedCmix clones a Cmix from remote storage.
// Parameters:
//   - ndfJSON - the NDF file used to connect to the network.
//   - storageDir - the local directory or path used for the encrypted key value
//     store.
//   - remoteStoragePathPrefix - the remote "directory" or path prefix used
//     by the RemoteStore when reading/writing files.
//   - password - the pssword used to decrypt the encrypted key value store.
//   - remote - the RemoteStore implementation to use for multi-device
//     synchronization.
func NewSynchronizedCmix(ndfJSON, storageDir, remoteStoragePathPrefix string,
	password []byte,
	remote RemoteStore) error {

	secret := copyAndClear(password)
	wrappedRemote := newRemoteStoreFileSystemWrapper(remote)
	jww.INFO.Printf("[BINDINGS] NewSynchronizedCmix, "+
		"storageDir: %s, remoteStoragePathPrefix: %s",
		storageDir, remoteStoragePathPrefix)
	return xxdk.NewSynchronizedCmix(ndfJSON, storageDir,
		remoteStoragePathPrefix, secret,
		wrappedRemote)
}

// LoadCmix will load an existing user storage from the storageDir using the
// password. This will fail if the user storage does not exist or the password
// is incorrect.
//
// The password is passed as a byte array so that it can be cleared from memory
// and stored as securely as possible using the MemGuard library.
//
// LoadCmix does not block on network connection and instead loads and starts
// subprocesses to perform network operations.
func LoadCmix(storageDir string, password []byte, cmixParamsJSON []byte) (*Cmix,
	error) {

	secret := copyAndClear(password)

	params, err := parseCMixParams(cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	net, err := xxdk.LoadCmix(storageDir, secret, params)
	if err != nil {
		return nil, errors.Errorf("LoadCmix failed: %+v", err)
	}

	return cmixTrackerSingleton.make(net), nil
}

// LoadCmixWithKV will load an existing user storage backed by a key-value store
// from the storageDir using the password. This will fail if the user storage
// does not exist or the password is incorrect.
//
// The password is passed as a byte array so that it can be cleared from memory
// and stored as securely as possible using the MemGuard library.
//
// LoadCmixWithKV does not block on network connection and instead loads and
// starts subprocesses to perform network operations.
func LoadCmixWithKV(kv GenericKeyValue, storageDir string, password []byte,
	cmixParamsJSON []byte) (*Cmix, error) {

	secret := copyAndClear(password)

	params, err := parseCMixParams(cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	wrappedKV := newGenericKeyValueWrapper(kv)
	net, err := xxdk.LoadCmixWithKV(wrappedKV, storageDir, secret, params)
	if err != nil {
		return nil, errors.Errorf("LoadCmixWithKV failed: %+v", err)
	}

	return cmixTrackerSingleton.make(net), nil
}

// LoadSynchronizedCmix will load an existing user storage from the
// storageDir along with a remote store object. Writes to any keys
// inside a synchronized prefix will be saved to a remote store
// transaction log, and writes from other cMix instances will be
// tracked by reading transaction logs written by other instances.
//
// The password is passed as a byte array so that it can be cleared from memory
// and stored as securely as possible using the MemGuard library.
//
// LoadCmix does not block on network connection and instead loads and
// starts subprocesses to perform network operations. This can take a
// while if there are a lot of transactions to replay by other
// instances.
func LoadSynchronizedCmix(storageDir, remoteStoragePathPrefix string, password []byte,
	remote RemoteStore, cmixParamsJSON []byte) (*Cmix, error) {

	secret := copyAndClear(password)

	params, err := parseCMixParams(cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	synchedPrefixes := []string{
		collective.StandardRemoteSyncPrefix,
		"channels",
	}

	wrappedRemote := newRemoteStoreFileSystemWrapper(remote)
	jww.INFO.Printf("[BINDINGS] LoadSynchronizedCmix, "+
		"storageDir: %s, remoteStoragePathPrefix: %s",
		storageDir, remoteStoragePathPrefix)

	net, err := xxdk.LoadSynchronizedCmix(storageDir,
		remoteStoragePathPrefix, secret,
		wrappedRemote, synchedPrefixes, params)
	if err != nil {
		return nil, errors.Errorf("LoadSynchronizedCmix failed: %+v",
			err)
	}

	return cmixTrackerSingleton.make(net), nil

}

// GetID returns the ID for this Cmix in the cmixTracker.
func (c *Cmix) GetID() int {
	return c.id
}

// GetReceptionID returns the Default Reception Identity for this cMix
// Instance
func (c *Cmix) GetReceptionID() []byte {
	rid := *c.Api.GetStorage().GetReceptionID()
	return rid.Bytes()
}

// GetRemoteKV returns the underlying [RemoteKV] storage so it can be
// interacted with directly.
// TODO: force this into a synchronized prefix?
func (c *Cmix) GetRemoteKV() *RemoteKV {
	local := c.Api.GetStorage().GetKV()
	remote, err := local.Prefix(collective.StandardRemoteSyncPrefix)
	if err != nil {
		jww.FATAL.Panicf("could not get remote KV: %+v", err)
	}

	return &RemoteKV{
		rkv:             remote,
		keyListenerLcks: make(map[string]*sync.Mutex),
		keyListeners: make(
			map[string]map[int]KeyChangedByRemoteCallback),
		mapListenerLcks: make(map[string]*sync.Mutex),
		mapListeners: make(
			map[string]map[int]MapChangedByRemoteCallback),
	}
}

// EKVGet allows access to a value inside secure encrypted key value store
func (c *Cmix) EKVGet(key string) ([]byte, error) {
	ekv := c.Api.GetStorage().GetKV()
	versionedVal, err := ekv.Get(key, 0)
	if err != nil {
		return nil, err
	}
	return versionedVal.Data, nil
}

// EKVSet allows user to set a value inside secure encrypted key value store
func (c *Cmix) EKVSet(key string, value []byte) error {
	ekv := c.Api.GetStorage().GetKV()
	versioned := versioned.Object{
		Version:   0,
		Data:      value,
		Timestamp: time.Now(),
	}
	return ekv.Set(key, &versioned)
}

////////////////////////////////////////////////////////////////////////////////
// cMix Tracker                                                               //
////////////////////////////////////////////////////////////////////////////////

// GetCMixInstance gets the bindings.Cmix for the given Cmix instanceID.
//
// This function is not used by Go bindings and is intended to be used by other
// wrappers.
func GetCMixInstance(instanceID int) (*Cmix, error) {
	instance, ok := cmixTrackerSingleton.tracked[instanceID]
	if !ok {
		return nil, errors.Errorf("no cmix instance id: %d", instanceID)
	}
	return instance, nil
}

func DeleteCmixInstance(instanceID int) error {
	cmix, ok := cmixTrackerSingleton.tracked[instanceID]
	if !ok {
		return errors.Errorf("no cmix instance id: %d", instanceID)
	}
	cmix.StopNetworkFollower()
	cmixTrackerSingleton.delete(instanceID)
	return nil
}

// cmixTracker is a singleton used to keep track of extant Cmix objects,
// preventing race conditions created by passing it over the bindings.
type cmixTracker struct {
	tracked map[int]*Cmix
	count   int
	mux     sync.RWMutex
}

// make creates a Cmix from a [xxdk.Cmix], assigns it a unique ID, and adds it
// to the cmixTracker.
func (ct *cmixTracker) make(c *xxdk.Cmix) *Cmix {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.tracked[id] = &Cmix{
		Api: c,
		id:  id,
	}

	return ct.tracked[id]
}

// get returns a Cmix from the cmixTracker given its ID.
func (ct *cmixTracker) get(id int) (*Cmix, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.tracked[id]
	if !exist {
		return nil, errors.Errorf(
			"Cannot get Cmix for ID %d, does not exist", id)
	}

	return c, nil
}

// delete a Cmix from the cmixTracker.
func (ct *cmixTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.tracked, id)
}

// Note: Copy is required because iOS will delete byte slices
// after a function returns.
func copyAndClear(inputPassword []byte) []byte {
	secret := make([]byte, len(inputPassword))
	copy(secret, inputPassword)
	// TODO: replace with clear() when moving to go 1.21
	for i := 0; i < len(inputPassword); i++ {
		inputPassword[i] = 0
	}
	return secret
}

////////////////////////////////////////////////////////////////////////////////
// GenericKeyValue Wrapper                                                   //
////////////////////////////////////////////////////////////////////////////////

// genericKeyValueWrapper is an internal Go wrapper for GenericKeyValue that
// adheres to the portable.GenericKeyValue interface. It is used to wrap the
// GenericKeyValue given by the bindings user for use in the xxdk system.
type genericKeyValueWrapper struct {
	kv GenericKeyValue
}

// newGenericKeyValueWrapper constructs a genericKeyValueWrapper.
func newGenericKeyValueWrapper(bindingsKV GenericKeyValue) *genericKeyValueWrapper {
	return &genericKeyValueWrapper{kv: bindingsKV}
}

// Get retrieves the value for the given key.
func (g *genericKeyValueWrapper) Get(key string) ([]byte, error) {
	return g.kv.Get(key)
}

// Set stores the value for the given key.
func (g *genericKeyValueWrapper) Set(key string, value []byte) error {
	return g.kv.Set(key, value)
}

// Delete removes the key and its value.
func (g *genericKeyValueWrapper) Delete(key string) error {
	return g.kv.Delete(key)
}

// Keys returns all keys in the store.
func (g *genericKeyValueWrapper) Keys() ([]string, error) {
	keysJSON, err := g.kv.Keys()
	if err != nil {
		return nil, err
	}
	var keys []string
	err = json.Unmarshal(keysJSON, &keys)
	return keys, err
}

var _ portable.GenericKeyValue = (*genericKeyValueWrapper)(nil)
