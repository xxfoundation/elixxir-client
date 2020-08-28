////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/ekv"
	"strings"
	"time"
)

const prefixKeySeparator = ":"

// MakeKeyWithPrefix creates a key for a type of data with a unique
// identifier using a globally defined separator character.
func MakeKeyWithPrefix(dataType string, uniqueID string) string {
	return fmt.Sprintf("%s%s%s", dataType, prefixKeySeparator, uniqueID)
}

// Object is used by VersionedKeyValue to keep track of
// versioning and time of storage
type Object struct {
	// Used to determine version Upgrade, if any
	Version uint64

	// Set when this object is written
	Timestamp time.Time

	// Serialized version of original object
	Data []byte
}

// Unmarshal deserializes a Object from a byte slice. It's used to
// make these storable in a KeyValue.
// Object exports all fields and they have simple types, so
// json.Unmarshal works fine.
func (v *Object) Unmarshal(data []byte) error {
	return json.Unmarshal(data, v)
}

// Marshal serializes a Object into a byte slice. It's used to
// make these storable in a KeyValue.
// Object exports all fields and they have simple types, so
// json.Marshal works fine.
func (v *Object) Marshal() []byte {
	d, err := json.Marshal(v)
	// Not being to marshal this simple object means something is really
	// wrong
	if err != nil {
		panic(fmt.Sprintf("Could not marshal: %+v", v))
	}
	return d
}

// Upgrade functions must be of this type
type Upgrade func(key string, oldObject *Object) (*Object,
	error)

// KV stores versioned data and Upgrade functions
type KV struct {
	upgradeTable map[string]Upgrade
	data         ekv.KeyValue
}

// Create a versioned key/value store backed by something implementing KeyValue
func NewKV(data ekv.KeyValue) *KV {
	newKV := new(KV)
	// Add new Upgrade functions to this Upgrade table
	newKV.upgradeTable = make(map[string]Upgrade)
	// All Upgrade functions should Upgrade to the latest version. You can
	// call older Upgrade functions if you need to. Upgrade functions don't
	// change the key or store the upgraded version of the data in the
	// key/value store. There's no mechanism built in for this -- users
	// should always make the key prefix before calling Set, and if they
	// want the upgraded data persisted they should call Set with the
	// upgraded data.
	newKV.upgradeTable[MakeKeyWithPrefix("test", "")] = func(key string,
		oldObject *Object) (*Object, error) {
		if oldObject.Version == 1 {
			return oldObject, nil
		}
		return &Object{
			Version: 1,
			// Upgrade functions don't need to update
			// the timestamp
			Timestamp: oldObject.Timestamp,
			Data: []byte("this object was upgraded from" +
				" v0 to v1"),
		}, nil
	}
	newKV.data = data
	return newKV
}

// Get gets and upgrades data stored in the key/value store
// Make sure to inspect the version returned in the versioned object
func (v *KV) Get(key string) (*Object, error) {
	// Get raw data
	result := Object{}
	err := v.data.Get(key, &result)
	if err != nil {
		return nil, err
	}
	// If the key starts with a version tag that we can find in the table,
	// we should call that function to Upgrade it
	for version, upgrade := range v.upgradeTable {
		if strings.HasPrefix(key, version) {
			// We should run this Upgrade function
			// The user of this function must update the key
			// based on the version returned in this
			// versioned object!
			return upgrade(key, &result)
		}
	}
	return &result, nil
}

// Delete removes a given key from the data store
func (v *KV) Delete(key string) error {
	return v.data.Delete(key)
}

// Set upserts new data into the storage
// When calling this, you are responsible for prefixing the key with the correct
// type optionally unique id! Call MakeKeyWithPrefix() to do so.
func (v *KV) Set(key string, object *Object) error {
	return v.data.Set(key, object)
}
