////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package sync covers logic regarding account synchronization.
package sync

import (
	"bytes"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"time"
)

const (
	collectorStoppable       = "collectorStoppable"
	collectorRunnerStoppable = "collectorRunnerStoppable"
)

const (
	collectorLogHeader = "COLLECTOR"
)

// todo: docstring
type (
	transactionChanges map[string][]Transaction
	// todo: docstring logs the time changes were made on a device?
	changeLogger map[string]time.Time
)

// todo: docstring
type Collector struct {
	// The base path for synchronization
	syncPath string
	// my device ID
	myID string
	// The last time each transaction log was read successfully
	// The keys are the device ID strings
	lastUpdates changeLogger

	// The max time we assume synchronization takes to happen.
	// This is constant across clients but stored in the object
	// for future parameterization.
	SynchronizationEpoch time.Duration

	// The local transaction log for this device
	txLog TransactionLog
	// The connection to the remote storage system for reading other device data.
	remote RemoteStore

	// The remote storage EKV wrapper
	kv RemoteKV
}

func (c *Collector) StartProcesses() (stoppable.Stoppable, error) {
	// Construct stoppables
	multiStoppable := stoppable.NewMulti(collectorStoppable)
	stopper := stoppable.NewSingle(collectorRunnerStoppable)
	go c.runner(stopper)

	return multiStoppable, nil
}

func (c *Collector) runner(stop *stoppable.Single) {
	t := time.NewTicker(c.SynchronizationEpoch)
	select {
	case <-t.C:
		c.collect()
	case <-stop.Quit():
		stop.ToStopped()
		jww.DEBUG.Printf("[%s] Stopping sync collector: stoppable triggered.")
		return
	}
}

// Perform a collection operation
func (c *Collector) collect() {
	// note this returns full device paths from the perspective of
	// the remote
	devices := readDeviceList(c.syncPath, c.remote)
	allowedUpdates := time.Now().Add(-2 * c.SynchronizationEpoch)
	changes, newUpdates := c.collectChanges(devices)

	// update this record only if we succeed in applying all changes!
	if err := c.applyChanges(changes); err != nil  {
		c.lastUpdates = newUpdates
		//debug( how many changes and how long it took)
		return
	} 

	// report applied changes failed
}


func (c *Collector) collectChanges(devices []string) (transactionChanges,
	changeLogger) {
	// Map of Device to list of (new) transactions
	changes, newUpdates := make(transactionChanges, 0), make(changeLogger, 0)
	oldestUpdate := time.Now()
	// drop as much of the following as makes sense into a collect changes func
	// we want to maximize testability!
	for _, d := range devices {
		lastUpdate, err := c.remote.GetLastModified(d)
		if err != nil {
			// todo: handle err
		}
		// If it’s you, always process. Otherwise only look at it if there are updates from the last
		// time we checked
		lastTrackedUpdate := c.lastUpdates[d]

		if d != c.myID &&
			(lastUpdate.Before(lastTrackedUpdate) ||
				lastUpdate.Equal(lastTrackedUpdate)) {
			continue
		}

		// During this pass, record the oldest update across devices
		if oldestUpdate.After(lastUpdate) {
			oldestUpdate = lastUpdate
		}

		// If us, read the local log, otherwise read the remote log
		// TODO: in the future this could work like an open call instead of sucking
		// the entire thing into memory.
		txlog := []byte{}
		if d != c.myID {
			txlog, err = c.remote.Read(d)
			if err != nil {
				// todo: handle err
			}
		} else {
			txlog, err = c.txLog.serialize()
			if err != nil {
				// todo: handle err
			}
		}

		// Read all transactions since the last time we saw an update from this device.
		// in the future we could turn this into like an iterator, where the “next” change across
		// all devices get read, but for now drop them into a list per device.
		changes[d] = readTransactionsAfter(txlog, c.lastUpdates[d], c.GetDeviceSecret(d))
		newUpdates[d] = lastUpdate
		//trace(number of changes for this device)
	}
	return changes, newUpdates
}


func (c *Collector) applyChanges(changes transactionChanges) error {
	// Now apply all collected changes
	ordered := orderChanges(changes) // map of key to list of transactions against it in order
	for k, txList := range ordered {
		// how do we get last write? Do we need to start storing it?
		localVal, err := c.remote.Read(k)
		if err != nil {
			// todo: handle err

		}
		lastWrite, err := c.remote.GetLastWrite()
		if err != nil {
			// todo: handle err
		}

		cur := txList[0]
		// If the local value last write is before the current then we need to upsert it, otherwise
		// the oldest transaction takes precedence over the local value (because local is ahead)
		// and we will overwrite it
		if txList[0].Timestamp.After(lastWrite) {
		 	cur = c.remote.Upsert(k, localVal, cur)
		}

		for i, tx := range(txList) {
			if i == 0 { continue}
			// Does the remote need to hold the upsert callback registration? Or should it
			// be done here?
			// I believe RemoteKV needs to hold the upsert functions because other callers
			// besides the collector object will need to call it.
			cur = c.remote.Upsert(k, cur, tx.Value)
		}
		// What about updates that happen between start and end of this loop?
		// It feels like maybe the entire upsert operation should be done
		// internally to RemoteKV, not here… Call it apply changes and send a list, right?
		// I leave this up to the implementer, but the following probably should not trigger
		// the transaction log.
		if bytes.Equal(cur.Value, localVal) {
			//trace(same val)
		} else {
			c.remote.Set(k, cur)
			debug(sync made on key.. details)
		}
	}

	return nil
}

func readDeviceList(syncPath string, store RemoteStore) []string {
	// fixme: how to implement?
}


func orderChanges(changes transactionChanges) transactionChanges {
	// ordering changes function
	// the key in the key value pair is the key for this map, when we apply we apply on a per key basis so we can make easy debug logs, etc.
	orderedChanges := make(map[string][]Transaction, 0)
	iterate := true
	var oldest *Transaction
	for iterate  {
		_, oldest, iterate = nextChange(changes)
		if oldest != nil {
			// add init when the key is new to the map…
			orderedChanges[oldest.Key] = append(orderedChanges[oldest.Key], oldest)
		}
	}

	return orderedChanges
}

// This looks at each device and returns the oldest change, so you can get them all in order
func nextChange(changes transactionChanges) (transactionChanges, *Transaction, bool) {
	var oldest *Transaction
	curd := ""
	for key, v := range changes{
		if v == nil || len(v) == 0{
		continue
	}
		if oldest == nil || oldest.Timestamp.After(v){
			oldest = &v[0]
			curd = key
		}
	}
	if curd == "" {
		return nil, nil, false
	}

	changes[curd] = changes[curd][1:]
	return changes, oldest, true
}

