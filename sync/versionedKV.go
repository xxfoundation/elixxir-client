////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"sync"

	"gitlab.com/elixxir/client/v4/storage/versioned"
)

// VersionedKV wraps a [sync.KV] inside of a [storage.versioned.KV] interface.
type VersionedKV struct {
	// synchronizedPrefixes are prefixes that trigger remote
	// synchronization calls.
	synchronizedPrefixes []string

	// hasSynchronizedPrefix tells us we are in a prefix that is synchronized.
	inSynchronizedPrefix bool

	// remoteKV is the remote synching KV instance. This is used
	// when we intercept Set calls because we are synchronizing this prefix.
	remoteKV *KV
	// vkv is a versioned KV instance that wraps the remoteKV, used
	// for all local operations.
	vkv versioned.KV

	lck sync.Mutex
}

// NewVersionedKV returns a versioned KV instance wrapping a remote KV
func NewVersionedKV(remote *KV) *VersionedKV {
	v := &VersionedKV{
		synchronizedPrefixes: remote.synchronizedPrefixes,
		remoteKV:             remote,
		vkv:                  versioned.NewKV(remote),
	}
	v.inSynchronizedPrefix = v.hasSynchronizedPrefix()

	return v
}

///////////////////////////////////////////////////////////////////////////////
// Begin Remote KV [storage.versioned.KV] interface implementation functions
///////////////////////////////////////////////////////////////////////////////

// Get implements [storage.versioned.KV.Get]
func (r *VersionedKV) Get(key string, version uint64) (*versioned.Object, error) {
	return r.vkv.Get(key, version)
}

// GetAndUpgrade implemenets [storage.versioned.KV.GetAndUpgrade]
func (r *VersionedKV) GetAndUpgrade(key string, ut versioned.UpgradeTable) (
	*versioned.Object, error) {
	return r.vkv.GetAndUpgrade(key, ut)
}

// Delete implements [storage.versioned.KV.Delete]
func (r *VersionedKV) Delete(key string, version uint64) error {
	return r.vkv.Delete(key, version)
}

// Set implements [storage.versioned.KV.Set]
// NOT: When calling this, you are responsible for prefixing the
// key with the correct type optionally unique id! Call
// [versioned.MakeKeyWithPrefix] to do so.
// The [Object] should contain the versioning if you are
// maintaining such a functionality.
func (r *VersionedKV) Set(key string, object *versioned.Object) error {
	if r.inSynchronizedPrefix {
		k := r.vkv.GetFullKey(key, object.Version)
		return r.remoteKV.SetRemote(k, object.Marshal(), nil)
	}
	return r.vkv.Set(key, object)
}

// GetPrefix implements [storage.versioned.KV.GetPrefix]
func (r *VersionedKV) GetPrefix() string {
	return r.vkv.GetPrefix()
}

// HasPrefix implements [storage.versioned.KV.HasPrefix]
func (r *VersionedKV) HasPrefix(prefix string) bool {
	return r.vkv.HasPrefix(prefix)
}

// Prefix implements [storage.versioned.KV.Prefix]
func (r *VersionedKV) Prefix(prefix string) (versioned.KV, error) {
	newKV, err := r.vkv.Prefix(prefix)
	if err == nil {
		v := &VersionedKV{
			synchronizedPrefixes: r.synchronizedPrefixes,
			remoteKV:             r.remoteKV,
			vkv:                  newKV,
		}
		v.inSynchronizedPrefix = v.hasSynchronizedPrefix()
		return v, nil
	}
	return nil, err
}

// IsMemStore implements [storage.versioned.KV.IsMemStore]
func (r *VersionedKV) IsMemStore() bool {
	return r.vkv.IsMemStore()
}

// GetFullKey implements [storage.versioned.KV.GetFullKey]
func (r *VersionedKV) GetFullKey(key string, version uint64) string {
	return r.vkv.GetFullKey(key, version)
}

// Exists implements [storage.versioned.KV.Exists]
func (r *VersionedKV) Exists(err error) bool {
	return r.vkv.Exists(err)
}

///////////////////////////////////////////////////////////////////////////////
// End Remote KV [storage.versioned.KV] interface
///////////////////////////////////////////////////////////////////////////////

// SyncPrefix adds a prefix to the synchronization list. Please note that when
// you call [VersionedKV.Prefix] it creates a new object, so calls to this will
// not update other instances of the kv.
func (r *VersionedKV) SyncPrefix(prefix string) {
	r.lck.Lock()
	defer r.lck.Unlock()
	for i := range r.synchronizedPrefixes {
		if prefix == r.synchronizedPrefixes[i] {
			return
		}
	}
	r.synchronizedPrefixes = append(r.synchronizedPrefixes, prefix)
	r.inSynchronizedPrefix = r.hasSynchronizedPrefix()
}

func (r *VersionedKV) hasSynchronizedPrefix() bool {
	r.lck.Lock()
	defer r.lck.Unlock()
	for i := range r.synchronizedPrefixes {
		if r.vkv.HasPrefix(r.synchronizedPrefixes[i]) {
			return true
		}
	}

	return false
}
