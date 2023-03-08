////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	// Thread stoppable name
	savedActionPurgeThreadStoppable = "SavedActionPurgeThread"

	// purgeFrequency is how often stale actions are purged.
	purgeFrequency = 10 * time.Minute

	// maxSavedActionAge is the age of a saved action before it is deleted.
	maxSavedActionAge = 24 * time.Hour

	// actionSaveNickname is the nickname used when replaying an action.
	actionSaveNickname = "ActionSaver"
)

func (sa *savedAction) String() string {
	fields := []string{
		// "Received:" + sa.Received.String(),
		"Received:" + fmt.Sprintf("%d", sa.Received.Unix()),
		"TargetMessage:" + fmt.Sprintf("%+v", sa.TargetMessage[:2]),
		// "TargetMessage:" + sa.TargetMessage.String(),
		// "CommandMessage:" + fmt.Sprintf("%+v", sa.CommandMessage),
	}

	return "{" + strings.Join(fields, " ") + "}"
}

// ActionSaver saves actions that are received that do not apply to any message
// in storage. The actions are saved and checked against each new message to see
// if they apply.
type ActionSaver struct {
	// actions is a map of actions that do not belong to any received messages
	// mapped to their target message and channel.
	actions map[id.ID]map[messageIdKey]*savedAction

	// triggerFn is called when a lease expired to trigger the undoing of the
	// action.
	triggerFn triggerActionEventFunc

	kv  *versioned.KV
	mux sync.RWMutex
}

type savedAction struct {
	// Received is the time that the message was received by the ActionSaver.
	Received time.Time `json:"received,omitempty"`

	// TargetMessage is the message ID of the message that the action applies to.
	TargetMessage message.ID `json:"targetMessage,omitempty"`

	CommandMessage `json:"commandMessage,omitempty"`
}

// NewActionSaver initialises a new empty ActionSaver.
func NewActionSaver(
	triggerFn triggerActionEventFunc, kv *versioned.KV) *ActionSaver {
	as := &ActionSaver{
		actions:   make(map[id.ID]map[messageIdKey]*savedAction),
		triggerFn: triggerFn,
		kv:        kv,
	}

	return as
}

// StartProcesses starts the thread that checks for expired action leases and
// undoes the action. This function adheres to the xxdk.Service type.
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
	jww.INFO.Printf("[CH] Starting stale saved action purge thread with "+
		"stoppable %s", stop.Name())

	ticker := time.NewTicker(purgeFrequency)
	for {
		select {
		case <-stop.Quit():
			jww.INFO.Printf("[CH] Stopping stale saved action purge thread: "+
				"stoppable %s quit", stop.Name())
			stop.ToStopped()
			return
		case <-ticker.C:
			jww.INFO.Printf("[CH] Purging stale saved actions.")
			err := as.purge(netTime.Now())
			if err != nil {
				jww.ERROR.Printf(
					"[CH] Stale saved action purge failed: %+v", err)
			}
		}
	}
}

// purge removes all stale actions from ActionSaver.
func (as *ActionSaver) purge(now time.Time) error {
	as.mux.Lock()
	defer as.mux.Unlock()

	// Find all stale actions
	for chanID, actions := range as.actions {
		var channelIdUpdate bool
		for key, sa := range actions {
			if now.Sub(sa.Received) > maxSavedActionAge {
				delete(as.actions[chanID], key)
				if len(as.actions[chanID]) == 0 {
					delete(as.actions, chanID)
					channelIdUpdate = true
				}
			}
		}
		err := as.updateStorage(&chanID, channelIdUpdate)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddAction inserts the saved action into the ordered list and map keyed on the
// target message.
func (as *ActionSaver) AddAction(channelID *id.ID, messageID,
	targetMessage message.ID, action MessageType, content,
	encryptedPayload []byte, timestamp, originatingTimestamp, received time.Time,
	lease time.Duration, originatingRound id.Round, round rounds.Round,
	fromAdmin bool) error {

	// Generate the savedAction object to save
	sa := &savedAction{
		Received:      received.Round(0).UTC(),
		TargetMessage: targetMessage,
		CommandMessage: CommandMessage{
			ChannelID:            channelID,
			MessageID:            messageID,
			MessageType:          action,
			Content:              content,
			EncryptedPayload:     encryptedPayload,
			Timestamp:            timestamp.Round(0).UTC(),
			OriginatingTimestamp: originatingTimestamp.Round(0).UTC(),
			Lease:                lease,
			OriginatingRound:     originatingRound,
			Round:                round,
			FromAdmin:            fromAdmin,
		},
	}
	jww.DEBUG.Printf(
		"[CH] Inserting new saved action for target message %s: %+v",
		targetMessage, sa)

	return as.addAction(sa)
}
func (as *ActionSaver) addAction(sa *savedAction) error {

	// When set to true, the list of channels IDs will be updated in storage
	var channelIdUpdate bool

	key := getMessageIdKey(sa.TargetMessage)

	as.mux.Lock()
	defer as.mux.Unlock()

	if messages, exists := as.actions[*sa.ChannelID]; !exists {
		// Add the channel with the saved action if it does not exist
		as.actions[*sa.ChannelID] = map[messageIdKey]*savedAction{key: sa}
		channelIdUpdate = true
	} else if loadedSa, exists2 := messages[key]; !exists2 {
		// Add the saved action for this target message if it does not exist
		as.actions[*sa.ChannelID][key] = sa
	} else {
		// If a saved action already exists for this target message, then
		// determine if this new action overwrites the saved one

		if loadedSa.MessageType != Delete {
			switch sa.MessageType {
			case Delete:
				// Delete replaces all other possible actions
				as.actions[*sa.ChannelID][key] = sa
			default:
				if sa.OriginatingTimestamp.After(loadedSa.OriginatingTimestamp) {
					// If this pin action is newer, then replace it
					as.actions[*sa.ChannelID][key] = sa
				} else {
					// Drop old pin messages
					return nil
				}
			}
		}
	}

	// Update storage
	return as.updateStorage(sa.ChannelID, channelIdUpdate)
}

// UpdateActionFn updates a message if it has a saved action. It returns a UUID
// and an error
type UpdateActionFn func() (uint64, error)

// CheckSavedActions checks if there is a saved action for the message and
// channel ID. If there are no saved actions for the message, then this function
// returns nil and false. If there is a saved delete action, then
// CheckSavedActions returns true and the message should be dropped and not be
// passed to the event model. If there is a non-delete action, then an
// UpdateActionFn is returned that must be called after the message is passed to
// the event model.
func (as *ActionSaver) CheckSavedActions(
	channelID *id.ID, targetMessage message.ID) (UpdateActionFn, bool) {
	as.mux.RLock()
	defer as.mux.RUnlock()

	if messages, exists := as.actions[*channelID]; exists {
		if sa, exists2 := messages[getMessageIdKey(targetMessage)]; exists2 {
			// Once the result has been returned, delete the action
			defer func(sa *savedAction) {
				go func(sa *savedAction) {
					as.mux.Lock()
					defer as.mux.Unlock()
					if err := as.deleteAction(sa); err != nil {
						jww.ERROR.Printf(
							"[CH] Failed to delete saved action: %+v", err)
					}
				}(sa)
			}(sa)

			if sa.MessageType == Delete {
				return nil, true
			} else {
				return func() (uint64, error) {
					return as.triggerFn(sa.ChannelID, sa.MessageID,
						sa.MessageType, actionSaveNickname, sa.Content,
						sa.EncryptedPayload, sa.Timestamp,
						sa.OriginatingTimestamp, sa.Lease, sa.OriginatingRound,
						sa.Round, sa.Status, sa.FromAdmin)
				}, false
			}
		}
	}

	return nil, false
}

// deleteAction removes the action from the map. This function also updates
// storage. If the message does not exist, nil is returned. This function is not
// thread safe.
func (as *ActionSaver) deleteAction(sa *savedAction) error {
	var loadedSa *savedAction
	key := getMessageIdKey(sa.TargetMessage)
	if messages, exists := as.actions[*sa.ChannelID]; !exists {
		return nil
	} else if loadedSa, exists = messages[key]; !exists {
		return nil
	}

	// When set to true, the list of channels IDs will be updated in storage
	var channelIdUpdate bool

	// Remove from message map
	delete(as.actions[*loadedSa.ChannelID], key)
	if len(as.actions[*loadedSa.ChannelID]) == 0 {
		delete(as.actions, *loadedSa.ChannelID)
		channelIdUpdate = true
	}

	// Update storage
	return as.updateStorage(sa.ChannelID, channelIdUpdate)
}

// RemoveChannel removes each saved action for the channel from the ordered list
// and removes the channel from the map. Also deletes from storage.
//
// RemoveChannel should be called when leaving a channel.
func (as *ActionSaver) RemoveChannel(channelID *id.ID) error {
	_, exists := as.actions[*channelID]
	if !exists {
		return nil
	}

	delete(as.actions, *channelID)

	// Update channel ID list
	if err := as.storeChannelList(); err != nil {
		return err
	}

	// Delete actions from storage
	return as.deleteActions(channelID)
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
	savedActionsVer               = 0
	savedActionsKey               = "channelSavedActions"
	savedActionsMessagesVer       = 0
	savedActionsMessagesKeyPrefix = "savedActionMessages/"
)

// Error messages.
const (
	// ActionSaver.load
	loadSavedActionsChanIDsErr  = "could not load list of channels"
	loadSavedActionsMessagesErr = "could not load saved action for channel %s"

	// ActionSaver.updateStorage
	storeSavedActionsMessagesErr = "could not store saved actions for channel %s"
	storeSavedActionChanIDsErr   = "could not store saved action channel IDs"
)

// load gets all the command messages from storage and loads them into the
// action list and map. If any of the messages have a lease trigger
// before now, then they are assigned a new lease trigger between 5 and 30
// minutes in the future to allow alternate replays a chance to be picked up.
func (as *ActionSaver) load(now time.Time) error {
	// Get list of channel IDs
	channelIDs, err := as.loadChannelList()
	if err != nil {
		return errors.Wrap(err, loadSavedActionsChanIDsErr)
	}

	// Get list of lease messages and load them into the message map and lease
	// list
	for _, channelID := range channelIDs {
		actionList, err2 := as.loadActions(channelID)
		if err2 != nil {
			return errors.Wrapf(err2, loadSavedActionsMessagesErr, channelID)
		}

		for key, sa := range actionList {
			// Check if the action is stale
			if now.Sub(sa.Received) > maxSavedActionAge {
				delete(actionList, key)
			}
		}

		if len(actionList) > 0 {
			as.actions[*channelID] = actionList
		}
	}

	return nil
}

// updateStorage updates the given channel lease list in storage. If
// channelIdUpdate is true, then the main list of channel IDs is also updated.
// Use this option when adding or removing a channel ID from the message map.
func (as *ActionSaver) updateStorage(
	channelID *id.ID, channelIdUpdate bool) error {
	if err := as.storeActions(channelID); err != nil {
		return errors.Wrapf(err, storeSavedActionsMessagesErr, channelID)
	} else if channelIdUpdate {
		if err = as.storeChannelList(); err != nil {
			return errors.Wrap(err, storeSavedActionChanIDsErr)
		}
	}
	return nil
}

// storeChannelList stores the list of all channel IDs in the action list to
// storage.
func (as *ActionSaver) storeChannelList() error {
	channelIDs := make([]*id.ID, 0, len(as.actions))
	for chanID := range as.actions {
		channelIDs = append(channelIDs, chanID.DeepCopy())
	}

	data, err := json.Marshal(&channelIDs)
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

// loadChannelList loads the list of all channel IDs in the action list from
// storage.
func (as *ActionSaver) loadChannelList() ([]*id.ID, error) {
	obj, err := as.kv.Get(savedActionsKey, savedActionsVer)
	if err != nil {
		return nil, err
	}

	var channelIDs []*id.ID
	return channelIDs, json.Unmarshal(obj.Data, &channelIDs)
}

// storeActions stores the list of savedAction objects for the given channel ID
// to storage keying on the channel ID.
func (as *ActionSaver) storeActions(channelID *id.ID) error {
	// If the list is empty, then delete it from storage
	if len(as.actions[*channelID]) == 0 {
		return as.deleteActions(channelID)
	}

	data, err := json.Marshal(as.actions[*channelID])
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   savedActionsMessagesVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return as.kv.Set(makeSavedActionMessagesKey(channelID), obj)
}

// loadActions loads the list of savedAction from storage keyed on the
// channel ID.
func (as *ActionSaver) loadActions(channelID *id.ID) (
	map[messageIdKey]*savedAction, error) {
	obj, err := as.kv.Get(
		makeSavedActionMessagesKey(channelID), savedActionsMessagesVer)
	if err != nil {
		return nil, err
	}

	var messages map[messageIdKey]*savedAction
	return messages, json.Unmarshal(obj.Data, &messages)
}

// deleteActions deletes the list of savedAction from storage that is
// keyed on the channel ID.
func (as *ActionSaver) deleteActions(channelID *id.ID) error {
	return as.kv.Delete(
		makeSavedActionMessagesKey(channelID), savedActionsMessagesVer)
}

// makeSavedActionMessagesKey creates a key for saving savedAction for
// actions for the given channel.
func makeSavedActionMessagesKey(channelID *id.ID) string {
	return savedActionsMessagesKeyPrefix +
		base64.StdEncoding.EncodeToString(channelID.Marshal())
}
