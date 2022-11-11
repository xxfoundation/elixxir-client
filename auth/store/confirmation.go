////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/base64"
	"encoding/json"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	confirmationKeyPrefix      = "Confirmation/"
	currentConfirmationVersion = 0
)

type storedConfirm struct {
	Payload []byte
	Mac     []byte
	Keyfp   []byte
}

// StoreConfirmation saves the confirmation to storage for the given partner and
// fingerprint.
func (s *Store) StoreConfirmation(partner *id.ID,
	confirmationPayload, mac []byte, fp format.Fingerprint) error {
	confirm := storedConfirm{
		Payload: confirmationPayload,
		Mac:     mac,
		Keyfp:   fp[:],
	}

	confirmBytes, err := json.Marshal(&confirm)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   currentConfirmationVersion,
		Timestamp: netTime.Now(),
		Data:      confirmBytes,
	}

	return s.kv.Set(makeConfirmationKey(partner), obj)
}

// LoadConfirmation loads the confirmation for the given partner and fingerprint
// from storage.
func (s *Store) LoadConfirmation(partner *id.ID) (
	[]byte, []byte, format.Fingerprint, error) {
	obj, err := s.kv.Get(
		makeConfirmationKey(partner), currentConfirmationVersion)
	if err != nil {
		return nil, nil, format.Fingerprint{}, err
	}

	confirm := storedConfirm{}
	if err = json.Unmarshal(obj.Data, &confirm); err != nil {
		return nil, nil, format.Fingerprint{}, err
	}

	fp := format.Fingerprint{}
	copy(fp[:], confirm.Keyfp)

	return confirm.Payload, confirm.Mac, fp, nil
}

// DeleteConfirmation deletes the confirmation for the given partner and
// fingerprint from storage.
func (s *Store) DeleteConfirmation(partner *id.ID) error {
	return s.kv.Delete(
		makeConfirmationKey(partner), currentConfirmationVersion)
}

// makeConfirmationKey generates the key used to load and store confirmations
// for the partner and fingerprint.
func makeConfirmationKey(partner *id.ID) string {
	return confirmationKeyPrefix + base64.StdEncoding.EncodeToString(
		partner.Marshal())
}
