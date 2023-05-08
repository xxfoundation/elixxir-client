////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package collective covers logic regarding account synchronization.
package collective

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/elixxir/client/v4/storage/versioned"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/xx_network/primitives/netTime"
)

// Stoppable constants.
const (
	collectorRunnerStoppable = "collectorRunnerStoppable"
	writerRunnerStoppable    = "writerRunnerStoppable"
)

// todo: determine actual value
const synchronizationEpoch = 5 * time.Second

// Log constants.
const (
	collectorLogHeader = "COLLECTOR"

	// FIXME: It should be: [name]-[deviceid]/[keyid]/txlog
	// but we don't have access to a name, so: [deviceid]/[keyid]/txlog
	txLogPathFmt = "%s/%s/txlog"
)

// Error messages.
const (
	deviceUpdateRetrievalErr = "failed to retrieve last update from %s: %+v"
)

// Log messages
const (
	serializeDeviceTxErr       = "[%s] Failed to Read for local values for %s: %v"
	retrieveTxLogFromDeviceErr = "[%s] Failed to serialize this device's mutate log: %+v"
	retrieveDeviceOffsetErr    = "[%s] Failed to retrieve offset for device %s, assuming it to be zero"
	localMoreRecentThanLocal   = "[%s] Mutate key %s has local val newer than remote value"
	upsertTxErr                = "[%s] Failed to upsert for mutate with key %s: %v"
	localSetErr                = "[%s] Failed to locally set for mutate with key %s: %v"
)

// lastMutationReadStorage
const lastMutationReadStorageKey = "lastMutationReadStorageKey_"

// collector is responsible for reading and collecting all device updates.
type collector struct {
	// The base path for synchronization
	syncPath string

	// This local instance ID
	myID InstanceID

	// The last time each mutate log was Read successfully
	lastUpdateRead     map[InstanceID]time.Time
	devicePatchTracker map[InstanceID]*Patch
	lastMutationRead   map[InstanceID]time.Time

	// The max time we assume synchronization takes to happen.
	// This is constant across clients but stored in the object
	// for future parameterization.
	synchronizationEpoch time.Duration

	// The local mutate log for this device
	txLog *remoteWriter
	// The connection to the remote storage system for reading other device data.
	remote RemoteStore

	// The remote storage EKV wrapper
	kv *internalKV

	encrypt encryptor

	//tracks connection state
	connected *uint32
	*notifier

	//tracks if the system has synched with remote
	synched *uint32
}

// newCollector constructs a collector object.
func newCollector(myID InstanceID, syncPath string,
	remote RemoteStore, kv *internalKV, encrypt encryptor,
	writer *remoteWriter) *collector {

	connected := uint32(0)
	synched := uint32(0)

	c := &collector{
		syncPath:             syncPath,
		myID:                 myID,
		lastUpdateRead:       make(map[InstanceID]time.Time),
		devicePatchTracker:   make(map[InstanceID]*Patch),
		lastMutationRead:     make(map[InstanceID]time.Time),
		synchronizationEpoch: synchronizationEpoch,
		txLog:                writer,
		remote:               remote,
		kv:                   kv,
		encrypt:              encrypt,
		connected:            &connected,
		synched:              &synched,
	}
	c.notifier = &notifier{}

	c.txLog.Register(func(state bool) {
		isConnected := c.isConnected()
		if state && isConnected {
			c.Notify(true)
		} else if !isConnected {
			c.Notify(false)
		}
	})

	c.loadLastMutationTime()

	return c

}

// runner is the long-running thread responsible for collecting changes and
// synchronizing changes across devices.
func (c *collector) runner(stop *stoppable.Single) {
	t := time.NewTicker(c.synchronizationEpoch)
	select {
	case <-t.C:
		c.collect()
	case <-stop.Quit():
		stop.ToStopped()
		jww.DEBUG.Printf(
			"[%s] Stopping collective collector: stoppable triggered.",
			collectorLogHeader)
		return
	}
}

func (c *collector) isConnected() bool {
	return atomic.LoadUint32(c.connected) == 1
}

// IsConnected returns true if the system is capable of both reading
// and writing to remote
func (c *collector) IsConnected() bool {
	return c.isConnected() && c.txLog.RemoteUpToDate()
}

// IsSynched returns true if the local data has been synchronized with remote
// during its lifetime
func (c *collector) IsSynched() bool {
	return atomic.LoadUint32(c.synched) == 1
}

// WaitUntilSynched waits until either timeout time has elapsed
// or the system successfully synchronizes with the remote
// polls every 100ms
func (c *collector) WaitUntilSynched(timeout time.Duration) bool {
	start := netTime.Now()

	for time.Now().Sub(start) < timeout {
		if c.IsSynched() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

func (c *collector) notify(state bool) {
	var toWrite uint32
	if state {
		toWrite = 1
	} else {
		toWrite = 0
	}
	old := atomic.SwapUint32(c.connected, toWrite)
	if toWrite != old {
		if !state {
			c.Notify(false)
		} else if writerConnected := c.txLog.RemoteUpToDate(); state && writerConnected {
			c.Notify(true)
		}
	}
}

// collect will collect, organize and apply all changes across devices.
func (c *collector) collect() {
	// note this returns full device paths from the perspective of
	// the remote
	devicePaths, err := c.remote.ReadDir(c.syncPath)
	if err != nil {
		c.notify(false)
		jww.WARN.Printf("[%s] Failed to Read devices: %+v",
			collectorLogHeader, err)
		return
	}

	start := netTime.Now()

	devices := c.initDevices(devicePaths)
	newUpdates, err := c.collectChanges(devices)
	if err != nil {
		jww.WARN.Printf("[%s] Failed to collect updates: %+v",
			collectorLogHeader, err)
		c.notify(false)
		return
	}

	c.notify(true)

	// update this record only if we succeed in applying all changes!
	if err = c.applyChanges(); err != nil {
		jww.WARN.Printf("[%s] Failed to apply updates: %+v",
			collectorLogHeader, err)
		return
	}

	atomic.StoreUint32(c.synched, 1)

	elapsed := netTime.Now().Sub(start).Milliseconds()
	jww.INFO.Printf("[%s] Applied new updates took %d ms",
		collectorLogHeader, elapsed)

	c.lastUpdateRead = newUpdates

}

// collectChanges will collate all changes across all devices.
func (c *collector) collectChanges(devices []InstanceID) (
	map[InstanceID]time.Time, error) {

	newUpdates := make(map[InstanceID]time.Time, 0)

	wg := &sync.WaitGroup{}
	connectionFailed := uint32(0)
	// Iterate over devices
	for _, deviceID := range devices {
		wg.Add(1)
		go func(deviceID InstanceID) {
			defer wg.Done()
			//do not get from remote for my data
			if deviceID == c.myID {
				return
			}

			// Get the last time the device log was written on the remote
			logPath := filepath.Join(c.syncPath,
				fmt.Sprintf(txLogPathFmt, deviceID, keyID))
			lastRemoteUpdate, err := c.remote.GetLastModified(logPath)
			if err != nil {
				atomic.AddUint32(&connectionFailed, 1)
				jww.WARN.Printf("Failed to get last modified for %s "+
					"at %s: %+v", deviceID, logPath, err)
				return
			}

			// determine the update timestamp we saw from the device
			lastTrackedUpdate := c.lastUpdateRead[deviceID]

			// If us, Read the local log, otherwise Read the remote log
			if !lastRemoteUpdate.After(lastTrackedUpdate) {
				return
			}

			// Read the remote log
			patchFile, err := c.readFromDevice(logPath)
			if err != nil {
				atomic.AddUint32(&connectionFailed, 1)
				jww.WARN.Printf("failed to Read the log from %s "+
					"at %s %+v", deviceID, logPath, err)
				return
			}

			_, patch, err := handleIncomingFile(patchFile, c.encrypt)
			if err != nil {
				jww.WARN.Printf("failed to handle incoming file for %s "+
					"at %s: %+v", deviceID, logPath, err)
				return
			}

			// preallocated
			c.devicePatchTracker[deviceID] = patch
			newUpdates[deviceID] = lastRemoteUpdate
		}(deviceID)
	}
	wg.Wait()

	if connectionFailed < 0 {
		err := errors.Errorf("failed to read %d logs, aborting "+
			"collection", connectionFailed)
		return nil, err
	}

	return newUpdates, nil
}

// readFromDevice is a helper function which will Read the patch from
// the DeviceID.
func (c *collector) readFromDevice(txLogPath string) (
	txLog []byte, err error) {

	// Retrieve device's mutate log if it is not this device
	txLog, err = c.remote.Read(txLogPath)
	if err != nil {
		return nil, errors.Errorf(
			retrieveTxLogFromDeviceErr, collectorLogHeader, err)
	}

	return txLog, nil
}

// applyChanges will order the transactionChanges and apply them to the collector.
func (c *collector) applyChanges() error {

	// get local patch and lock transaction log so no remote wite are witten
	// while changes are applied
	localPatch, unlock := c.txLog.Read()
	defer unlock()
	c.devicePatchTracker[c.myID] = localPatch
	c.lastMutationRead[c.myID] = netTime.Now()

	//prepare the data for the diff
	devices, patches, ignoreBefore := prepareDiff(c.devicePatchTracker, c.lastMutationRead)

	//execute the diff
	updates, lastSeen := localPatch.Diff(patches, ignoreBefore)

	// store the timestamps
	for i, device := range devices {
		c.lastMutationRead[device] = lastSeen[i]
	}
	c.saveLastMutationTime()

	// Sort the updates by map and execute the key operations
	wg := sync.WaitGroup{}
	mapUpdates := make(map[string]map[string]*Mutate)
	for key := range updates {
		isMapElement, mapName, _ := versioned.DetectMapElement(key)
		if isMapElement {
			mapObj, exists := mapUpdates[mapName]
			if !exists {
				mapObj = make(map[string]*Mutate)
				mapUpdates[mapName] = mapObj
			}
			mapObj[key] = updates[key]
		} else {
			wg.Add(1)
			func(key string, m *Mutate) {
				if m.Deletion {
					err := c.kv.DeleteFromRemote(key)
					if err != nil {
						jww.WARN.Printf("Failed to Delete %s "+
							"from remote: %+v", key, err)
					}
				} else {
					err := c.kv.SetBytesFromRemote(key, m.Value)
					if err != nil {
						jww.FATAL.Panicf("Failed to set %s from remote: "+
							"%+v", key, err)
					}
				}
			}(key, updates[key])
		}
	}

	//apply the map updates
	for mapName := range mapUpdates {
		mapUpdate := mapUpdates[mapName]
		err := c.kv.MapTransactionFromRemote(mapName, mapUpdate)
		if err != nil {
			jww.FATAL.Panicf("Failed to update map %sL %+v", mapName, err)
		}
	}
	return nil
}

func prepareDiff(devicePatchTracker map[InstanceID]*Patch,
	lastMutationRead map[InstanceID]time.Time) ([]InstanceID, []*Patch, []time.Time) {
	//sort the devices so they are in supremacy order
	devices := make([]InstanceID, 0, len(devicePatchTracker))
	for deviceID := range devicePatchTracker {
		devices = append(devices, deviceID)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Cmp(devices[j]) == 1
	})

	patches := make([]*Patch, len(devices))
	lastSeen := make([]time.Time, len(devices))

	for i := 0; i < len(devices); i++ {
		patches[i] = devicePatchTracker[devices[i]]
		lastSeen[i] = lastMutationRead[devices[i]]
	}

	return devices, patches, lastSeen
}

func (c *collector) saveLastMutationTime() {
	storageKey := makeLastMutationKey(c.myID)
	data, err := json.Marshal(&c.lastMutationRead)
	if err != nil {
		jww.WARN.Printf("Failed to encode lastMutationRead to store "+
			"to disk at %s, data may be replayed: %+v", storageKey, err)
		return
	}

	if err = c.kv.SetBytes(makeLastMutationKey(c.myID), data); err != nil {
		jww.WARN.Printf("Failed to store lastMutationRead to "+
			"to disk at %s, data may be replayed: %+v", storageKey, err)
	}
}

func (c *collector) loadLastMutationTime() {
	storageKey := makeLastMutationKey(c.myID)
	data, err := c.kv.GetBytes(storageKey)
	if err != nil {
		jww.WARN.Printf("Failed to load lastMutationRead from "+
			"to disk at %s, data may be replayed: %+v", storageKey, err)
		return
	}

	c.lastMutationRead = make(map[InstanceID]time.Time)
	err = json.Unmarshal(data, &c.lastMutationRead)
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal lastMutationRead loaded "+
			"from disk at %s, data may be replayed: %+v", storageKey, err)
		return
	}
}

func (c *collector) initDevices(devicePaths []string) []InstanceID {
	devices := make([]InstanceID, 0, len(devicePaths))

	for i, deviceIDStr := range devicePaths {
		deviceID, err := NewInstanceIDFromString(deviceIDStr)
		if err == nil {
			jww.WARN.Printf("Failed to decode device ID for "+
				"index %d: %s, skipping", i, deviceIDStr)
			continue
		}
		devices = append(devices, deviceID)
		if _, exists := c.lastUpdateRead[deviceID]; !exists {
			c.lastUpdateRead[deviceID] = time.Unix(0, 0).UTC()
			c.lastMutationRead[deviceID] = time.Unix(0, 0).UTC()
			c.devicePatchTracker[deviceID] = newPatch()
		}
	}
	return devices
}

func handleIncomingFile(patchFile []byte, decrypt encryptor) (*header, *Patch, error) {
	h, ecrPatchBytes, err := decodeFile(patchFile)
	if err != nil {
		err = errors.WithMessagef(err, "failed to decode the file")
		return nil, nil, err
	}

	patchBytes, err := decrypt.Decrypt(ecrPatchBytes)
	if err != nil {
		err = errors.WithMessagef(err, "failed to decrypt the patch")
		return h, nil, err
	}
	patch := newPatch()
	err = patch.Deserialize(patchBytes)
	if err != nil {
		err = errors.WithMessagef(err, "failed to decode the patch from file")
		return h, nil, err
	}

	return h, patch, nil
}

func makeLastMutationKey(deviceID InstanceID) string {
	return lastMutationReadStorageKey + deviceID.String()
}
