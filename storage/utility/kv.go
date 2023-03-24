package utility

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	accountSync "gitlab.com/elixxir/client/v4/sync"
)

const CurrentSessionVersion = 0

// KV wraps the [versioned.KV] and [accountSync.RemoteKV] objects and provides
// a generic Get & Set operation.
type KV struct {
	Local  *versioned.KV
	Remote *accountSync.RemoteKV
}

// Set writes to [accountSync.RemoteKV] if it is initialized (refer to
// [Session.InitRemoteKV]). Otherwise, it writes to [versioned.KV].
func (k *KV) Set(key string, data []byte) error {
	if k.Remote != nil {
		return k.Remote.Set(key, data, nil)
	}

	obj := &versioned.Object{}
	if err := json.Unmarshal(data, obj); err != nil {
		return err
	}

	return k.Local.Set(key, obj)
}

// Get reads from [accountSync.RemoteKV] if it is initialized (refer to
// [Session.InitRemoteKV]). Otherwise, it reads from [versioned.KV].
func (k *KV) Get(key string, v uint64) ([]byte, error) {
	if k.Remote != nil {
		data, err := k.Remote.Get(key)
		if err != nil {
			return nil, err
		}

		obj := &versioned.Object{}
		if err := json.Unmarshal(data, obj); err != nil {
			return nil, err
		}

		return obj.Data, nil
	}

	obj, err := k.Local.Get(key, v)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

// Delete removes the data stored at the given key from storage.
func (k *KV) Delete(key string, version uint64) error {
	return k.Local.Delete(key, version)
}

// Exists determines if the error message is known to report the key does not
// exist. Returns true if the error does not specify or it is nil and false
// otherwise.
func (k *KV) Exists(err error) bool {
	return k.Local.Exists(err)
}
