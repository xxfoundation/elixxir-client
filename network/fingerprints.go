///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/primitives/format"
	"sync"
)

// Processor is an object which ties an interfaces.MessageProcessorFP
// to a lock. This prevents the processor from being used in multiple
// different threads.
type Processor struct {
	interfaces.MessageProcessorFP
	sync.Mutex
}

// NewFingerprints is a constructor function for the Processor object.
func newProcessor(mp interfaces.MessageProcessorFP) *Processor {
	return &Processor{
		MessageProcessorFP: mp,
		Mutex:              sync.Mutex{},
	}
}

// Fingerprints is a thread-safe map, mapping format.Fingerprint's to
// a Processor object.
type Fingerprints struct {
	fingerprints map[format.Fingerprint]*Processor
	sync.RWMutex
}

// NewFingerprints is a constructor function for the Fingerprints tracker.
func NewFingerprints() *Fingerprints {
	return &Fingerprints{
		fingerprints: make(map[format.Fingerprint]*Processor),
		RWMutex:      sync.RWMutex{},
	}
}

// Get is a thread-safe getter for the Fingerprints map. Get returns the mapped
// processor and true (representing that it exists in the map) if the provided
// fingerprint has an entry. Otherwise, Get returns nil and false.
func (f *Fingerprints) Get(fingerprint format.Fingerprint) (*Processor, bool) {
	f.RLock()
	defer f.RUnlock()
	fp, exists := f.fingerprints[fingerprint]
	if !exists {
		return nil, false
	}
	return fp, true
}

// AddFingerprint is a thread-safe setter for the Fingerprints map. AddFingerprint
// maps the given fingerprint key to the processor value. If there is already
// an entry for this fingerprint, the method returns with no write operation.
func (f *Fingerprints) AddFingerprint(fingerprint format.Fingerprint,
	processor interfaces.MessageProcessorFP) {
	f.Lock()
	defer f.Unlock()

	f.addFingerprint(fingerprint, processor)

}

// AddFingerprints is a thread-safe setter for multiple entries into
// the Fingerprints map. If there is not a 1:1 relationship between
// fingerprints and  processors slices (i.e. the lengths of these slices
// are equivalent), an error will be returned.
// Otherwise, each fingerprint is written to the associated processor.
// If there is already an entry for the given fingerprint/processor pair,
// no write operation will occur for this pair.
func (f *Fingerprints) AddFingerprints(fps []format.Fingerprint,
	processors []interfaces.MessageProcessorFP) error {
	f.Lock()
	defer f.Unlock()

	if len(fps) != len(processors) {
		return errors.Errorf("Canot perform a batch add when there are "+
			"not an equal amount of fingerprints and processors. "+
			"Given %d fingerprints and %d processors.", len(fps), len(processors))
	}

	for i, fp := range fps {
		f.addFingerprint(fp, processors[i])
	}

	return nil
}

// addFingerprint is a non-thread-safe helper function which writes a Processor
// to the given fingerprint key. If an entry already exists for this fingerprint key,
// no write operation occurs.
func (f *Fingerprints) addFingerprint(fingerprint format.Fingerprint,
	processor interfaces.MessageProcessorFP) {

	if _, exists := f.fingerprints[fingerprint]; exists {
		return
	}

	newMsgProc := newProcessor(processor)

	f.fingerprints[fingerprint] = newMsgProc
}

// RemoveFingerprint is a thread-safe deletion operation on the Fingerprints map.
// It will remove the entry for the given fingerprint from the map.
func (f *Fingerprints) RemoveFingerprint(fingerprint format.Fingerprint) {
	f.Lock()
	defer f.Unlock()

	delete(f.fingerprints, fingerprint)
}

// RemoveFingerprints is a thread-safe batch deletion operation on the Fingerprints map.
// It will remove the entries for the given fingerprints from the map.
func (f *Fingerprints) RemoveFingerprints(fingerprint []format.Fingerprint) {
	f.Lock()
	defer f.Unlock()

	for _, fp := range fingerprint {
		delete(f.fingerprints, fp)
	}
}
