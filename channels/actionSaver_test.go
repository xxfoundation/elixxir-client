////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
)

// Tests that NewActionSaver returned the expected new ActionSaver.
func TestNewActionSaver(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &ActionSaver{
		actions: make(map[id.ID]map[messageIdKey]*savedAction),
		kv:      kv,
	}

	as := NewActionSaver(nil, kv)
	if !reflect.DeepEqual(expected, as) {
		t.Errorf("Unexpected new ActionSaver.\nexpected: %+v\nrecieved: %+v",
			expected, as)
	}
}

// Tests that ActionSaver.purge clears only the stale saved actions.
func TestActionSaver_purge(t *testing.T) {
	prng := rand.New(rand.NewSource(3523))
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	now := time.Unix(200, 0)
	expected := make(map[id.ID]map[messageIdKey]*savedAction)
	for i := 0; i < 6; i++ {
		chanID := id.NewIdFromUInt(uint64(i), id.User, t)
		for j := 0; j < 10; j++ {
			var received time.Time
			switch i {
			case 0:
				// All messages in the first channel will be purged
				received = time.Unix(prng.Int63n(200), 0)
			case 1:
				// All messages in the second channel will be preserved
				received = time.Unix(200+prng.Int63n(200), 0)
			default:
				// All other messages are randomly chosen to be purged or not
				received = time.Unix(100+prng.Int63n(200), 0)
			}
			sa := &savedAction{received, message.ID{byte(i), byte(j)},
				CommandMessage{chanID, message.ID{byte(i), byte(j)}, Delete, "",
					[]byte("content"), []byte("encryptedPayload"), nil, 0,
					time.Unix(36, 0), time.Unix(35, 0), 5 * time.Minute, 35,
					rounds.Round{}, 0, false, false}}

			err := as.addAction(sa)
			if err != nil {
				t.Fatalf("Failed to add action: %+v", err)
			}

			if !sa.Received.Before(now) {
				key := getMessageIdKey(sa.TargetMessage)
				if _, exists := expected[*chanID]; !exists {
					expected[*chanID] = map[messageIdKey]*savedAction{key: sa}
				} else {
					expected[*chanID][key] = sa
				}
			}
		}
	}

	now = now.Add(maxSavedActionAge)
	err := as.purge(now)
	if err != nil {
		t.Fatalf("Purge failed: %+v", err)
	}

	if !reflect.DeepEqual(expected, as.actions) {
		t.Errorf("Unexpected actions map after purge at %s."+
			"\nexpected: %+v\nreceived: %+v", now, expected, as.actions)
	}
}

// Tests ActionSaver.AddAction adds a new action when the list is empty,
// overwrites an action if it is older, and that the delete action overwrites
// any present actions (even if they are newer), and that a newer action will
// not overwrite an older delete.
func TestActionSaver_AddAction(t *testing.T) {
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	// Test adding new action
	e := &savedAction{
		Received:      time.Unix(37, 0).Round(0).UTC(),
		TargetMessage: message.ID{7},
		CommandMessage: CommandMessage{
			ChannelID:            id.NewIdFromString("channelID", id.User, t),
			MessageID:            message.ID{36},
			MessageType:          Pinned,
			Content:              []byte("content"),
			EncryptedPayload:     []byte("encryptedPayload"),
			Timestamp:            time.Unix(36, 0).Round(0).UTC(),
			OriginatingTimestamp: time.Unix(35, 0).Round(0).UTC(),
			Lease:                5 * time.Minute,
			OriginatingRound:     35,
			Round:                rounds.Round{},
			FromAdmin:            false,
		},
	}
	err := as.AddAction(e.ChannelID, e.MessageID, e.TargetMessage, e.MessageType,
		e.Content, e.EncryptedPayload, e.Timestamp, e.OriginatingTimestamp,
		e.Received, e.Lease, e.OriginatingRound, e.Round, e.FromAdmin)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	key := getMessageIdKey(e.TargetMessage)
	if sa, exists := as.actions[*e.ChannelID][key]; !exists {
		t.Errorf("Action not saved.")
	} else if !reflect.DeepEqual(e, sa) {
		t.Errorf("Unexpected savedAction.\nexpected: %+v\nreceived: %+v", e, sa)
	}

	// Test adding newer action (it should overwrite the existing one)
	e2 := &savedAction{time.Unix(47, 0), e.TargetMessage, CommandMessage{
		e.ChannelID, message.ID{46}, Pinned, "", []byte("1content"),
		[]byte("1encryptedPayload"), nil, 0, time.Unix(46, 0), time.Unix(45, 0),
		5 * time.Minute, 45, rounds.Round{}, 0, false, false}}
	err = as.addAction(e2)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	if sa, exists := as.actions[*e.ChannelID][key]; !exists {
		t.Errorf("Action not saved.")
	} else if !reflect.DeepEqual(e2, sa) {
		t.Errorf("Unexpected savedAction.\nexpected: %+v\nreceived: %+v", e2, sa)
	}

	// Test adding initial (older) action again (it should be ignored)
	err = as.addAction(e)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	if sa, exists := as.actions[*e.ChannelID][key]; !exists {
		t.Errorf("Action not saved.")
	} else if !reflect.DeepEqual(e2, sa) {
		t.Errorf("Unexpected savedAction.\nexpected: %+v\nreceived: %+v", e2, sa)
	}

	// Test adding an older delete action (it should overwrite)
	e = &savedAction{time.Unix(5, 0), e.TargetMessage, CommandMessage{
		e.ChannelID, message.ID{2}, Delete, "", []byte("2content"),
		[]byte("2encryptedPayload"), nil, 0, time.Unix(6, 0), time.Unix(4, 0),
		5 * time.Minute, 2, rounds.Round{}, 0, false, false}}
	err = as.addAction(e)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	if sa, exists := as.actions[*e.ChannelID][key]; !exists {
		t.Errorf("Action not saved.")
	} else if !reflect.DeepEqual(e, sa) {
		t.Errorf("Unexpected savedAction.\nexpected: %+v\nreceived: %+v", e, sa)
	}

	// Test adding newer action after a deletion (it should be ignored)
	e2 = &savedAction{time.Unix(27, 0), e.TargetMessage, CommandMessage{
		e.ChannelID, message.ID{16}, Pinned, "", []byte("3content"),
		[]byte("3encryptedPayload"), nil, 0, time.Unix(16, 0), time.Unix(15, 0),
		25 * time.Minute, 25, rounds.Round{}, 0, false, false}}
	err = as.addAction(e2)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	if sa, exists := as.actions[*e.ChannelID][key]; !exists {
		t.Errorf("Action not saved.")
	} else if !reflect.DeepEqual(e, sa) {
		t.Errorf("Unexpected savedAction.\nexpected: %+v\nreceived: %+v", e, sa)
	}

	// Test adding action for new target ID to same channel
	e2 = &savedAction{time.Unix(27, 0), message.ID{50}, CommandMessage{
		e.ChannelID, message.ID{16}, Pinned, "", []byte("3content"),
		[]byte("3encryptedPayload"), nil, 0, time.Unix(16, 0), time.Unix(15, 0),
		25 * time.Minute, 25, rounds.Round{}, 0, false, false}}
	err = as.addAction(e2)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	key = getMessageIdKey(e2.TargetMessage)
	if sa, exists := as.actions[*e.ChannelID][key]; !exists {
		t.Errorf("Action not saved.")
	} else if !reflect.DeepEqual(e2, sa) {
		t.Errorf("Unexpected savedAction.\nexpected: %+v\nreceived: %+v", e, sa)
	}
}

// Tests that ActionSaver.CheckSavedActions returns the correct update function
// for the action.
func TestActionSaver_CheckSavedActions(t *testing.T) {
	const expectedUUID = 5
	triggerFn := func(channelID *id.ID, messageID message.ID,
		messageType MessageType, nickname string, payload,
		encryptedPayload []byte, timestamp, originatingTimestamp time.Time,
		lease time.Duration, originatingRound id.Round, round rounds.Round,
		status SentStatus, fromAdmin bool) (uint64, error) {
		return expectedUUID, nil
	}
	as := NewActionSaver(triggerFn, versioned.NewKV(ekv.MakeMemstore()))

	e1 := &savedAction{time.Unix(37, 0), message.ID{7}, CommandMessage{
		id.NewIdFromString("channelID", id.User, t), message.ID{36}, Pinned, "",
		[]byte("content"), []byte("encryptedPayload"), nil, 0, time.Unix(36, 0),
		time.Unix(35, 0), 5 * time.Minute, 35, rounds.Round{}, 0, false, false}}

	err := as.addAction(e1)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	updateFn, deleted := as.CheckSavedActions(e1.ChannelID, e1.TargetMessage)
	if deleted {
		t.Errorf("Saved action should not have been deleted")
	}

	uuid, err := updateFn()
	if err != nil {
		t.Fatal(err)
	} else if uuid != expectedUUID {
		t.Errorf("Incorrect UUID.\nexpected: %d\nreceived: %d",
			expectedUUID, uuid)
	}
}

// Tests that ActionSaver.CheckSavedActions returns true with a nil update
// function when the action is for deletion.
func TestActionSaver_CheckSavedActions_DeletedAction(t *testing.T) {
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	e1 := &savedAction{time.Unix(37, 0), message.ID{7}, CommandMessage{
		id.NewIdFromString("channelID", id.User, t), message.ID{36}, Delete, "",
		[]byte("content"), []byte("encryptedPayload"), nil, 0, time.Unix(36, 0),
		time.Unix(35, 0), 5 * time.Minute, 35, rounds.Round{}, 0, false, false}}

	err := as.addAction(e1)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	updateFn, deleted := as.CheckSavedActions(e1.ChannelID, e1.TargetMessage)
	if !deleted {
		t.Error("Saved action should be marked for deletion")
	}

	if updateFn != nil {
		t.Errorf("Did not receive nil update function")
	}
}

// Tests that ActionSaver.deleteAction deletes the correct element and removes
// the channel from the map when it is empty.
func TestActionSaver_deleteAction(t *testing.T) {
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	e1 := &savedAction{time.Unix(37, 0), message.ID{7}, CommandMessage{
		id.NewIdFromString("channelID", id.User, t), message.ID{36}, Pinned, "",
		[]byte("content"), []byte("encryptedPayload"), nil, 0, time.Unix(36, 0),
		time.Unix(35, 0), 5 * time.Minute, 35, rounds.Round{}, 0, false, false}}
	key1 := getMessageIdKey(e1.TargetMessage)

	err := as.addAction(e1)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	e2 := &savedAction{Received: time.Unix(2, 0), TargetMessage: message.ID{72},
		CommandMessage: CommandMessage{e1.ChannelID, message.ID{36}, Pinned, "",
			[]byte("content2"), []byte("encryptedPayload2"), nil, 0,
			time.Unix(346, 0), time.Unix(335, 0), 5 * time.Minute, 354,
			rounds.Round{}, 0, false, false}}
	err = as.addAction(e2)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	// Test deleting from a channel that does not exist
	err = as.deleteAction(&savedAction{TargetMessage: e2.TargetMessage,
		CommandMessage: CommandMessage{ChannelID: &id.ID{5}}})
	if err != nil {
		t.Fatal(err)
	}

	if len(as.actions) != 1 {
		t.Errorf("Incorrect number of channels.\nexpected: %d\nreceived:%d",
			1, len(as.actions))
	} else if len(as.actions[*e1.ChannelID]) != 2 {
		t.Errorf("Incorrect number of actions for %s.\nexpected: %d\nreceived:%d",
			e1.ChannelID, 2, len(as.actions[*e1.ChannelID]))
	}

	// Test deleting a target ID that does not exist from a channel that does
	// exist
	err = as.deleteAction(&savedAction{TargetMessage: message.ID{90},
		CommandMessage: CommandMessage{ChannelID: e1.ChannelID}})
	if err != nil {
		t.Fatal(err)
	}

	if len(as.actions) != 1 {
		t.Errorf("Incorrect number of channels.\nexpected: %d\nreceived:%d",
			1, len(as.actions))
	} else if len(as.actions[*e1.ChannelID]) != 2 {
		t.Errorf("Incorrect number of actions for %s.\nexpected: %d\nreceived:%d",
			e1.ChannelID, 2, len(as.actions[*e1.ChannelID]))
	}

	// Test deleting an existing action
	err = as.deleteAction(e2)
	if err != nil {
		t.Fatal(err)
	}

	if len(as.actions) != 1 {
		t.Errorf("Incorrect number of channels.\nexpected: %d\nreceived:%d",
			1, len(as.actions))
	} else if len(as.actions[*e1.ChannelID]) != 1 {
		t.Errorf("Incorrect number of actions for %s.\nexpected: %d\nreceived:%d",
			e1.ChannelID, 1, len(as.actions[*e1.ChannelID]))
	} else if !reflect.DeepEqual(e1, as.actions[*e1.ChannelID][key1]) {
		t.Errorf("Incorrect saved action deleted.\nexpected: %+v\nreceived: %+v",
			e1, as.actions[*e1.ChannelID][key1])
	}

	// Test deleting an existing action
	err = as.deleteAction(e1)
	if err != nil {
		t.Fatal(err)
	}

	if len(as.actions) != 0 {
		t.Errorf("Incorrect number of channels.\nexpected: %d\nreceived:%d",
			1, len(as.actions))
	}

}

// Tests that ActionSaver.RemoveChannel removes the given channel but none
// others.
func TestActionSaver_RemoveChannel(t *testing.T) {
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	// Add two new actions to two different channels
	e1 := &savedAction{time.Unix(37, 0), message.ID{7}, CommandMessage{
		id.NewIdFromString("channelID", id.User, t), message.ID{36}, Pinned,
		"", []byte("content"), []byte("encryptedPayload"), nil, 0,
		time.Unix(36, 0), time.Unix(35, 0), 5 * time.Minute, 35, rounds.Round{},
		0, false, false}}
	err := as.addAction(e1)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	e2 := &savedAction{time.Unix(37, 0), message.ID{7}, CommandMessage{
		id.NewIdFromString("channelID2", id.User, t), message.ID{36}, Pinned,
		"", []byte("content"), []byte("encryptedPayload"), nil, 0,
		time.Unix(36, 0), time.Unix(35, 0), 5 * time.Minute, 35, rounds.Round{},
		0, false, false}}
	err = as.addAction(e2)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	// Test removing nonexistent channel
	err = as.RemoveChannel(id.NewIdFromString("channelID3", id.User, t))
	if err != nil {
		t.Fatalf("Failed to remove channel: %+v", err)
	}

	if len(as.actions) != 2 {
		t.Errorf("Incorrect number of channels.\nexpected: %d\nreceived:%d",
			2, len(as.actions))
	}

	// Remove existing channel
	err = as.RemoveChannel(e1.ChannelID)
	if err != nil {
		t.Fatalf("Failed to remove channel: %+v", err)
	}

	if len(as.actions) != 1 {
		t.Errorf("Incorrect number of channels.\nexpected: %d\nreceived:%d",
			1, len(as.actions))
	}

	if _, exists := as.actions[*e1.ChannelID]; exists {
		t.Errorf("Channel was not deleted.")
	}
}

////////////////////////////////////////////////////////////////////////////////
// Message ID Key                                                             //
////////////////////////////////////////////////////////////////////////////////

// Consistency test of getMessageIdKey.
func Test_getMessageIdKey(t *testing.T) {
	prng := rand.New(rand.NewSource(11))

	expectedKeys := []messageIdKey{
		"1JLig6TQMcYcH+Chq8NhwL63Fi61cXeHn5BEqG3iMSk=",
		"sO4/jjtVnQM2/KzOqKpj12YsLbVCrmsUrAPsRp986Bs=",
		"fVuWkVIm/qXl+sHso36XqdFhpBHS6JkzEwGUjBK21nQ=",
		"+nhb3T+rTL4itrNbVFhogYXVg8bA5DP3tBoV///jYb8=",
		"q5uFDJbi7JihzZJ7wIX5bgNMQNVw5IOYK8OYH0+ZZns=",
		"NVUOugb37iJPfeKRRvRMzZspnlDBM7FpCAWwiCmCqDw=",
		"wp8ZzT/8hyaqrkE6CfWwpHlyQN+idhxkVZDRTIqiDkk=",
		"hoNKYd1ElmSBQmQuYa+5NF4SCeH47fxKbHjR8OAMagU=",
		"xZwqDFSqDiRwzjLXBAGe7G/GtDxBvtiH+Qwkuzy7JXI=",
		"9kO20VZDZEW7t6c07lA0nGKqYzYZv35+7Elkd3m94bo=",
		"k6kOxvNLEEbqmiHLb4y5TwTR6EfYfaoY20m3xePwp+8=",
		"RM/uDCZ5DepdGszn7RA+DbEiFiP1wa/EpkVh1ZYW+ZU=",
		"sKiubk2pipg1slpYXnU1LoqrnDZnfaMZY3KIdwqPsfY=",
		"b40mzzjvwY011eG35JZpUaV7Ys4E6O2UrycH1uFJaFE=",
		"dxmwZhaT+lu3mXvSIyK3HEe/Re+Es/1/sa1rMdInFAE=",
		"TgOmIM+BgsVbLSOUvxxvhDkuSMOOxhreeN5e3RJ5eys=",
	}
	for i, expected := range expectedKeys {
		key := getMessageIdKey(randMessageID(prng, t))

		if expected != key {
			t.Errorf("Key does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that ActionSaver.load loads an ActionSaver from storage that matches
// the original.
func TestActionSaver_load(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	as := NewActionSaver(nil, kv)

	now := time.Unix(200, 0)
	expected := make(map[id.ID]map[messageIdKey]*savedAction)
	for i := 0; i < 10; i++ {
		channelID := randChannelID(prng, t)
		as.actions[*channelID] = make(map[messageIdKey]*savedAction)
		for j := 0; j < 5; j++ {
			var received time.Time
			switch i {
			case 0:
				// All messages in the first channel will be purged
				received = time.Unix(prng.Int63n(200), 0)
			case 1:
				// All messages in the second channel will be preserved
				received = time.Unix(200+prng.Int63n(200), 0)
			default:
				// All other messages are randomly chosen to be purged or not
				received = time.Unix(100+prng.Int63n(200), 0)
			}

			sa := &savedAction{received, message.ID{byte(i), byte(j)},
				CommandMessage{channelID, message.ID{byte(i), byte(j)}, Delete, "",
					[]byte("content"), []byte("encryptedPayload"), nil, 0,
					time.Unix(36, 0), time.Unix(35, 0), 5 * time.Minute, 35,
					rounds.Round{}, 0, false, false}}

			err := as.addAction(sa)
			if err != nil {
				t.Fatalf("Failed to add action: %+v", err)
			}

			if !sa.Received.Before(now) {
				key := getMessageIdKey(sa.TargetMessage)
				if _, exists := expected[*channelID]; !exists {
					expected[*channelID] = map[messageIdKey]*savedAction{key: sa}
				} else {
					expected[*channelID][key] = sa
				}
			}
		}
		err := as.updateStorage(channelID, true)
		if err != nil {
			t.Errorf("Failed to save action list to storage: %+v", err)
		}
	}

	// Create new list and load old contents into it
	loadedAs := NewActionSaver(nil, kv)
	now = now.Add(maxSavedActionAge)
	err := loadedAs.load(now)
	if err != nil {
		t.Errorf("Failed to load ActionLeaseList from storage: %+v", err)
	}

	if !reflect.DeepEqual(expected, loadedAs.actions) {
		t.Errorf("Unexpected actions map after purge at %s."+
			"\nexpected: %+v\nreceived: %+v", now, expected, loadedAs.actions)
	}

	// Check that the loaded message map matches the original after it has been
	// purged
	err = as.purge(now)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loadedAs.actions, as.actions) {
		t.Errorf("Unexpected actions map after purge at %s."+
			"\nexpected: %+v\nreceived: %+v", now, loadedAs.actions, as.actions)
	}
}

// Error path: Tests that ActionSaver.load returns the expected error when no
// channel IDs can be loaded from storage.
func TestActionSaver_load_ChannelListLoadError(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	as := NewActionSaver(nil, kv)

	channelID := randChannelID(prng, t)
	targetID := randMessageID(prng, t)
	err := as.AddAction(channelID, message.ID{}, targetID, 0, nil, nil,
		time.Time{}, time.Time{}, time.Time{}, 0, 0, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	err = kv.Delete(
		savedActionsKey, savedActionsVer)
	if err != nil {
		t.Fatal(err)
	}

	// Create new list and load old contents into it
	loadedAs := NewActionSaver(nil, kv)

	expectedErr := loadSavedActionsChanIDsErr
	err = loadedAs.load(time.Unix(0, 0))
	if err == nil || kv.Exists(err) || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to get expected error when the channel ID was "+
			"deleted from straoge.\nexpected: %s\nreceived: %v",
			expectedErr, err)
	}
}

// Error path: Tests that ActionSaver.load returns the expected error when no
// messages can be loaded from storage.
func TestActionSaver_load_MessagesLoadError(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	as := NewActionSaver(nil, kv)

	channelID := randChannelID(prng, t)
	targetID := randMessageID(prng, t)
	err := as.AddAction(channelID, message.ID{}, targetID, 0, nil, nil,
		time.Time{}, time.Time{}, time.Time{}, 0, 0, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}

	err = kv.Delete(
		makeSavedActionMessagesKey(channelID), savedActionsMessagesVer)
	if err != nil {
		t.Fatal(err)
	}

	// Create new list and load old contents into it
	loadedAs := NewActionSaver(nil, kv)

	expectedErr := fmt.Sprintf(loadSavedActionsMessagesErr, channelID)
	err = loadedAs.load(time.Unix(0, 0))
	if err == nil || kv.Exists(err) || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to get expected error when the actions were deleted "+
			"from straoge.\nexpected: %s\nreceived: %v", expectedErr, err)
	}
}

// Tests that the list of channel IDs in the message map can be saved and loaded
// to and from storage with ActionSaver.storeChannelList and
// ActionSaver.loadChannelList.
func TestActionSaver_storeChannelList_loadChannelList(t *testing.T) {
	const n = 10
	prng := rand.New(rand.NewSource(32))
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))
	expectedIDs := make([]*id.ID, n)

	for i := 0; i < n; i++ {
		channelID := randChannelID(prng, t)
		as.actions[*channelID] = make(map[messageIdKey]*savedAction)
		for j := 0; j < 5; j++ {
			sa := &savedAction{
				CommandMessage: CommandMessage{
					ChannelID:   channelID,
					MessageID:   randMessageID(prng, t),
					MessageType: randAction(prng),
					Content:     randPayload(prng, t),
				},
				Received: randTimestamp(prng),
			}
			key := getMessageIdKey(randMessageID(prng, t))
			as.actions[*channelID][key] = sa
		}
		expectedIDs[i] = channelID
	}

	err := as.storeChannelList()
	if err != nil {
		t.Errorf("Failed to store channel IDs: %+v", err)
	}

	loadedIDs, err := as.loadChannelList()
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

// Error path: Tests that ActionSaver.loadChannelList returns an error when
// trying to load from storage when nothing was saved.
func TestActionSaver_loadChannelList_StorageError(t *testing.T) {
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	_, err := as.loadChannelList()
	if err == nil || as.kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that a list of messages can be stored and loaded using
// ActionSaver.storeActions and ActionSaver.loadActions.
func TestActionSaver_storeActions_loadActions(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))
	channelID := randChannelID(prng, t)
	as.actions[*channelID] = make(map[messageIdKey]*savedAction)

	for i := 0; i < 15; i++ {
		sa := &savedAction{
			CommandMessage: CommandMessage{
				ChannelID:            channelID,
				MessageID:            randMessageID(prng, t),
				MessageType:          randAction(prng),
				Nickname:             "A",
				Content:              randPayload(prng, t),
				EncryptedPayload:     randPayload(prng, t),
				PubKey:               randPayload(prng, t),
				Codeset:              uint8(i),
				Timestamp:            randTimestamp(prng),
				OriginatingTimestamp: randTimestamp(prng),
				Lease:                randLease(prng),
				OriginatingRound:     id.Round(i),
				Round:                rounds.Round{},
			},
			Received: randTimestamp(prng),
		}
		key := getMessageIdKey(randMessageID(prng, t))
		as.actions[*channelID][key] = sa
	}

	err := as.storeActions(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	loadedMessages, err := as.loadActions(channelID)
	if err != nil {
		t.Errorf("Failed to load messages: %+v", err)
	}

	if !reflect.DeepEqual(as.actions[*channelID], loadedMessages) {
		t.Errorf("Loaded messages do not match original."+
			"\nexpected: %+v\nreceived: %+v",
			as.actions[*channelID], loadedMessages)
	}
}

// Tests that ActionSaver.storeActions deletes the lease message file from
// storage when the list is empty.
func TestActionSaver_storeActions_EmptyList(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))
	channelID := randChannelID(prng, t)
	as.actions[*channelID] = make(map[messageIdKey]*savedAction)

	for i := 0; i < 15; i++ {
		sa := &savedAction{
			CommandMessage: CommandMessage{
				ChannelID:   channelID,
				MessageID:   randMessageID(prng, t),
				MessageType: randAction(prng),
				Content:     randPayload(prng, t),
			},
			Received: randTimestamp(prng),
		}
		key := getMessageIdKey(randMessageID(prng, t))
		as.actions[*channelID][key] = sa
	}

	err := as.storeActions(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	as.actions[*channelID] = make(map[messageIdKey]*savedAction)
	err = as.storeActions(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	_, err = as.loadActions(channelID)
	if err == nil || as.kv.Exists(err) {
		t.Fatalf("Failed to delete lease messages: %+v", err)
	}
}

// Error path: Tests that ActionSaver.loadActions returns an error when trying
// to load from storage when nothing was saved.
func TestActionSaver_loadActions_StorageError(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	_, err := as.loadActions(randChannelID(prng, t))
	if err == nil || as.kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that ActionSaver.deleteActions removes the messages from storage.
func TestActionSaver_deleteActions(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))
	channelID := randChannelID(prng, t)
	as.actions[*channelID] = make(map[messageIdKey]*savedAction)

	for i := 0; i < 15; i++ {
		sa := &savedAction{
			CommandMessage: CommandMessage{
				ChannelID:   channelID,
				MessageID:   randMessageID(prng, t),
				MessageType: randAction(prng),
				Content:     randPayload(prng, t),
			},
			Received: randTimestamp(prng),
		}
		key := getMessageIdKey(randMessageID(prng, t))
		as.actions[*channelID][key] = sa
	}

	err := as.storeActions(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	err = as.deleteActions(channelID)
	if err != nil {
		t.Errorf("Failed to delete messages: %+v", err)
	}

	_, err = as.loadActions(channelID)
	if err == nil || as.kv.Exists(err) {
		t.Fatalf("Failed to delete lease messages: %+v", err)
	}
}

// Consistency test of makeSavedActionMessagesKey.
func Test_makeSavedActionMessagesKey(t *testing.T) {
	prng := rand.New(rand.NewSource(11))

	expectedKeys := []string{
		"savedActionMessages/WQwUQJiItbB9UagX7gfD8hRZNbxxVePHp2SQw+CqC2oD",
		"savedActionMessages/WGLDLvh5GdCZH3r4XpU7dEKP71tXeJvJAi/UyPkxnakD",
		"savedActionMessages/mo59OR72CzZlLvnGxzfhscEY4AxjhmvE6b5W+yK1BQUD",
		"savedActionMessages/TOFI3iGP8TNZJ/V1/E4SrgW2MiS9LRxIzM0LoMnUmukD",
		"savedActionMessages/xfUsHf4FuGVcwFkKywinHo7mCdaXppXef4RU7l0vUQwD",
		"savedActionMessages/dpBGwqS9/xi7eiT+cPNRzC3BmdDg/aY3MR2IPdHBUCAD",
		"savedActionMessages/ZnT0fZYP2dCHlxxDo6DSpBplgaM3cj7RPgTZ+OF7MiED",
		"savedActionMessages/rXartsxcv2+tIPfN2x9r3wgxPqp77YK2/kSqqKzgw5ID",
		"savedActionMessages/6G0Z4gfi6u2yUp9opRTgcB0FpSv/x55HgRo6tNNi5lYD",
		"savedActionMessages/7aHvDBG6RsPXxMHvw21NIl273F0CzDN5aixeq5VRD+8D",
		"savedActionMessages/v0Pw6w7z7XAaebDUOAv6AkcMKzr+2eOIxLcDMMr/i2gD",
		"savedActionMessages/7OI/yTc2sr0m0kONaiV3uolWpyvJHXAtts4bZMm7o14D",
		"savedActionMessages/jDQqEBKqNhLpKtsIwIaW5hzUy+JdQ0JkXfkbae5iLCgD",
		"savedActionMessages/TCTUC3AblwtJiOHcvDNrmY1o+xm6VueZXhXDm3qDwT4D",
		"savedActionMessages/niQssT7H/lGZ0QoQWqLwLM24xSJeDBKKadamDlVM340D",
		"savedActionMessages/EYzeEw5VzugCW1QGXgq0jWVc5qbeoot+LH+Pt136xIED",
	}
	for i, expected := range expectedKeys {
		key := makeSavedActionMessagesKey(randChannelID(prng, t))

		if expected != key {
			t.Errorf("Key does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}
	}
}
