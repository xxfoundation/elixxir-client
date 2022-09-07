////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"sync"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/crypto/csprng"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// FingerprintsManager is a thread-safe map, mapping format.Fingerprint's to a
// Handler object.
type FingerprintsManager struct {
	fpMap      map[id.ID]map[format.Fingerprint]Processor
	standardID *id.ID
	sync.Mutex
}

// newFingerprints is a constructor function for the FingerprintsManager.
func newFingerprints(standardID *id.ID) *FingerprintsManager {
	return &FingerprintsManager{
		fpMap:      make(map[id.ID]map[format.Fingerprint]Processor),
		standardID: standardID,
	}
}

// pop returns the associated handler to the fingerprint and removes it from the
// list.
// CRITICAL: It is never ok to process a fingerprint twice. This is a security
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

// AddFingerprint is a thread-safe setter for the FingerprintsManager map.
// AddFingerprint maps the given fingerprint key to the handler value. If there
// is already an entry for this fingerprint, the method returns with no write
// operation.
// If a nil identity is passed, it will automatically use the default
// identity in the session
func (f *FingerprintsManager) AddFingerprint(clientID *id.ID,
	fingerprint format.Fingerprint, mp Processor) error {
	jww.TRACE.Printf("AddFingerprint: %s", fingerprint)
	f.Lock()
	defer f.Unlock()

	if clientID == nil {
		clientID = f.standardID
	}

	cid := *clientID

	if _, exists := f.fpMap[cid]; !exists {
		f.fpMap[cid] = make(
			map[format.Fingerprint]Processor)
	}

	if _, exists := f.fpMap[cid][fingerprint]; exists {
		return errors.Errorf("fingerprint %s already exists", fingerprint)
	}

	f.fpMap[cid][fingerprint] = mp
	return nil
}

// DeleteFingerprint is a thread-safe deletion operation on the Fingerprints
// map. It will remove the entry for the given fingerprint from the map.
func (f *FingerprintsManager) DeleteFingerprint(clientID *id.ID,
	fingerprint format.Fingerprint) {
	f.Lock()
	defer f.Unlock()

	if clientID == nil {
		clientID = f.standardID
	}

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

func RandomFingerprint(rng csprng.Source) format.Fingerprint {
	fpBuf := make([]byte, format.KeyFPLen)
	if _, err := rng.Read(fpBuf); err != nil {
		jww.FATAL.Panicf("Failed to generate fingerprint: %+v", err)
	}

	// The first bit must be 0.
	fpBuf[0] &= 0x7F

	return format.NewFingerprint(fpBuf)
}
