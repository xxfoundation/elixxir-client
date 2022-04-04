////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/base64"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	confirmationKeyPrefix      = "Confirmation/"
	currentConfirmationVersion = 0
)

// StoreConfirmation saves the confirmation to storage for the given partner and
// fingerprint.
func (s *Store) StoreConfirmation(
	partner *id.ID, fingerprint, confirmation []byte) error {
	obj := &versioned.Object{
		Version:   currentConfirmationVersion,
		Timestamp: netTime.Now(),
		Data:      confirmation,
	}

	return s.kv.Set(makeConfirmationKey(partner, fingerprint),
		currentConfirmationVersion, obj)
}

// LoadConfirmation loads the confirmation for the given partner and fingerprint
// from storage.
func (s *Store) LoadConfirmation(partner *id.ID, fingerprint []byte) (
	[]byte, error) {
	obj, err := s.kv.Get(
		makeConfirmationKey(partner, fingerprint), currentConfirmationVersion)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

// deleteConfirmation deletes the confirmation for the given partner and
// fingerprint from storage.
func (s *Store) deleteConfirmation(partner *id.ID, fingerprint []byte) error {
	return s.kv.Delete(
		makeConfirmationKey(partner, fingerprint), currentConfirmationVersion)
}

// makeConfirmationKey generates the key used to load and store confirmations
// for the partner and fingerprint.
func makeConfirmationKey(partner *id.ID, fingerprint []byte) string {
	return confirmationKeyPrefix + base64.StdEncoding.EncodeToString(
		partner.Marshal()) + "/" + base64.StdEncoding.EncodeToString(fingerprint)
}
