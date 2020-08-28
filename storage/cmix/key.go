package cmix

import (
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

func NewKey(kv *versioned.KV, k *cyclic.Int, id *id.ID) (*key, error) {
	newKey := &key{
		kv:       kv,
		k:        k,
		storeKey: keyKey(id),
	}

	return newKey, newKey.save()
}

// returns the cyclic key
func (k *key) Get() *cyclic.Int {
	return k.k
}

// loads the key for the given node id from the versioned keystore
func loadKey(kv *versioned.KV, id *id.ID) (*key, error) {
	k := &key{}

	key := keyKey(id)

	obj, err := kv.Get(key)
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

	return k.kv.Set(k.storeKey, &obj)
}

// deletes the key from the versioned keystore
func (k *key) delete(kv *versioned.KV, id *id.ID) error {
	key := keyKey(id)
	return kv.Delete(key)
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

// generates the key used in the keystore for the given key
func keyKey(id *id.ID) string {
	return "nodeKey:" + id.String()
}
