///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ephemeral

import "gitlab.com/elixxir/client/storage/reception"

type IdentityStoreInterface interface {
	IsNewIdentity(identity reception.Identity) bool
	AddIdentity(identity reception.Identity) error
	InsertIdentity(identity reception.Identity) error
}

type IdentityStore struct {
	*reception.Store
	tracker map[reception.Identity]bool
}

func newTracker(store *reception.Store) *IdentityStore {
	return &IdentityStore{
		tracker: make(map[reception.Identity]bool),
		Store:   store,
	}
}

func (is *IdentityStore) IsNewIdentity(identity reception.Identity) bool {
	return is.tracker[identity]
}

func (is *IdentityStore) InsertIdentity(identity reception.Identity) error {
	is.tracker[identity] = true
	return is.AddIdentity(identity)
}
