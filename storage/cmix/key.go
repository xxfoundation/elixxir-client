///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmix

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const currentKeyVersion = 0

type key struct {
	kv *versioned.KV
	k  *cyclic.Int

	storeKey string
}

func newKey(kv *versioned.KV, k *cyclic.Int, id *id.ID) *key {
	nk := &key{
		kv:       kv,
		k:        k,
		storeKey: keyKey(id),
	}

	if err := nk.save(); err != nil {
		jww.FATAL.Panicf("Failed to make nodeKey for %s: %s", id, err)
	}

	return nk
}

// returns the cyclic key
func (k *key) Get() *cyclic.Int {
	return k.k
}

// loads the key for the given node id from the versioned keystore
func loadKey(kv *versioned.KV, id *id.ID) (*key, error) {
	k := &key{}

	key := keyKey(id)

	obj, err := kv.Get(key, currentKeyVersion)
	if err != nil {
		return nil, err
	}

	err = k.unmarshal(obj.Data)

	if err != nil {
		return nil, err
	}

	return k, nil
}

// saves the key as the key for the given node ID in the passed keystore
func (k *key) save() error {
	now := time.Now()

	data, err := k.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentKeyVersion,
		Timestamp: now,
		Data:      data,
	}

	return k.kv.Set(k.storeKey, currentKeyVersion, &obj)
}

// deletes the key from the versioned keystore
func (k *key) delete(kv *versioned.KV, id *id.ID) {
	key := keyKey(id)
	if err := kv.Delete(key, currentKeyVersion); err != nil {
		jww.FATAL.Panicf("Failed to delete key %s: %s", k, err)
	}
}

// makes a binary representation of the given key in the keystore
func (k *key) marshal() ([]byte, error) {
	return k.k.GobEncode()
}

// resets the data of the key from the binary representation of the key passed in
func (k *key) unmarshal(b []byte) error {
	k.k = &cyclic.Int{}
	return k.k.GobDecode(b)
}

// Adheres to the stringer interface
func (k *key) String() string {
	return k.storeKey

}

// generates the key used in the keystore for the given key
func keyKey(id *id.ID) string {
	return "nodeKey:" + id.String()
}
