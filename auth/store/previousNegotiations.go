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
	"strings"
)

const (
	negotiationPartnersKey                = "NegotiationPartners"
	negotiationPartnersVersion            = 1
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
func (s *Store) CheckIfNegotiationIsNew(partner, myID *id.ID, negotiationFingerprint []byte) (
	newFingerprint bool, position uint) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// If the partner does not exist, add it to the list and store a new
	// fingerprint to storage
	aid := makeAuthIdentity(partner, myID)
	_, exists := s.previousNegotiations[aid]
	if !exists {
		s.previousNegotiations[aid] = true

		// Save fingerprint to storage
		err := saveNegotiationFingerprints(partner, myID, s.kv, negotiationFingerprint)
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
	fingerprints, err := loadNegotiationFingerprints(partner, myID, s.kv, myID.Cmp(s.defaultID))
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
	err = saveNegotiationFingerprints(partner, myID, s.kv, fingerprints...)
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
func (s *Store) newOrLoadPreviousNegotiations() (map[authIdentity]bool, error) {

	obj, err := s.kv.Get(negotiationPartnersKey, negotiationPartnersVersion)
	if err != nil {
		if strings.Contains(err.Error(), "object not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			obj, err = s.kv.Get(negotiationPartnersKey, 0)
			if err != nil {
				if strings.Contains(err.Error(), "object not found") ||
					strings.Contains(err.Error(), "no such file or directory") {
					return make(map[authIdentity]bool), nil
				} else {
					return nil, err
				}
			}
			return unmarshalOldPreviousNegotiations(obj.Data, s.defaultID), nil
		} else {
			return nil, err
		}
	}

	return unmarshalPreviousNegotiations(obj.Data)
}

// marshalPreviousNegotiations marshals the list of partners into a byte slice.
func marshalPreviousNegotiations(partners map[authIdentity]bool) []byte {
	toMarshal := make([]authIdentity, 0, len(partners))

	for aid := range partners {
		toMarshal = append(toMarshal, aid)
	}

	b, err := json.Marshal(&toMarshal)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal previous negotations", err)
	}

	return b
}

// unmarshalPreviousNegotiations unmarshalls the marshalled json into a
//// list of partner IDs.
func unmarshalPreviousNegotiations(b []byte) (map[authIdentity]bool,
	error) {
	unmarshal := make([]authIdentity, 0)

	if err := json.Unmarshal(b, &unmarshal); err != nil {
		return nil, err
	}

	partners := make(map[authIdentity]bool)

	for _, aid := range unmarshal {
		partners[aid] = true
	}

	return partners, nil
}

// unmarshalOldPreviousNegotiations unmarshalls the marshalled json into a
// list of partner IDs.
func unmarshalOldPreviousNegotiations(buf []byte, defaultID *id.ID) map[authIdentity]bool {
	buff := bytes.NewBuffer(buf)

	numberOfPartners := binary.LittleEndian.Uint64(buff.Next(8))
	partners := make(map[authIdentity]bool, numberOfPartners)

	for i := uint64(0); i < numberOfPartners; i++ {
		partner, err := id.Unmarshal(buff.Next(id.ArrIDLen))
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to unmarshal negotiation partner ID: %+v", err)
		}

		partners[makeAuthIdentity(partner, defaultID)] = false
	}

	return partners
}

// saveNegotiationFingerprints saves the list of sentByFingerprints for the given
// partner to storage.
func saveNegotiationFingerprints(
	partner, myID *id.ID, kv *versioned.KV, fingerprints ...[]byte) error {

	obj := &versioned.Object{
		Version:   currentNegotiationFingerprintsVersion,
		Timestamp: netTime.Now(),
		Data:      marshalNegotiationFingerprints(fingerprints...),
	}

	return kv.Set(makeNegotiationFingerprintsKey(partner, myID),
		currentNegotiationFingerprintsVersion, obj)
}

// loadNegotiationFingerprints loads the list of sentByFingerprints for the given
// partner from storage.
func loadNegotiationFingerprints(partner, myID *id.ID, kv *versioned.KV, possibleOld bool) ([][]byte, error) {
	obj, err := kv.Get(makeNegotiationFingerprintsKey(partner, myID),
		currentNegotiationFingerprintsVersion)
	if err != nil {
		if possibleOld {
			obj, err = kv.Get(makeOldNegotiationFingerprintsKey(partner),
				currentNegotiationFingerprintsVersion)
			if err != nil {
				return nil, err
			}
			if err = kv.Set(makeNegotiationFingerprintsKey(partner, myID),
				currentNegotiationFingerprintsVersion, obj); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
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
func makeOldNegotiationFingerprintsKey(partner *id.ID) string {
	return negotiationFingerprintsKeyPrefix +
		string(base64.StdEncoding.EncodeToString(partner.Marshal()))
}

// makeNegotiationFingerprintsKey generates the key used to load and store
// negotiation sentByFingerprints for the partner.
func makeNegotiationFingerprintsKey(partner, myID *id.ID) string {
	return negotiationFingerprintsKeyPrefix +
		string(base64.StdEncoding.EncodeToString(partner.Marshal())) +
		string(base64.StdEncoding.EncodeToString(myID.Marshal()))
}
