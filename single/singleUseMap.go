///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"sync/atomic"
	"time"
)

// pending contains a map of all pending single-use states.
type pending struct {
	singleUse map[id.ID]*state
	sync.RWMutex
}

type ReplyComm func(payload []byte, err error)

// state contains the information and state of each single-use message that is
// being transmitted.
type state struct {
	dhKey    *cyclic.Int
	fpMap    *fingerprintMap // List of fingerprints for each response part
	c        *collator       // Collects all response message parts
	callback ReplyComm       // Returns the error status of the communication
	quitChan chan struct{}   // Sending on channel kills the timeout handler
}

// newPending creates a pending object with an empty map.
func newPending() *pending {
	return &pending{
		singleUse: map[id.ID]*state{},
	}
}

// newState generates a new state object with the fingerprint map and collator
// initialised.
func newState(dhKey *cyclic.Int, messageCount uint8, callback ReplyComm) *state {
	return &state{
		dhKey:    dhKey,
		fpMap:    newFingerprintMap(dhKey, uint64(messageCount)),
		c:        newCollator(uint64(messageCount)),
		callback: callback,
		quitChan: make(chan struct{}),
	}
}

// addState adds a new state to the map and starts a thread waiting for all the
// message parts or for the timeout to occur.
func (p *pending) addState(rid *id.ID, dhKey *cyclic.Int, maxMsgs uint8,
	callback ReplyComm, timeout time.Duration) (chan struct{}, *int32, error) {
	p.Lock()

	// Check if the state already exists
	if _, exists := p.singleUse[*rid]; exists {
		return nil, nil, errors.Errorf("a state already exists in the map with "+
			"the ID %s.", rid)
	}

	jww.DEBUG.Printf("Successfully added single-use state with the ID %s to "+
		"the map.", rid)

	// Add the state
	p.singleUse[*rid] = newState(dhKey, maxMsgs, callback)
	quitChan := p.singleUse[*rid].quitChan
	p.Unlock()

	// Create atomic which is set when the timeoutHandler thread is killed
	quit := int32(0)

	go p.timeoutHandler(rid, callback, timeout, quitChan, &quit)

	return quitChan, &quit, nil
}

// timeoutHandler waits for the signal to complete or times out and deletes the
// state.
func (p *pending) timeoutHandler(rid *id.ID, callback ReplyComm,
	timeout time.Duration, quitChan chan struct{}, quit *int32) {
	jww.DEBUG.Printf("Starting handler for sending single-use transmission "+
		"that will timeout after %s.", timeout)

	timer := time.NewTimer(timeout)

	// Signal on the atomic when this thread quits
	defer func() {
		atomic.StoreInt32(quit, 1)
	}()

	select {
	case <-quitChan:
		jww.DEBUG.Print("Single-use transmission timeout handler quitting.")
		return
	case <-timer.C:
		jww.WARN.Printf("Single-use transmission timeout handler timed out "+
			"after %s.", timeout)

		p.Lock()
		if _, exists := p.singleUse[*rid]; !exists {
			p.Unlock()
			return
		}
		delete(p.singleUse, *rid)

		p.Unlock()

		err := errors.Errorf("waiting for response to single-use transmission "+
			"timed out after %s.", timeout.String())
		jww.DEBUG.Printf("Deleted single-use from map. Calling callback with "+
			"error: %+v", err)

		callback(nil, err)
	}
}
