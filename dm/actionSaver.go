////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	// Thread stoppable name
	savedActionPurgeThreadStoppable = "SavedActionPurgeThread"

	// purgeFrequency is how often stale actions are purged.
	purgeFrequency = 10 * time.Minute

	// maxSavedActionAge is the age of a saved action before it is deleted.
	maxSavedActionAge = 24 * time.Hour
)

// ActionSaver saves actions that target messages that have not been received
// yet and saves them until they are received to apply the action.
type ActionSaver struct {
	actions map[messageIdKey]time.Time
	kv      versioned.KV
	mux     sync.RWMutex
}

// NewActionSaver initialises a new empty ActionSaver.
func NewActionSaver(kv versioned.KV) *ActionSaver {
	return &ActionSaver{
		actions: make(map[messageIdKey]time.Time),
		kv:      kv,
	}
}

// StartProcesses starts the thread that checks for expired action leases and
// undoes the action. This function adheres to the [xxdk.Service] type.
//
// This function always returns a nil error.
func (as *ActionSaver) StartProcesses() (stoppable.Stoppable, error) {
	actionThreadStop := stoppable.NewSingle(savedActionPurgeThreadStoppable)

	// Start the thread
	go as.purgeThread(actionThreadStop)

	return actionThreadStop, nil
}

// purge removes all stale actions from ActionSaver.
func (as *ActionSaver) purgeThread(stop *stoppable.Single) {
	jww.INFO.Printf("[DM] Starting stale saved action purge thread with "+
		"stoppable %s", stop.Name())

	ticker := time.NewTicker(purgeFrequency)
	for {
		select {
		case <-stop.Quit():
			jww.INFO.Printf("[DM] Stopping stale saved action purge thread: "+
				"stoppable %s quit", stop.Name())
			stop.ToStopped()
			return
		case <-ticker.C:
			jww.INFO.Printf("[DM] Purging stale saved actions.")
			if err := as.purge(netTime.Now()); err != nil {
				jww.ERROR.Printf(
					"[DM] Stale saved action purge failed: %+v", err)
			}
		}
	}
}

// purge removes all stale actions from ActionSaver.
func (as *ActionSaver) purge(now time.Time) error {
	as.mux.Lock()
	defer as.mux.Unlock()

	// Find all stale actions
	for key, received := range as.actions {
		if now.Sub(received) > maxSavedActionAge {
			delete(as.actions, key)
		}
	}

	return as.storeActionList()
}

// AddAction saves the action to the list.
func (as *ActionSaver) AddAction(targetMessage message.ID, received time.Time) error {
	jww.DEBUG.Printf(
		"[DM] Inserting new saved action for target message %s", targetMessage)

	as.mux.Lock()
	defer as.mux.Unlock()

	// Add the saved action for this target message if it does not exist
	as.actions[getMessageIdKey(targetMessage)] = received.UTC().Round(0)

	// Update storage
	return as.storeActionList()
}

// CheckSavedActions checks if there is a saved action for the message.
func (as *ActionSaver) CheckSavedActions(targetMessage message.ID) bool {
	as.mux.RLock()
	defer as.mux.RUnlock()

	key := getMessageIdKey(targetMessage)
	if _, exists := as.actions[key]; exists {
		// Once the result has been returned, delete the action
		go func(key messageIdKey) {
			as.mux.Lock()
			defer as.mux.Unlock()
			delete(as.actions, key)
			if err := as.storeActionList(); err != nil {
				jww.ERROR.Printf("[DM] Failed to delete saved action for "+
					"message %s from storage: %+v", key, err)
			}
		}(key)

		return true
	}

	return false
}

////////////////////////////////////////////////////////////////////////////////
// Message ID Key                                                             //
////////////////////////////////////////////////////////////////////////////////

// messageIdKey is the base 64 string encoding of the message.ID. It is used as
// a key in a map so the map can be JSON marshalled/unmarshalled.
type messageIdKey string

// getMessageIdKey creates a messageIdKey from a message.ID.
func getMessageIdKey(msgID message.ID) messageIdKey {
	return messageIdKey(base64.StdEncoding.EncodeToString(msgID.Marshal()))
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Storage values.
const (
	savedActionsVer = 0
	savedActionsKey = "channelSavedActions"
)

// Error messages.
const (
	// ActionSaver.load
	loadSavedActionsChanIDsErr = "could not load list of channels"
)

// load gets all the command messages from storage and loads them into the
// action list and map. If any of the messages have a lease trigger
// before now, then they are assigned a new lease trigger between 5 and 30
// minutes in the future to allow alternate replays a chance to be picked up.
func (as *ActionSaver) load(now time.Time) error {
	// Get list of channel IDs
	actionList, err := as.loadActionList()
	if err != nil {
		return errors.Wrap(err, loadSavedActionsChanIDsErr)
	}

	for key, received := range actionList {
		if now.Sub(received) < maxSavedActionAge {
			as.actions[key] = received
		}
	}

	return nil
}

// storeActionList stores the list of all channel IDs in the action list to
// storage.
func (as *ActionSaver) storeActionList() error {
	data, err := json.Marshal(as.actions)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   savedActionsVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return as.kv.Set(savedActionsKey, obj)
}

// loadActionList loads the list of all channel IDs in the action list from
// storage.
func (as *ActionSaver) loadActionList() (map[messageIdKey]time.Time, error) {
	obj, err := as.kv.Get(savedActionsKey, savedActionsVer)
	if err != nil {
		return nil, err
	}

	var actions map[messageIdKey]time.Time
	return actions, json.Unmarshal(obj.Data, &actions)
}
