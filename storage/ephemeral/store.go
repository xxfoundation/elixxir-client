///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package ephemeral

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"sync"
	"time"
)

const (
	storePrefix = "ephemeralTimestamp"
    ephemeralVersion = 0
    timestampKey = "timestampKeyStore"
)

type Store struct {
	kv        *versioned.KV
	// Timestamp of last check for ephemeral Ids
	timestamp time.Time
	mux       sync.RWMutex
}

// NewStore creates a new store.
func NewStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(storePrefix)
	s := &Store{
		kv:        kv,
		timestamp: time.Time{},
	}

	return s, s.save()
}

// loads the ephemeral storage object
func LoadStore(kv *versioned.KV) (*Store, error) {
	kv = kv.Prefix(storePrefix)
	s := &Store{
		timestamp: time.Time{},
		kv:        kv,
	}

	obj, err := kv.Get(timestampKey)
	if err != nil {
		return nil, err
	}

	if err = s.unmarshal(obj.Data); err != nil {
		return nil, err
	}

	return s, nil
}

// Returns the stored timestamp. If a timestamp is empty, we check disk.
// If the disk's timestamp is empty. we return an error.
// Otherwise, we return the valid timestamp found
func (s *Store) GetTimestamp() (time.Time, error) {
	s.mux.RLock()
	defer 	s.mux.RUnlock()

	ts := s.timestamp

	// Check that t
	if ts.Equal(time.Time{}) {
		obj, err := s.kv.Get(timestampKey)
		if err != nil {
			return time.Time{}, err
		}

		ts = time.Time{}
		if err := ts.UnmarshalBinary(obj.Data); err != nil {
			return time.Time{}, err
		}

		// If still an empty time object, then no timestamp exists
		if ts.Equal(time.Time{}) {
			return time.Time{}, errors.Errorf("No timestamp has been found")
		}
	}

	return ts, nil
}

// Updates the stored time stamp with the time passed in
func (s *Store) UpdateTimestamp(ts time.Time) error {
	s.mux.Lock()
	defer s.mux.Unlock()


	s.timestamp = ts

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to update timestamp of last check for ephemeral IDs")
	}

	return nil
}

// stores the ephemeral store
func (s *Store) save() error {

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   ephemeralVersion,
		Timestamp: time.Now(),
		Data:      data,
	}

	return s.kv.Set(timestampKey, &obj)
}

// builds a byte representation of the store
func (s *Store) marshal() ([]byte, error) {
	return s.timestamp.MarshalBinary()
}

// restores the data for a store from the byte representation of the store
func (s *Store) unmarshal(b []byte) error {
	ts := time.Time{}
	if err := ts.UnmarshalBinary(b); err != nil {
		return err
	}

	s.timestamp = ts

	return nil
}
