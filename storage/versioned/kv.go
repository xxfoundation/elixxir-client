///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"fmt"
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
func (v *KV) Get(key string) (*Object, error) {
	key = v.makeKey(key)
	jww.TRACE.Printf("Get %p with key %v", v.r.data, key)
	// Get raw data
	result := Object{}
	err := v.r.data.Get(key, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get gets and upgrades data stored in the key/value store
// Make sure to inspect the version returned in the versioned object
func (v *KV) GetUpgrade(key string, table []Upgrade) (*Object, error) {
	key = v.makeKey(key)
	jww.TRACE.Printf("Get %p with key %v", v.r.data, key)
	// Get raw data
	result := &Object{}
	err := v.r.data.Get(key, result)
	if err != nil {
		return nil, err
	}

	initialVersion := result.Version
	for result.Version<uint64(len(table)){
		oldVersion := result.Version
		result, err = table[oldVersion](result)
		if err!=nil{
			jww.FATAL.Panicf("failed to upgrade key %s from " +
				"version %v, initla version %v",  key, oldVersion,
				initialVersion)
		}
	}

	return result, nil
}


// delete removes a given key from the data store
func (v *KV) Delete(key string) error {
	key = v.makeKey(key)
	jww.TRACE.Printf("delete %p with key %v", v.r.data, key)
	return v.r.data.Delete(key)
}

// Set upserts new data into the storage
// When calling this, you are responsible for prefixing the key with the correct
// type optionally unique id! Call MakeKeyWithPrefix() to do so.
func (v *KV) Set(key string, object *Object) error {
	key = v.makeKey(key)
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
func (v *KV) GetFullKey(key string) string {
	return v.prefix + key
}

func (v *KV)makeKey(key string)string{
	return v.prefix + key
}