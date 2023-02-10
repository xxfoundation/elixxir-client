////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"math/rand"
	"os"
	"reflect"
	"sort"
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

// TODO: Finish test
func TestActionSaver_StartProcesses(t *testing.T) {
}

// TODO: Finish test
func TestActionSaver_purgeThread(t *testing.T) {
}

// TODO: Finish test
func TestActionSaver_purge(t *testing.T) {
}

// Tests ActionSaver.AddAction
// TODO: Finish test
func TestActionSaver_AddAction(t *testing.T) {

	as := NewActionSaver(nil, versioned.NewKV(ekv.MakeMemstore()))

	chanID := &id.ID{5}
	msgID, targetID := message.ID{6}, message.ID{7}
	action := Pinned
	content, encryptedPayload := []byte("content"), []byte("encryptedPayload")
	timestamp, originatingTimestamp := time.Unix(6, 0), time.Unix(5, 0)
	received := time.Unix(7, 0)
	lease := 5 * time.Minute
	err := as.AddAction(chanID, msgID, targetID, action, content, encryptedPayload,
		timestamp, originatingTimestamp, received, lease, 5, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Failed to add action: %+v", err)
	}
}

// TODO: Finish test
func TestActionSaver_CheckSavedActions(t *testing.T) {
}

// TODO: Finish test
func TestActionSaver_deleteAction(t *testing.T) {
}

// TODO: Finish test
func TestActionSaver_RemoveChannel(t *testing.T) {
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

	for i := 0; i < 10; i++ {
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
		err := as.updateStorage(channelID, true)
		if err != nil {
			t.Errorf("Failed to save action list to storage: %+v", err)
		}
	}

	// Create new list and load old contents into it
	loadedAs := NewActionSaver(nil, kv)
	err := loadedAs.load(time.Unix(0, 0))
	if err != nil {
		t.Errorf("Failed to load ActionLeaseList from storage: %+v", err)
	}

	// Check that the loaded message map matches the original
	for chanID, messages := range as.actions {
		loadedMessages, exists := as.actions[chanID]
		if !exists {
			t.Errorf("Channel ID %s does not exist in map.", chanID)
		}

		for fp, sa := range messages {
			loadedSa, exists2 := loadedMessages[fp]
			if !exists2 {
				t.Errorf("Message does not exist in map: %+v", sa)
			}

			if !reflect.DeepEqual(sa, loadedSa) {
				t.Errorf("Message does not match expected."+
					"\nexpected: %+v\nreceived: %+v", sa, loadedSa)
			}
		}
	}
}

// Tests that when ActionSaver.load loads a savedAction with a received time in
// the past, that a new one is randomly calculated between replayWaitMin and
// replayWaitMax.
func TestActionSaver_load_LeaseModify(t *testing.T) {

}

// Error path: Tests that ActionSaver.load returns the expected error when no
// channel IDs can be loaded from storage.
func TestActionSaver_load_ChannelListLoadError(t *testing.T) {

}

// Error path: Tests that ActionSaver.load returns the expected error when no
// messages can be loaded from storage.
func TestActionSaver_load_MessagesLoadError(t *testing.T) {
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
