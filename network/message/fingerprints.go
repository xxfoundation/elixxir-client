///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"sync"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// FingerprintsManager is a thread-safe map, mapping format.Fingerprint's to
// a Handler object.
type FingerprintsManager struct {
	fpMap map[id.ID]map[format.Fingerprint]Processor
	sync.Mutex
}

// newFingerprints is a constructor function for the Fingerprints tracker.
func newFingerprints() *FingerprintsManager {
	return &FingerprintsManager{
		fpMap: make(map[id.ID]map[format.Fingerprint]Processor),
	}
}

// Pop returns the associated handler to the fingerprint and removes
// it from our list.
// CRITICAL: it is never ok to process a fingerprint twice. This is a security
// vulnerability.
func (f *FingerprintsManager) pop(clientID *id.ID,
	fingerprint format.Fingerprint) (
	Processor, bool) {
	f.Lock()
	defer f.Unlock()
	cid := *clientID
	if idFpMap, exists := f.fpMap[cid]; exists {
		if proc, exists := idFpMap[fingerprint]; exists {
			delete(f.fpMap[cid], fingerprint)
			if len(f.fpMap[cid]) == 0 {
				delete(f.fpMap, cid)
			}
			return proc, true
		}
	}

	return nil, false
}

// AddFingerprint is a thread-safe setter for the Fingerprints
// map. AddFingerprint maps the given fingerprint key to the handler
// value. If there is already an entry for this fingerprint, the
// method returns with no write operation.
func (f *FingerprintsManager) AddFingerprint(clientID *id.ID,
	fingerprint format.Fingerprint, mp Processor) error {
	f.Lock()
	defer f.Unlock()

	cid := *clientID

	if _, exists := f.fpMap[cid]; !exists {
		f.fpMap[cid] = make(
			map[format.Fingerprint]Processor)
	}

	if _, exists := f.fpMap[cid][fingerprint]; exists {
		return errors.Errorf("fingerprint %s already exists",
			fingerprint)
	}

	f.fpMap[cid][fingerprint] = mp
	return nil
}

// DeleteFingerprint is a thread-safe deletion operation on the Fingerprints map.
// It will remove the entry for the given fingerprint from the map.
func (f *FingerprintsManager) DeleteFingerprint(clientID *id.ID,
	fingerprint format.Fingerprint) {
	f.Lock()
	defer f.Unlock()

	cid := *clientID

	if _, exists := f.fpMap[cid]; exists {
		if _, exists = f.fpMap[cid][fingerprint]; exists {
			delete(f.fpMap[cid], fingerprint)
		}
		if len(f.fpMap[cid]) == 0 {
			delete(f.fpMap, cid)
		}
	}
}

// DeleteClientFingerprints is a thread-safe deletion operation on the
// fingerprints map. It will remove all entries for the given clientID from the
// map.
func (f *FingerprintsManager) DeleteClientFingerprints(clientID *id.ID) {
	f.Lock()
	defer f.Unlock()
	delete(f.fpMap, *clientID)
}
