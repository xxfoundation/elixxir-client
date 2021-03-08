///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
)

const PrefixSeparator = "/"

// MakeKeyWithPrefix creates a key for a type of data with a unique
// identifier using a globally defined separator character.
func MakeKeyWithPrefix(dataType string, uniqueID string) string {
	return fmt.Sprintf("%s%s%s", dataType, PrefixSeparator, uniqueID)
}

// MakePartnerPrefix creates a string prefix
// to denote who a conversation or relationship is with
func MakePartnerPrefix(id *id.ID) string {
	return fmt.Sprintf("Partner:%v", id.String())
}

// Upgrade functions must be of this type
type Upgrade func(oldObject *Object) (*Object,
	error)


type root struct {
	data         ekv.KeyValue
}

// KV stores versioned data and Upgrade functions
type KV struct {
	r      *root
	prefix string
}

// Create a versioned key/value store backed by something implementing KeyValue
func NewKV(data ekv.KeyValue) *KV {
	newKV := KV{}
	root := root{}

	root.data = data

	newKV.r = &root

	return &newKV
}

// Get gets and upgrades data stored in the key/value store
// Make sure to inspect the version returned in the versioned object
func (v *KV) Get(key string, version uint64) (*Object, error) {
	key = v.makeKey(key, version)
	jww.TRACE.Printf("Get %p with key %v", v.r.data, key)
	// Get raw data
	result := Object{}
	err := v.r.data.Get(key, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type UpgradeTable struct{
	CurrentVersion uint64
	Table []Upgrade
}

// Get gets and upgrades data stored in the key/value store
// Make sure to inspect the version returned in the versioned object
func (v *KV) GetUpgrade(key string, ut UpgradeTable) (*Object, error) {
	version := ut.CurrentVersion
	key = v.makeKey(key, version)

	if uint64(len(ut.Table))!=version{
		jww.FATAL.Panicf("Cannot get upgrade for %s: table lengh (%d) " +
			"does not match current version (%d)", key, len(ut.Table),
			version)
	}
	var result *Object
	for ;version>=0;version--{

		key = v.makeKey(key, version)
		jww.TRACE.Printf("Get %p with key %v", v.r.data, key)

		// Get raw data
		result = &Object{}
		err := v.r.data.Get(key, result)
		if err != nil {
			jww.WARN.Printf("Failed to get keyvalue %s: %s", key, err)
		}else{
			break
		}
	}

	if version < 0{
		return nil, errors.Errorf("Failed to get key and upgrade it for %s", v.makeKey(key, ut.CurrentVersion))
	}

	var err error
	initialVersion := result.Version
	for result.Version<uint64(len(ut.Table)){
		oldVersion := result.Version
		result, err = ut.Table[oldVersion](result)
		if err!=nil{
			jww.FATAL.Panicf("failed to upgrade key %s from " +
				"version %v, initla version %v",  key, oldVersion,
				initialVersion)
		}
	}

	return result, nil
}


// delete removes a given key from the data store
func (v *KV) Delete(key string, version uint64) error {
	key = v.makeKey(key, version)
	jww.TRACE.Printf("delete %p with key %v", v.r.data, key)
	return v.r.data.Delete(key)
}

// Set upserts new data into the storage
// When calling this, you are responsible for prefixing the key with the correct
// type optionally unique id! Call MakeKeyWithPrefix() to do so.
func (v *KV) Set(key string, version uint64, object *Object) error {
	key = v.makeKey(key, version)
	jww.TRACE.Printf("Set %p with key %v", v.r.data, key)
	return v.r.data.Set(key, object)
}

//Returns a new KV with the new prefix
func (v *KV) Prefix(prefix string) *KV {
	kvPrefix := KV{
		r:      v.r,
		prefix: v.prefix + prefix + PrefixSeparator,
	}
	return &kvPrefix
}

//Returns the key with all prefixes appended
func (v *KV) GetFullKey(key string, version uint64) string {
	return v.makeKey(key, version)
}

func (v *KV)makeKey(key string, version uint64)string{
	return fmt.Sprintf("%s%s_%d", v.prefix, key, version)
}