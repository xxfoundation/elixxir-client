////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"fmt"
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

const (
	PrefixContainingSeparatorErr = "cannot accept prefix with the default separator"
	DuplicatePrefixErr           = "prefix has already been added, cannot overwrite"
)

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

	// GetPrefix returns the full Prefix of the KV
	GetPrefix() string

	// HasPrefix returns whether this prefix exists in the KV
	HasPrefix(prefix string) bool

	// Prefix returns a new KV with the new prefix appending
	Prefix(prefix string) (KV, error)

	// IsMemStore returns true if the underlying KV is memory based
	IsMemStore() bool

	// GetFullKey returns the key with all prefixes appended
	GetFullKey(key string, version uint64) string

	// Exists returns if the error indicates a KV error showing
	// the key exists.
	Exists(err error) bool
}

// MakePartnerPrefix creates a string prefix
// to denote who a conversation or relationship is with
func MakePartnerPrefix(id *id.ID) string {
	return fmt.Sprintf("Partner:%v", id.String())
}

// Upgrade functions must be of this type
type Upgrade func(oldObject *Object) (*Object,
	error)

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

// Prefix returns a new kv with the new prefix appending.
func (v *kv) Prefix(prefix string) (KV, error) {
	//// Reject invalid prefixes
	if strings.Contains(prefix, PrefixSeparator) {
		return nil, errors.Errorf(PrefixContainingSeparatorErr)
	}

	// Reject duplicate prefixes
	if v.HasPrefix(prefix) {
		return nil, errors.Errorf(DuplicatePrefixErr)
	}

	v.offset++

	kvPrefix := kv{
		r:         v.r,
		prefix:    v.prefix + prefix + PrefixSeparator,
		prefixMap: v.prefixMap,
	}

	v.prefixMap[kvPrefix.prefix] = v.offset

	return &kvPrefix, nil
}

func (v *kv) IsMemStore() bool {
	_, success := v.r.data.(*ekv.Memstore)
	return success
}

// Returns the key with all prefixes appended
func (v *kv) GetFullKey(key string, version uint64) string {
	return v.makeKey(key, version)
}

func (v *kv) makeKey(key string, version uint64) string {
	return fmt.Sprintf("%s%s_%d", v.prefix, key, version)
}

// Exists returns false if the error indicates the element doesn't
// exist.
func (v *kv) Exists(err error) bool {
	return ekv.Exists(err)
}
