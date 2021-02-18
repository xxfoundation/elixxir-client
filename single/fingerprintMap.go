///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"sync"
)

// fingerprintMap stores a map of fingerprint to key numbers.
type fingerprintMap struct {
	fps map[format.Fingerprint]uint64
	sync.Mutex
}

// newFingerprintMap returns a map of fingerprints generated from the provided
// key that is messageCount long.
func newFingerprintMap(dhKey *cyclic.Int, messageCount uint64) *fingerprintMap {
	fpm := &fingerprintMap{
		fps: make(map[format.Fingerprint]uint64, messageCount),
	}

	for i := uint64(0); i < messageCount; i++ {
		fp := singleUse.NewResponseFingerprint(dhKey, i)
		fpm.fps[fp] = i
	}

	return fpm
}

// getKey returns true and the corresponding key of the fingerprint exists in
// the map and returns false otherwise. If the fingerprint exists, then it is
// deleted prior to returning the key.
func (fpm *fingerprintMap) getKey(dhKey *cyclic.Int, fp format.Fingerprint) ([]byte, bool) {
	fpm.Lock()
	defer fpm.Unlock()

	num, exists := fpm.fps[fp]
	if !exists {
		return nil, false
	}

	// Delete found fingerprint
	delete(fpm.fps, fp)

	// Generate and return the key
	return singleUse.NewResponseKey(dhKey, num), true
}
