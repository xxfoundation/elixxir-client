////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

// Error messages.
const (
	// replayBlocker.verifyReplay
	saveReplayCommandMessageErr = "failed to save command message"
)

// replayBlocker ensures that any channel commands received as a replay messages
// are newer than the most recent command message. If it is not, then the
// replayBlocker rejects it and replays the correct command.
type replayBlocker struct {
	// List of command messages grouped by the channel and keyed on a unique
	// fingerprint.
	commandsByChannel map[id.ID]map[commandFingerprintKey]*commandMessage

	// replay allows a command to be replayed.
	replay triggerLeaseReplay

	store *CommandStore
	kv    versioned.KV
	mux   sync.Mutex
}

// triggerLeaseReplay takes the information needed to schedule a replay on the
// lease system.
type triggerLeaseReplay func(
	channelID *id.ID, action MessageType, payload []byte) error

// commandMessage contains the information to uniquely identify a command in a
// channel and which round it originated from and the round it was last replayed
// on.
type commandMessage struct {
	// ChannelID is the ID of the channel that his message is in.
	ChannelID *id.ID `json:"channelID"`

	// Action is the action applied to the message (currently only Pinned and
	// Mute).
	Action MessageType `json:"action"`

	// Payload is the contents of the ChannelMessage.Payload.
	Payload []byte `json:"payload"`

	// OriginatingRound is the ID of the round the message was originally sent
	// on.
	OriginatingRound id.Round `json:"originatingRound"`

	// UnsanitizedFP is the first 8 bytes of the commandFingerprint generated
	// using the unsanitized payload.
	UnsanitizedFP uint64 `json:"unsanitizedFP"`
}

// newOrLoadReplayBlocker loads an existing replayBlocker from storage, if it
// exists. Otherwise, it initialises a new empty replayBlocker.
func newOrLoadReplayBlocker(replay triggerLeaseReplay, store *CommandStore,
	kv versioned.KV) (*replayBlocker, error) {
	rb := newReplayBlocker(replay, store, kv)

	err := rb.load()
	if err != nil && kv.Exists(err) {
		return nil, err
	}

	return rb, nil
}

// newReplayBlocker initialises a new empty replayBlocker.
func newReplayBlocker(replay triggerLeaseReplay, store *CommandStore,
	kv versioned.KV) *replayBlocker {
	kv, err := kv.Prefix(replayBlockerStoragePrefix)
	if err != nil {
		jww.FATAL.Panicf("[CH] Failed to add prefix %s to KV: %+v", replayBlockerStoragePrefix, err)
	}
	return &replayBlocker{
		commandsByChannel: make(map[id.ID]map[commandFingerprintKey]*commandMessage),
		replay:            replay,
		store:             store,
		kv:                kv,
	}
}

// verifyReplay verifies if the replay is valid by checking if it is the newest
// version (i.e. the originating round is newer). If it is not, verifyReplay
// returns false. Otherwise, the replay is valid, and it returns true.
func (rb *replayBlocker) verifyReplay(channelID *id.ID, messageID message.ID,
	action MessageType, unsanitizedPayload, sanitizedPayload,
	encryptedPayload []byte, timestamp, originatingTimestamp time.Time,
	lease time.Duration, originatingRound id.Round, round rounds.Round,
	fromAdmin bool) (bool, error) {
	fp := newCommandFingerprint(channelID, action, sanitizedPayload)
	unsanitizedFp := newCommandFingerprint(channelID, action, unsanitizedPayload)

	newCm := &commandMessage{
		ChannelID:        channelID,
		Action:           action,
		Payload:          sanitizedPayload,
		OriginatingRound: originatingRound,
		UnsanitizedFP:    binary.LittleEndian.Uint64(unsanitizedFp[:]),
	}

	var cm *commandMessage
	var channelIdUpdate bool

	start := netTime.Now()
	rb.mux.Lock()
	defer rb.mux.Unlock()

	// Check that the mux did not block for too long and print a warning if it
	// does. This is done to make sure that the replay blocker does not cause
	// too much of a delay on message handling.
	if since := netTime.Since(start); since > 100*time.Millisecond {
		jww.WARN.Printf("Replay blocker waited %s at mux. This is too long "+
			"and indicates that the replay blocker needs to be modified to "+
			"fix this.", since)
	}

	if messages, exists := rb.commandsByChannel[*channelID]; exists {
		if cm, exists = messages[fp.key()]; exists &&
			cm.OriginatingRound >= newCm.OriginatingRound &&
			cm.UnsanitizedFP != newCm.UnsanitizedFP {
			// If the message is replaying an older command, then reject the
			// message (return false) and replay the correct command
			go func(cm *commandMessage) {
				err := rb.replay(cm.ChannelID, cm.Action, cm.Payload)
				if err != nil {
					jww.ERROR.Printf(
						"[CH] Failed to replay %s on channel %s: %+v",
						cm.Action, cm.ChannelID, err)
				}
			}(cm)
			return false, nil
		} else {
			// Add the command message if it does not exist or overwrite if the
			// new message occurred on a newer round
			rb.commandsByChannel[*channelID][fp.key()] = newCm
		}
	} else {
		// Add the channel if it does not exist
		rb.commandsByChannel[*channelID] =
			map[commandFingerprintKey]*commandMessage{fp.key(): newCm}
		channelIdUpdate = true
	}

	// Save message details to storage
	err := rb.store.SaveCommand(channelID, messageID, action, "",
		sanitizedPayload, encryptedPayload, nil, 0, timestamp,
		originatingTimestamp, lease, originatingRound, round, 0, fromAdmin,
		false)
	if err != nil {
		return true, errors.Wrap(err, saveReplayCommandMessageErr)
	}

	// Update storage
	return true, rb.updateStorage(channelID, channelIdUpdate)
}

// removeCommand removes the command from the command list and removes it from
// storage.
func (rb *replayBlocker) removeCommand(
	channelID *id.ID, action MessageType, payload []byte) error {
	fp := newCommandFingerprint(channelID, action, payload)

	rb.mux.Lock()
	defer rb.mux.Unlock()

	if messages, exists := rb.commandsByChannel[*channelID]; !exists {
		return nil
	} else if _, exists = messages[fp.key()]; !exists {
		return nil
	}

	// When set to true, the list of channels IDs will be updated in storage
	var channelIdUpdate bool

	delete(rb.commandsByChannel[*channelID], fp.key())
	if len(rb.commandsByChannel[*channelID]) == 0 {
		delete(rb.commandsByChannel, *channelID)
		channelIdUpdate = true
	}

	// Update storage
	return rb.updateStorage(channelID, channelIdUpdate)
}

// removeChannelCommands removes all commands for the channel from the messages
// map. Also deletes from storage.
func (rb *replayBlocker) removeChannelCommands(channelID *id.ID) error {
	rb.mux.Lock()
	defer rb.mux.Unlock()

	commands, exists := rb.commandsByChannel[*channelID]
	if !exists {
		return nil
	}

	for _, cm := range commands {
		err := rb.store.DeleteCommand(cm.ChannelID, cm.Action, cm.Payload)
		if err != nil && rb.store.kv.Exists(err) {
			jww.ERROR.Printf("[CH] Failed to delete command %s for channel %s "+
				"from storage: %+v", cm.Action, cm.ChannelID, err)
		}
	}

	delete(rb.commandsByChannel, *channelID)

	err := rb.storeCommandChannelsList()
	if err != nil {
		return err
	}

	return rb.deleteCommandMessages(channelID)
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Storage values.
const (
	replayBlockerStoragePrefix = "channelReplayBlocker"

	commandChannelListVer     = 0
	commandChannelListKey     = "channelCommandList"
	channelCommandMessagesVer = 0
)

// Error messages.
const (
	// replayBlocker.updateStorage
	storeCommandMessagesErr = "could not store command messages for channel %s: %+v"
	storeCommandChanIDsErr  = "could not store command channel IDs: %+v"

	// replayBlocker.load
	loadCommandChanIDsErr  = "could not load list of channels"
	loadCommandMessagesErr = "could not load command messages for channel %s"
)

// load gets all the command messages from storage and loads them into the
// message map.
func (rb *replayBlocker) load() error {
	// Get list of channel IDs
	channelIDs, err := rb.loadCommandChannelsList()
	if err != nil {
		return errors.Wrap(err, loadCommandChanIDsErr)
	}

	// Get list of command messages and load them into the message map
	for _, channelID := range channelIDs {
		rb.commandsByChannel[*channelID], err = rb.loadCommandMessages(channelID)
		if err != nil {
			return errors.Wrapf(err, loadCommandMessagesErr, channelID)
		}
	}

	return nil
}

// updateStorage updates the given channel command list in storage. If
// channelIdUpdate is true, then the main list of channel IDs is also updated.
// Use this option when adding or removing a channel ID from the message map.
func (rb *replayBlocker) updateStorage(
	channelID *id.ID, channelIdUpdate bool) error {
	if err := rb.storeCommandMessages(channelID); err != nil {
		return errors.Errorf(storeCommandMessagesErr, channelID, err)
	} else if channelIdUpdate {
		if err = rb.storeCommandChannelsList(); err != nil {
			return errors.Errorf(storeCommandChanIDsErr, err)
		}
	}
	return nil
}

// storeCommandChannelsList stores the list of all channel IDs in the command
// list to storage.
func (rb *replayBlocker) storeCommandChannelsList() error {
	channelIDs := make([]*id.ID, 0, len(rb.commandsByChannel))
	for chanID := range rb.commandsByChannel {
		channelIDs = append(channelIDs, chanID.DeepCopy())
	}

	data, err := json.Marshal(&channelIDs)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   commandChannelListVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return rb.kv.Set(commandChannelListKey, obj)
}

// loadCommandChannelsList loads the list of all channel IDs in the command list
// from storage.
func (rb *replayBlocker) loadCommandChannelsList() ([]*id.ID, error) {
	obj, err := rb.kv.Get(commandChannelListKey, commandChannelListVer)
	if err != nil {
		return nil, err
	}

	var channelIDs []*id.ID
	return channelIDs, json.Unmarshal(obj.Data, &channelIDs)
}

// storeCommandMessages stores the map of commandMessage objects for the given
// channel ID to storage keying on the channel ID.
func (rb *replayBlocker) storeCommandMessages(channelID *id.ID) error {
	// If the list is empty, then delete it from storage
	if len(rb.commandsByChannel[*channelID]) == 0 {
		return rb.deleteCommandMessages(channelID)
	}

	data, err := json.Marshal(rb.commandsByChannel[*channelID])
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   channelCommandMessagesVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return rb.kv.Set(makeChannelCommandMessagesKey(channelID), obj)
}

// loadCommandMessages loads the map of commandMessage from storage keyed on the
// channel ID.
func (rb *replayBlocker) loadCommandMessages(channelID *id.ID) (
	map[commandFingerprintKey]*commandMessage, error) {
	obj, err := rb.kv.Get(
		makeChannelCommandMessagesKey(channelID), channelCommandMessagesVer)
	if err != nil {
		return nil, err
	}

	var messages map[commandFingerprintKey]*commandMessage
	return messages, json.Unmarshal(obj.Data, &messages)
}

// deleteCommandMessages deletes the map of commandMessage from storage that is
// keyed on the channel ID.
func (rb *replayBlocker) deleteCommandMessages(channelID *id.ID) error {
	return rb.kv.Delete(
		makeChannelCommandMessagesKey(channelID), channelCommandMessagesVer)
}

// makeChannelCommandMessagesKey creates a key for saving channel replay
// messages to storage.
func makeChannelCommandMessagesKey(channelID *id.ID) string {
	return hex.EncodeToString(channelID.Marshal())
}
