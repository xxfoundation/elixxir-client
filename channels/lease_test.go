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

// Tests that newOrLoadActionLeaseList initialises a new empty actionLeaseList
// when called for the first time and that it loads the actionLeaseList from
// storage after the original has been saved.
func Test_newOrLoadActionLeaseList(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	expected := newActionLeaseList(nil, kv, rng)

	all, err := newOrLoadActionLeaseList(nil, kv, rng)
	if err != nil {
		t.Errorf("Failed to create new actionLeaseList: %+v", err)
	}

	all.addLeaseMessage = expected.addLeaseMessage
	all.removeLeaseMessage = expected.removeLeaseMessage
	all.removeChannelCh = expected.removeChannelCh
	if !reflect.DeepEqual(expected, all) {
		t.Errorf("New actionLeaseList does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, all)
	}

	lm := &leaseMessage{
		ChannelID:         randChannelID(prng, t),
		Action:            randAction(prng),
		Payload:           randPayload(prng, t),
		Timestamp:         randTimestamp(prng),
		OriginalTimestamp: randTimestamp(prng),
		Lease:             time.Hour,
	}
	err = all.addMessage(lm)
	if err != nil {
		t.Errorf("Failed to add message: %+v", err)
	}
	for _, l := range all.messagesByChannel[*lm.ChannelID] {
		lm.LeaseEnd = l.LeaseEnd
		lm.LeaseTrigger = l.LeaseTrigger
	}

	loadedAll, err := newOrLoadActionLeaseList(nil, kv, rng)
	if err != nil {
		t.Errorf("Failed to load actionLeaseList: %+v", err)
	}

	all.addLeaseMessage = loadedAll.addLeaseMessage
	all.removeLeaseMessage = loadedAll.removeLeaseMessage
	all.removeChannelCh = loadedAll.removeChannelCh
	if !reflect.DeepEqual(all, loadedAll) {
		t.Errorf("Loaded actionLeaseList does not match expected."+
			"\nexpected: %+v\nreceived: %+v\nexpected: %+v\nreceived: %+v",
			all, loadedAll, all.messagesByChannel, loadedAll.messagesByChannel)
	}
}

// Tests that newActionLeaseList returns the expected new actionLeaseList.
func Test_newActionLeaseList(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	expected := &actionLeaseList{
		leases:             list.New(),
		messagesByChannel:  make(map[id.ID]map[leaseFingerprintKey]*leaseMessage),
		addLeaseMessage:    make(chan *leaseMessage, addLeaseMessageChanSize),
		removeLeaseMessage: make(chan *leaseMessage, removeLeaseMessageChanSize),
		removeChannelCh:    make(chan *id.ID, removeChannelChChanSize),
		triggerFn:          nil,
		kv:                 kv,
		rng:                rng,
	}

	all := newActionLeaseList(nil, kv, rng)
	all.addLeaseMessage = expected.addLeaseMessage
	all.removeLeaseMessage = expected.removeLeaseMessage
	all.removeChannelCh = expected.removeChannelCh

	if !reflect.DeepEqual(expected, all) {
		t.Errorf("New actionLeaseList does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, all)
	}
}

// Tests that actionLeaseList.StartProcesses returns an error until
// actionLeaseList.RegisterReplayFn has been called.
func TestActionLeaseList_StartProcesses_RegisterReplayFn(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	all := newActionLeaseList(nil, kv, rng)

	_, err := all.StartProcesses()
	if err == nil {
		t.Errorf("StartProcesses did not fail when the replay function has " +
			"not been set.")
	}

	all.RegisterReplayFn(func(*id.ID, []byte) {})

	_, err = all.StartProcesses()
	if err != nil {
		t.Errorf("StartProcesses failed: %+v", err)
	}
}

// Tests that actionLeaseList.updateLeasesThread removes the expected number of
// lease messages when they expire.
func Test_actionLeaseList_updateLeasesThread(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	triggerChan := make(chan *leaseMessage, 3)
	trigger := func(channelID *id.ID, _ message.ID,
		messageType MessageType, nickname string, payload, _ []byte, timestamp,
		originalTimestamp time.Time, lease time.Duration, _ rounds.Round,
		_ SentStatus, _ bool) (uint64, error) {
		triggerChan <- &leaseMessage{
			ChannelID:         channelID,
			Action:            messageType,
			Payload:           payload,
			OriginalTimestamp: originalTimestamp,
		}
		return 0, nil
	}
	replay := func(channelID *id.ID, encryptedPayload []byte) {
		triggerChan <- &leaseMessage{
			ChannelID: channelID,
			Payload:   encryptedPayload,
		}
	}
	all := newActionLeaseList(trigger, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	all.RegisterReplayFn(replay)

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	timestamp := netTime.Now().UTC().Round(0)
	expectedMessages := map[time.Duration]*leaseMessage{
		50 * time.Millisecond: {
			ChannelID:         randChannelID(prng, t),
			Action:            randAction(prng),
			Payload:           randPayload(prng, t),
			OriginalTimestamp: timestamp,
		},
		200 * time.Millisecond: {
			ChannelID:         randChannelID(prng, t),
			Action:            randAction(prng),
			Payload:           randPayload(prng, t),
			OriginalTimestamp: timestamp,
		},
		400 * time.Millisecond: {
			ChannelID:         randChannelID(prng, t),
			Action:            randAction(prng),
			Payload:           randPayload(prng, t),
			OriginalTimestamp: timestamp,
		},
		600 * time.Hour: { // This tests the replay code
			ChannelID:         randChannelID(prng, t),
			Action:            randAction(prng),
			EncryptedPayload:  randPayload(prng, t),
			OriginalTimestamp: timestamp.Add(-time.Hour),
		},
	}

	for lease, e := range expectedMessages {
		all.AddMessage(e.ChannelID, e.MessageID, e.Action, e.Payload,
			e.EncryptedPayload, e.Timestamp, e.OriginalTimestamp, lease,
			e.FromAdmin)
	}

	fp := newLeaseFingerprint(expectedMessages[600*time.Hour].ChannelID,
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
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be triggered.")
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[200*time.Millisecond]
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be triggered.")
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[400*time.Millisecond]
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
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
		if !bytes.Equal(expected.EncryptedPayload, lm.Payload) {
			t.Errorf("Did not receive expected EncryptedPayload."+
				"\nexpected: %v\nreceived: %v",
				expected.EncryptedPayload, lm.Payload)
		}
	case <-time.After(800 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be triggered.")
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to close thread: %+v", err)
	}
}

// Tests that actionLeaseList.updateLeasesThread stops the stoppable when
// triggered and returns.
func Test_actionLeaseList_updateLeasesThread_Stoppable(t *testing.T) {
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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

// Tests that actionLeaseList.updateLeasesThread adds and removes a lease
// channel.
func Test_actionLeaseList_updateLeasesThread_AddAndRemove(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	lease := 200 * time.Hour
	randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
	timestamp := netTime.Now().UTC().Add(-randDuration).Round(0)
	exp := &leaseMessage{
		ChannelID:         randChannelID(prng, t),
		MessageID:         randMessageID(prng, t),
		Action:            randAction(prng),
		Payload:           randPayload(prng, t),
		EncryptedPayload:  randPayload(prng, t),
		OriginalTimestamp: timestamp,
		Lease:             lease,
		LeaseEnd:          timestamp.Add(lease),
		FromAdmin:         false,
		e:                 nil,
	}
	fp := newLeaseFingerprint(
		exp.ChannelID, exp.Action, exp.Payload)

	all.AddMessage(exp.ChannelID, exp.MessageID, exp.Action, exp.Payload,
		exp.EncryptedPayload, timestamp, timestamp, lease, exp.FromAdmin)

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
	exp.Timestamp, exp.OriginalTimestamp = lm.Timestamp, lm.OriginalTimestamp
	if !reflect.DeepEqual(exp, lm) {
		t.Errorf("Unexpected lease message added to lease list."+
			"\nexpected: %+v\nreceived: %+v", exp, lm)
	}

	if messages, exists := all.messagesByChannel[*exp.ChannelID]; !exists {
		t.Errorf("Channel %s not found in message map.", exp.ChannelID)
	} else if lm, exists = messages[fp.key()]; !exists {
		t.Errorf("Message with fingerprint %s not found in message map.", fp)
	} else if !reflect.DeepEqual(exp, lm) {
		t.Errorf("Unexpected lease message added to message map."+
			"\nexpected: %+v\nreceived: %+v", exp, lm)
	}

	all.RemoveMessage(exp.ChannelID, exp.Action, exp.Payload)

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

// Tests that actionLeaseList.AddMessage sends the expected leaseMessage on the
// addLeaseMessage channel.
func Test_actionLeaseList_AddMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	timestamp := randTimestamp(prng)
	lease := randLease(prng)
	exp := &leaseMessage{
		ChannelID:         randChannelID(prng, t),
		MessageID:         randMessageID(prng, t),
		Action:            randAction(prng),
		Payload:           randPayload(prng, t),
		EncryptedPayload:  randPayload(prng, t),
		OriginalTimestamp: timestamp,
		Lease:             lease,
		FromAdmin:         false,
		e:                 nil,
	}

	all.AddMessage(exp.ChannelID, exp.MessageID, exp.Action, exp.Payload,
		exp.EncryptedPayload, exp.Timestamp, exp.OriginalTimestamp, exp.Lease,
		exp.FromAdmin)

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

// Tests that actionLeaseList.addMessage adds all the messages to both the
// lease list and the message map and that the lease list is in the correct
// order.
func Test_actionLeaseList_addMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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
				lm := &leaseMessage{
					ChannelID:         channelID,
					Action:            MessageType(k),
					Payload:           payload,
					EncryptedPayload:  encrypted,
					OriginalTimestamp: timestamp,
					Lease:             lease,
					LeaseEnd:          timestamp.Add(lease),
					LeaseTrigger:      timestamp.Add(lease),
				}
				expected = append(expected, lm)

				err := all.addMessage(lm)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}
			}
		}
	}

	// Check that the message map has all the expected messages
	for i, exp := range expected {
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messagesByChannel[*exp.ChannelID]; !exists {
			t.Errorf("Channel %s does not exist (%d).", exp.ChannelID, i)
		} else if lm, exists2 := messages[fp.key()]; !exists2 {
			t.Errorf("No lease message found with key %s (%d).", fp.key(), i)
		} else {
			lm.e = nil
			if !reflect.DeepEqual(exp, lm) {
				t.Errorf("leaseMessage does not match expected (%d)."+
					"\nexpected: %+v\nreceived: %+v", i, exp, lm)
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

// Tests that after updating half the messages, actionLeaseList.addMessage moves
// the messages to the lease list is still in order.
func Test_actionLeaseList_addMessage_Update(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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
				timestamp := randTimestamp(prng)
				lease := randLease(prng)
				lm := &leaseMessage{
					ChannelID:        channelID,
					Action:           MessageType(k),
					Payload:          payload,
					EncryptedPayload: encrypted,
					LeaseEnd:         timestamp.Add(lease),
					LeaseTrigger:     timestamp.Add(lease),
				}
				expected = append(expected, lm)

				err := all.addMessage(lm)
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

// Tests that actionLeaseList.insertLease inserts all the leaseMessage in the
// correct order, from smallest LeaseTrigger to largest.
func Test_actionLeaseList_insertLease(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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
// actionLeaseList.updateLease correctly moves elements to the correct order
// when their LeaseTrigger changes.
func Test_actionLeaseList_updateLease(t *testing.T) {
	prng := rand.New(rand.NewSource(32_142))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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

// Tests that actionLeaseList.RemoveMessage sends the expected leaseMessage on
// the removeLeaseMessage channel.
func Test_actionLeaseList_RemoveMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	exp := &leaseMessage{
		ChannelID: randChannelID(prng, t),
		Action:    randAction(prng),
		Payload:   randPayload(prng, t),
	}

	all.RemoveMessage(exp.ChannelID, exp.Action, exp.Payload)

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

// Tests that actionLeaseList.removeMessage removes all the messages from both
// the lease list and the message map and that the lease list remains in the
// correct order after every removal.
func Test_actionLeaseList_removeMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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
				lm := &leaseMessage{
					ChannelID:         channelID,
					Action:            MessageType(k),
					Payload:           payload,
					EncryptedPayload:  encrypted,
					OriginalTimestamp: timestamp,
					Lease:             lease,
					LeaseEnd:          timestamp.Add(lease),
					LeaseTrigger:      timestamp.Add(lease),
				}
				fp := newLeaseFingerprint(channelID, lm.Action, payload)
				err := all.addMessage(lm)
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
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
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
		err := all.removeMessage(exp)
		if err != nil {
			t.Errorf("Failed to remove message %d: %+v", i, exp)
		}

		// Check that the message was removed from the map
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
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

// Tests that actionLeaseList.removeMessage does nothing and returns nil when
// removing a message that does not exist.
func Test_actionLeaseList_removeMessage_NonExistentMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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
				lm := &leaseMessage{
					ChannelID:         channelID,
					Action:            MessageType(k),
					Payload:           payload,
					EncryptedPayload:  encrypted,
					OriginalTimestamp: timestamp,
					Lease:             lease,
					LeaseEnd:          timestamp.Add(lease),
					LeaseTrigger:      timestamp.Add(lease),
				}
				fp := newLeaseFingerprint(channelID, lm.Action, payload)
				err := all.addMessage(lm)
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
	})
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

// Test that actionLeaseList.updateLeaseTrigger updates the LeaseTrigger and
// that the list is in order.
func Test_actionLeaseList_updateLeaseTrigger(t *testing.T) {
	prng := rand.New(rand.NewSource(8_175_178))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	const numMessages = 50
	messages := make([]*leaseMessage, numMessages)
	now := netTime.Now().UTC().Round(0)
	for i := 0; i < numMessages; i++ {
		lease := 600 * time.Hour
		timestamp := now.Add(-5 * time.Hour)
		lm := &leaseMessage{
			ChannelID:         randChannelID(prng, t),
			Action:            randAction(prng),
			Payload:           randPayload(prng, t),
			OriginalTimestamp: timestamp,
			Lease:             lease,
		}
		err := all.addMessage(lm)
		if err != nil {
			t.Errorf("Failed to add lease message (%d): %+v", i, err)
		}
		messages[i] = lm
	}

	for i, lm := range messages {
		oldLeaseTrigger := lm.LeaseTrigger
		err := all.updateLeaseTrigger(lm, now)
		if err != nil {
			t.Errorf("Failed to update lease trigger (%d): %+v", i, err)
		}

		if oldLeaseTrigger == lm.LeaseTrigger {
			t.Errorf("Failed to update lease trigger %d/%d. New should be "+
				"different from old.\nold: %s\nnew: %s", i+1, numMessages,
				oldLeaseTrigger, lm.LeaseTrigger)
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

// Tests that actionLeaseList.RemoveChannel removes all leases for the channel
// from the list.
func Test_actionLeaseList_RemoveChannel(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	var channelID *id.ID
	for i := 0; i < 5; i++ {
		channelID = randChannelID(prng, t)
		for j := 0; j < 5; j++ {
			lease := 200 * time.Hour
			timestamp := netTime.Now().UTC().Round(0).Add(-5 * time.Hour)
			exp := &leaseMessage{
				ChannelID:         channelID,
				MessageID:         randMessageID(prng, t),
				Action:            randAction(prng),
				Payload:           randPayload(prng, t),
				EncryptedPayload:  randPayload(prng, t),
				OriginalTimestamp: timestamp,
				Lease:             lease,
				LeaseEnd:          timestamp.Add(lease),
				LeaseTrigger:      timestamp.Add(lease),
			}

			all.AddMessage(exp.ChannelID, exp.MessageID, exp.Action,
				exp.Payload, exp.EncryptedPayload, exp.Timestamp,
				exp.OriginalTimestamp, exp.Lease, exp.FromAdmin)
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
	case <-time.After(80 * time.Millisecond):
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

// Tests that actionLeaseList.removeChannel removes all the messages from both
// the lease list and the message map for the given channel and that the lease
// list remains in the correct order after removal.
func Test_actionLeaseList_removeChannel(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
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
				lm := &leaseMessage{
					ChannelID:         channelID,
					Action:            MessageType(k),
					Payload:           payload,
					EncryptedPayload:  encrypted,
					OriginalTimestamp: timestamp,
					Lease:             lease,
					LeaseEnd:          timestamp.Add(lease),
					LeaseTrigger:      timestamp.Add(lease),
				}
				fp := newLeaseFingerprint(channelID, lm.Action, payload)
				err := all.addMessage(lm)
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
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
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
		lease                            time.Duration
		now, originalTimestamp, expected time.Time
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
		leaseTrigger, validLease := calculateLeaseTrigger(
			tt.now, tt.originalTimestamp, tt.lease, rng)
		if !validLease {
			t.Errorf("Invalid lease (%d).", j)
		} else if tt.expected != (time.Time{}) {
			if !leaseTrigger.Equal(tt.expected) {
				t.Errorf("lease trigger duration does not match expected "+
					"(%d).\nexpected: %s\nreceived: %s",
					j, tt.expected, leaseTrigger)
			}
		} else {
			if tt.lease == ValidForever {
				tt.originalTimestamp = tt.now
			}
			floor := tt.originalTimestamp.Add(MessageLife / 2)
			ceiling := tt.originalTimestamp.Add(MessageLife)
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

// Tests that actionLeaseList.load loads an actionLeaseList from storage that
// matches the original.
func Test_actionLeaseList_load(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := newActionLeaseList(
		nil, kv, fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	for i := 0; i < 10; i++ {
		channelID := randChannelID(prng, t)
		for j := 0; j < 5; j++ {
			lease := 200 * time.Hour
			randDuration := randDurationInRange(time.Hour, 5*time.Hour, prng)
			timestamp := netTime.Now().UTC().Round(0).Add(-randDuration)
			lm := &leaseMessage{
				ChannelID:         channelID,
				MessageID:         randMessageID(prng, t),
				Action:            randAction(prng),
				Payload:           randPayload(prng, t),
				EncryptedPayload:  randPayload(prng, t),
				Timestamp:         timestamp,
				OriginalTimestamp: timestamp,
				Lease:             lease,
				FromAdmin:         false,
				e:                 nil,
			}

			err := all.addMessage(lm)
			if err != nil {
				t.Errorf("Failed to add message: %+v", err)
			}
		}
	}

	// Create new list and load old contents into it
	loadedAll := newActionLeaseList(
		nil, kv, fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	err := loadedAll.load(time.Unix(0, 0))
	if err != nil {
		t.Errorf("Failed to load actionLeaseList from storage: %+v", err)
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

// Tests that when actionLeaseList.load loads a leaseMessage with a lease
// trigger in the past, that a new one is randomly calculated between
// replayWaitMin and replayWaitMax.
func Test_actionLeaseList_load_LeaseModify(t *testing.T) {
	prng := rand.New(rand.NewSource(23))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := newActionLeaseList(
		nil, kv, fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	now := netTime.Now().UTC().Round(0)
	lease := 1200 * time.Hour
	lm := &leaseMessage{
		ChannelID:         randChannelID(prng, t),
		MessageID:         randMessageID(prng, t),
		Action:            randAction(prng),
		Payload:           randPayload(prng, t),
		EncryptedPayload:  randPayload(prng, t),
		OriginalTimestamp: now,
		Lease:             lease,
	}

	err := all.addMessage(lm)
	if err != nil {
		t.Errorf("Failed to add message: %+v", err)
	}

	// Create new list and load old contents into it
	loadedAll := newActionLeaseList(
		nil, kv, fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	now = now.Add(MessageLife)
	err = loadedAll.load(now)
	if err != nil {
		t.Errorf("Failed to load actionLeaseList from storage: %+v", err)
	}

	fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
	leaseEnd := loadedAll.messagesByChannel[*lm.ChannelID][fp.key()].LeaseEnd
	leaseTrigger := loadedAll.messagesByChannel[*lm.ChannelID][fp.key()].LeaseTrigger
	all.messagesByChannel[*lm.ChannelID][fp.key()].LeaseEnd = leaseEnd
	all.messagesByChannel[*lm.ChannelID][fp.key()].LeaseTrigger = leaseTrigger
	if !reflect.DeepEqual(all.messagesByChannel[*lm.ChannelID][fp.key()],
		loadedAll.messagesByChannel[*lm.ChannelID][fp.key()]) {
		t.Errorf("Loaded lease message does not match original."+
			"\nexpected: %+v\nreceived: %+v",
			all.messagesByChannel[*lm.ChannelID][fp.key()],
			loadedAll.messagesByChannel[*lm.ChannelID][fp.key()])
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

// Error path: Tests that actionLeaseList.load returns the expected error when
// no channel IDs can be loaded from storage.
func Test_actionLeaseList_load_ChannelListLoadError(t *testing.T) {
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	expectedErr := loadLeaseChanIDsErr

	err := all.load(time.Unix(0, 0))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to return expected error no channel ID list exists."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: Tests that actionLeaseList.load returns the expected error when
// no lease messages can be loaded from storage.
func Test_actionLeaseList_load_LeaseMessagesLoadError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := newActionLeaseList(
		nil, kv, fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	chanID := randChannelID(rand.New(rand.NewSource(32)), t)
	all.messagesByChannel[*chanID] = make(map[leaseFingerprintKey]*leaseMessage)
	err := all.storeLeaseChannels()
	if err != nil {
		t.Errorf("Failed to store lease channels: %+v", err)
	}

	expectedErr := fmt.Sprintf(loadLeaseMessagesErr, chanID)

	err = all.load(time.Unix(0, 0))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to return expected error no lease messages exists."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the list of channel IDs in the message map can be saved and loaded
// to and from storage with actionLeaseList.storeLeaseChannels and
// actionLeaseList.loadLeaseChannels.
func Test_actionLeaseList_storeLeaseChannels_loadLeaseChannels(t *testing.T) {
	const n = 10
	prng := rand.New(rand.NewSource(32))
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := newActionLeaseList(
		nil, kv, fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	expectedIDs := make([]*id.ID, n)

	for i := 0; i < n; i++ {
		channelID := randChannelID(prng, t)
		all.messagesByChannel[*channelID] =
			make(map[leaseFingerprintKey]*leaseMessage)
		for j := 0; j < 5; j++ {
			payload, action := randPayload(prng, t), randAction(prng)
			encrypted := randPayload(prng, t)
			fp := newLeaseFingerprint(channelID, action, payload)
			all.messagesByChannel[*channelID][fp.key()] = &leaseMessage{
				ChannelID:        channelID,
				Action:           action,
				Payload:          payload,
				EncryptedPayload: encrypted,
			}
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

// Error path: Tests that actionLeaseList.loadLeaseChannels returns an error
// when trying to load when nothing was saved.
func Test_actionLeaseList_loadLeaseChannels_StorageError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	_, err := all.loadLeaseChannels()
	if err == nil || kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that a list of leaseMessage can be stored and loaded using
// actionLeaseList.storeLeaseMessages and actionLeaseList.loadLeaseMessages.
func Test_actionLeaseList_storeLeaseMessages_loadLeaseMessages(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	channelID := randChannelID(prng, t)
	all.messagesByChannel[*channelID] =
		make(map[leaseFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID:         channelID,
			MessageID:         randMessageID(prng, t),
			Action:            randAction(prng),
			Payload:           randPayload(prng, t),
			EncryptedPayload:  randPayload(prng, t),
			Timestamp:         randTimestamp(prng),
			OriginalTimestamp: randTimestamp(prng),
			Lease:             randLease(prng),
			LeaseEnd:          randTimestamp(prng),
			LeaseTrigger:      randTimestamp(prng),
			FromAdmin:         false,
			e:                 nil,
		}
		fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
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

// Tests that actionLeaseList.storeLeaseMessages deletes the lease message file
// from storage when the list is empty.
func Test_actionLeaseList_storeLeaseMessages_EmptyList(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	channelID := randChannelID(prng, t)
	all.messagesByChannel[*channelID] =
		make(map[leaseFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID:        channelID,
			Action:           randAction(prng),
			Payload:          randPayload(prng, t),
			EncryptedPayload: randPayload(prng, t),
			LeaseEnd:         randTimestamp(prng),
			LeaseTrigger:     randTimestamp(prng),
		}
		fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		all.messagesByChannel[*channelID][fp.key()] = lm
	}

	err := all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	all.messagesByChannel[*channelID] =
		make(map[leaseFingerprintKey]*leaseMessage)
	err = all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	_, err = all.loadLeaseMessages(channelID)
	if err == nil || all.kv.Exists(err) {
		t.Fatalf("Failed to delete lease messages: %+v", err)
	}
}

// Error path: Tests that actionLeaseList.loadLeaseMessages returns an error
// when trying to load when nothing was saved.
func Test_actionLeaseList_loadLeaseMessages_StorageError(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	_, err := all.loadLeaseMessages(randChannelID(prng, t))
	if err == nil || all.kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that actionLeaseList.deleteLeaseMessages removes the lease messages
// from storage.
func Test_actionLeaseList_deleteLeaseMessages(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	channelID := randChannelID(prng, t)
	all.messagesByChannel[*channelID] =
		make(map[leaseFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID:        channelID,
			Action:           randAction(prng),
			Payload:          randPayload(prng, t),
			EncryptedPayload: randPayload(prng, t),
			LeaseEnd:         randTimestamp(prng),
			LeaseTrigger:     randTimestamp(prng),
		}
		fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
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
	channelID := randChannelID(prng, t)
	payload := []byte("payload")
	encrypted := []byte("encrypted")
	timestamp, lease := netTime.Now().UTC().Round(0), 6*time.Minute+30*time.Second

	lm := leaseMessage{
		ChannelID:         channelID,
		MessageID:         cryptoChannel.MakeMessageID(payload, channelID),
		Action:            randAction(prng),
		Payload:           payload,
		EncryptedPayload:  encrypted,
		Timestamp:         timestamp,
		OriginalTimestamp: timestamp,
		Lease:             lease,
		LeaseEnd:          timestamp.Add(lease),
		LeaseTrigger:      timestamp.Add(lease),
		FromAdmin:         true,
		e:                 nil,
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
	messages := make(map[leaseFingerprintKey]*leaseMessage, n)

	for i := 0; i < n; i++ {
		timestamp := randTimestamp(prng)
		lm := &leaseMessage{
			ChannelID:         randChannelID(prng, t),
			MessageID:         randMessageID(prng, t),
			Action:            randAction(prng),
			Payload:           randPayload(prng, t),
			EncryptedPayload:  randPayload(prng, t),
			Timestamp:         timestamp,
			OriginalTimestamp: timestamp,
			Lease:             5 * time.Hour,
			LeaseEnd:          timestamp,
			LeaseTrigger:      timestamp,
			FromAdmin:         false,
			e:                 nil,
		}
		fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		messages[fp.key()] = lm
	}

	data, err := json.Marshal(&messages)
	if err != nil {
		t.Errorf("Failed to JSON marshal map of leaseMessage: %+v", err)
	}

	var loadedMessages map[leaseFingerprintKey]*leaseMessage
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
// Fingerprint                                                                //
////////////////////////////////////////////////////////////////////////////////

// Consistency test of newLeaseFingerprint.
func Test_newLeaseFingerprint_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(420))
	expectedFingerprints := []string{
		"HPplU+CG9P872SORbI4BeFxgjkuBPUlF3gSNm371U3c=",
		"PU4zKeWyqHwrFMbPUMT7BMIVwAkF8vPFsBB4bLf+Arw=",
		"+OqBwVfOwR1tracbff/TxrlT8AIcO2JD+AZ3pmyEgvQ=",
		"7hVYQ1cvCou0O4tFLcipa2IXZSDbRAs+sPhrlTFiF64=",
		"xzddIIMaEZh9q47YDt7umTZtfFOl6T+dzgzfhpneEB4=",
		"Ls2aePoiD7kYeJmzjb5CKS5KNYSr2LbHnW/7UTvkGh8=",
		"r0pqAaciOdWTpWOirV0xv07uZ8fFNmN+F0I6hbQRMZE=",
		"fDl6jf6l/g2+gOZPz/LepdxlTIwKmeEEaNW5gXrxcQ0=",
		"nS2bu34dC6tfKFz6nZu/w9ORA+bcbfow2qomMh5+2NI=",
		"Q8WhfIucZ4fNSfXjfQT6HRkZfV6HMurSgO2BU917f4E=",
		"nUgCKjHnAEX06S0Gocb5I/H2ADWMeSPKii4PND9Hjm4=",
		"zJFC3E3SZhfPxSY/sxziRG1pX5pp/g7ba9/nP6kTFyU=",
		"u8jPvEekbPEBUZyVN9ra2BqRvjlfHpdQwuu5dZHg7U8=",
		"PWEf6L9yPjeMl/xP0fI62FzCCLQklT28XWTYHDi+1FU=",
		"ntnLOuShjBY2f3clP3Adp5tv7PHJxcs7biernqnXa38=",
		"D1NIRb3FdEJKC3Kh84LDC5wtUwICURcGLeyLXF/c6vw=",
	}

	for i, expected := range expectedFingerprints {
		fp := newLeaseFingerprint(randChannelID(prng, t),
			randAction(prng), randPayload(prng, t))

		if expected != fp.String() {
			t.Errorf("leaseFingerprint does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, fp)
		}
	}
}

// Tests that any changes to any of the inputs to newLeaseFingerprint result in
// different fingerprints.
func Test_newLeaseFingerprint_Uniqueness(t *testing.T) {
	rng := csprng.NewSystemRNG()
	const n = 100
	chanIDs := make([]*id.ID, n)
	payloads, encryptedPayloads := make([][]byte, n), make([][]byte, n)
	for i := 0; i < n; i++ {
		chanIDs[i] = randChannelID(rng, t)
		payloads[i] = randPayload(rng, t)
		encryptedPayloads[i] = randPayload(rng, t)

	}
	actions := []MessageType{Delete, Pinned, Mute}

	fingerprints := make(map[string]bool)
	for _, channelID := range chanIDs {
		for _, payload := range payloads {
			for _, action := range actions {
				fp := newLeaseFingerprint(channelID, action, payload)
				if fingerprints[fp.String()] {
					t.Errorf("Fingerprint %s already exists.", fp)
				}

				fingerprints[fp.String()] = true
			}
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
