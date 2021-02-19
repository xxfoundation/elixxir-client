///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"sync"
)

type receiveComm func(payload []byte, c Contact)

// callbackMap stores a list of possible callbacks that can be called when a
// message is received. To receive a transmission, each transmitted message must
// use the same tag as the registered callback. The tag fingerprint is a hash of
// a tag string that is used to identify the module that the transmission
// message belongs to. The tag can be anything, but should be long enough so
// that it is unique.
type callbackMap struct {
	callbacks map[singleUse.TagFP]receiveComm
	sync.RWMutex
}

// newCallbackMap initialises a new map.
func newCallbackMap() *callbackMap {
	return &callbackMap{
		callbacks: map[singleUse.TagFP]receiveComm{},
	}
}

// registerCallback adds a callback function to the map that associates it with
// its tag. The tag should be a unique string identifying the module using the
// callback.
func (cbm *callbackMap) registerCallback(tag string, callback receiveComm) {
	cbm.Lock()
	defer cbm.Unlock()

	tagFP := singleUse.NewTagFP(tag)
	cbm.callbacks[tagFP] = callback
}

// getCallback returns the callback registered with the given tag fingerprint.
// An error is returned if no associated callback exists.
func (cbm *callbackMap) getCallback(tagFP singleUse.TagFP) (receiveComm, error) {
	cbm.RLock()
	defer cbm.RUnlock()

	cb, exists := cbm.callbacks[tagFP]
	if !exists {
		return nil, errors.Errorf("no callback registered for the tag "+
			"fingerprint %s.", tagFP)
	}

	return cb, nil
}
