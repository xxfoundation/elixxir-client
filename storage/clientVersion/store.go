////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package clientVersion

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	prefix       = "clientVersionStore"
	storeKey     = "clientVersion"
	storeVersion = 0
)

// Store stores the version of the client's storage.
type Store struct {
	version version.Version
	kv      *utility.KV
	sync.RWMutex
}

// NewStore returns a new clientVersion store.
func NewStore(newVersion version.Version, kv *utility.KV) (*Store, error) {
	s := &Store{
		version: newVersion,
		kv:      kv,
	}

	return s, s.save()
}

// LoadStore loads the clientVersion storage object.
func LoadStore(kv *utility.KV) (*Store, error) {
	s := &Store{
		kv: kv,
	}

	data, err := s.kv.Get(storeKey, storeVersion)
	if err != nil {
		return nil, err
	}

	s.version, err = version.ParseVersion(string(data))
	if err != nil {
		return nil, errors.Errorf("failed to parse client version: %+v", err)
	}

	return s, nil
}

// Get returns the stored version.
func (s *Store) Get() version.Version {
	s.RLock()
	defer s.RUnlock()

	return s.version
}

// CheckUpdateRequired determines if the storage needs to be upgraded to the new
// client version. It returns true if an update is required (new > stored) and
// false otherwise. The old stored version is returned to be used to determine
// how to upgrade storage. If the new version is older than the stored version,
// an error is returned.
func (s *Store) CheckUpdateRequired(newVersion version.Version) (bool, version.Version, error) {
	s.Lock()
	defer s.Unlock()

	oldVersion := s.version
	diff := version.Cmp(oldVersion, newVersion)

	switch {
	case diff < 0:
		return true, oldVersion, s.update(newVersion)
	case diff > 0:
		return false, oldVersion, errors.Errorf("new version (%s) is older "+
			"than stored version (%s).", &newVersion, &oldVersion)
	default:
		return false, oldVersion, nil
	}
}

// update replaces the current version with the new version if it is newer. Note
// that this function does not take a lock.
func (s *Store) update(newVersion version.Version) error {
	jww.DEBUG.Printf("Updating stored client version from %s to %s.",
		&s.version, &newVersion)

	// Update version
	s.version = newVersion

	// Save new version to storage
	return s.save()
}

// save stores the clientVersion store. Note that this function does not take
// a lock.
func (s *Store) save() error {
	timeNow := netTime.Now()

	obj := &versioned.Object{
		Version:   storeVersion,
		Timestamp: timeNow,
		Data:      []byte(s.version.String()),
	}

	return s.kv.Set(storeKey, obj.Marshal())
}
