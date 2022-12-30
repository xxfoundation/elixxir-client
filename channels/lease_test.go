////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Tests that NewOrLoadActionLeaseList initialises a new empty ActionLeaseList
// when called for the first time and that it loads the ActionLeaseList from
// storage after the original has been saved.
func TestNewOrLoadActionLeaseList(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	s := NewCommandStore(kv)
	expected := NewActionLeaseList(nil, s, kv, rng)

	all, err := NewOrLoadActionLeaseList(nil, s, kv, rng)
	if err != nil {
		t.Fatalf("Failed to create new ActionLeaseList: %+v", err)
	}

	all.addLeaseMessage = expected.addLeaseMessage
	all.removeLeaseMessage = expected.removeLeaseMessage
	all.removeChannelCh = expected.removeChannelCh
	expected.rb.replay = nil
	all.rb.replay = nil
	if !reflect.DeepEqual(expected.rb, all.rb) {
		t.Errorf("New ActionLeaseList does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected.rb, all.rb)
	}
	if !reflect.DeepEqual(expected, all) {
		t.Errorf("New ActionLeaseList does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, all)
	}

	timestamp := randTimestamp(prng)
	lmp := &leaseMessagePacket{
		leaseMessage: &leaseMessage{
			ChannelID:            randChannelID(prng, t),
			Action:               randAction(prng),
			Payload:              randPayload(prng, t),
			OriginatingTimestamp: timestamp,
			Lease:                time.Hour,
			LeaseEnd:             timestamp.Add(time.Hour),
			LeaseTrigger:         timestamp.Add(time.Hour),
			e:                    nil,
		},
	}
	lmp.cm = &CommandMessage{
		ChannelID:            lmp.ChannelID,
		MessageID:            randMessageID(prng, t),
		MessageType:          lmp.Action,
		Content:              lmp.Payload,
		EncryptedPayload:     randPayload(prng, t),
		Timestamp:            randTimestamp(prng),
		OriginatingTimestamp: lmp.OriginatingTimestamp,
		Lease:                lmp.Lease,
		Round:                rounds.Round{ID: 5},
		FromAdmin:            true,
	}
	err = all.addMessage(lmp)
	if err != nil {
		t.Errorf("Failed to add message: %+v", err)
	}
	for _, l := range all.messagesByChannel[*lmp.ChannelID] {
		lmp.LeaseEnd = l.LeaseEnd
		lmp.LeaseTrigger = l.LeaseTrigger
	}

	loadedAll, err := NewOrLoadActionLeaseList(nil, s, kv, rng)
	if err != nil {
		t.Errorf("Failed to load ActionLeaseList: %+v", err)
	}

	all.addLeaseMessage = loadedAll.addLeaseMessage
	all.removeLeaseMessage = loadedAll.removeLeaseMessage
	all.removeChannelCh = loadedAll.removeChannelCh
	all.rb.replay = nil
	loadedAll.rb.replay = nil
	if !reflect.DeepEqual(all, loadedAll) {
		t.Errorf("Loaded ActionLeaseList does not match expected."+
			"\nexpected: %+v\nreceived: %+v\nexpected: %+v\nreceived: %+v",
			all, loadedAll, all.messagesByChannel, loadedAll.messagesByChannel)
	}
}

// Tests that NewActionLeaseList returns the expected new ActionLeaseList.
func TestNewActionLeaseList(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewCommandStore(kv)
	expected := &ActionLeaseList{
		leases:             list.New(),
		messagesByChannel:  make(map[id.ID]map[commandFingerprintKey]*leaseMessage),
		addLeaseMessage:    make(chan *leaseMessagePacket, addLeaseMessageChanSize),
		removeLeaseMessage: make(chan *leaseMessage, removeLeaseMessageChanSize),
		removeChannelCh:    make(chan *id.ID, removeChannelChChanSize),
		store:              s,
		kv:                 kv,
		rng:                rng,
	}
	expected.rb = newReplayBlocker(expected.AddOrOverwrite, s, kv)

	all := NewActionLeaseList(nil, s, kv, rng)
	all.addLeaseMessage = expected.addLeaseMessage
	all.removeLeaseMessage = expected.removeLeaseMessage
	all.removeChannelCh = expected.removeChannelCh
	all.rb.replay = nil
	expected.rb.replay = nil

	if !reflect.DeepEqual(expected, all) {
		t.Errorf("New ActionLeaseList does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, all)
	}
}

// Tests that ActionLeaseList.StartProcesses returns an error until
// ActionLeaseList.RegisterReplayFn has been called.
func TestActionLeaseList_StartProcesses_RegisterReplayFn(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv, rng)

	_, err := all.StartProcesses()
	if err == nil || err.Error() != noReplayFuncErr {
		t.Errorf("StartProcesses did not return the expected error when the"+
			"replay function was not set.\nexpected: %s\nreceived: %+v",
			noReplayFuncErr, err)
	}

	all.RegisterReplayFn(func(*id.ID, []byte) {})

	_, err = all.StartProcesses()
	if err != nil {
		t.Errorf("StartProcesses failed: %+v", err)
	}
}

// Tests that ActionLeaseList.updateLeasesThread removes the expected number of
// lease messages when they expire.
func TestActionLeaseList_updateLeasesThread(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	triggerChan := make(chan *leaseMessage, 3)
	trigger := func(channelID *id.ID, _ message.ID, messageType MessageType,
		nickname string, payload, _ []byte, timestamp,
		originatingTimestamp time.Time, lease time.Duration, _ id.Round,
		_ rounds.Round, _ SentStatus, _ bool) (uint64, error) {
		triggerChan <- &leaseMessage{
			ChannelID:            channelID,
			Action:               messageType,
			Payload:              payload,
			OriginatingTimestamp: originatingTimestamp,
		}
		return 0, nil
	}
	replay := func(channelID *id.ID, encryptedPayload []byte) {
		triggerChan <- &leaseMessage{
			ChannelID: channelID,
			Payload:   encryptedPayload,
		}
	}
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(trigger, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	all.RegisterReplayFn(replay)

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	timestamp := netTime.Now().UTC().Round(0)
	expectedMessages := map[time.Duration]*leaseMessagePacket{
		50 * time.Millisecond: {
			leaseMessage: &leaseMessage{
				ChannelID:            randChannelID(prng, t),
				Action:               randAction(prng),
				Payload:              randPayload(prng, t),
				OriginatingTimestamp: timestamp,
			},
			cm: &CommandMessage{},
		},
		200 * time.Millisecond: {
			leaseMessage: &leaseMessage{
				ChannelID:            randChannelID(prng, t),
				Action:               randAction(prng),
				Payload:              randPayload(prng, t),
				OriginatingTimestamp: timestamp,
			},
			cm: &CommandMessage{},
		},
		400 * time.Millisecond: {
			leaseMessage: &leaseMessage{
				ChannelID:            randChannelID(prng, t),
				Action:               randAction(prng),
				Payload:              randPayload(prng, t),
				OriginatingTimestamp: timestamp,
			},
			cm: &CommandMessage{},
		},
		600 * time.Hour: { // This tests the replay code
			leaseMessage: &leaseMessage{
				ChannelID:            randChannelID(prng, t),
				Action:               randAction(prng),
				OriginatingTimestamp: timestamp.Add(-time.Hour),
			},
			cm: &CommandMessage{
				EncryptedPayload: randPayload(prng, t),
			},
		},
	}

	for lease, e := range expectedMessages {
		err := all.AddMessage(e.ChannelID, e.cm.MessageID, e.Action,
			randPayload(prng, t), e.Payload, e.cm.EncryptedPayload,
			e.cm.Timestamp, e.OriginatingTimestamp, lease,
			e.cm.OriginatingRound, e.cm.Round, e.cm.FromAdmin)
		if err != nil {
			t.Fatalf("Failed to add message for lease %s: %+v", lease, err)
		}
	}

	fp := newCommandFingerprint(expectedMessages[600*time.Hour].ChannelID,
		expectedMessages[600*time.Hour].Action,
		expectedMessages[600*time.Hour].Payload)

	// Modify lease trigger of 600*time.Hour so the test doesn't take hours
	for {
		messages, exists :=
			all.messagesByChannel[*expectedMessages[600*time.Hour].ChannelID]
		if exists {
			if _, exists = messages[fp.key()]; exists {
				all.messagesByChannel[*expectedMessages[600*time.Hour].
					ChannelID][fp.key()].LeaseTrigger =
					netTime.Now().Add(600 * time.Millisecond)
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[50*time.Millisecond]
		if !reflect.DeepEqual(expected.leaseMessage, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected.leaseMessage, lm)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be triggered.")
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[200*time.Millisecond]
		if !reflect.DeepEqual(expected.leaseMessage, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected.leaseMessage, lm)
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be triggered.")
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[400*time.Millisecond]
		if !reflect.DeepEqual(expected.leaseMessage, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected.leaseMessage, lm)
		}
	case <-time.After(400 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be triggered.")
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[600*time.Hour]
		if !expected.ChannelID.Cmp(lm.ChannelID) {
			t.Errorf("Did not receive expected channel ID."+
				"\nexpected: %+v\nreceived: %+v",
				expected.ChannelID, lm.ChannelID)
		}
		if !bytes.Equal(expected.cm.EncryptedPayload, lm.Payload) {
			t.Errorf("Did not receive expected EncryptedPayload."+
				"\nexpected: %v\nreceived: %v",
				expected.cm.EncryptedPayload, lm.Payload)
		}
	case <-time.After(800 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be triggered.")
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to close thread: %+v", err)
	}
}

// Tests that ActionLeaseList.updateLeasesThread stops the stoppable when
// triggered and returns.
func TestActionLeaseList_updateLeasesThread_Stoppable(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	stop := stoppable.NewSingle(leaseThreadStoppable)
	stopped := make(chan struct{})
	go func() {
		all.updateLeasesThread(stop)
		stopped <- struct{}{}
	}()

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to close thread: %+v", err)
	}

	select {
	case <-stopped:
		if !stop.IsStopped() {
			t.Errorf("Stoppable not stopped.")
		}
	case <-time.After(5 * time.Millisecond):
		t.Errorf("Timed out waitinf for updateLeasesThread to return")
	}
}

// Tests that ActionLeaseList.updateLeasesThread adds and removes a lease
// channel.
func TestActionLeaseList_updateLeasesThread_AddAndRemove(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	lease := 200 * time.Hour
	randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
	timestamp := netTime.Now().UTC().Add(-randDuration).Round(0)
	exp := &leaseMessagePacket{
		leaseMessage: &leaseMessage{
			ChannelID:            randChannelID(prng, t),
			Action:               randAction(prng),
			Payload:              randPayload(prng, t),
			OriginatingTimestamp: timestamp,
			Lease:                lease,
			LeaseEnd:             timestamp.Add(lease),
			LeaseTrigger:         timestamp.Add(lease),
		},
		cm: &CommandMessage{
			MessageID:        randMessageID(prng, t),
			EncryptedPayload: randPayload(prng, t),
			FromAdmin:        false,
		},
	}
	fp := newCommandFingerprint(
		exp.ChannelID, exp.Action, exp.Payload)

	err := all.AddMessage(exp.ChannelID, exp.cm.MessageID, exp.Action,
		randPayload(prng, t), exp.Payload, exp.cm.EncryptedPayload, timestamp,
		timestamp, lease, exp.cm.OriginatingRound, exp.cm.Round,
		exp.cm.FromAdmin)
	if err != nil {
		t.Fatalf("Failed to add message: %+v", err)
	}

	done := make(chan struct{})
	go func() {
		for all.leases.Len() < 1 {
			time.Sleep(time.Millisecond)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Millisecond):
		t.Fatalf("Timed out waiting for message to be added to message map.")
	}

	lm := all.leases.Front().Value.(*leaseMessage)
	exp.e = lm.e
	exp.LeaseTrigger = lm.LeaseTrigger
	exp.OriginatingTimestamp = lm.OriginatingTimestamp
	if !reflect.DeepEqual(exp.leaseMessage, lm) {
		t.Errorf("Unexpected lease message added to lease list."+
			"\nexpected: %+v\nreceived: %+v", exp.leaseMessage, lm)
	}

	if messages, exists := all.messagesByChannel[*exp.ChannelID]; !exists {
		t.Errorf("Channel %s not found in message map.", exp.ChannelID)
	} else if lm, exists = messages[fp.key()]; !exists {
		t.Errorf("Message with fingerprint %s not found in message map.", fp)
	} else if !reflect.DeepEqual(exp.leaseMessage, lm) {
		t.Errorf("Unexpected lease message added to message map."+
			"\nexpected: %+v\nreceived: %+v", exp.leaseMessage, lm)
	}

	err = all.RemoveMessage(exp.ChannelID, exp.cm.MessageID, exp.Action,
		randPayload(prng, t), exp.Payload, exp.cm.EncryptedPayload,
		exp.cm.Timestamp, exp.OriginatingTimestamp, exp.Lease,
		exp.cm.OriginatingRound+1, exp.cm.Round, exp.cm.FromAdmin)
	if err != nil {
		t.Fatalf("Failed to remove message: %+v", err)
	}

	done = make(chan struct{})
	go func() {
		for len(all.messagesByChannel) != 0 {
			time.Sleep(time.Millisecond)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Millisecond):
		t.Fatalf("Timed out waiting for message to be removed from message map.")
	}

	if all.leases.Len() != 0 {
		t.Errorf("%d messages left in lease list.", all.leases.Len())
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to close thread: %+v", err)
	}
}

// Tests that ActionLeaseList.AddMessage sends the expected leaseMessage on the
// addLeaseMessage channel.
func TestActionLeaseList_AddMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	timestamp := randTimestamp(prng)
	lease := randLease(prng)
	exp := &leaseMessagePacket{
		leaseMessage: &leaseMessage{
			ChannelID:            randChannelID(prng, t),
			Action:               randAction(prng),
			Payload:              randPayload(prng, t),
			OriginatingTimestamp: timestamp,
			Lease:                lease,
			LeaseEnd:             timestamp.Add(lease),
			e:                    nil,
		},
	}
	exp.cm = &CommandMessage{
		ChannelID:            exp.ChannelID,
		MessageID:            randMessageID(prng, t),
		MessageType:          exp.Action,
		Content:              exp.Payload,
		EncryptedPayload:     randPayload(prng, t),
		Timestamp:            timestamp,
		OriginatingTimestamp: timestamp,
		Lease:                lease,
	}

	err := all.AddMessage(exp.ChannelID, exp.cm.MessageID, exp.Action,
		randPayload(prng, t), exp.Payload, exp.cm.EncryptedPayload,
		exp.cm.Timestamp, exp.OriginatingTimestamp, exp.Lease,
		exp.cm.OriginatingRound, exp.cm.Round, exp.cm.FromAdmin)
	if err != nil {
		t.Fatalf("Failed to add message: %+v", err)
	}

	select {
	case lm := <-all.addLeaseMessage:
		exp.LeaseTrigger = lm.LeaseTrigger
		if !reflect.DeepEqual(exp, lm) {
			t.Errorf("leaseMessage does not match expected."+
				"\nexpected: %+v\nreceived: %+v", exp, lm)
		}
	case <-time.After(5 * time.Millisecond):
		t.Error("Timed out waiting on addLeaseMessage.")
	}
}

func TestActionLeaseList_AddOrOverwrite(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	timestamp := randTimestamp(prng)
	lease := randLease(prng)
	exp := &leaseMessagePacket{
		leaseMessage: &leaseMessage{
			ChannelID:            randChannelID(prng, t),
			Action:               randAction(prng),
			Payload:              randPayload(prng, t),
			OriginatingTimestamp: timestamp,
			Lease:                lease,
			LeaseEnd:             timestamp.Add(lease),
			e:                    nil,
		},
	}
	exp.cm = &CommandMessage{
		ChannelID:            exp.ChannelID,
		MessageID:            randMessageID(prng, t),
		MessageType:          exp.Action,
		Content:              exp.Payload,
		EncryptedPayload:     randPayload(prng, t),
		Timestamp:            timestamp,
		OriginatingTimestamp: timestamp,
		Lease:                lease,
	}

	err := all.store.SaveCommand(exp.ChannelID, exp.cm.MessageID, exp.Action,
		exp.cm.Nickname, exp.cm.Content, exp.cm.EncryptedPayload, exp.cm.PubKey,
		exp.cm.Codeset, exp.cm.Timestamp, exp.cm.OriginatingTimestamp,
		exp.cm.Lease, exp.cm.OriginatingRound, exp.cm.Round, exp.cm.Status,
		exp.cm.FromAdmin, exp.cm.UserMuted)
	if err != nil {
		t.Fatalf("Failed to store command message: %+v", err)
	}

	err = all.AddOrOverwrite(exp.ChannelID, exp.Action, exp.Payload)
	if err != nil {
		t.Fatalf("Failed to AddOrOverwrite: %+v", err)
	}

	select {
	case lm := <-all.addLeaseMessage:
		floor := netTime.Now().Add(quickReplayFloor - time.Nanosecond)
		ceiling := netTime.Now().Add(quickReplayCeiling)
		if lm.LeaseTrigger.Before(floor) {
			t.Errorf("Lease trigger smaller than floor."+
				"\nLeaseTrigger: %s\nfloor:        %s", lm.leaseMessage, floor)
		} else if lm.LeaseTrigger.After(ceiling) {
			t.Errorf("Lease trigger greater than ceiling."+
				"\nLeaseTrigger: %s\nceiling:      %s", lm.leaseMessage, ceiling)
		}

		exp.LeaseTrigger = lm.LeaseTrigger
		if !reflect.DeepEqual(exp, lm) {
			t.Errorf("leaseMessage does not match expected."+
				"\nexpected: %+v\nreceived: %+v", exp, lm)
		}
	case <-time.After(5 * time.Millisecond):
		t.Error("Timed out waiting on addLeaseMessage.")
	}
}

// Tests that ActionLeaseList.addMessage adds all the messages to both the
// lease list and the message map and that the lease list is in the correct
// order.
func TestActionLeaseList_addMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessagePacket, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := randChannelID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := randPayload(prng, t)
			encrypted := randPayload(prng, t)

			for k := 0; k < o; k++ {
				lease := 200 * time.Hour
				randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
				timestamp := netTime.Now().UTC().Add(-randDuration).Round(0)
				lmp := &leaseMessagePacket{
					leaseMessage: &leaseMessage{
						ChannelID:            channelID,
						Action:               MessageType(k),
						Payload:              payload,
						OriginatingTimestamp: timestamp,
						Lease:                lease,
					},
					cm: &CommandMessage{
						ChannelID:        channelID,
						MessageID:        randMessageID(prng, t),
						EncryptedPayload: encrypted,
					},
				}

				expected = append(expected, lmp)

				err := all.addMessage(lmp)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}
			}
		}
	}

	// Check that the message map has all the expected messages
	for i, exp := range expected {
		fp := newCommandFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messagesByChannel[*exp.ChannelID]; !exists {
			t.Errorf("Channel %s does not exist (%d).", exp.ChannelID, i)
		} else if lm, exists2 := messages[fp.key()]; !exists2 {
			t.Errorf("No lease message found with key %s (%d).", fp.key(), i)
		} else {
			lm.e = nil
			if !reflect.DeepEqual(exp.leaseMessage, lm) {
				t.Errorf("leaseMessage does not match expected (%d)."+
					"\nexpected: %+v\nreceived: %+v", i, exp.leaseMessage, lm)
			}
		}
	}

	// Check that the lease list has all the expected messages in the correct
	// order
	sort.SliceStable(expected, func(i, j int) bool {
		return expected[i].LeaseTrigger.Before(expected[j].LeaseTrigger)
	})
	for i, e := 0, all.leases.Front(); e != nil; i, e = i+1, e.Next() {
		if expected[i].LeaseTrigger != e.Value.(*leaseMessage).LeaseTrigger {
			t.Errorf("leaseMessage %d not in correct order."+
				"\nexpected: %+v\nreceived: %+v",
				i, expected[i], e.Value.(*leaseMessage))
		}
	}
}

// Tests that after updating half the messages, ActionLeaseList.addMessage moves
// the messages to the lease list is still in order.
func TestActionLeaseList_addMessage_Update(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessagePacket, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := randChannelID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := randPayload(prng, t)
			encrypted := randPayload(prng, t)

			for k := 0; k < o; k++ {
				timestamp := randTimestamp(prng)
				lease := randLease(prng)
				lmp := &leaseMessagePacket{
					leaseMessage: &leaseMessage{
						ChannelID:            channelID,
						Action:               MessageType(k),
						Payload:              payload,
						OriginatingTimestamp: timestamp,
						Lease:                lease,
					},
					cm: &CommandMessage{
						ChannelID:        channelID,
						EncryptedPayload: encrypted,
					},
				}
				expected = append(expected, lmp)

				err := all.addMessage(lmp)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}
			}
		}
	}

	// Update the time of half the messages.
	for i, lm := range expected {
		if i%2 == 0 {
			timestamp := randTimestamp(prng)
			lease := time.Minute
			lm.LeaseTrigger = timestamp.Add(lease)

			err := all.addMessage(lm)
			if err != nil {
				t.Errorf("Failed to add message: %+v", err)
			}
		}
	}

	// Check that the order is still correct
	sort.SliceStable(expected, func(i, j int) bool {
		return expected[i].LeaseTrigger.Before(expected[j].LeaseTrigger)
	})
	for i, e := 0, all.leases.Front(); e != nil; i, e = i+1, e.Next() {
		if expected[i].LeaseTrigger != e.Value.(*leaseMessage).LeaseTrigger {
			t.Errorf("leaseMessage %d not in correct order."+
				"\nexpected: %+v\nreceived: %+v",
				i, expected[i], e.Value.(*leaseMessage))
		}
	}
}

// Tests that ActionLeaseList.insertLease inserts all the leaseMessage in the
// correct order, from smallest LeaseTrigger to largest.
func TestActionLeaseList_insertLease(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	expected := make([]time.Time, 50)

	for i := range expected {
		randomTime := time.Unix(0, prng.Int63())
		all.insertLease(&leaseMessage{LeaseTrigger: randomTime})
		expected[i] = randomTime
	}

	sort.SliceStable(expected, func(i, j int) bool {
		return expected[i].Before(expected[j])
	})

	for i, e := 0, all.leases.Front(); e != nil; i, e = i+1, e.Next() {
		if expected[i] != e.Value.(*leaseMessage).LeaseTrigger {
			t.Errorf("Timestamp %d not in correct order."+
				"\nexpected: %s\nreceived: %s",
				i, expected[i], e.Value.(*leaseMessage).LeaseTrigger)
		}
	}
}

// Fills the lease list with in-order messages and tests that
// ActionLeaseList.updateLease correctly moves elements to the correct order
// when their LeaseTrigger changes.
func TestActionLeaseList_updateLease(t *testing.T) {
	prng := rand.New(rand.NewSource(32_142))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	for i := 0; i < 50; i++ {
		randomTime := time.Unix(0, prng.Int63())
		all.insertLease(&leaseMessage{LeaseTrigger: randomTime})
	}

	tests := []struct {
		randomTime time.Time
		e          *list.Element
	}{
		// Change the first element to a random time
		{time.Unix(0, prng.Int63()), all.leases.Front()},

		// Change an element to a random time
		{time.Unix(0, prng.Int63()), all.leases.Front().Next().Next().Next()},

		// Change the last element to a random time
		{time.Unix(0, prng.Int63()), all.leases.Back()},

		// Change an element to the first element
		{all.leases.Front().Value.(*leaseMessage).LeaseTrigger.Add(-1),
			all.leases.Front().Next().Next()},

		// Change an element to the last element
		{all.leases.Back().Value.(*leaseMessage).LeaseTrigger.Add(1),
			all.leases.Front().Next().Next().Next().Next().Next()},
	}

	for i, tt := range tests {
		tt.e.Value.(*leaseMessage).LeaseTrigger = tt.randomTime
		all.updateLease(tt.e)

		// Check that the list is in order
		for j, n := 0, all.leases.Front(); n.Next() != nil; j, n = j+1, n.Next() {
			lt1 := n.Value.(*leaseMessage).LeaseTrigger
			lt2 := n.Next().Value.(*leaseMessage).LeaseTrigger
			if lt1.After(lt2) {
				t.Errorf("Element #%d is greater than element #%d (%d)."+
					"\nelement #%d: %s\nelement #%d: %s",
					j, j+1, i, j, lt1, j+1, lt2)
			}
		}
	}
}

// Tests that ActionLeaseList.RemoveMessage sends the expected leaseMessage on
// the removeLeaseMessage channel.
func TestActionLeaseList_RemoveMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	exp := &leaseMessage{
		ChannelID: randChannelID(prng, t),
		Action:    randAction(prng),
		Payload:   randPayload(prng, t),
	}

	err := all.RemoveMessage(exp.ChannelID, cryptoChannel.MessageID{},
	exp.Action, randPayload(prng, t), exp.Payload, []byte{}, netTime.Now(),
	netTime.Now(), 200*time.Hour, 5, rounds.Round{}, false)
	if err != nil {
		t.Fatalf("Failed to remove message: %+v", err)
	}

	select {
	case lm := <-all.removeLeaseMessage:
		if !reflect.DeepEqual(exp, lm) {
			t.Errorf("leaseMessage does not match expected."+
				"\nexpected: %+v\nreceived: %+v", exp, lm)
		}
	case <-time.After(5 * time.Millisecond):
		t.Error("Timed out waiting on removeLeaseMessage.")
	}
}

// Tests that ActionLeaseList.removeMessage removes all the messages from both
// the lease list and the message map and that the lease list remains in the
// correct order after every removal.
func TestActionLeaseList_removeMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := randChannelID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := randPayload(prng, t)
			encrypted := randPayload(prng, t)

			for k := 0; k < o; k++ {
				lease := 200 * time.Hour
				randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
				timestamp := netTime.Now().UTC().Add(-randDuration).Round(0)
				lmp := &leaseMessagePacket{
					leaseMessage: &leaseMessage{
						ChannelID:            channelID,
						Action:               MessageType(k),
						Payload:              payload,
						OriginatingTimestamp: timestamp,
						Lease:                lease,
					},
					cm: &CommandMessage{
						ChannelID:        channelID,
						EncryptedPayload: encrypted,
					},
				}
				err := all.addMessage(lmp)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}

				fp := newCommandFingerprint(channelID, lmp.Action, payload)
				expected = append(
					expected, all.messagesByChannel[*channelID][fp.key()])
			}
		}
	}

	// Check that the message map has all the expected messages
	for i, exp := range expected {
		fp := newCommandFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messagesByChannel[*exp.ChannelID]; !exists {
			t.Errorf("Channel %s does not exist (%d).", exp.ChannelID, i)
		} else if lm, exists2 := messages[fp.key()]; !exists2 {
			t.Errorf("No lease message found with key %s (%d).", fp.key(), i)
		} else {
			if !reflect.DeepEqual(exp, lm) {
				t.Errorf("leaseMessage does not match expected (%d)."+
					"\nexpected: %+v\nreceived: %+v", i, exp, lm)
			}
		}
	}

	for i, exp := range expected {
		err := all.removeMessage(exp, true)
		if err != nil {
			t.Errorf("Failed to remove message %d: %+v", i, exp)
		}

		// Check that the message was removed from the map
		fp := newCommandFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messagesByChannel[*exp.ChannelID]; exists {
			if _, exists = messages[fp.key()]; exists {
				t.Errorf(
					"Removed leaseMessage found with key %s (%d).", fp.key(), i)
			}
		}

		// Check that the lease list is in order
		for e := all.leases.Front(); e != nil && e.Next() != nil; e = e.Next() {
			// Check that the message does not exist in the list
			if reflect.DeepEqual(exp, e.Value) {
				t.Errorf(
					"Removed leaseMessage found in list (%d): %+v", i, e.Value)
			}
			if e.Value.(*leaseMessage).LeaseTrigger.After(
				e.Next().Value.(*leaseMessage).LeaseTrigger) {
				t.Errorf("Lease list not in order.")
			}
		}
	}
}

// Tests that ActionLeaseList.removeMessage does nothing and returns nil when
// removing a message that does not exist.
func TestActionLeaseList_removeMessage_NonExistentMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := randChannelID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := randPayload(prng, t)
			encrypted := randPayload(prng, t)

			for k := 0; k < o; k++ {
				lease := 200 * time.Hour
				randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
				timestamp := netTime.Now().UTC().Add(-randDuration).Round(0)
				lmp := &leaseMessagePacket{
					leaseMessage: &leaseMessage{
						ChannelID:            channelID,
						Action:               MessageType(k),
						Payload:              payload,
						OriginatingTimestamp: timestamp,
						Lease:                lease,
					},
					cm: &CommandMessage{
						ChannelID:        channelID,
						EncryptedPayload: encrypted,
					},
				}
				fp := newCommandFingerprint(channelID, lmp.Action, payload)
				err := all.addMessage(lmp)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}

				expected = append(
					expected, all.messagesByChannel[*channelID][fp.key()])
			}
		}
	}

	err := all.removeMessage(&leaseMessage{
		ChannelID:    randChannelID(prng, t),
		Action:       randAction(prng),
		Payload:      randPayload(prng, t),
		LeaseEnd:     randTimestamp(prng),
		LeaseTrigger: randTimestamp(prng),
	}, true)
	if err != nil {
		t.Errorf("Error removing message that does not exist: %+v", err)
	}

	if all.leases.Len() != len(expected) {
		t.Errorf("Unexpected length of lease list.\nexpected: %d\nreceived: %d",
			len(expected), all.leases.Len())
	}

	if len(all.messagesByChannel) != m {
		t.Errorf("Unexpected length of message channels."+
			"\nexpected: %d\nreceived: %d", m, len(all.messagesByChannel))
	}

	for chID, messages := range all.messagesByChannel {
		if len(messages) != n*o {
			t.Errorf("Unexpected length of messages for channel %s."+
				"\nexpected: %d\nreceived: %d", chID, n*o, len(messages))
		}
	}
}

// Test that ActionLeaseList.updateLeaseTrigger updates the LeaseTrigger and
// that the list is in order.
func TestActionLeaseList_updateLeaseTrigger(t *testing.T) {
	prng := rand.New(rand.NewSource(8_175_178))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	const numMessages = 50
	messages := make([]*leaseMessagePacket, numMessages)
	now := netTime.Now().UTC().Round(0)
	for i := 0; i < numMessages; i++ {
		lease := 600 * time.Hour
		timestamp := now.Add(-5 * time.Hour)
		lmp := &leaseMessagePacket{
			leaseMessage: &leaseMessage{
				ChannelID:            randChannelID(prng, t),
				Action:               randAction(prng),
				Payload:              randPayload(prng, t),
				OriginatingTimestamp: timestamp,
				Lease:                lease,
			},
			cm: &CommandMessage{
				ChannelID: randChannelID(prng, t),
			},
		}
		err := all.addMessage(lmp)
		if err != nil {
			t.Errorf("Failed to add lease message (%d): %+v", i, err)
		}
		messages[i] = lmp
	}

	for i, lmp := range messages {
		oldLeaseTrigger := lmp.LeaseTrigger
		err := all.updateLeaseTrigger(lmp.leaseMessage, now)
		if err != nil {
			t.Errorf("Failed to update lease trigger (%d): %+v", i, err)
		}

		if oldLeaseTrigger == lmp.LeaseTrigger {
			t.Errorf("Failed to update lease trigger %d/%d. New should be "+
				"different from old.\nold: %s\nnew: %s", i+1, numMessages,
				oldLeaseTrigger, lmp.LeaseTrigger)
		}

		// Check that the list is in order
		for j, n := 0, all.leases.Front(); n.Next() != nil; j, n = j+1, n.Next() {
			lt1 := n.Value.(*leaseMessage).LeaseTrigger
			lt2 := n.Next().Value.(*leaseMessage).LeaseTrigger
			if lt1.After(lt2) {
				t.Errorf("Element #%d is greater than element #%d (%d)."+
					"\nelement #%d: %s\nelement #%d: %s",
					j, j+1, i, j, lt1, j+1, lt2)
			}
		}
	}
}

// Tests that ActionLeaseList.RemoveChannel removes all leases for the channel
// from the list.
func TestActionLeaseList_RemoveChannel(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	var channelID *id.ID
	for i := 0; i < 5; i++ {
		channelID = randChannelID(prng, t)
		for j := 0; j < 5; j++ {
			lease := 200 * time.Hour
			timestamp := netTime.Now().UTC().Round(0).Add(-5 * time.Hour)
			exp := &leaseMessagePacket{
				leaseMessage: &leaseMessage{
					ChannelID:            channelID,
					Action:               randAction(prng),
					Payload:              randPayload(prng, t),
					OriginatingTimestamp: timestamp,
					Lease:                lease,
					LeaseEnd:             timestamp.Add(lease),
					LeaseTrigger:         timestamp.Add(lease),
				},
				cm: &CommandMessage{
					ChannelID:        nil,
					MessageID:        randMessageID(prng, t),
					EncryptedPayload: randPayload(prng, t),
				},
			}

			err := all.AddMessage(exp.ChannelID, exp.cm.MessageID, exp.Action,
				randPayload(prng, t), exp.Payload, exp.cm.EncryptedPayload,
				exp.cm.Timestamp, exp.OriginatingTimestamp, exp.Lease,
				exp.cm.OriginatingRound, exp.cm.Round, exp.cm.FromAdmin)
			if err != nil {
				t.Fatalf("Failed to add message: %+v", err)
			}
		}
	}

	done := make(chan struct{})
	go func() {
		for len(all.messagesByChannel) < 5 {
			time.Sleep(time.Millisecond)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(80 * time.Millisecond):
		t.Errorf("Timed out waiting for messages to be added to message map.")
	}

	all.RemoveChannel(channelID)

	done = make(chan struct{})
	go func() {
		for len(all.messagesByChannel) > 4 {
			time.Sleep(time.Millisecond)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Error("Timed out waiting for message to be removed from message map.")
	}

	if all.leases.Len() != 20 {
		t.Errorf("%d messages left in lease list when %d expected.",
			all.leases.Len(), 20)
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to close thread: %+v", err)
	}
}

// Tests that ActionLeaseList.removeChannel removes all the messages from both
// the lease list and the message map for the given channel and that the lease
// list remains in the correct order after removal.
func TestActionLeaseList_removeChannel(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := randChannelID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := randPayload(prng, t)
			encrypted := randPayload(prng, t)

			for k := 0; k < o; k++ {
				lease := 200 * time.Hour
				randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
				timestamp := netTime.Now().UTC().Add(-randDuration).Round(0)
				lmp := &leaseMessagePacket{
					leaseMessage: &leaseMessage{
						ChannelID:            channelID,
						Action:               MessageType(k),
						Payload:              payload,
						OriginatingTimestamp: timestamp,
						Lease:                lease,
						LeaseEnd:             timestamp.Add(lease),
						LeaseTrigger:         timestamp.Add(lease),
					},
					cm: &CommandMessage{
						ChannelID:        channelID,
						EncryptedPayload: encrypted,
					},
				}
				fp := newCommandFingerprint(channelID, lmp.Action, payload)
				err := all.addMessage(lmp)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}

				expected = append(
					expected, all.messagesByChannel[*channelID][fp.key()])
			}
		}
	}

	// Check that the message map has all the expected messages
	for i, exp := range expected {
		fp := newCommandFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messagesByChannel[*exp.ChannelID]; !exists {
			t.Errorf("Channel %s does not exist (%d).", exp.ChannelID, i)
		} else if lm, exists2 := messages[fp.key()]; !exists2 {
			t.Errorf("No lease message found with key %s (%d).", fp.key(), i)
		} else {
			if !reflect.DeepEqual(exp, lm) {
				t.Errorf("leaseMessage does not match expected (%d)."+
					"\nexpected: %+v\nreceived: %+v", i, exp, lm)
			}
		}
	}

	// Get random channel ID
	var channelID id.ID
	for channelID = range all.messagesByChannel {
		break
	}

	err := all.removeChannel(&channelID)
	if err != nil {
		t.Errorf("Failed to remove channel: %+v", err)
	}

	for e := all.leases.Front(); e != nil && e.Next() != nil; e = e.Next() {
		// Check that the message does not exist in the list
		if e.Value.(*leaseMessage).ChannelID.Cmp(&channelID) {
			t.Errorf(
				"Found lease message from channel %s: %+v", channelID, e.Value)
		}
		if e.Value.(*leaseMessage).LeaseTrigger.After(
			e.Next().Value.(*leaseMessage).LeaseTrigger) {
			t.Errorf("Lease list not in order.")
		}
	}

	// Test removing a channel that does not exist
	err = all.removeChannel(randChannelID(prng, t))
	if err != nil {
		t.Errorf("Error when removing non-existent channel: %+v", err)
	}
}

// Tests that calculateLeaseTrigger returns times within the expected
// window. Runs the test many times to ensure no numbers fall outside the range.
func Test_calculateLeaseTrigger(t *testing.T) {
	rng := csprng.NewSystemRNG()
	ts := time.Date(1955, 11, 5, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		lease                               time.Duration
		now, originatingTimestamp, expected time.Time
	}{
		{time.Hour, ts, ts, ts.Add(time.Hour)},
		{time.Hour, ts, ts.Add(-time.Minute), ts.Add(time.Hour - time.Minute)},
		{MessageLife, ts, ts.Add(-MessageLife / 2), ts.Add(MessageLife / 2)},
		{MessageLife, ts, ts, time.Time{}},
		{MessageLife * 3 / 2, ts, ts.Add(-time.Minute), time.Time{}},
		{ValidForever, ts, ts.Add(-2000 * time.Hour), time.Time{}},
	}

	// for i := 0; i < 100; i++ {
	for j, tt := range tests {
		leaseTrigger, leaseActive := calculateLeaseTrigger(
			tt.now, tt.originatingTimestamp, tt.lease, rng)
		if !leaseActive {
			t.Errorf("Lease is expired (%d).", j)
		} else if tt.expected != (time.Time{}) {
			if !leaseTrigger.Equal(tt.expected) {
				t.Errorf("lease trigger duration does not match expected "+
					"(%d).\nexpected: %s\nreceived: %s",
					j, tt.expected, leaseTrigger)
			}
		} else {
			if tt.lease == ValidForever {
				tt.originatingTimestamp = tt.now
			}
			floor := tt.originatingTimestamp.Add(MessageLife / 2)
			ceiling := tt.originatingTimestamp.Add(MessageLife)
			if leaseTrigger.Before(floor) {
				t.Errorf("lease trigger occurs before the floor (%d)."+
					"\nfloor:   %s\ntrigger: %s", j, floor, leaseTrigger)
			} else if leaseTrigger.After(ceiling) {
				t.Errorf("lease trigger occurs after the ceiling (%d)."+
					"\nceiling:  %s\ntrigger: %s", j, ceiling, leaseTrigger)
			}
		}
	}
}

// Tests that randDurationInRange returns positive unique numbers in range.
func Test_randDurationInRange(t *testing.T) {
	prng := rand.New(rand.NewSource(684_532))
	rng := csprng.NewSystemRNG()
	const n = 10_000
	ints := make(map[time.Duration]struct{}, n)

	for i := 0; i < n; i++ {
		start := time.Duration(prng.Int63()) / 2
		end := start + time.Duration(prng.Int63())/2

		num := randDurationInRange(start, end, rng)
		if num < start {
			t.Errorf("Int #%d is less than start.\nstart:   %d\nreceived: %d",
				i, start, num)
		} else if num > end {
			t.Errorf("Int #%d is greater than end.\nend:     %d\nreceived: %d",
				i, end, num)
		}

		if _, exists := ints[num]; exists {
			t.Errorf("Int #%d already generated: %d", i, num)
		} else {
			ints[num] = struct{}{}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that ActionLeaseList.load loads an ActionLeaseList from storage that
// matches the original.
func TestActionLeaseList_load(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewCommandStore(kv)
	crng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	all := NewActionLeaseList(nil, s, kv, crng)

	for i := 0; i < 10; i++ {
		channelID := randChannelID(prng, t)
		for j := 0; j < 5; j++ {
			lease := 200 * time.Hour
			randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
			timestamp := netTime.Now().UTC().Round(0).Add(-randDuration)
			lmp := &leaseMessagePacket{
				leaseMessage: &leaseMessage{
					ChannelID:            channelID,
					Action:               randAction(prng),
					Payload:              randPayload(prng, t),
					OriginatingTimestamp: timestamp,
					Lease:                lease,
					LeaseEnd:             timestamp.Add(lease),
					LeaseTrigger:         timestamp.Add(lease),
				},
				cm: &CommandMessage{
					ChannelID:        channelID,
					MessageID:        randMessageID(prng, t),
					EncryptedPayload: randPayload(prng, t),
					Timestamp:        timestamp,
				},
			}

			err := all.addMessage(lmp)
			if err != nil {
				t.Errorf("Failed to add message: %+v", err)
			}
		}
	}

	// Create new list and load old contents into it
	loadedAll := NewActionLeaseList(nil, s, kv, crng)
	err := loadedAll.load(time.Unix(0, 0))
	if err != nil {
		t.Errorf("Failed to load ActionLeaseList from storage: %+v", err)
	}

	// Check that the loaded message map matches the original
	for chanID, messages := range all.messagesByChannel {
		loadedMessages, exists := loadedAll.messagesByChannel[chanID]
		if !exists {
			t.Errorf("Channel ID %s does not exist in map.", chanID)
		}

		for fp, lm := range messages {
			loadedLm, exists2 := loadedMessages[fp]
			if !exists2 {
				t.Errorf("Lease message does not exist in map: %+v", lm)
			}

			lm.e, loadedLm.e = nil, nil
			if !reflect.DeepEqual(lm, loadedLm) {
				t.Errorf("leaseMessage does not match expected."+
					"\nexpected: %+v\nreceived: %+v", lm, loadedLm)
			}
		}
	}

	// Check that the loaded lease list matches the original
	e1, e2 := all.leases.Front(), loadedAll.leases.Front()
	for i := 0; e1 != nil; i, e1, e2 = i+1, e1.Next(), e2.Next() {
		if !reflect.DeepEqual(e1.Value, e2.Value) {
			t.Errorf("Element %d does not match expected."+
				"\nexpected: %+v\nreceived: %+v", i, e1.Value, e2.Value)
		}
	}
}

// Tests that when ActionLeaseList.load loads a leaseMessage with a lease
// trigger in the past, that a new one is randomly calculated between
// replayWaitMin and replayWaitMax.
func TestActionLeaseList_load_LeaseModify(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewCommandStore(kv)
	crng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	all := NewActionLeaseList(nil, s, kv, crng)

	now := netTime.Now().UTC().Round(0)
	lease := 1200 * time.Hour
	lmp := &leaseMessagePacket{
		leaseMessage: &leaseMessage{
			ChannelID:            randChannelID(prng, t),
			Action:               randAction(prng),
			Payload:              randPayload(prng, t),
			OriginatingTimestamp: now,
			Lease:                lease,
		},
		cm: &CommandMessage{
			ChannelID:        randChannelID(prng, t),
			MessageID:        randMessageID(prng, t),
			EncryptedPayload: randPayload(prng, t),
		},
	}

	err := all.addMessage(lmp)
	if err != nil {
		t.Errorf("Failed to add message: %+v", err)
	}

	// Create new list and load old contents into it
	loadedAll := NewActionLeaseList(nil, s, kv, crng)
	now = now.Add(MessageLife)
	err = loadedAll.load(now)
	if err != nil {
		t.Errorf("Failed to load ActionLeaseList from storage: %+v", err)
	}

	fp := newCommandFingerprint(lmp.ChannelID, lmp.Action, lmp.Payload)
	leaseEnd := loadedAll.messagesByChannel[*lmp.ChannelID][fp.key()].LeaseEnd
	leaseTrigger := loadedAll.messagesByChannel[*lmp.ChannelID][fp.key()].LeaseTrigger
	all.messagesByChannel[*lmp.ChannelID][fp.key()].LeaseEnd = leaseEnd
	all.messagesByChannel[*lmp.ChannelID][fp.key()].LeaseTrigger = leaseTrigger
	if !reflect.DeepEqual(all.messagesByChannel[*lmp.ChannelID][fp.key()],
		loadedAll.messagesByChannel[*lmp.ChannelID][fp.key()]) {
		t.Errorf("Loaded lease message does not match original."+
			"\nexpected: %+v\nreceived: %+v",
			all.messagesByChannel[*lmp.ChannelID][fp.key()],
			loadedAll.messagesByChannel[*lmp.ChannelID][fp.key()])
	}

	if leaseTrigger.Before(now.Add(replayWaitMin)) ||
		leaseTrigger.After(now.Add(replayWaitMax)) {
		t.Errorf("Lease trigger out of range.\nfloor:        %s"+
			"\nceiling:      %s\nleaseTrigger: %s",
			now.Add(replayWaitMin), now.Add(replayWaitMax), leaseTrigger)
	}

	// Check that the loaded lease list matches the original
	e1, e2 := all.leases.Front(), loadedAll.leases.Front()
	for ; e1 != nil; e1, e2 = e1.Next(), e2.Next() {
		if !reflect.DeepEqual(e1.Value, e2.Value) {
			t.Errorf("Element does not match expected."+
				"\nexpected: %+v\nreceived: %+v", e1.Value, e2.Value)
		}
	}
}

// Error path: Tests that ActionLeaseList.load returns the expected error when
// no channel IDs can be loaded from storage.
func TestActionLeaseList_load_ChannelListLoadError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	expectedErr := loadLeaseChanIDsErr

	err := all.load(time.Unix(0, 0))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to return expected error no channel ID list exists."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: Tests that ActionLeaseList.load returns the expected error when
// no lease messages can be loaded from storage.
func TestActionLeaseList_load_LeaseMessagesLoadError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	channelID := randChannelID(rand.New(rand.NewSource(32)), t)
	all.messagesByChannel[*channelID] =
		make(map[commandFingerprintKey]*leaseMessage)
	err := all.storeLeaseChannels()
	if err != nil {
		t.Fatalf("Failed to store lease channels: %+v", err)
	}

	expectedErr := fmt.Sprintf(loadLeaseMessagesErr, channelID)

	err = all.load(time.Unix(0, 0))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to return expected error no lease messages exist."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the list of channel IDs in the message map can be saved and loaded
// to and from storage with ActionLeaseList.storeLeaseChannels and
// ActionLeaseList.loadLeaseChannels.
func TestActionLeaseList_storeLeaseChannels_loadLeaseChannels(t *testing.T) {
	const n = 10
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	s := NewCommandStore(kv)
	all := NewActionLeaseList(
		nil, s, kv, fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	expectedIDs := make([]*id.ID, n)

	for i := 0; i < n; i++ {
		channelID := randChannelID(prng, t)
		all.messagesByChannel[*channelID] =
			make(map[commandFingerprintKey]*leaseMessage)
		for j := 0; j < 5; j++ {
			lm := &leaseMessage{
				ChannelID: channelID,
				Action:    randAction(prng),
				Payload:   randPayload(prng, t),
			}
			fp := newCommandFingerprint(channelID, lm.Action, lm.Payload)
			all.messagesByChannel[*channelID][fp.key()] = lm
		}
		expectedIDs[i] = channelID
	}

	err := all.storeLeaseChannels()
	if err != nil {
		t.Errorf("Failed to store channel IDs: %+v", err)
	}

	loadedIDs, err := all.loadLeaseChannels()
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

// Error path: Tests that ActionLeaseList.loadLeaseChannels returns an error
// when trying to load from storage when nothing was saved.
func TestActionLeaseList_loadLeaseChannels_StorageError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	_, err := all.loadLeaseChannels()
	if err == nil || kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that a list of leaseMessage can be stored and loaded using
// ActionLeaseList.storeLeaseMessages and ActionLeaseList.loadLeaseMessages.
func TestActionLeaseList_storeLeaseMessages_loadLeaseMessages(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	channelID := randChannelID(prng, t)
	all.messagesByChannel[*channelID] =
		make(map[commandFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID:            channelID,
			Action:               randAction(prng),
			Payload:              randPayload(prng, t),
			OriginatingTimestamp: randTimestamp(prng),
			Lease:                randLease(prng),
			LeaseEnd:             randTimestamp(prng),
			LeaseTrigger:         randTimestamp(prng),
			e:                    nil,
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		all.messagesByChannel[*channelID][fp.key()] = lm
	}

	err := all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	loadedMessages, err := all.loadLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to load messages: %+v", err)
	}

	if !reflect.DeepEqual(all.messagesByChannel[*channelID], loadedMessages) {
		t.Errorf("Loaded messages do not match original."+
			"\nexpected: %+v\nreceived: %+v",
			all.messagesByChannel[*channelID], loadedMessages)
	}
}

// Tests that ActionLeaseList.storeLeaseMessages deletes the lease message file
// from storage when the list is empty.
func TestActionLeaseList_storeLeaseMessages_EmptyList(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	channelID := randChannelID(prng, t)
	all.messagesByChannel[*channelID] =
		make(map[commandFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID:    channelID,
			Action:       randAction(prng),
			Payload:      randPayload(prng, t),
			LeaseEnd:     randTimestamp(prng),
			LeaseTrigger: randTimestamp(prng),
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		all.messagesByChannel[*channelID][fp.key()] = lm
	}

	err := all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	all.messagesByChannel[*channelID] =
		make(map[commandFingerprintKey]*leaseMessage)
	err = all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	_, err = all.loadLeaseMessages(channelID)
	if err == nil || all.kv.Exists(err) {
		t.Fatalf("Failed to delete lease messages: %+v", err)
	}
}

// Error path: Tests that ActionLeaseList.loadLeaseMessages returns an error
// when trying to load from storage when nothing was saved.
func TestActionLeaseList_loadLeaseMessages_StorageError(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	_, err := all.loadLeaseMessages(randChannelID(prng, t))
	if err == nil || all.kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that ActionLeaseList.deleteLeaseMessages removes the lease messages
// from storage.
func TestActionLeaseList_deleteLeaseMessages(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := NewActionLeaseList(nil, NewCommandStore(kv), kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	channelID := randChannelID(prng, t)
	all.messagesByChannel[*channelID] =
		make(map[commandFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID:    channelID,
			Action:       randAction(prng),
			Payload:      randPayload(prng, t),
			LeaseEnd:     randTimestamp(prng),
			LeaseTrigger: randTimestamp(prng),
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		all.messagesByChannel[*channelID][fp.key()] = lm
	}

	err := all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	err = all.deleteLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to delete messages: %+v", err)
	}

	_, err = all.loadLeaseMessages(channelID)
	if err == nil || all.kv.Exists(err) {
		t.Fatalf("Failed to delete lease messages: %+v", err)
	}
}

// Tests that a leaseMessage object can be JSON marshalled and unmarshalled.
func Test_leaseMessage_JSON(t *testing.T) {
	prng := rand.New(rand.NewSource(12))
	timestamp, lease := netTime.Now().UTC().Round(0), randLease(prng)

	lm := leaseMessage{
		ChannelID:            randChannelID(prng, t),
		Action:               randAction(prng),
		Payload:              randPayload(prng, t),
		OriginatingTimestamp: timestamp,
		Lease:                lease,
		LeaseEnd:             timestamp.Add(lease),
		LeaseTrigger:         randTimestamp(prng).Add(lease),
		e:                    nil,
	}

	data, err := json.Marshal(&lm)
	if err != nil {
		t.Errorf("Failed to JSON marshal leaseMessage: %+v", err)
	}

	var loadedLm leaseMessage
	err = json.Unmarshal(data, &loadedLm)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal leaseMessage: %+v", err)
	}

	if !reflect.DeepEqual(lm, loadedLm) {
		t.Errorf("Loaded leaseMessage does not match original."+
			"\nexpected: %#v\nreceived: %#v", lm, loadedLm)
	}
}

// Tests that a map of leaseMessage objects can be JSON marshalled and
// unmarshalled.
func Test_leaseMessageMap_JSON(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	const n = 15
	messages := make(map[commandFingerprintKey]*leaseMessage, n)

	for i := 0; i < n; i++ {
		timestamp := randTimestamp(prng)
		lm := &leaseMessage{
			ChannelID:            randChannelID(prng, t),
			Action:               randAction(prng),
			Payload:              randPayload(prng, t),
			OriginatingTimestamp: timestamp,
			Lease:                5 * time.Hour,
			LeaseEnd:             timestamp,
			LeaseTrigger:         timestamp,
			e:                    nil,
		}
		fp := newCommandFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		messages[fp.key()] = lm
	}

	data, err := json.Marshal(&messages)
	if err != nil {
		t.Errorf("Failed to JSON marshal map of leaseMessage: %+v", err)
	}

	var loadedMessages map[commandFingerprintKey]*leaseMessage
	err = json.Unmarshal(data, &loadedMessages)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal map of leaseMessage: %+v", err)
	}

	if !reflect.DeepEqual(messages, loadedMessages) {
		t.Errorf("Loaded map of leaseMessage does not match original."+
			"\nexpected: %+v\nreceived: %+v", messages, loadedMessages)
	}
}

// Consistency test of makeChannelLeaseMessagesKey.
func Test_makeChannelLeaseMessagesKey_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(11))

	expectedKeys := []string{
		"channelLeaseMessages/WQwUQJiItbB9UagX7gfD8hRZNbxxVePHp2SQw+CqC2oD",
		"channelLeaseMessages/WGLDLvh5GdCZH3r4XpU7dEKP71tXeJvJAi/UyPkxnakD",
		"channelLeaseMessages/mo59OR72CzZlLvnGxzfhscEY4AxjhmvE6b5W+yK1BQUD",
		"channelLeaseMessages/TOFI3iGP8TNZJ/V1/E4SrgW2MiS9LRxIzM0LoMnUmukD",
		"channelLeaseMessages/xfUsHf4FuGVcwFkKywinHo7mCdaXppXef4RU7l0vUQwD",
		"channelLeaseMessages/dpBGwqS9/xi7eiT+cPNRzC3BmdDg/aY3MR2IPdHBUCAD",
		"channelLeaseMessages/ZnT0fZYP2dCHlxxDo6DSpBplgaM3cj7RPgTZ+OF7MiED",
		"channelLeaseMessages/rXartsxcv2+tIPfN2x9r3wgxPqp77YK2/kSqqKzgw5ID",
		"channelLeaseMessages/6G0Z4gfi6u2yUp9opRTgcB0FpSv/x55HgRo6tNNi5lYD",
		"channelLeaseMessages/7aHvDBG6RsPXxMHvw21NIl273F0CzDN5aixeq5VRD+8D",
		"channelLeaseMessages/v0Pw6w7z7XAaebDUOAv6AkcMKzr+2eOIxLcDMMr/i2gD",
		"channelLeaseMessages/7OI/yTc2sr0m0kONaiV3uolWpyvJHXAtts4bZMm7o14D",
		"channelLeaseMessages/jDQqEBKqNhLpKtsIwIaW5hzUy+JdQ0JkXfkbae5iLCgD",
		"channelLeaseMessages/TCTUC3AblwtJiOHcvDNrmY1o+xm6VueZXhXDm3qDwT4D",
		"channelLeaseMessages/niQssT7H/lGZ0QoQWqLwLM24xSJeDBKKadamDlVM340D",
		"channelLeaseMessages/EYzeEw5VzugCW1QGXgq0jWVc5qbeoot+LH+Pt136xIED",
	}
	for i, expected := range expectedKeys {
		key := makeChannelLeaseMessagesKey(randChannelID(prng, t))

		if expected != key {
			t.Errorf("Key does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Test Utility Function                                                      //
////////////////////////////////////////////////////////////////////////////////

// randChannelID creates a new random channel id.ID for testing.
func randChannelID(rng io.Reader, t *testing.T) *id.ID {
	channelID, err := id.NewRandomID(rng, id.User)
	if err != nil {
		t.Fatalf("Failed to generate new channel ID: %+v", err)
	}

	return channelID
}

// randMessageID creates a new random channel.MessageID for testing.
func randMessageID(rng io.Reader, t *testing.T) message.ID {
	msg := make([]byte, 256)
	if _, err := rng.Read(msg); err != nil {
		t.Fatalf("Failed to generate random message: %+v", err)
	}

	channelID, err := id.NewRandomID(rng, id.User)
	if err != nil {
		t.Fatalf("Failed to generate new channel ID: %+v", err)
	}

	return message.DeriveChannelMessageID(channelID, 5, msg)
}

// randPayload creates a new random payload for testing.
func randPayload(rng io.Reader, t *testing.T) []byte {
	const payloadSize = 256
	payload := make([]byte, payloadSize)

	if n, err := rng.Read(payload); err != nil {
		t.Fatalf("Failed to generate new payload: %+v", err)
	} else if n != payloadSize {
		t.Fatalf("Only generated %d bytes when %d bytes required for payload.",
			n, payloadSize)
	}

	return payload
}

// randAction creates a new random action MessageType for testing.
func randAction(rng io.Reader) MessageType {
	return MessageType(
		randomness.ReadRangeUint32(uint32(Delete), uint32(Mute)+1, rng))
}

// randTimestamp creates a new random action lease end for testing.
func randTimestamp(rng io.Reader) time.Time {
	lease := randDurationInRange(1*time.Hour, 1000*time.Hour, rng)
	return netTime.Now().Add(lease).UTC().Round(0)
}

// randLease creates a new random lease duration end for testing.
func randLease(rng io.Reader) time.Duration {
	return randDurationInRange(1*time.Minute, 1000*time.Hour, rng)
}
