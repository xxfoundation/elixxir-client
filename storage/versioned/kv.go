////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
)

// PrefixSeparator is the separator added when prefixing a KV (see [KV.Prefix]).
// No prefix may contain this character. This value has been defined as
// `backwards-slash` (\) rather than the typical `forward-slash` (/) for
// explicit purposes. There are many valid reasons for the consumer to want to
// define their prefixes with a `/`, such as using the base64 encoding of some
// value (see base64.StdEncoding].
const PrefixSeparator = `\`

const StandardRemoteSyncPrefix = "remoteSync"

var (
	EmptyPrefixErr               = errors.New("empty prefix")
	PrefixContainingSeparatorErr = errors.New("cannot accept prefix with the default separator")
	DuplicatePrefixErr           = errors.New("prefix has already been added, cannot overwrite")
	UnimplementedErr             = errors.New("not implemented")
)

// KV is a key value store interface that supports versioned and
// upgradable entries.
type KV interface {
	// Get returns the object stored at the specified version.
	Get(key string, version uint64) (*Object, error)

	// GetAndUpgrade gets and upgrades data stored in the key/value store.
	// Make sure to inspect the version returned in the versioned object.
	GetAndUpgrade(key string, ut UpgradeTable) (*Object, error)

	// Delete removes a given key from the data store.
	Delete(key string, version uint64) error

	// Set upserts new data into the storage
	// When calling this, you are responsible for prefixing the
	// key with the correct type optionally unique id! Call
	// MakeKeyWithPrefix() to do so.
	// The [Object] should contain the versioning if you are
	// maintaining such a functionality.
	Set(key string, object *Object) error

	// StoreMapElement stores a versioned map element into the KV. This relies
	// on the underlying remote [KV.StoreMapElement] function to lock and control
	// updates, but it uses [versioned.Object] values.
	// The version of the value must match the version of the map.
	// All Map storage functions update the remote.
	StoreMapElement(mapName, elementName string, mapVersion uint64,
		value *Object) error

	// StoreMap saves a versioned map element into the KV. This relies
	// on the underlying remote [KV.StoreMap] function to lock and control
	// updates, but it uses [versioned.Object] values.
	// the version of values must match the version of the map
	// All Map storage functions update the remote.
	StoreMap(mapName string, mapVersion uint64, values map[string]*Object) error

	// GetMap loads a versioned map from the KV. This relies
	// on the underlying remote [KV.GetMap] function to lock and control
	// updates, but it uses [versioned.Object] values.
	GetMap(mapName string, mapVersion uint64) (map[string]*Object, error)

	// GetMapElement loads a versioned map element from the KV. This relies
	// on the underlying remote [KV.GetMapElement] function to lock and control
	// updates, but it uses [versioned.Object] values.
	GetMapElement(mapName, elementName string, mapVersion uint64) (
		*Object, error)

	// Transaction locks a key while it is being mutated then stores the result
	// and returns the old value if it existed.
	// If the op returns an error, the operation will be aborted.
	Transaction(key string, op TransactionOperation, version uint64) (
		old *Object, existed bool, err error)

	// ListenOnRemoteKey allows the caller to receive updates when
	// a key is updated by synching with another client.
	// Only one callback can be written per key.
	ListenOnRemoteKey(key string, version uint64, callback KeyChangedByRemoteCallback)

	// ListenOnRemoteMap allows the caller to receive updates when
	// the map or map elements are updated
	ListenOnRemoteMap(mapName string, version uint64, callback MapChangedByRemoteCallback)

	// GetPrefix returns the full Prefix of the KV
	GetPrefix() string

	// HasPrefix returns whether this prefix exists in the KV
	HasPrefix(prefix string) bool

	// Prefix returns a new KV with the new prefix appending
	Prefix(prefix string) (KV, error)

	// Root returns the KV with no prefixes
	Root() KV

	// IsMemStore returns true if the underlying KV is memory based
	IsMemStore() bool

	// GetFullKey returns the key with all prefixes appended
	GetFullKey(key string, version uint64) string

	// Exists returns if the error indicates a KV error showing
	// the key exists.
	Exists(err error) bool
}

// KeyChangedByRemoteCallback is the callback used to report local updates caused
// by a remote client editing their EKV
type KeyChangedByRemoteCallback func(key string, old, new *Object, op KeyOperation)

// MapChangedByRemoteCallback is the callback used to report local updates caused
// by a remote client editing their EKV
type MapChangedByRemoteCallback func(mapName string, edits map[string]ElementEdit)
type ElementEdit struct {
	OldElement *Object
	NewElement *Object
	Operation  KeyOperation
}

type KeyOperation uint8

const (
	Created KeyOperation = iota
	Updated
	Deleted
)

type TransactionOperation func(old *Object, existed bool) (data *Object,
	err error)

type MutualTransactionOperation func(map[string]Value) (
	updates map[string]Value, err error)

type Value struct {
	Obj    *Object
	Exists bool
}

// MakePartnerPrefix creates a string prefix
// to denote who a conversation or relationship is with
func MakePartnerPrefix(id *id.ID) string {
	return fmt.Sprintf("Partner:%v", id.String())
}

// Upgrade functions must be of this type
type Upgrade func(oldObject *Object) (*Object,
	error)

// UpgradeTable contains a table of upgrade functions for a
// versioned KV.
type UpgradeTable struct {
	CurrentVersion uint64
	Table          []Upgrade
}

type root struct {
	data ekv.KeyValue
}

// kv stores versioned data and Upgrade functions
type kv struct {
	r         *root
	prefix    string
	prefixMap map[string]int
	offset    int
}

// Create a versioned key/value store backed by something implementing KeyValue
func NewKV(data ekv.KeyValue) KV {
	newkv := kv{
		prefixMap: make(map[string]int, 0),
	}
	root := root{}

	root.data = data

	newkv.r = &root

	jww.WARN.Printf("storage/versioned.KV is deprecated. " +
		"Please use sync.VersionedKV")

	return &newkv
}

// Get gets and upgrades data stored in the key/value store
// Make sure to inspect the version returned in the versioned object
func (v *kv) Get(key string, version uint64) (*Object, error) {
	key = v.makeKey(key, version)
	jww.TRACE.Printf("get %p with key %v", v.r.data, key)
	// get raw data
	result := Object{}
	err := v.r.data.Get(key, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAndUpgrade gets and upgrades data stored in the key/value store.
// Make sure to inspect the version returned in the versioned object.
func (v *kv) GetAndUpgrade(key string, ut UpgradeTable) (*Object, error) {
	version := ut.CurrentVersion
	baseKey := key
	key = v.makeKey(baseKey, version)

	if uint64(len(ut.Table)) != version {
		jww.FATAL.Panicf("Cannot get upgrade for %s: table lengh (%d) "+
			"does not match current version (%d)", key, len(ut.Table),
			version)
	}
	var result *Object
	// NOTE: Upgrades do not happen on the current version, so we check to
	// see if version-1, version-2, and so on exist to find out if an
	// earlier version of this object exists.
	version++
	for version != 0 {
		version--
		key = v.makeKey(baseKey, version)
		jww.TRACE.Printf("get %p with key %v", v.r.data, key)

		// get raw data
		result = &Object{}
		err := v.r.data.Get(key, result)
		// Break when we find the *newest* version of the object
		// in the data store.
		if err == nil {
			break
		}
	}

	if result == nil || len(result.Data) == 0 {
		return nil, errors.Errorf(
			"Failed to get key and upgrade it for %s",
			v.makeKey(baseKey, ut.CurrentVersion))
	}

	var err error
	initialVersion := result.Version
	for result.Version < uint64(len(ut.Table)) {
		oldVersion := result.Version
		result, err = ut.Table[oldVersion](result)
		if err != nil || oldVersion == result.Version {
			jww.FATAL.Panicf("failed to upgrade key %s from "+
				"version %v, initial version %v", key,
				oldVersion, initialVersion)
		}
	}

	return result, nil
}

// Delete removes a given key from the data store.
func (v *kv) Delete(key string, version uint64) error {
	key = v.makeKey(key, version)
	jww.TRACE.Printf("delete %p with key %v", v.r.data, key)
	return v.r.data.Delete(key)
}

// Set upserts new data into the storage
// When calling this, you are responsible for prefixing the key with the correct
// type optionally unique id! Call MakeKeyWithPrefix() to do so.
// The [Object] should contain the versioning if you are maintaining such
// a functionality.
func (v *kv) Set(key string, object *Object) error {
	key = v.makeKey(key, object.Version)
	jww.TRACE.Printf("Set %p with key %v", v.r.data, key)
	return v.r.data.Set(key, object)
}

// GetPrefix returns the prefix of the kv.
func (v *kv) GetPrefix() string {
	return v.prefix
}

// HasPrefix returns whether this prefix exists in the kv.
func (v *kv) HasPrefix(prefix string) bool {
	_, exists := v.prefixMap[prefix]
	return exists
}

// Prefix returns a new kv with the new prefix appending, implements
// [KV.Prefix].
func (v *kv) Prefix(prefix string) (KV, error) {
	if prefix == "" {
		return nil, EmptyPrefixErr
	}

	//// Reject invalid prefixes
	if strings.Contains(prefix, PrefixSeparator) {
		return nil, PrefixContainingSeparatorErr
	}

	// Reject duplicate prefixes
	if v.HasPrefix(prefix) {
		return nil, DuplicatePrefixErr
	}

	newPrefixMap := make(map[string]int)
	for k, v := range v.prefixMap {
		newPrefixMap[k] = v
	}
	newPrefixMap[prefix] = v.offset + 1

	// Intialize a new KV with the prefix appending
	kvPrefix := kv{
		r:         v.r,
		prefix:    v.prefix + prefix + PrefixSeparator,
		prefixMap: newPrefixMap,
		offset:    v.offset + 1,
	}

	return &kvPrefix, nil
}

// Root implements [KV.Root]
func (v *kv) Root() KV {
	kvPrefix := kv{
		r:         v.r,
		prefix:    "",
		prefixMap: make(map[string]int),
		offset:    0,
	}
	return &kvPrefix
}

func (v *kv) IsMemStore() bool {
	_, success := v.r.data.(*ekv.Memstore)
	return success
}

// GetFullKey returns the key with all prefixes appended
func (v *kv) GetFullKey(key string, version uint64) string {
	return v.makeKey(key, version)
}

func (v *kv) makeKey(key string, version uint64) string {
	return v.prefix + key + "_" + strconv.Itoa(int(version))
}

// Exists returns false if the error indicates the element doesn't
// exist.
func (v *kv) Exists(err error) bool {
	return ekv.Exists(err)
}

// StoreMapElement is not implemented for local KVs
func (v *kv) StoreMapElement(mapName, elementName string, mapVersion uint64,
	value *Object) error {
	return UnimplementedErr
}

// StoreMap is not implemented for local KVs
func (v *kv) StoreMap(mapName string, mapVersion uint64,
	values map[string]*Object) error {
	return UnimplementedErr
}

// GetMap is not implemented for local KVs
func (v *kv) GetMap(mapName string, version uint64) (
	map[string]*Object, error) {
	return nil, UnimplementedErr
}

// GetMapElement is not implemented for local KVs
func (v *kv) GetMapElement(mapName, element string, version uint64) (
	*Object, error) {
	return nil, UnimplementedErr
}

// Transaction is not implemented for local KVs
func (v *kv) Transaction(key string, op TransactionOperation, version uint64) (
	old *Object, existed bool, err error) {
	return nil, false, UnimplementedErr
}

// ListenOnRemoteKey is not implemented for local KVs
func (v *kv) ListenOnRemoteKey(key string, version uint64,
	callback KeyChangedByRemoteCallback) {
	jww.ERROR.Printf("%+v", errors.Wrapf(UnimplementedErr,
		"ListenOnRemoteKey"))
}

// ListenOnRemoteMap is not implemented for local KVs
func (v *kv) ListenOnRemoteMap(mapName string, version uint64,
	callback MapChangedByRemoteCallback) {
	jww.ERROR.Printf("%+v", errors.Wrapf(UnimplementedErr,
		"ListenOnRemoteMap"))
}
