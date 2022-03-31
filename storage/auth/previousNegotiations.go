////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"bytes"
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"strings"
)

const (
	negotiationPartnersKey                = "NegotiationPartners"
	negotiationPartnersVersion            = 0
	negotiationFingerprintsKeyPrefix      = "NegotiationFingerprints/"
	currentNegotiationFingerprintsVersion = 0
)

// AddIfNew adds a new negotiation fingerprint if it is new.
// If the partner does not exist, it will add it and the new fingerprint and
// return newFingerprint = true, latest = true.
// If the partner exists and the fingerprint does not exist, add it adds it as
// the latest fingerprint and returns newFingerprint = true, latest = true
// If the partner exists and the fingerprint exists, return
// newFingerprint = false, latest = false or latest = true if it is the last one
// in the list.
func (s *Store) AddIfNew(partner *id.ID, negotiationFingerprint []byte) (
	newFingerprint, latest bool) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// If the partner does not exist, add it to the list and store a new
	// fingerprint to storage
	_, exists := s.previousNegotiations[*partner]
	if !exists {
		s.previousNegotiations[*partner] = struct{}{}

		// Save fingerprint to storage
		err := s.saveNegotiationFingerprints(partner, negotiationFingerprint)
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
		latest = true

		return
	}

	// get the fingerprint list from storage
	fingerprints, err := s.loadNegotiationFingerprints(partner)
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
			latest = i == len(fingerprints)-1

			return
		}
	}

	// If the partner does exist and the fingerprint does not exist, then add
	// the fingerprint to the list as latest
	fingerprints = append(fingerprints, negotiationFingerprint)
	err = s.saveNegotiationFingerprints(partner, fingerprints...)
	if err != nil {
		jww.FATAL.Panicf("Failed to save negotiation sentByFingerprints for "+
			"partner %s: %+v", partner, err)
	}

	newFingerprint = true
	latest = true

	return
}

// deletePreviousNegotiationPartner removes the partner, its sentByFingerprints, and
// its confirmations from memory and storage.
func (s *Store) deletePreviousNegotiationPartner(partner *id.ID) error {

	// Do nothing if the partner does not exist
	if _, exists := s.previousNegotiations[*partner]; !exists {
		return nil
	}

	// Delete partner from memory
	delete(s.previousNegotiations, *partner)

	// Delete partner from storage and return an error
	err := s.savePreviousNegotiations()
	if err != nil {
		return err
	}

	// Check if sentByFingerprints exist
	fingerprints, err := s.loadNegotiationFingerprints(partner)

	// If sentByFingerprints exist for this partner, delete them from storage and any
	// accompanying confirmations
	if err == nil {
		// Delete the fingerprint list from storage but do not return the error
		// until after attempting to delete the confirmations
		err = s.kv.Delete(makeNegotiationFingerprintsKey(partner),
			currentNegotiationFingerprintsVersion)

		// Delete all confirmations from storage
		for _, fp := range fingerprints {
			// Ignore the error since confirmations rarely exist
			_ = s.deleteConfirmation(partner, fp)
		}
	}

	// Return any error from loading or deleting sentByFingerprints
	return err
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
func (s *Store) newOrLoadPreviousNegotiations() (map[id.ID]struct{}, error) {
	obj, err := s.kv.Get(negotiationPartnersKey, negotiationPartnersVersion)
	if err != nil {
		if strings.Contains(err.Error(), "object not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			return make(map[id.ID]struct{}), nil
		}
		return nil, err
	}

	return unmarshalPreviousNegotiations(obj.Data), nil
}

// marshalPreviousNegotiations marshals the list of partners into a byte slice.
func marshalPreviousNegotiations(partners map[id.ID]struct{}) []byte {
	buff := bytes.NewBuffer(nil)
	buff.Grow(8 + (len(partners) * id.ArrIDLen))

	// Write number of partners to buffer
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(partners)))
	buff.Write(b)

	// Write each partner ID to buffer
	for partner := range partners {
		buff.Write(partner.Marshal())
	}

	return buff.Bytes()
}

// unmarshalPreviousNegotiations unmarshalls the marshalled byte slice into a
// list of partner IDs.
func unmarshalPreviousNegotiations(buf []byte) map[id.ID]struct{} {
	buff := bytes.NewBuffer(buf)

	numberOfPartners := binary.LittleEndian.Uint64(buff.Next(8))
	partners := make(map[id.ID]struct{}, numberOfPartners)

	for i := uint64(0); i < numberOfPartners; i++ {
		partner, err := id.Unmarshal(buff.Next(id.ArrIDLen))
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to unmarshal negotiation partner ID: %+v", err)
		}

		partners[*partner] = struct{}{}
	}

	return partners
}

// saveNegotiationFingerprints saves the list of sentByFingerprints for the given
// partner to storage.
func (s *Store) saveNegotiationFingerprints(
	partner *id.ID, fingerprints ...[]byte) error {

	obj := &versioned.Object{
		Version:   currentNegotiationFingerprintsVersion,
		Timestamp: netTime.Now(),
		Data:      marshalNegotiationFingerprints(fingerprints...),
	}

	return s.kv.Set(makeNegotiationFingerprintsKey(partner),
		currentNegotiationFingerprintsVersion, obj)
}

// loadNegotiationFingerprints loads the list of sentByFingerprints for the given
// partner from storage.
func (s *Store) loadNegotiationFingerprints(partner *id.ID) ([][]byte, error) {
	obj, err := s.kv.Get(makeNegotiationFingerprintsKey(partner),
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

// makeNegotiationFingerprintsKey generates the key used to load and store
// negotiation sentByFingerprints for the partner.
func makeNegotiationFingerprintsKey(partner *id.ID) string {
	return negotiationFingerprintsKeyPrefix + partner.String()
}
