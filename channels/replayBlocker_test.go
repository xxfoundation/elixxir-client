////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Tests that newOrLoadReplayBlocker initialises a new empty replayBlocker when
// called for the first time and that it loads the replayBlocker from storage
// after the original has been saved.
func Test_newOrLoadReplayBlocker(t *testing.T) {
	prng := rand.New(rand.NewSource(986))
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewCommandStore(kv)
	expected := newReplayBlocker(nil, s, kv)

	rb, err := newOrLoadReplayBlocker(nil, s, kv)
	if err != nil {
		t.Fatalf("Failed to create new replayBlocker: %+v", err)
	}
	if !reflect.DeepEqual(expected, rb) {
		t.Errorf("New replayBlocker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, rb)
	}

	cm := &commandMessage{
		ChannelID:        randChannelID(prng, t),
		Action:           randAction(prng),
		Payload:          randPayload(prng, t),
		OriginatingRound: id.Round(prng.Uint64()),
	}
	unsanitizedPayload := randPayload(prng, t)
	cm.UnsanitizedFP =
		makeUnsanitizedFP(cm.ChannelID, cm.Action, unsanitizedPayload)

	valid, err := rb.verifyReplay(cm.ChannelID, message.ID{}, cm.Action,
		randPayload(prng, t), cm.Payload, nil, time.Time{}, time.Time{}, 0,
		cm.OriginatingRound, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Error verifying replay: %+v", err)
	}

	if !valid {
		t.Errorf("Replay not valid when it should be.")
	}

	// Create new list and load old contents into it
	loadedRb, err := newOrLoadReplayBlocker(nil, s, kv)
	if err != nil {
		t.Errorf("Failed to load replayBlocker from storage: %+v", err)
	}
	if !reflect.DeepEqual(rb, loadedRb) {
		t.Errorf("Loaded replayBlocker does not match expected."+
			"\nexpected: %+v\nreceived: %+v\nexpected: %+v\nreceived: %+v",
			rb, loadedRb, rb.commandsByChannel, loadedRb.commandsByChannel)
	}
}

// Tests that newReplayBlocker returns the expected new replayBlocker.
func Test_newReplayBlocker(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewCommandStore(kv)

	expected := &replayBlocker{
		commandsByChannel: make(map[id.ID]map[commandFingerprintKey]*commandMessage),
		store:             s,
		kv:                kv.Prefix(replayBlockerStoragePrefix),
	}

	rb := newReplayBlocker(nil, s, kv)

	if !reflect.DeepEqual(expected, rb) {
		t.Errorf("New replayBlocker does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, rb)
	}
}

// Tests that verifyReplay only adds messages that are verified and that
// messages with older originating rounds IDs are rejected.
func Test_replayBlocker_verifyReplay(t *testing.T) {
	prng := rand.New(rand.NewSource(321585))
	replayChan := make(chan *commandMessage)
	replay := func(channelID *id.ID, action MessageType, payload []byte) error {
		replayChan <- &commandMessage{channelID, action, payload, 0, 0}
		return nil
	}
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(replay, NewCommandStore(kv), kv)

	cm := &commandMessage{
		ChannelID:        randChannelID(prng, t),
		Action:           randAction(prng),
		Payload:          randPayload(prng, t),
		OriginatingRound: id.Round(prng.Uint64()),
	}
	unsanitizedPayload := randPayload(prng, t)
	cm.UnsanitizedFP =
		makeUnsanitizedFP(cm.ChannelID, cm.Action, unsanitizedPayload)

	// Insert the command message and test that it was inserted
	valid, err := rb.verifyReplay(cm.ChannelID, message.ID{}, cm.Action,
		unsanitizedPayload, cm.Payload, nil, time.Time{}, time.Time{}, 0,
		cm.OriginatingRound, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Error verifying replay: %+v", err)
	} else if !valid {
		t.Errorf("Replay not valid when it should be.")
	}

	fp := newCommandFingerprint(cm.ChannelID, cm.Action, cm.Payload)
	if rm2, exists := rb.commandsByChannel[*cm.ChannelID][fp.key()]; !exists {
		t.Errorf("commandMessage not inserted into map.")
	} else if !reflect.DeepEqual(cm, rm2) {
		t.Errorf("Incorrect commandMessage.\nexpected: %+v\nreceived: %+v",
			cm, rm2)
	}

	// Increase the round and test that the message gets overwritten
	cm.OriginatingRound++
	valid, err = rb.verifyReplay(cm.ChannelID, message.ID{}, cm.Action,
		unsanitizedPayload, cm.Payload, nil, time.Time{}, time.Time{}, 0,
		cm.OriginatingRound, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Error verifying replay: %+v", err)
	} else if !valid {
		t.Errorf("Replay not valid when it should be.")
	}

	if rm2, exists := rb.commandsByChannel[*cm.ChannelID][fp.key()]; !exists {
		t.Errorf("commandMessage not inserted into map.")
	} else if !reflect.DeepEqual(cm, rm2) {
		t.Errorf("Incorrect commandMessage.\nexpected: %+v\nreceived: %+v",
			cm, rm2)
	}

	// Decrease the round and test that the message is not overwritten and that
	// verifyReplay returns false
	cm.OriginatingRound--
	valid, err = rb.verifyReplay(cm.ChannelID, message.ID{}, cm.Action,
		randPayload(prng, t), cm.Payload, nil, time.Time{}, time.Time{}, 0,
		cm.OriginatingRound, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Error verifying replay: %+v", err)
	} else if valid {
		t.Errorf("Replay valid when it should not be.")
	}

	if cm2, exists := rb.commandsByChannel[*cm.ChannelID][fp.key()]; !exists {
		t.Errorf("commandMessage not inserted into map.")
	} else if reflect.DeepEqual(cm, cm2) {
		t.Errorf("commandMessage was changed when it is invalid."+
			"\nexpected: %+v\nreceived: %+v", cm, cm2)
	}

	select {
	case <-replayChan:
	case <-netTime.After(20 * time.Millisecond):
		t.Errorf("Timed out waiting for replay trigger to be called.")
	}
}

// Tests that replayBlocker.removeCommand removes all the messages from the
// message map.
func Test_replayBlocker_removeCommand(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)

	const m, n, o = 20, 5, 3
	expected := make([]*commandMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := randChannelID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and originating rounds)
			payload := randPayload(prng, t)

			for k := 0; k < o; k++ {
				cm := &commandMessage{
					ChannelID:        channelID,
					Action:           MessageType(k),
					Payload:          payload,
					OriginatingRound: id.Round(k),
				}
				verified, err := rb.verifyReplay(cm.ChannelID, message.ID{},
					cm.Action, randPayload(prng, t), cm.Payload, nil,
					time.Time{}, time.Time{}, 0, cm.OriginatingRound,
					rounds.Round{}, false)
				if err != nil {
					t.Fatalf("Error verfying command (%d, %d, %d): %+v",
						i, j, k, err)
				} else if !verified {
					t.Errorf("Command could not be verified (%d, %d, %d)",
						i, j, k)
				}

				fp := newCommandFingerprint(channelID, cm.Action, payload)
				expected = append(
					expected, rb.commandsByChannel[*channelID][fp.key()])
			}
		}
	}

	for i, exp := range expected {
		err := rb.removeCommand(exp.ChannelID, exp.Action, exp.Payload)
		if err != nil {
			t.Errorf("Failed to remove message %d: %+v", i, exp)
		}

		// Check that the message was removed from the map
		fp := newCommandFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := rb.commandsByChannel[*exp.ChannelID]; exists {
			if _, exists = messages[fp.key()]; exists {
				t.Errorf("Removed commandMessage found with key %s (%d).",
					fp.key(), i)
			}
		}
	}
}

// Tests that replayBlocker.removeCommand returns nil when trying to remove a
// message for a channel that does not exist.
func Test_replayBlocker_removeCommand_NoChannel(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)

	err := rb.removeCommand(
		randChannelID(prng, t), randAction(prng), randPayload(prng, t))
	if err != nil {
		t.Errorf("Error removoing message for channel that does not exist")
	}

	expected := newReplayBlocker(rb.replay, rb.store, kv)
	if !reflect.DeepEqual(expected, rb) {
		t.Errorf("Unexpected replayBlocker after removing command that does "+
			"not exist.\nexpected: %+v\nreceoved: %+v", expected, rb)
	}
}

// Tests that replayBlocker.removeCommand returns nil when trying to remove a
// message that does not exist (but the channel does)
func Test_replayBlocker_removeCommand_NoMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)

	cm := &commandMessage{
		ChannelID:        randChannelID(prng, t),
		Action:           randAction(prng),
		Payload:          randPayload(prng, t),
		OriginatingRound: id.Round(prng.Uint64()),
	}

	_, _ = rb.verifyReplay(cm.ChannelID, message.ID{}, cm.Action,
		randPayload(prng, t), cm.Payload, nil, time.Time{}, time.Time{}, 0,
		cm.OriginatingRound, rounds.Round{}, false)
	err := rb.removeCommand(cm.ChannelID, cm.Action, randPayload(prng, t))
	if err != nil {
		t.Errorf("Error removoing message for channel that does not exist")
	}
}

// Tests that replayBlocker.removeChannelCommands removes all the messages from
// the message map for the given channel.
func Test_replayBlocker_removeChannelCommands(t *testing.T) {
	prng := rand.New(rand.NewSource(2345))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)

	const m, n, o = 20, 5, 3
	expected := make([]*commandMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := randChannelID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and originating rounds)
			payload := randPayload(prng, t)

			for k := 0; k < o; k++ {
				cm := &commandMessage{
					ChannelID:        channelID,
					Action:           MessageType(k),
					Payload:          payload,
					OriginatingRound: id.Round(k),
				}
				verified, err := rb.verifyReplay(cm.ChannelID, message.ID{},
					cm.Action, randPayload(prng, t), cm.Payload, nil,
					time.Time{}, time.Time{}, 0, cm.OriginatingRound,
					rounds.Round{}, false)
				if err != nil {
					t.Fatalf("Error verfying command (%d, %d, %d): %+v",
						i, j, k, err)
				} else if !verified {
					t.Errorf("Command could not be verified (%d, %d, %d)",
						i, j, k)
				}

				fp := newCommandFingerprint(channelID, cm.Action, payload)
				expected = append(
					expected, rb.commandsByChannel[*channelID][fp.key()])
			}
		}
	}

	// Get random channel ID
	var channelID id.ID
	for channelID = range rb.commandsByChannel {
		break
	}

	err := rb.removeChannelCommands(&channelID)
	if err != nil {
		t.Errorf("Failed to remove channel: %+v", err)
	}

	if messages, exists := rb.commandsByChannel[channelID]; exists {
		t.Errorf("Channel commands not deleted: %+v", messages)
	}

	// Test removing a channel that does not exist
	err = rb.removeChannelCommands(randChannelID(prng, t))
	if err != nil {
		t.Errorf("Error when removing non-existent channel: %+v", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that replayBlocker.load loads a replayBlocker from storage that matches
// the original.
func Test_replayBlocker_load(t *testing.T) {
	prng := rand.New(rand.NewSource(986))
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewCommandStore(kv)
	rb := newReplayBlocker(nil, s, kv)

	for i := 0; i < 10; i++ {
		channelID := randChannelID(prng, t)
		rb.commandsByChannel[*channelID] =
			make(map[commandFingerprintKey]*commandMessage)
		for j := 0; j < 5; j++ {
			cm := &commandMessage{
				ChannelID:        channelID,
				Action:           randAction(prng),
				Payload:          randPayload(prng, t),
				OriginatingRound: id.Round(prng.Uint64()),
			}

			fp := newCommandFingerprint(channelID, cm.Action, cm.Payload)
			rb.commandsByChannel[*channelID][fp.key()] = cm
		}

		err := rb.updateStorage(channelID, true)
		if err != nil {
			t.Errorf("Failed to update storage for channel %s (%d): %+v",
				channelID, i, err)
		}
	}

	// Create new list and load old contents into it
	loadedRb := newReplayBlocker(nil, s, kv)
	err := loadedRb.load()
	if err != nil {
		t.Errorf("Failed to load replayBlocker from storage: %+v", err)
	}

	// Check that the loaded message map matches the original
	for chanID, messages := range rb.commandsByChannel {
		loadedMessages, exists := rb.commandsByChannel[chanID]
		if !exists {
			t.Errorf("Channel ID %s does not exist in map.", chanID)
		}

		for fp, cm := range messages {
			loadedRm, exists2 := loadedMessages[fp]
			if !exists2 {
				t.Errorf("Command message does not exist in map: %+v", cm)
			}
			if !reflect.DeepEqual(cm, loadedRm) {
				t.Errorf("commandMessage does not match expected."+
					"\nexpected: %+v\nreceived: %+v", cm, loadedRm)
			}
		}
	}
}

// Error path: Tests that replayBlocker.load returns the expected error when no
// channel IDs can be loaded from storage.
func Test_replayBlocker_load_ChannelListLoadError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)
	expectedErr := loadCommandChanIDsErr

	err := rb.load()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to return expected error no channel ID list exists."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: Tests that replayBlocker.load returns the expected error when no
// command messages can be loaded from storage.
func Test_replayBlocker_load_CommandMessagesLoadError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)

	channelID := randChannelID(rand.New(rand.NewSource(456)), t)
	rb.commandsByChannel[*channelID] =
		make(map[commandFingerprintKey]*commandMessage)
	err := rb.storeCommandChannelsList()
	if err != nil {
		t.Fatalf("Failed to store channel list: %+v", err)
	}

	expectedErr := fmt.Sprintf(loadCommandMessagesErr, channelID)

	err = rb.load()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to return expected error no command messages exist."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the list of channel IDs in the message map can be saved and loaded
// to and from storage with replayBlocker.storeCommandChannelsList and
// replayBlocker.loadCommandChannelsList.
func Test_replayBlocker_storeCommandChannelsList_loadCommandChannelsList(t *testing.T) {
	const n = 10
	prng := rand.New(rand.NewSource(986))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)
	expectedIDs := make([]*id.ID, n)

	for i := 0; i < n; i++ {
		channelID := randChannelID(prng, t)
		rb.commandsByChannel[*channelID] =
			make(map[commandFingerprintKey]*commandMessage)
		for j := 0; j < 5; j++ {
			action, payload := randAction(prng), randPayload(prng, t)
			fp := newCommandFingerprint(channelID, action, payload)
			rb.commandsByChannel[*channelID][fp.key()] = &commandMessage{
				ChannelID:        channelID,
				Action:           action,
				Payload:          payload,
				OriginatingRound: id.Round(prng.Uint64()),
			}
		}
		expectedIDs[i] = channelID
	}

	err := rb.storeCommandChannelsList()
	if err != nil {
		t.Errorf("Failed to store channel IDs: %+v", err)
	}

	loadedIDs, err := rb.loadCommandChannelsList()
	if err != nil {
		t.Errorf("Failed to load channel IDs: %+v", err)
	}

	sort.SliceStable(expectedIDs, func(i, j int) bool {
		return bytes.Compare(expectedIDs[i][:], expectedIDs[j][:]) == -1
	})
	sort.SliceStable(loadedIDs, func(i, j int) bool {
		return bytes.Compare(loadedIDs[i][:], loadedIDs[j][:]) == -1
	})

	if !reflect.DeepEqual(expectedIDs, loadedIDs) {
		t.Errorf("Loaded channel IDs do not match original."+
			"\nexpected: %+v\nreceived: %+v", expectedIDs, loadedIDs)
	}
}

// Error path: Tests that replayBlocker.loadCommandChannelsList returns an error
// when trying to load from storage when nothing was saved.
func Test_replayBlocker_loadCommandChannelsList_StorageError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)

	_, err := rb.loadCommandChannelsList()
	if err == nil || kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that a list of commandMessage can be stored and loaded using
// replayBlocker.storeCommandMessages and replayBlocker.loadCommandMessages.
func Test_replayBlocker_storeCommandMessages_loadCommandMessages(t *testing.T) {
	prng := rand.New(rand.NewSource(986))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)
	channelID := randChannelID(prng, t)
	rb.commandsByChannel[*channelID] =
		make(map[commandFingerprintKey]*commandMessage)

	for i := 0; i < 15; i++ {
		lm := &commandMessage{
			ChannelID:        randChannelID(prng, t),
			Action:           randAction(prng),
			Payload:          randPayload(prng, t),
			OriginatingRound: id.Round(prng.Uint64()),
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		rb.commandsByChannel[*channelID][fp.key()] = lm
	}

	err := rb.storeCommandMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	loadedMessages, err := rb.loadCommandMessages(channelID)
	if err != nil {
		t.Errorf("Failed to load messages: %+v", err)
	}

	if !reflect.DeepEqual(rb.commandsByChannel[*channelID], loadedMessages) {
		t.Errorf("Loaded messages do not match original."+
			"\nexpected: %+v\nreceived: %+v",
			rb.commandsByChannel[*channelID], loadedMessages)
	}
}

// Tests that replayBlocker.storeCommandMessages deletes the Command message
// file from storage when the list is empty.
func Test_replayBlocker_storeCommandMessages_EmptyList(t *testing.T) {
	prng := rand.New(rand.NewSource(986))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)
	channelID := randChannelID(prng, t)
	rb.commandsByChannel[*channelID] =
		make(map[commandFingerprintKey]*commandMessage)

	for i := 0; i < 15; i++ {
		lm := &commandMessage{
			ChannelID:        randChannelID(prng, t),
			Action:           randAction(prng),
			Payload:          randPayload(prng, t),
			OriginatingRound: id.Round(prng.Uint64()),
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		rb.commandsByChannel[*channelID][fp.key()] = lm
	}

	err := rb.storeCommandMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	rb.commandsByChannel[*channelID] =
		make(map[commandFingerprintKey]*commandMessage)
	err = rb.storeCommandMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	_, err = rb.loadCommandMessages(channelID)
	if err == nil || rb.kv.Exists(err) {
		t.Fatalf("Failed to delete command messages: %+v", err)
	}
}

// Error path: Tests that replayBlocker.loadCommandMessages returns an error
// when trying to load from storage when nothing was saved.
func Test_replayBlocker_loadCommandMessages_StorageError(t *testing.T) {
	prng := rand.New(rand.NewSource(986))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)

	_, err := rb.loadCommandMessages(randChannelID(prng, t))
	if err == nil || rb.kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that replayBlocker.deleteCommandMessages removes the command messages
// from storage.
func Test_replayBlocker_deleteCommandMessages(t *testing.T) {
	prng := rand.New(rand.NewSource(986))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rb := newReplayBlocker(nil, NewCommandStore(kv), kv)
	channelID := randChannelID(prng, t)
	rb.commandsByChannel[*channelID] =
		make(map[commandFingerprintKey]*commandMessage)

	for i := 0; i < 15; i++ {
		lm := &commandMessage{
			ChannelID:        randChannelID(prng, t),
			Action:           randAction(prng),
			Payload:          randPayload(prng, t),
			OriginatingRound: id.Round(prng.Uint64()),
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		rb.commandsByChannel[*channelID][fp.key()] = lm
	}

	err := rb.storeCommandMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	err = rb.deleteCommandMessages(channelID)
	if err != nil {
		t.Errorf("Failed to delete messages: %+v", err)
	}

	_, err = rb.loadCommandMessages(channelID)
	if err == nil || rb.kv.Exists(err) {
		t.Fatalf("Failed to delete command messages: %+v", err)
	}
}

// Tests that a commandMessage object can be JSON marshalled and unmarshalled.
func Test_commandMessage_JSON(t *testing.T) {
	prng := rand.New(rand.NewSource(9685))

	cm := commandMessage{
		ChannelID:        randChannelID(prng, t),
		Action:           randAction(prng),
		Payload:          randPayload(prng, t),
		OriginatingRound: id.Round(prng.Uint64()),
	}

	data, err := json.Marshal(&cm)
	if err != nil {
		t.Errorf("Failed to JSON marshal commandMessage: %+v", err)
	}

	var loadedRm commandMessage
	err = json.Unmarshal(data, &loadedRm)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal commandMessage: %+v", err)
	}

	if !reflect.DeepEqual(cm, loadedRm) {
		t.Errorf("Loaded commandMessage does not match original."+
			"\nexpected: %#v\nreceived: %#v", cm, loadedRm)
	}
}

// Tests that a map of commandMessage objects can be JSON marshalled and
// unmarshalled.
func Test_commandMessageMap_JSON(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	const n = 15
	messages := make(map[commandFingerprintKey]*commandMessage, n)

	for i := 0; i < n; i++ {
		lm := &commandMessage{
			ChannelID:        randChannelID(prng, t),
			Action:           randAction(prng),
			Payload:          randPayload(prng, t),
			OriginatingRound: id.Round(prng.Uint64()),
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		messages[fp.key()] = lm
	}

	data, err := json.Marshal(&messages)
	if err != nil {
		t.Errorf("Failed to JSON marshal map of commandMessage: %+v", err)
	}

	var loadedMessages map[commandFingerprintKey]*commandMessage
	err = json.Unmarshal(data, &loadedMessages)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal map of commandMessage: %+v", err)
	}

	if !reflect.DeepEqual(messages, loadedMessages) {
		t.Errorf("Loaded map of commandMessage does not match original."+
			"\nexpected: %+v\nreceived: %+v", messages, loadedMessages)
	}
}

// Consistency test of makeChannelCommandMessagesKey.
func Test_makeChannelCommandMessagesKey_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(4978))

	expectedKeys := []string{
		"42ab84b199deac60a60ffd86874d04af1d225930223f4c1d6dd9e2a9f9d8e6c003",
		"0c9532f8e6ed4285f80ed260e04732a9641e66baa3a7d4d8a88a44cd2f363a8603",
		"623b05551182e5c1cad1a193543e938f5c5f69bce5e4efac1707649421a0934b03",
		"c21fc83c25e502237d2f4faeb3a42b786823ff637f7ba6c6512411186c17b7c303",
		"22b323be76037f9e97d443cc47a0e45884f1b178c0d056b8361ead091cc9ae4003",
		"7da2d0d3ea7004ad57da6d95e6a3ed7f1bb32738ac556a80a2a8c5a6e446014d03",
		"a03b1fe700cae64411c56ef4a1a7de2c641d34f79ce3a6b3940b9648d800cf9603",
		"f61f471981e005d0ef720204bbea600fa1d660f1591f16ca93dc5d61ceaf2af603",
		"ed783b3743e9207cc7651b5f4864d61e8556b4898f42df715e590ed90d24078b03",
		"b634426a3007d4cbf2e103517e04b1e81ead5bbcc5ddb210c75c228cf5acd1d903",
		"721bc300fae39398d82e31972107ab5864e46e8658cd7043dcb0cdcfb161688903",
		"4fcd4542546819ddeca86246a894e824e930ef48627a0277eb7873a086000d6403",
		"1fedb554c4bf5c335860a02d93529a421a213cc0a8494840aa45c78f1d46a58803",
		"e6e161e6620cd74a967a09736a439de7f145fd88f6e422d3e09c075820ce6a4103",
		"8313bda62b20b564611bf018630f472d54149eead4d85dcf2e7da043c70ccf7b03",
		"cf67c35c5f19086098cd7a3bffd2e8975267d65f202f733b29faca624a6ba8cf03",
	}
	for i, expected := range expectedKeys {
		key := makeChannelCommandMessagesKey(randChannelID(prng, t))

		if expected != key {
			t.Errorf("Key does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}
	}
}

// makeUnsanitizedFP generates a sanitised fingerprint.
func makeUnsanitizedFP(
	channelID *id.ID, action MessageType, unsanitizedPayload []byte) uint64 {
	fp := newCommandFingerprint(channelID, action, unsanitizedPayload)
	return binary.LittleEndian.Uint64(fp[:])
}
