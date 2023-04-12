////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package sync covers logic regarding account synchronization.
package sync

import (
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/xx_network/primitives/netTime"
)

// Stoppable constants.
const (
	collectorStoppable       = "collectorStoppable"
	collectorRunnerStoppable = "collectorRunnerStoppable"
)

// todo: determine actual value
const synchronizationEpoch = 5 * time.Second

// Log constants.
const (
	collectorLogHeader = "COLLECTOR"
)

// Error messages.
const (
	deviceUpdateRetrievalErr = "failed to retrieve last update from %s: %+v"
)

// Log messages
const (
	serializeDeviceTxErr       = "[%s] Failed to read for local values for %s: %v"
	retrieveTxLogFromDeviceErr = "[%s] Failed to serialize this device's transaction log: %+v"
	retrieveDeviceOffsetErr    = "[%s] Failed to retrieve offset for device %s, assuming it to be zero"
	localMoreRecentThanLocal   = "[%s] Transaction key %s has local val newer than remote value"
	upsertTxErr                = "[%s] Failed to upsert for transaction with key %s: %v"
	localSetErr                = "[%s] Failed to locally set for transaction with key %s: %v"
)

// Collector is responsible for reading and collecting all device updates.
type Collector struct {
	// The base path for synchronization
	syncPath string
	// my device ID
	myID cmix.InstanceID
	// The last time each transaction log was read successfully
	// The keys are the device ID strings
	lastUpdates map[cmix.InstanceID]time.Time

	// The max time we assume synchronization takes to happen.
	// This is constant across clients but stored in the object
	// for future parameterization.
	SynchronizationEpoch time.Duration

	// The local transaction log for this device
	txLog *TransactionLog
	// The connection to the remote storage system for reading other device data.
	remote RemoteStore

	// The remote storage EKV wrapper
	kv *VersionedKV

	deviceTxTracker *deviceTransactionTracker
}

// NewCollector constructs a collector object.
func NewCollector(syncPath string, txLog *TransactionLog,
	remote RemoteStore, kv *VersionedKV) *Collector {
	myID, err := cmix.LoadInstanceID(kv)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return &Collector{
		syncPath:             syncPath,
		myID:                 myID,
		lastUpdates:          make(map[cmix.InstanceID]time.Time, 0),
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

	newUpdates, err := c.collectChanges(devices)
	if err != nil {
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

	c.lastUpdates = newUpdates

}

// collectChanges will collate all changes across all devices.
func (c *Collector) collectChanges(devices []string) (
	map[cmix.InstanceID]time.Time, error) {
	// Map of Device to list of (new) transactions
	oldestUpdate := time.Now().Add(-2 * c.SynchronizationEpoch)

	newUpdates := make(map[cmix.InstanceID]time.Time, 0)

	// Iterate over devices
	for _, deviceIDStr := range devices {
		deviceID, err := cmix.InstanceIDFromString(deviceIDStr)
		// Retrieve updates from device
		lastUpdate, err := c.remote.GetLastModified(deviceIDStr)
		if err != nil {
			return nil, errors.Errorf(deviceUpdateRetrievalErr,
				deviceIDStr, err)
		}
		// Get the last update
		lastTrackedUpdate := c.lastUpdates[deviceID]
		// If us, read the local log, otherwise read the remote log
		if deviceID != c.myID &&
			(lastUpdate.Before(lastTrackedUpdate) ||
				lastUpdate.Equal(lastTrackedUpdate)) {
			continue
		}

		// During this pass, record the oldest update across devices
		if oldestUpdate.After(lastUpdate) {
			oldestUpdate = lastUpdate
		}

		// If us, read the local log, otherwise read the remote log
		// TODO: in the future this could work like an open call instead of
		//  sucking the entire thing into memory.
		txLog, err := c.readFromDevice(deviceID)
		if err != nil {
			jww.WARN.Printf("%s", err)
			continue
		}

		offset := c.getTxLogOffset(deviceIDStr)

		// Read all transactions since the last time we saw an update from this
		// device.
		// TODO: in the future we could turn this into like an iterator, where
		//  the “next” change across all devices get read, but for now drop them
		//  into a list per device.
		deviceChanges, err := c.readTransactionsFromLog(
			txLog, deviceIDStr, offset, c.GetDeviceSecret(deviceIDStr))
		if err != nil {
			jww.WARN.Printf("failed to read transaction log for %s: %+v",
				deviceIDStr, err)
			continue
		}

		c.deviceTxTracker.AddToDevice(deviceID, deviceChanges)

		jww.TRACE.Printf("Recorded %d changed for device %s",
			len(deviceChanges), deviceIDStr)

		newUpdates[deviceID] = lastUpdate

	}
	return newUpdates, nil
}

// readFromDevice is a helper function which will read the transaction logs from
// the DeviceID.
func (c *Collector) readFromDevice(deviceId cmix.InstanceID) (
	txLog []byte, err error) {

	if deviceId != c.myID {
		// Retrieve device's transaction log if it is not this device
		txLog, err = c.remote.Read(string(deviceId.String()))
		if err != nil {
			// todo: continue or return here?
			return nil, errors.Errorf(
				retrieveTxLogFromDeviceErr, collectorLogHeader, err)
		}
	} else {
		txLog, err = c.txLog.serialize()
		if err != nil {
			// todo: continue or return here?
			return nil, errors.Errorf(
				retrieveTxLogFromDeviceErr, collectorLogHeader, err)
		}
	}
	return txLog, nil
}

// applyChanges will order the transactionChanges and apply them to the Collector.
func (c *Collector) applyChanges() error {
	// Now apply all collected changes
	ordered := c.deviceTxTracker.Sort()
	for _, tx := range ordered {
		lastWrite, _ := c.remote.GetLastWrite()
		localVal, err := c.remote.Read(tx.Key)
		if err != nil {
			jww.WARN.Printf(serializeDeviceTxErr, collectorLogHeader, tx.Key, err)
			continue
		}

		// If the local value last write is before the current then we need to
		// upsert it, otherwise the oldest transaction takes precedence over the
		// local value (because local is ahead) and we will overwrite it
		if tx.Timestamp.After(lastWrite) {
			jww.WARN.Printf(localMoreRecentThanLocal, collectorLogHeader, tx.Key)
			if err = c.kv.Remote().UpsertLocal(tx.Key, localVal); err != nil {
				jww.WARN.Printf(upsertTxErr, collectorLogHeader, tx.Key, err)
				continue
			}

		} else {

			if err = c.kv.Remote().SetBytes(tx.Key, tx.Value); err != nil {
				jww.WARN.Printf(localSetErr, collectorLogHeader, tx.Key, err)

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

// readTransactionsFromLog is a utility function which reads all Transaction's
// after the last read. This deserializes a TransactionLog and must have the
// device's secret passed in to decrypt transactions.
func (c *Collector) readTransactionsFromLog(txLogSerialized []byte, deviceId string,
	offset int, deviceSecret []byte) ([]Transaction, error) {
	txLog := &TransactionLog{
		deviceSecret: deviceSecret,
	}
	if err := txLog.deserialize(txLogSerialized); err != nil {
		return nil, errors.Errorf(
			"failed to deserialize transaction log: %+v", err)
	}

	if err := c.setTxLogOffset(deviceId, len(txLog.txs)); err != nil {
		return nil, err
	}

	return txLog.txs[offset:], nil
}

// getTxLogOffset is a helper function which will read a device ID's offset
// from storage. If it cannot retrieve the offset from local, it will assume
// zero value.
func (c *Collector) getTxLogOffset(deviceId string) int {
	offsetData, err := c.kv.Remote().GetBytes(deviceOffsetKey(deviceId))
	if err != nil {
		jww.WARN.Printf(retrieveDeviceOffsetErr, collectorLogHeader, deviceId)
	}

	if len(offsetData) != 8 {
		return 0
	}

	return int(deserializeInt(offsetData))
}

// setTxLogOffset is a helper function which writes the latest offset value for
// the given device to local storage.
func (c *Collector) setTxLogOffset(deviceId string, offset int) error {
	data := serializeInt(offset)
	return c.kv.Remote().SetBytes(deviceOffsetKey(deviceId), data)
}

// deviceOffsetKey is a helper function which creates the key for
// setTxLogOffset & getTxLogOffset.
func deviceOffsetKey(id string) string {
	return id + "offset"
}

// deviceTransactionTracker is a structure which tracks the ordered changes of a
// deviceID and the device's Transaction's.
type deviceTransactionTracker struct {
	// Map deviceId -> ordered list of Transactions
	changes map[cmix.InstanceID][]Transaction
}

// newDeviceTransactionTracker is the constructor of the
// deviceTransactionTracker.
func newDeviceTransactionTracker() *deviceTransactionTracker {
	return &deviceTransactionTracker{
		changes: make(map[cmix.InstanceID][]Transaction, 0),
	}
}

// AddToDevice will add a list of Transaction's to the tracked changes.
func (d *deviceTransactionTracker) AddToDevice(deviceId cmix.InstanceID,
	changes []Transaction) {
	d.changes[deviceId] = append(d.changes[deviceId], changes...)
}

// Sort will return the list of Transaction's since the last call to Sort on
// this DeviceID.
func (d *deviceTransactionTracker) Sort() []Transaction {
	sorted := make([]Transaction, 0)
	// todo: these transaction lists are already sorted, so there
	//  is a more efficient way of doing this. Implement this
	//  when iterating over this code. Example code:
	//  changesToCompare = []change
	//  for device, changeList := range changes {
	//    if changeList != nil && len(changeList) != 0 {
	//      changesToCompare = append(changesToCompare, changeList[0])
	//      // pop the first change off this change list, then save it
	//      changeList = changeList[1:]
	//      changes[device] = changeList
	//  }
	//  if len(changesToCompare) == 0 { // return error/end/whatever }
	//  /*key code below*/
	//  oldest := changesToCompare[0]
	//  for i := 1; i < len(changesToCompare); i++ {
	//     change = changesToCompare[i]
	//     if oldest.Timestamp.After(change.Timestamp) {
	//        oldest = change
	//    }
	//  }
	// Iterate over all device transactions
	for _, txs := range d.changes {
		// Insert into sorted list
		for _, tx := range txs {
			sorted = insertionSort(sorted, tx)
		}
	}

	d.changes = make(map[cmix.InstanceID][]Transaction, 0)
	return sorted
}
