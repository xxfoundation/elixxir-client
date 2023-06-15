////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Unit test of NewActionSaver.
func TestNewActionSaver(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &ActionSaver{
		actions: make(map[messageIdKey]time.Time),
		kv:      kv,
	}

	as := NewActionSaver(kv)

	if !reflect.DeepEqual(expected, as) {
		t.Errorf("Unexpected new ActionSaver.\nexpected: %+v\nreceived: %+v",
			expected, as)
	}
}

// Tests that ActionSaver.purge removes all stale actions.
func TestActionSaver_purge(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	as := NewActionSaver(kv)

	now := time.Unix(prng.Int63n(200), 0).Round(0).UTC()
	expected := make(map[messageIdKey]time.Time)
	for i := 0; i < 50; i++ {
		tm := randMessageID(prng, t)
		key := getMessageIdKey(tm)
		netTime.Now().UnixNano()
		var received time.Time
		switch prng.Intn(2) {
		case 0:
			received = now.Add(-(maxSavedActionAge +
				time.Duration(prng.Int63n(int64(maxSavedActionAge)))))
		case 1:
			received = now.Add(time.Duration(
				prng.Int63n(int64(maxSavedActionAge))))
			expected[key] = received
		}

		err := as.AddAction(tm, received)
		if err != nil {
			t.Fatalf("Failed to add action: %+v", err)
		}
	}
	err := as.purge(now)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, as.actions) {
		t.Errorf("Unexpected actions map after purge at %s."+
			"\nexpected: %+v\nreceived: %+v", now, expected, as.actions)
	}
}

// Unit test of ActionSaver.AddAction.
func TestActionSaver_AddAction(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	as := NewActionSaver(versioned.NewKV(ekv.MakeMemstore()))

	expected := make(map[messageIdKey]time.Time)
	for i := 0; i < 25; i++ {
		tm, received := randMessageID(prng, t), randTimestamp(prng)
		expected[getMessageIdKey(tm)] = received
		err := as.AddAction(tm, received)
		if err != nil {
			t.Errorf("Failed to add action (%d): %+v", i, err)
		}
	}

	if !reflect.DeepEqual(expected, as.actions) {
		t.Errorf("Unexpected actions map.\nexpected: %+v\nreceived: %+v",
			expected, as.actions)
	}

	// Check that all actions were saved to storage
	if loaded, err := as.loadActionList(); err != nil {
		t.Errorf("Failed to load action list from storage: %+v", err)
	} else if !reflect.DeepEqual(expected, loaded) {
		t.Errorf("Unexpected actions map.\nexpected: %+v\nreceived: %+v",
			expected, loaded)
	}
}

// Tests that ActionSaver.CheckSavedActions correctly determines a message exist
// or not and deletes it if it does.
func TestActionSaver_CheckSavedActions(t *testing.T) {
	prng := rand.New(rand.NewSource(85466))
	as := NewActionSaver(versioned.NewKV(ekv.MakeMemstore()))

	expected := make(map[message.ID]bool)
	for i := 0; i < 25; i++ {
		tm, received := randMessageID(prng, t), randTimestamp(prng)
		if prng.Intn(2) == 0 {
			expected[tm] = true
			err := as.AddAction(tm, received)
			if err != nil {
				t.Errorf("Failed to add action (%d): %+v", i, err)
			}
		} else {
			expected[tm] = false
		}
	}

	for tm, exp := range expected {
		if exp && !as.CheckSavedActions(tm) {
			t.Errorf("Target message %s should be in list.", tm)
		} else if !exp && as.CheckSavedActions(tm) {
			t.Errorf(
				"Target message %s is in the list when it should not be.", tm)
		}
	}

	time.Sleep(5 * time.Millisecond)
	as.mux.Lock()
	for tm, exp := range expected {
		if exp {
			if _, exists := as.actions[getMessageIdKey(tm)]; exists {
				t.Errorf("Message %s not deleted.", tm)
			}
		}
	}
	as.mux.Unlock()
}

////////////////////////////////////////////////////////////////////////////////
// Message ID Key                                                             //
////////////////////////////////////////////////////////////////////////////////

// Consistency test of getMessageIdKey.
func Test_getMessageIdKey(t *testing.T) {
	prng := rand.New(rand.NewSource(563567))

	expectedKeys := []messageIdKey{
		"yhGkDSXbu5/ekOUGs0OJrU+LIzCkHQVZJh+TLGyqZYw=",
		"tFI7DwxlhaEJfulja3Bg0Y6FY5IXOQqqtBbcp4Cey3g=",
		"vwS9upSLZAL/unHYVm4/gmT02u2XAC5cMMfUtttTjI0=",
		"zXhj5rTfq9dXDqVCM93RebCPBU/57rv4EpFC6/MWX2s=",
		"jy/vtD3VruoRQsW+aYpEd1ER/zVsp3rNpipqTDEHXR8=",
		"M3W3dqPM4fhxchPnXpLdKXLx9UqrTCTFStvnJbxyF50=",
		"Y0wT0JaHdjs56AD6+hdkZCiP1mMG1XH0mnt2mieQ+Ko=",
		"ZFL1YkVxUILm0LvZ2EE7Ay+JXxFx8rL8ticL2FP6Ltg=",
		"bHwyo+tGiMGCn4D+UYzUqCA/Tu/vxJ/w1EHJO2ddYxc=",
		"2hbXvk4Gqw0T1GvGtbdywrR+7bHtpnEw6MiWuaIEkFs=",
		"9e+OEY9zve1ohsjOsHWMjy/u3Tn4pzohk+8Hkca8h+c=",
		"a2/ga4Cet2jsVrYY2UQxB/OVsGyquIgLT/aHntR2ltg=",
		"hQ7a3E82RvKikQNDT0+RHITQdyG6Yt6owxG5Racj96E=",
		"vHlCrGMbel3YXLVYUS7Tf/3F5zYhDDjJK/B5XeMyA0E=",
		"6TRcXXzW/UP0KKIRfK/IdAP8eYH+JaJhVHAs8quXmeQ=",
		"kkKXjui/OldEbE0YDFMm2c+mgeEWLCubGSVs5DHromc=",
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
// the original but with all stale actions removed.
func TestActionSaver_load(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	as := NewActionSaver(kv)

	now := time.Unix(prng.Int63n(200), 0).Round(0).UTC()
	expected := make(map[messageIdKey]time.Time)
	for i := 0; i < 50; i++ {
		tm := randMessageID(prng, t)
		key := getMessageIdKey(tm)
		netTime.Now().UnixNano()
		var received time.Time
		switch prng.Intn(2) {
		case 0:
			received = now.Add(-(maxSavedActionAge +
				time.Duration(prng.Int63n(int64(maxSavedActionAge)))))
		case 1:
			received = now.Add(time.Duration(
				prng.Int63n(int64(maxSavedActionAge))))
			expected[key] = received
		}

		err := as.AddAction(tm, received)
		if err != nil {
			t.Fatalf("Failed to add action: %+v", err)
		}
	}

	// Create new list and load old contents into it
	loadedAs := NewActionSaver(kv)
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

// Tests that a list of actions can be stored and loaded using
// ActionSaver.storeActionList and ActionSaver.loadActionList.
func TestActionSaver_storeActionList_loadActionList(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	prng := rand.New(rand.NewSource(33678545))
	as := &ActionSaver{
		actions: map[messageIdKey]time.Time{
			getMessageIdKey(randMessageID(prng, t)): randTimestamp(prng),
			getMessageIdKey(randMessageID(prng, t)): randTimestamp(prng),
			getMessageIdKey(randMessageID(prng, t)): randTimestamp(prng),
			getMessageIdKey(randMessageID(prng, t)): randTimestamp(prng),
			getMessageIdKey(randMessageID(prng, t)): randTimestamp(prng),
			getMessageIdKey(randMessageID(prng, t)): randTimestamp(prng),
		},
		kv: kv,
	}

	err := as.storeActionList()
	if err != nil {
		t.Errorf("Failed to store action saver list: %+v", err)
	}

	as2 := &ActionSaver{
		actions: make(map[messageIdKey]time.Time),
		kv:      kv,
	}

	loaded, err := as2.loadActionList()
	if err != nil {
		t.Errorf("Failed to load action saver list: %+v", err)
	}

	if !reflect.DeepEqual(as.actions, loaded) {
		t.Errorf("Unxpected loaded action list.\nexpected: %+v\nreceived: %+v",
			as.actions, loaded)
	}
}

// randMessageID creates a new random channel.MessageID for testing.
func randMessageID(prng *rand.Rand, t testing.TB) message.ID {
	receptionID, err := id.NewRandomID(prng, id.User)
	if err != nil {
		t.Fatalf("Failed to generate random ID: %+v", err)
	}

	types := []MessageType{
		TextType, ReplyType, ReactionType, SilentType, DeleteType}

	dm := &DirectMessage{
		RoundID:        prng.Uint64(),
		PayloadType:    uint32(types[prng.Intn(len(types))]),
		Payload:        make([]byte, prng.Intn(64)),
		Nickname:       string(make([]byte, prng.Intn(12))),
		Nonce:          make([]byte, prng.Intn(16)),
		LocalTimestamp: prng.Int63(),
	}

	prng.Read(dm.Payload)
	prng.Read([]byte(dm.Nickname))

	return message.DeriveDirectMessageID(receptionID, dm)
}

// randTimestamp creates a new random action lease end for testing.
func randTimestamp(prng *rand.Rand) time.Time {
	return netTime.Now().Add(time.Duration(prng.Int63n(int64(1000 * time.Hour)))).UTC().Round(0)
}
