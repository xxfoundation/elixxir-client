////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"bytes"
	"encoding/binary"
	"gitlab.com/elixxir/crypto/cyclic"
)

type key struct {
	k          *cyclic.Int
	keyId      []byte
	validUntil uint64
}

func newKey(k *cyclic.Int, validUntil uint64, keyId []byte) *key {
	return &key{
		k:          k,
		keyId:      keyId,
		validUntil: validUntil,
	}
}

// get returns the cyclic key.
func (k *key) get() *cyclic.Int {
	return k.k
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
