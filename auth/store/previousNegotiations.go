////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	negotiationPartnersKey                = "NegotiationPartners"
	negotiationPartnersVersion            = 0
	negotiationFingerprintsKeyPrefix      = "NegotiationFingerprints/"
	currentNegotiationFingerprintsVersion = 0
)

// CheckIfNegotiationIsNew adds a new negotiation fingerprint if it is new.
// If the partner does not exist, it will add it and the new fingerprint and
// return newFingerprint = true.
// If the partner exists and the fingerprint does not exist, add it adds it as
// the latest fingerprint and returns newFingerprint = true,
// If the partner exists and the fingerprint exists, return
// newFingerprint = false
// in all cases it will return the position of the fingerprint, with the newest
// always at position 0
func (s *Store) CheckIfNegotiationIsNew(partner *id.ID, negotiationFingerprint []byte) (
	newFingerprint bool, position uint) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// If the partner does not exist, add it to the list and store a new
	// fingerprint to storage
	_, exists := s.previousNegotiations[*partner]
	if !exists {
		s.previousNegotiations[*partner] = true

		// Save fingerprint to storage
		err := saveNegotiationFingerprints(partner, s.kv, negotiationFingerprint)
		if err != nil {
			jww.FATAL.Panicf("Failed to save negotiation sentByFingerprints for "+
				"partner %s: %+v", partner, err)
		}

		// Save partner list to storage
		err = s.savePreviousNegotiations()
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to save negotiation partners %s: %+v", partner, err)
		}

		newFingerprint = true
		position = 0
		return
	}

	// get the fingerprint list from storage
	fingerprints, err := loadNegotiationFingerprints(partner, s.kv)
	if err != nil {
		jww.FATAL.Panicf("Failed to load negotiation sentByFingerprints for "+
			"partner %s: %+v", partner, err)
	}

	// If the partner does exist and the fingerprint exists, then make no
	// changes to the list
	for i, fp := range fingerprints {
		if bytes.Equal(fp, negotiationFingerprint) {
			newFingerprint = false

			// Latest = true if it is the last fingerprint in the list
			lastPost := len(fingerprints) - 1
			position = uint(lastPost - i)

			return
		}
	}

	// If the partner does exist and the fingerprint does not exist, then add
	// the fingerprint to the list as latest
	fingerprints = append(fingerprints, negotiationFingerprint)
	err = saveNegotiationFingerprints(partner, s.kv, fingerprints...)
	if err != nil {
		jww.FATAL.Panicf("Failed to save negotiation sentByFingerprints for "+
			"partner %s: %+v", partner, err)
	}

	newFingerprint = true
	position = 0

	return
}

// savePreviousNegotiations saves the list of previousNegotiations partners to
// storage.
func (s *Store) savePreviousNegotiations() error {
	obj := &versioned.Object{
		Version:   negotiationPartnersVersion,
		Timestamp: netTime.Now(),
		Data:      marshalPreviousNegotiations(s.previousNegotiations),
	}

	return s.kv.Set(negotiationPartnersKey, negotiationPartnersVersion, obj)
}

// newOrLoadPreviousNegotiations loads the list of previousNegotiations partners
// from storage.
func (s *Store) newOrLoadPreviousNegotiations() (map[id.ID]bool, error) {

	obj, err := s.kv.Get(negotiationPartnersKey, negotiationPartnersVersion)
	if err != nil {
		return nil, err
	}

	return unmarshalPreviousNegotiations(obj.Data)
}

// marshalPreviousNegotiations marshals the list of partners into a byte slice.
func marshalPreviousNegotiations(partners map[id.ID]bool) []byte {
	toMarshal := make([]id.ID, 0, len(partners))

	for partner := range partners {
		toMarshal = append(toMarshal, partner)
	}

	b, err := json.Marshal(&toMarshal)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal previous negotations", err)
	}

	return b
}

// unmarshalPreviousNegotiations unmarshalls the marshalled json into a
//// list of partner IDs.
func unmarshalPreviousNegotiations(b []byte) (map[id.ID]bool,
	error) {
	unmarshal := make([]id.ID, 0)

	if err := json.Unmarshal(b, &unmarshal); err != nil {
		return nil, err
	}

	partners := make(map[id.ID]bool)

	for _, aid := range unmarshal {
		partners[aid] = true
	}

	return partners, nil
}

// saveNegotiationFingerprints saves the list of sentByFingerprints for the given
// partner to storage.
func saveNegotiationFingerprints(
	partner *id.ID, kv *versioned.KV, fingerprints ...[]byte) error {

	obj := &versioned.Object{
		Version:   currentNegotiationFingerprintsVersion,
		Timestamp: netTime.Now(),
		Data:      marshalNegotiationFingerprints(fingerprints...),
	}

	return kv.Set(makeNegotiationFingerprintsKey(partner),
		currentNegotiationFingerprintsVersion, obj)
}

// loadNegotiationFingerprints loads the list of sentByFingerprints for the given
// partner from storage.
func loadNegotiationFingerprints(partner *id.ID, kv *versioned.KV) ([][]byte, error) {
	obj, err := kv.Get(makeNegotiationFingerprintsKey(partner),
		currentNegotiationFingerprintsVersion)
	if err != nil {
		return nil, err
	}

	return unmarshalNegotiationFingerprints(obj.Data), nil
}

// marshalNegotiationFingerprints marshals the list of sentByFingerprints into a byte
// slice for storage.
func marshalNegotiationFingerprints(fingerprints ...[]byte) []byte {
	buff := bytes.NewBuffer(nil)
	buff.Grow(8 + (len(fingerprints) * auth.NegotiationFingerprintLen))

	// Write number of sentByFingerprints to buffer
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(fingerprints)))
	buff.Write(b)

	for _, fp := range fingerprints {
		// Write fingerprint to buffer
		buff.Write(fp[:auth.NegotiationFingerprintLen])
	}

	return buff.Bytes()
}

// unmarshalNegotiationFingerprints unmarshalls the marshalled byte slice into a
// list of sentByFingerprints.
func unmarshalNegotiationFingerprints(buf []byte) [][]byte {
	buff := bytes.NewBuffer(buf)

	listLen := binary.LittleEndian.Uint64(buff.Next(8))
	fingerprints := make([][]byte, listLen)

	for i := range fingerprints {
		fingerprints[i] = make([]byte, auth.NegotiationFingerprintLen)
		copy(fingerprints[i], buff.Next(auth.NegotiationFingerprintLen))
	}

	return fingerprints
}

// makeOldNegotiationFingerprintsKey generates the key used to load and store
// negotiation sentByFingerprints for the partner.
func makeNegotiationFingerprintsKey(partner *id.ID) string {
	return negotiationFingerprintsKeyPrefix +
		string(base64.StdEncoding.EncodeToString(partner.Marshal()))
}
