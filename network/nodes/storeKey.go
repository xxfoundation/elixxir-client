///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"bytes"
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentKeyVersion = 0

type key struct {
	kv         *versioned.KV
	k          *cyclic.Int
	keyId      []byte
	validUntil uint64
	storeKey   string
}

func newKey(kv *versioned.KV, k *cyclic.Int, id *id.ID, validUntil uint64,
	keyId []byte) *key {
	nk := &key{
		kv:         kv,
		k:          k,
		keyId:      keyId,
		validUntil: validUntil,
		storeKey:   keyKey(id),
	}

	if err := nk.save(); err != nil {
		jww.FATAL.Panicf("Failed to make nodeKey for %s: %s", id, err)
	}

	return nk
}

// get returns the cyclic key.
func (k *key) get() *cyclic.Int {
	return k.k
}

// loadKey loads the key for the given node ID from the versioned keystore.
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

// save stores the key as the key for the given nodes ID in the keystore.
func (k *key) save() error {
	now := netTime.Now()

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

// delete deletes the key from the versioned keystore.
func (k *key) delete(kv *versioned.KV, id *id.ID) {
	key := keyKey(id)
	if err := kv.Delete(key, currentKeyVersion); err != nil {
		jww.FATAL.Panicf("Failed to delete key %s: %s", k, err)
	}
}

// marshal makes a binary representation of the given key and key values in the
// keystore.
func (k *key) marshal() ([]byte, error) {
	buff := bytes.NewBuffer(nil)
	keyBytes, err := k.k.GobEncode()
	if err != nil {
		return nil, err
	}

	// Write key length
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(keyBytes)))
	buff.Write(b)

	// Write key
	buff.Write(keyBytes)

	// Write the keyId length
	binary.LittleEndian.PutUint64(b, uint64(len(k.keyId)))
	buff.Write(b)

	// Write keyID
	buff.Write(k.keyId)

	// Write valid until
	binary.LittleEndian.PutUint64(b, k.validUntil)
	buff.Write(b)

	return buff.Bytes(), nil
}

// unmarshal resets the data of the key from the binary representation of the
// key passed in.
func (k *key) unmarshal(b []byte) error {
	buff := bytes.NewBuffer(b)

	// get the key length
	keyLen := int(binary.LittleEndian.Uint64(buff.Next(8)))

	// Decode the key length
	k.k = &cyclic.Int{}
	err := k.k.GobDecode(buff.Next(keyLen))
	if err != nil {
		return err
	}

	// get the keyID length
	keyIDLen := int(binary.LittleEndian.Uint64(buff.Next(8)))
	k.keyId = buff.Next(keyIDLen)

	// get the valid until value
	k.validUntil = binary.LittleEndian.Uint64(buff.Next(8))

	return nil
}

// String returns a string representation of key. This functions adheres to the
// fmt.Stringer interface.
func (k *key) String() string {
	return k.storeKey
}

// keyKey generates the key used in the keystore for the given key.
func keyKey(id *id.ID) string {
	return "nodeKey:" + id.String()
}
