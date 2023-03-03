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
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

// Stoppable constants.
const (
	collectorStoppable       = "collectorStoppable"
	collectorRunnerStoppable = "collectorRunnerStoppable"
)

// todo: determine actual value
const synchronizationEpoch = 2 * time.Hour

const (
	collectorLogHeader = "COLLECTOR"
)

const (
	deviceUpdateRetrievalErr = "failed to retrieve last update from %s"
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
	txLog *TransactionLog
	// The connection to the remote storage system for reading other device data.
	remote RemoteStore

	// The remote storage EKV wrapper
	kv *RemoteKV
}

// NewCollector constructs a collector object.
func NewCollector(syncPath, myId string, txLog *TransactionLog,
	remote RemoteStore, kv *RemoteKV) *Collector {
	return &Collector{
		syncPath:             syncPath,
		myID:                 myId,
		lastUpdates:          make(changeLogger, 0),
		SynchronizationEpoch: synchronizationEpoch,
		txLog:                txLog,
		remote:               remote,
		kv:                   kv,
	}

}

// StartProcesses will begin a long-running thread to collect and
// synchronize changes across devices.
func (c *Collector) StartProcesses() (stoppable.Stoppable, error) {
	// Construct stoppables
	multiStoppable := stoppable.NewMulti(collectorStoppable)
	stopper := stoppable.NewSingle(collectorRunnerStoppable)
	go c.runner(stopper)
	return multiStoppable, nil
}

// runner is the long-running thread responsible for collecting changes and
// synchronizing changes across devices.
func (c *Collector) runner(stop *stoppable.Single) {
	t := time.NewTicker(c.SynchronizationEpoch)
	select {
	case <-t.C:
		c.collect()
	case <-stop.Quit():
		stop.ToStopped()
		jww.DEBUG.Printf(
			"[%s] Stopping sync collector: stoppable triggered.",
			collectorLogHeader)
		return
	}
}

// collect will collect, organize and apply all changes across devices.
func (c *Collector) collect() {
	// note this returns full device paths from the perspective of
	// the remote
	devices, err := c.remote.ReadDir(c.syncPath)
	if err != nil {
		// todo: handle err
		jww.WARN.Printf("[%s] Failed to read devices: %+v",
			collectorLogHeader, err)
		return
	}

	start := netTime.Now()

	changes, newUpdates, err := c.collectChanges(devices)

	// update this record only if we succeed in applying all changes!
	if err = c.applyChanges(changes); err != nil {
		jww.WARN.Printf("[%s] Failed to apply updates: %+v",
			collectorLogHeader, err)
		return
	}

	elapsed := netTime.Now().Sub(start).Milliseconds()
	jww.INFO.Printf("[%s] Applied %d new updates took %s",
		collectorLogHeader, len(newUpdates), elapsed)

	c.lastUpdates = newUpdates
}

// collectChanges will collate all changes across all devices.
func (c *Collector) collectChanges(devices []string) (transactionChanges,
	changeLogger, error) {
	// Map of Device to list of (new) transactions
	changes, newUpdates := make(transactionChanges, 0), make(changeLogger, 0)
	oldestUpdate := time.Now().Add(-2 * c.SynchronizationEpoch)

	// Iterate over devices
	for _, deviceId := range devices {

		// Retrieve updates from device
		lastUpdate, err := c.remote.GetLastModified(deviceId)
		if err != nil {
			return nil, nil, errors.Errorf(deviceUpdateRetrievalErr, deviceId)
		}

		// Get the last update
		lastTrackedUpdate := c.lastUpdates[deviceId]

		// If us, read the local log, otherwise read the remote log
		if deviceId != c.myID ||
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
		//  the entire thing into memory.
		txLog := []byte{}
		if deviceId != c.myID {
			// Retrieve device's transaction log if it is not this device
			txLog, err = c.remote.Read(deviceId)
			if err != nil {
				// todo: continue or return here?
				jww.WARN.Printf("failed to retrieve transaction log from device %s", deviceId)
				continue
			}
		} else {
			txLog, err = c.txLog.serialize()
			if err != nil {
				// todo: continue or return here?
				jww.WARN.Printf("failed to serialize this device's transaction log: %+v", err)
				continue
			}
		}

		// Read all transactions since the last time we saw an update from this device.
		// in the future we could turn this into like an iterator, where the “next” change across
		// all devices get read, but for now drop them into a list per device.
		// d - identifier (string)
		changes[deviceId], err = readTransactionsAfter(txLog, c.lastUpdates[deviceId], c.GetDeviceSecret(deviceId))
		if err != nil {
			// todo: continue or return here?
			jww.WARN.Printf("failed to read transaction log for %d: %+v",
				deviceId, err)
			continue
		}
		newUpdates[deviceId] = lastUpdate
		//trace(number of changes for this device)
		jww.TRACE.Printf("Recorded %d changed for device %s",
			len(changes[deviceId]), deviceId)
	}
	return changes, newUpdates, nil
}

// applyChanges will order the transactionChanges and apply them to the
// Collector.
func (c *Collector) applyChanges(changes transactionChanges) error {
	// Now apply all collected changes
	ordered := orderChanges(changes) // map of key to list of transactions against it in order
	for k, txList := range ordered {
		// how do we get last write? Do we need to start storing it?
		// fixme: is this meant to be the KV? I assumed remote store
		localVal, lastWrite, err := c.remote.ReadAndGetLastWrite(k)
		if err != nil {
			// todo: continue or return here?
			jww.WARN.Printf("failed to read for local values for %s: %v", k, err)
			continue
		}

		cur := txList[0]
		// If the local value last write is before the current then we need to upsert it, otherwise
		// the oldest transaction takes precedence over the local value (because local is ahead)
		// and we will overwrite it
		if txList[0].Timestamp.After(lastWrite) {
			// upset func
			updateCb := func(newTx Transaction, err error) {
				cur = newTx
			}

			if err = c.kv.Set(k, localVal, updateCb); err != nil {
				// todo: continue or return here?
				jww.WARN.Printf("failed to dry for local values for %s: %v", k, err)
				continue
			}

		}

		// Go through all transactions
		for i, tx := range txList {
			if i == 0 {
				continue
			}

			updateCb := func(newTx Transaction, err error) {
				cur = newTx
			}

			if err = c.kv.Set(k, tx.Value, updateCb); err != nil {
				// todo: continue or return here?
				jww.WARN.Printf("failed to read for local values for %s: %v", k, err)
				continue
			}
		}

		if bytes.Equal(cur.Value, localVal) {
			//trace(same val)
			jww.TRACE.Printf("Same value for transaction %d", cur)
		} else {
			if err = c.kv.Set(k, cur.Value, nil); err != nil {
				// todo: continue or return here?
				jww.WARN.Printf("failed to read for local values for %s: %v", k, err)
				continue
			}
		}
	}

	return nil
}

// GetDeviceSecret will return the device secret for the given device identifier.
//
// Fixme: For now, it will return the master secret, this will be an rpc in
//  future, return master secret
func (c *Collector) GetDeviceSecret(d string) []byte {
	return c.txLog.deviceSecret
}

func orderChanges(changes transactionChanges) transactionChanges {
	// ordering changes function
	// the key in the key value pair is the key for this map, when we apply we apply on a per key basis so we can make easy debug logs, etc.
	orderedChanges := make(map[string][]Transaction, 0)
	iterate := true
	var oldest *Transaction
	for iterate {
		_, oldest, iterate = nextChange(changes)
		if oldest != nil {
			// add init when the key is new to the map…
			orderedChanges[oldest.Key] = append(orderedChanges[oldest.Key], *oldest)
		}
	}

	return orderedChanges
}

// nextChange will look at each device and return the oldest change.
func nextChange(changes transactionChanges) (transactionChanges, *Transaction,
	bool) {
	var oldest *Transaction
	curd := ""
	for key, v := range changes {
		if v == nil || len(v) == 0 {
			continue
		}
		if oldest == nil || oldest.Timestamp.IsZero() ||
			oldest.Timestamp.After(v[0].Timestamp) {
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

// readTransactionsAfter is a utility function which reads all Transaction's
// after the given time. This deserializes a TransactionLog and must have the
// device's secret passed in to decrypt transactions.
func readTransactionsAfter(txLogSerialized []byte, t time.Time,
	deviceSecret []byte) ([]Transaction, error) {
	txLog := &TransactionLog{
		deviceSecret: deviceSecret,
	}
	if err := txLog.deserialize(txLogSerialized); err != nil {
		// todo: handle err
		return nil, errors.Errorf("failed to deserialize transaction log: %+v", err)
	}

	res := make([]Transaction, 0)
	for _, tx := range txLog.txs {
		if tx.Timestamp.After(t) {
			res = append(res, tx)
		}
	}

	return res
}
