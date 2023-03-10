////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package sync covers logic regarding account synchronization.
package sync

import (
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
	deviceUpdateRetrievalErr = "failed to retrieve last update from %s: %+v"
)

// todo: docstring
// todo: move to interfaces.go
type (
	// transactionChanges maps a DeviceId to the list of transactions.
	transactionChanges map[DeviceId][]Transaction
	// todo: docstring logs the time changes were made on a device?
	changeLogger map[DeviceId]time.Time
)

// todo: docstring
type Collector struct {
	// The base path for synchronization
	syncPath string
	// my device ID
	myID DeviceId
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

	deviceTxTracker *deviceTransactionTracker
}

// NewCollector constructs a collector object.
func NewCollector(syncPath string, myId string, txLog *TransactionLog,
	remote RemoteStore, kv *RemoteKV) *Collector {
	return &Collector{
		syncPath:             syncPath,
		myID:                 DeviceId(myId),
		lastUpdates:          make(changeLogger, 0),
		deviceTxTracker:      newDeviceTransactionTracker(),
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

	if err = c.collectChanges(devices); err != nil {
		jww.WARN.Printf("[%s] Failed to collect updates: %+v",
			collectorLogHeader, err)
	}

	// update this record only if we succeed in applying all changes!
	if err = c.applyChanges(); err != nil {
		jww.WARN.Printf("[%s] Failed to apply updates: %+v",
			collectorLogHeader, err)
		return
	}

	elapsed := netTime.Now().Sub(start).Milliseconds()
	jww.INFO.Printf("[%s] Applied new updates took %d ms",
		collectorLogHeader, elapsed)

}

// collectChanges will collate all changes across all devices.
func (c *Collector) collectChanges(devices []string) error {
	// Map of Device to list of (new) transactions
	oldestUpdate := time.Now().Add(-2 * c.SynchronizationEpoch)

	// Iterate over devices
	for _, deviceId := range devices {
		// Retrieve updates from device
		lastUpdate, err := c.remote.GetLastModified(deviceId)
		if err != nil {
			return errors.Errorf(deviceUpdateRetrievalErr, deviceId, err)
		}
		// Get the last update
		lastTrackedUpdate := c.lastUpdates[DeviceId(deviceId)]
		// If us, read the local log, otherwise read the remote log
		if DeviceId(deviceId) != c.myID &&
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
		if DeviceId(deviceId) != c.myID {
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

		// Read all transactions since the last time we saw an update from this
		// device.
		// TODO: in the future we could turn this into like an iterator, where
		//  the “next” change across all devices get read, but for now drop them
		//  into a list per device.
		deviceChanges, err := readTransactionsAfter(txLog, DeviceId(deviceId),
			c.GetDeviceSecret(deviceId))
		if err != nil {
			jww.WARN.Printf("failed to read transaction log for %s: %+v",
				deviceId, err)
			continue
		}

		c.deviceTxTracker.AddToDevice(DeviceId(deviceId), deviceChanges)

		jww.TRACE.Printf("Recorded %d changed for device %s",
			len(deviceChanges), deviceId)

	}
	return nil
}

// applyChanges will order the transactionChanges and apply them to the
// Collector.
func (c *Collector) applyChanges() error {
	// Now apply all collected changes
	ordered := c.deviceTxTracker.Sort()
	for _, tx := range ordered {

		localVal, lastWrite, err := c.remote.ReadAndGetLastWrite(tx.Key)
		if err != nil {
			jww.WARN.Printf("failed to read for local values for %s: %v", tx.Key, err)
			continue
		}

		// If the local value last write is before the current then we need to
		// upsert it, otherwise the oldest transaction takes precedence over the
		// local value (because local is ahead) and we will overwrite it
		if tx.Timestamp.After(lastWrite) {
			if err = c.kv.UpsertLocal(tx.Key, localVal); err != nil {
				jww.WARN.Printf("failed to upsert for transaction with key %s: %v", tx.Key, err)
				continue
			}

		} else {
			if err = c.kv.localSet(tx.Key, tx.Value); err != nil {
				jww.WARN.Printf("failed to locally set for transaction with key %s: %v", tx.Key, err)

			}

		}

	}

	return nil
}

// GetDeviceSecret will return the device secret for the given device identifier.
//
// Fixme: For now, it will return the master secret, this will be an rpc in
//
//	future, return master secret
func (c *Collector) GetDeviceSecret(d string) []byte {
	return c.txLog.deviceSecret
}

// readTransactionsAfter is a utility function which reads all Transaction's
// after the last read. This deserializes a TransactionLog and must have the
// device's secret passed in to decrypt transactions.
func readTransactionsAfter(txLogSerialized []byte, deviceId DeviceId,
	deviceSecret []byte) ([]Transaction, error) {
	txLog := &TransactionLog{
		deviceSecret: deviceSecret,
	}
	if err := txLog.deserialize(txLogSerialized); err != nil {
		return nil, errors.Errorf("failed to deserialize transaction log: %+v", err)
	}

	offset := txLog.offsets[deviceId]

	return txLog.txs[offset:], nil
}

// deviceTransactionTracker is a structure which tracks the ordered changes of a
// deviceID and the device's Transaction's.
type deviceTransactionTracker struct {
	// Map deviceId -> ordered list of Transactions
	changes transactionChanges
}

// newDeviceTransactionTracker is the constructor of the
// deviceTransactionTracker.
func newDeviceTransactionTracker() *deviceTransactionTracker {
	return &deviceTransactionTracker{
		changes: make(transactionChanges, 0),
	}
}

// AddToDevice will add a list of Transaction's to the tracked changes.
func (d *deviceTransactionTracker) AddToDevice(deviceId DeviceId,
	changes []Transaction) {
	d.changes[deviceId] = append(d.changes[deviceId], changes...)
}

// Sort will return the list of Transaction's since the last call to Sort on
// this DeviceId.
func (d *deviceTransactionTracker) Sort() []Transaction {
	sorted := make([]Transaction, 0)
	// Iterate over all device transactions
	for _, txs := range d.changes {
		// Insert into sorted list
		for _, tx := range txs {
			sorted = insertionSort(sorted, tx)
		}
	}

	d.changes = make(transactionChanges, 0)
	return sorted
}
