////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package callbackTracker

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"strconv"
	"sync"
	"time"
)

// Manager tracks the callbacks for each transfer.
type Manager struct {
	// Map of transfers and their list of callbacks
	callbacks map[ftCrypto.TransferID][]*callbackTracker

	// List of multi stoppables used to stop callback trackers; each multi
	// stoppable contains a single stoppable for each callback.
	stops map[ftCrypto.TransferID]*stoppable.Multi

	mux sync.RWMutex
}

// NewManager initializes a new callback tracker Manager.
func NewManager() *Manager {
	m := &Manager{
		callbacks: make(map[ftCrypto.TransferID][]*callbackTracker),
		stops:     make(map[ftCrypto.TransferID]*stoppable.Multi),
	}

	return m
}

// AddCallback adds a callback to the list of callbacks for the given transfer
// ID and calls it regardless of the callback tracker status.
func (m *Manager) AddCallback(
	tid *ftCrypto.TransferID, cb callback, period time.Duration) {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Create new entries for this transfer ID if none exist
	if _, exists := m.callbacks[*tid]; !exists {
		m.callbacks[*tid] = []*callbackTracker{}
		m.stops[*tid] = stoppable.NewMulti("FileTransfer/" + tid.String())
	}

	// Generate the stoppable and add it to the transfer's multi stoppable
	stop := stoppable.NewSingle(makeStoppableName(tid, len(m.callbacks[*tid])))
	m.stops[*tid].Add(stop)

	// Create new callback tracker and add to the map
	ct := newCallbackTracker(cb, period, stop)
	m.callbacks[*tid] = append(m.callbacks[*tid], ct)

	// Call the callback
	go cb(nil)
}

// Call triggers each callback for the given transfer ID and passes along the
// given error.
func (m *Manager) Call(tid *ftCrypto.TransferID, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, cb := range m.callbacks[*tid] {
		go cb.call(err)
	}
}

// Delete stops all scheduled stoppables for the given transfer and deletes the
// callbacks from the map.
func (m *Manager) Delete(tid *ftCrypto.TransferID) {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Stop the stoppable if the stoppable still exists
	stop, exists := m.stops[*tid]
	if exists {
		if err := stop.Close(); err != nil {
			jww.ERROR.Printf("[FT] Failed to stop progress callbacks: %+v", err)
		}
	}

	// Delete callbacks and stoppables
	delete(m.callbacks, *tid)
	delete(m.stops, *tid)
}

// makeStoppableName generates a unique name for the callback stoppable.
func makeStoppableName(tid *ftCrypto.TransferID, callbackNum int) string {
	return tid.String() + "/" + strconv.Itoa(callbackNum)
}
