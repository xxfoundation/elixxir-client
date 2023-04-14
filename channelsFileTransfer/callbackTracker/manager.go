////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package callbackTracker

import (
	"strconv"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

// Manager tracks the callbacks for each transfer.
type Manager struct {
	// Map of files and their list of registered callbacks
	callbacks map[ftCrypto.ID][]*callbackTracker

	// List of multi stoppables used to stop callback trackers; each multi
	// stoppable contains a single stoppable for each callback.
	stops map[ftCrypto.ID]*stoppable.Multi

	mux sync.RWMutex
}

// NewManager initializes a new callback tracker Manager.
func NewManager() *Manager {
	m := &Manager{
		callbacks: make(map[ftCrypto.ID][]*callbackTracker),
		stops:     make(map[ftCrypto.ID]*stoppable.Multi),
	}

	return m
}

// AddCallback adds a callback to the list of callbacks for the given transfer
// ID and calls it regardless of the callback tracker status.
func (m *Manager) AddCallback(
	fid ftCrypto.ID, cb callback, period time.Duration) {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Create new entries for this file ID if none exist
	if _, exists := m.callbacks[fid]; !exists {
		m.callbacks[fid] = []*callbackTracker{}
		m.stops[fid] = stoppable.NewMulti("FileTransfer/" + fid.String())
	}

	// Generate the stoppable and add it to the transfer's multi stoppable
	stop := stoppable.NewSingle(makeStoppableName(fid, len(m.callbacks[fid])))
	m.stops[fid].Add(stop)

	// Create new callback tracker and add to the map
	ct := newCallbackTracker(cb, period, stop)
	m.callbacks[fid] = append(m.callbacks[fid], ct)

	// Call the callback
	go cb(nil)
}

// Call triggers each callback for the given file ID and passes along the given
// error.
func (m *Manager) Call(fid ftCrypto.ID, err error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, cb := range m.callbacks[fid] {
		go cb.call(err)
	}
}

// Delete stops all scheduled stoppables for the given transfer and deletes the
// callbacks from the map.
func (m *Manager) Delete(fid ftCrypto.ID) {
	m.mux.Lock()
	defer m.mux.Unlock()

	// Stop the stoppable if the stoppable still exists
	stop, exists := m.stops[fid]
	if exists {
		if err := stop.Close(); err != nil {
			jww.ERROR.Printf("[FT] Failed to stop progress callbacks: %+v", err)
		}
	}

	// Delete callbacks and stoppables
	delete(m.callbacks, fid)
	delete(m.stops, fid)
}

// makeStoppableName generates a unique name for the callback stoppable.
func makeStoppableName(fid ftCrypto.ID, callbackNum int) string {
	return fid.String() + "/" + strconv.Itoa(callbackNum)
}
