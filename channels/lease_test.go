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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/comms/mixmessages"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
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
	expected := newActionLeaseList(nil, kv)

	all, err := newOrLoadActionLeaseList(nil, kv)
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
		ChannelID: newRandomChanID(prng, t),
		Action:    newRandomAction(prng, t),
		Payload:   newRandomPayload(prng, t),
		LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
	}
	err = all._addMessage(lm)
	if err != nil {
		t.Errorf("Failed to add message: %+v", err)
	}

	loadedAll, err := newOrLoadActionLeaseList(nil, kv)
	if err != nil {
		t.Errorf("Failed to load actionLeaseList: %+v", err)
	}

	all.addLeaseMessage = loadedAll.addLeaseMessage
	all.removeLeaseMessage = loadedAll.removeLeaseMessage
	all.removeChannelCh = loadedAll.removeChannelCh
	if !reflect.DeepEqual(all, loadedAll) {
		t.Errorf("Loaded actionLeaseList does not match expected."+
			"\nexpected: %+v\nreceived: %+v", all, loadedAll)
	}
}

// Tests that newActionLeaseList returns the expected new actionLeaseList.
func Test_newActionLeaseList(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &actionLeaseList{
		leases:             list.New(),
		messages:           make(map[id.ID]map[leaseFingerprintKey]*leaseMessage),
		addLeaseMessage:    make(chan *leaseMessage, addLeaseMessageChanSize),
		removeLeaseMessage: make(chan *leaseMessage, removeLeaseMessageChanSize),
		removeChannelCh:    make(chan *id.ID, removeChannelChChanSize),
		triggerFn:          nil,
		kv:                 kv,
	}

	all := newActionLeaseList(nil, kv)
	all.addLeaseMessage = expected.addLeaseMessage
	all.removeLeaseMessage = expected.removeLeaseMessage
	all.removeChannelCh = expected.removeChannelCh

	if !reflect.DeepEqual(expected, all) {
		t.Errorf("New actionLeaseList does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, all)
	}
}

// Tests that actionLeaseList.updateLeasesThread removes the expected number of
// lease messages when they expire.
func Test_actionLeaseList(t *testing.T) {
	// jww.SetStdoutThreshold(jww.LevelTrace)
	prng := rand.New(rand.NewSource(32))
	triggerChan := make(chan *leaseMessage, 3)
	trigger := func(channelID *id.ID, _ cryptoChannel.MessageID,
		messageType MessageType, _ string, payload []byte, timestamp time.Time,
		lease time.Duration, _ rounds.Round, _ SentStatus, _ bool) (
		uint64, error) {
		triggerChan <- &leaseMessage{
			ChannelID: channelID,
			Action:    messageType,
			Payload:   payload,
			LeaseEnd:  timestamp.Add(lease).UnixNano(),
		}
		return 0, nil
	}
	all := newActionLeaseList(trigger, versioned.NewKV(ekv.MakeMemstore()))

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	expectedMessages := map[time.Duration]*leaseMessage{
		50 * time.Millisecond: {
			ChannelID: newRandomChanID(prng, t),
			Action:    newRandomAction(prng, t),
			Payload:   newRandomPayload(prng, t),
		},
		200 * time.Millisecond: {
			ChannelID: newRandomChanID(prng, t),
			Action:    newRandomAction(prng, t),
			Payload:   newRandomPayload(prng, t),
		},
		400 * time.Millisecond: {
			ChannelID: newRandomChanID(prng, t),
			Action:    newRandomAction(prng, t),
			Payload:   newRandomPayload(prng, t),
		},
	}

	timeNow := netTime.Now()
	for lease, e := range expectedMessages {
		all.addMessage(e.ChannelID, cryptoChannel.MessageID{}, e.Action, "",
			e.Payload, timeNow, lease, rounds.Round{ID: 5}, 0)
		expectedMessages[lease].LeaseEnd = timeNow.Add(lease).UnixNano()
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[50*time.Millisecond]
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
		}
		all.removeMessage(lm.ChannelID, lm.Action, lm.Payload)
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be removed.")
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[200*time.Millisecond]
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
		}
		all.removeMessage(lm.ChannelID, lm.Action, lm.Payload)
	case <-time.After(200 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be removed.")
	}

	select {
	case lm := <-triggerChan:
		expected := expectedMessages[400*time.Millisecond]
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("Did not receive expected lease message."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
		}
		all.removeMessage(lm.ChannelID, lm.Action, lm.Payload)
	case <-time.After(400 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be removed.")
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to close thread: %+v", err)
	}
}

// Tests that actionLeaseList.updateLeasesThread adds and removes a lease
// channel.
func Test_actionLeaseList_updateLeasesThread_AddAndRemove(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	stop := stoppable.NewSingle(leaseThreadStoppable)
	go all.updateLeasesThread(stop)

	timestamp, lease := netTime.Now(), time.Hour
	expected := &leaseMessage{
		ChannelID: newRandomChanID(prng, t),
		Action:    newRandomAction(prng, t),
		Payload:   newRandomPayload(prng, t),
		Timestamp: timestamp,
		Lease:     lease,
		LeaseEnd:  timestamp.Add(lease).UnixNano(),
		Round:     rounds.Round{ID: 5},
	}
	fp := newLeaseFingerprint(
		expected.ChannelID, expected.Action, expected.Payload)

	all.addMessage(expected.ChannelID, cryptoChannel.MessageID{},
		expected.Action, "", expected.Payload, timestamp, lease,
		expected.Round, 0)

	done := make(chan struct{})
	go func() {
		for len(all.messages) < 1 {
			time.Sleep(time.Millisecond)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be added to message map.")
	}

	lm := all.leases.Front().Value.(*leaseMessage)
	expected.e = lm.e
	if !reflect.DeepEqual(expected, lm) {
		t.Errorf("Unexpected lease message added to lease list."+
			"\nexpected: %+v\nreceived: %+v", expected, lm)
	}

	if messages, exist := all.messages[*expected.ChannelID]; !exist {
		t.Errorf("Channel %s not found in message map.", expected.ChannelID)
	} else if lm, exists2 := messages[fp.key()]; !exists2 {
		t.Errorf("Message with fingerprint %s not found in message map.", fp)
	} else if !reflect.DeepEqual(expected, lm) {
		t.Errorf("Unexpected lease message added to message map."+
			"\nexpected: %+v\nreceived: %+v", expected, lm)
	}

	all.removeMessage(expected.ChannelID, expected.Action, expected.Payload)

	done = make(chan struct{})
	go func() {
		for len(all.messages) != 0 {
			time.Sleep(time.Millisecond)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Millisecond):
		t.Errorf("Timed out waiting for message to be removed from message map.")
	}

	if all.leases.Len() != 0 {
		t.Errorf("%d messages left in lease list.", all.leases.Len())
	}

	if err := stop.Close(); err != nil {
		t.Errorf("Failed to close thread: %+v", err)
	}
}

// Tests that actionLeaseList.updateLeasesThread stops the stoppable when
// triggered and returns.
func Test_actionLeaseList_updateLeasesThread_Stoppable(t *testing.T) {
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

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

// Tests that actionLeaseList.addMessage sends the expected leaseMessage on the
// addLeaseMessage channel.
func Test_actionLeaseList_addMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	timestamp := newRandomLeaseEnd(prng, t)
	lease := newRandomLease(prng, t)
	expected := &leaseMessage{
		ChannelID: newRandomChanID(prng, t),
		MessageID: cryptoChannel.MessageID{},
		Action:    newRandomAction(prng, t),
		Payload:   newRandomPayload(prng, t),
		Timestamp: timestamp,
		Lease:     lease,
		LeaseEnd:  timestamp.Add(lease).UnixNano(),
		Round:     rounds.Round{ID: 5},
	}

	all.addMessage(expected.ChannelID, cryptoChannel.MessageID{},
		expected.Action, "", expected.Payload, timestamp, lease,
		expected.Round, 0)

	select {
	case lm := <-all.addLeaseMessage:
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("leaseMessage does not match expected."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
		}
	case <-time.After(5 * time.Millisecond):
		t.Error("Timed out waiting on addLeaseMessage.")
	}
}

// Tests that actionLeaseList._addMessage adds all the messages to both the
// lease list and the message map and that the lease list is in the correct
// order.
func Test_actionLeaseList__addMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := newRandomChanID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := newRandomPayload(prng, t)

			for k := 0; k < o; k++ {
				timestamp := newRandomLeaseEnd(prng, t)
				lease := newRandomLease(prng, t)
				lm := &leaseMessage{
					ChannelID: channelID,
					Action:    MessageType(k),
					Payload:   payload,
					LeaseEnd:  timestamp.Add(lease).UnixNano(),
				}
				expected = append(expected, lm)

				err := all._addMessage(lm)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}
			}
		}
	}

	// Check that the message map has all the expected messages
	for i, exp := range expected {
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messages[*exp.ChannelID]; !exists {
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
		return expected[i].LeaseEnd < expected[j].LeaseEnd
	})
	for i, e := 0, all.leases.Front(); e != nil; i, e = i+1, e.Next() {
		if expected[i].LeaseEnd != e.Value.(*leaseMessage).LeaseEnd {
			t.Errorf("leaseMessage %d not in correct order."+
				"\nexpected: %+v\nreceived: %+v",
				i, expected[i], e.Value.(*leaseMessage))
		}
	}
}

// Tests that after updating half the messages, actionLeaseList._addMessage
// moves the messages to the lease list is still in order.
func Test_actionLeaseList__addMessage_Update(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := newRandomChanID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := newRandomPayload(prng, t)

			for k := 0; k < o; k++ {
				timestamp := newRandomLeaseEnd(prng, t)
				lease := newRandomLease(prng, t)
				lm := &leaseMessage{
					ChannelID: channelID,
					Action:    MessageType(k),
					Payload:   payload,
					LeaseEnd:  timestamp.Add(lease).UnixNano(),
				}
				expected = append(expected, lm)

				err := all._addMessage(lm)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}
			}
		}
	}

	// Update the time of half the messages.
	for i, lm := range expected {
		if i%2 == 0 {
			timestamp := newRandomLeaseEnd(prng, t)
			lease := time.Minute
			lm.LeaseEnd = timestamp.Add(lease).UnixNano()

			err := all._addMessage(lm)
			if err != nil {
				t.Errorf("Failed to add message: %+v", err)
			}
		}
	}

	// Check that the order is still correct
	sort.SliceStable(expected, func(i, j int) bool {
		return expected[i].LeaseEnd < expected[j].LeaseEnd
	})
	for i, e := 0, all.leases.Front(); e != nil; i, e = i+1, e.Next() {
		if expected[i].LeaseEnd != e.Value.(*leaseMessage).LeaseEnd {
			t.Errorf("leaseMessage %d not in correct order."+
				"\nexpected: %+v\nreceived: %+v",
				i, expected[i], e.Value.(*leaseMessage))
		}
	}
}

// Tests that actionLeaseList.removeMessage sends the expected leaseMessage on
// the removeLeaseMessage channel.
func Test_actionLeaseList_removeMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	expected := &leaseMessage{
		ChannelID: newRandomChanID(prng, t),
		Action:    newRandomAction(prng, t),
		Payload:   newRandomPayload(prng, t),
	}

	all.removeMessage(expected.ChannelID, expected.Action, expected.Payload)

	select {
	case lm := <-all.removeLeaseMessage:
		if !reflect.DeepEqual(expected, lm) {
			t.Errorf("leaseMessage does not match expected."+
				"\nexpected: %+v\nreceived: %+v", expected, lm)
		}
	case <-time.After(5 * time.Millisecond):
		t.Error("Timed out waiting on removeLeaseMessage.")
	}
}

// Tests that actionLeaseList._removeMessage removes all the messages from both
// the lease list and the message map and that the lease list remains in the
// correct order after every removal.
func Test_actionLeaseList__removeMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := newRandomChanID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := newRandomPayload(prng, t)

			for k := 0; k < o; k++ {
				lm := &leaseMessage{
					ChannelID: channelID,
					Action:    MessageType(k),
					Payload:   payload,
					LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
				}
				fp := newLeaseFingerprint(channelID, lm.Action, payload)
				err := all._addMessage(lm)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}

				expected = append(expected, all.messages[*channelID][fp.key()])
			}
		}
	}

	// Check that the message map has all the expected messages
	for i, exp := range expected {
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messages[*exp.ChannelID]; !exists {
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
		err := all._removeMessage(exp)
		if err != nil {
			t.Errorf("Failed to remove message %d: %+v", i, exp)
		}

		// Check that the message was removed from the map
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messages[*exp.ChannelID]; exists {
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
			if e.Value.(*leaseMessage).LeaseEnd >
				e.Next().Value.(*leaseMessage).LeaseEnd {
				t.Errorf("Lease list not in order.")
			}
		}
	}
}

// Tests that actionLeaseList._removeMessage does nothing and returns nil when
// removing a message that does not exist.
func Test_actionLeaseList__removeMessage_NonExistentMessage(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := newRandomChanID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := newRandomPayload(prng, t)

			for k := 0; k < o; k++ {
				lm := &leaseMessage{
					ChannelID: channelID,
					Action:    MessageType(k),
					Payload:   payload,
					LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
				}
				fp := newLeaseFingerprint(channelID, lm.Action, payload)
				err := all._addMessage(lm)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}

				expected = append(expected, all.messages[*channelID][fp.key()])
			}
		}
	}

	err := all._removeMessage(&leaseMessage{
		ChannelID: newRandomChanID(prng, t),
		Action:    newRandomAction(prng, t),
		Payload:   newRandomPayload(prng, t),
		LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
	})
	if err != nil {
		t.Errorf("Error removing message that does not exist: %+v", err)
	}

	if all.leases.Len() != len(expected) {
		t.Errorf("Unexpected length of lease list.\nexpected: %d\nreceived: %d",
			len(expected), all.leases.Len())
	}

	if len(all.messages) != m {
		t.Errorf("Unexpected length of message channels."+
			"\nexpected: %d\nreceived: %d", m, len(all.messages))
	}

	for chID, messages := range all.messages {
		if len(messages) != n*o {
			t.Errorf("Unexpected length of messages for channel %s."+
				"\nexpected: %d\nreceived: %d", chID, n*o, len(messages))
		}
	}
}

// Tests that actionLeaseList.insertLease inserts all the leaseMessage in the
// correct order, from smallest LeaseEnd to largest.
func Test_actionLeaseList_insertLease(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))
	expected := make([]int64, 50)

	for i := range expected {
		randomTime := time.Unix(0, prng.Int63())
		all.insertLease(&leaseMessage{LeaseEnd: randomTime.UnixNano()})
		expected[i] = randomTime.UnixNano()
	}

	sort.SliceStable(expected, func(i, j int) bool {
		return expected[i] < expected[j]
	})

	for i, e := 0, all.leases.Front(); e != nil; i, e = i+1, e.Next() {
		if expected[i] != e.Value.(*leaseMessage).LeaseEnd {
			t.Errorf("Timestamp %d not in correct order."+
				"\nexpected: %d\nreceived: %d",
				i, expected[i], e.Value.(*leaseMessage).LeaseEnd)
		}
	}
}

// Fills the lease list with in-order messages and tests that
// actionLeaseList.updateLease correctly moves elements to the correct order
// when their LeaseEnd changes.
func Test_actionLeaseList_updateLease(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	for i := 0; i < 50; i++ {
		randomTime := time.Unix(0, prng.Int63())
		all.insertLease(&leaseMessage{LeaseEnd: randomTime.UnixNano()})
	}

	tests := []struct {
		randomTime int64
		e          *list.Element
	}{
		// Change the first element to a random time
		{prng.Int63(), all.leases.Front()},

		// Change an element to a random time
		{prng.Int63(), all.leases.Front().Next().Next().Next()},

		// Change the last element to a random time
		{prng.Int63(), all.leases.Back()},

		// Change an element to the first element
		{all.leases.Front().Value.(*leaseMessage).LeaseEnd - 1,
			all.leases.Front().Next().Next()},

		// Change an element to the last element
		{all.leases.Back().Value.(*leaseMessage).LeaseEnd + 1,
			all.leases.Front().Next().Next().Next().Next().Next()},
	}

	for i, tt := range tests {
		tt.e.Value.(*leaseMessage).LeaseEnd = tt.randomTime
		all.updateLease(tt.e)

		// Check that the list is in order
		for n := all.leases.Front(); n.Next() != nil; n = n.Next() {
			if n.Value.(*leaseMessage).LeaseEnd >
				n.Next().Value.(*leaseMessage).LeaseEnd {
				t.Errorf("List out of order (%d).", i)
			}
		}
	}
}

// Tests that actionLeaseList._removeChannel removes all the messages from both
// the lease list and the message map for the given channel and that the lease
// list remains in the correct order after removal.
func Test_actionLeaseList__removeChannel(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	const m, n, o = 20, 5, 3
	expected := make([]*leaseMessage, 0, m*n*o)
	for i := 0; i < m; i++ {
		// Make multiple messages with same channel ID
		channelID := newRandomChanID(prng, t)

		for j := 0; j < n; j++ {
			// Make multiple messages with same payload (but different actions
			// and leases)
			payload := newRandomPayload(prng, t)

			for k := 0; k < o; k++ {
				lm := &leaseMessage{
					ChannelID: channelID,
					Action:    MessageType(k),
					Payload:   payload,
					LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
				}
				fp := newLeaseFingerprint(channelID, lm.Action, payload)
				err := all._addMessage(lm)
				if err != nil {
					t.Errorf("Failed to add message: %+v", err)
				}

				expected = append(expected, all.messages[*channelID][fp.key()])
			}
		}
	}

	// Check that the message map has all the expected messages
	for i, exp := range expected {
		fp := newLeaseFingerprint(exp.ChannelID, exp.Action, exp.Payload)
		if messages, exists := all.messages[*exp.ChannelID]; !exists {
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
	for channelID = range all.messages {
		break
	}

	err := all._removeChannel(&channelID)
	if err != nil {
		t.Errorf("Failed to remove channel: %+v", err)
	}

	for e := all.leases.Front(); e != nil && e.Next() != nil; e = e.Next() {
		// Check that the message does not exist in the list
		if e.Value.(*leaseMessage).ChannelID.Cmp(&channelID) {
			t.Errorf(
				"Found lease message from channel %s: %+v", channelID, e.Value)
		}
		if e.Value.(*leaseMessage).LeaseEnd >
			e.Next().Value.(*leaseMessage).LeaseEnd {
			t.Errorf("Lease list not in order.")
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
	all := newActionLeaseList(nil, kv)

	for i := 0; i < 10; i++ {
		channelID := newRandomChanID(prng, t)
		for j := 0; j < 5; j++ {
			timestamp := newRandomLeaseEnd(prng, t)
			lease := time.Minute
			lm := &leaseMessage{
				ChannelID: channelID,
				Action:    newRandomAction(prng, t),
				Payload:   newRandomPayload(prng, t),
				LeaseEnd:  timestamp.Add(lease).UnixNano(),
			}

			err := all._addMessage(lm)
			if err != nil {
				t.Errorf("Failed to add message: %+v", err)
			}
		}
	}

	// Create new list and load old contents into it
	loadedAll := newActionLeaseList(nil, kv)
	err := loadedAll.load()
	if err != nil {
		t.Errorf("Failed to load actionLeaseList from storage: %+v", err)
	}

	// Check that the loaded message map matches the original
	for chanID, messages := range all.messages {
		loadedMessages, exists := loadedAll.messages[chanID]
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
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))
	expectedErr := loadLeaseChanIDsErr

	err := all.load()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Failed to return expected error no channel ID list exists."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: Tests that actionLeaseList.load returns the expected error when
// no lease messages can be loaded from storage.
func Test_actionLeaseList_load_LeaseMessagesLoadError(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	all := newActionLeaseList(nil, kv)
	chanID := newRandomChanID(rand.New(rand.NewSource(32)), t)
	all.messages[*chanID] = make(map[leaseFingerprintKey]*leaseMessage)
	err := all.storeLeaseChannels()
	if err != nil {
		t.Errorf("Failed to store lease channels: %+v", err)
	}

	expectedErr := fmt.Sprintf(loadLeaseMessagesErr, chanID)

	err = all.load()
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
	all := newActionLeaseList(nil, kv)
	expectedIDs := make([]*id.ID, n)

	for i := 0; i < n; i++ {
		channelID := newRandomChanID(prng, t)
		all.messages[*channelID] = make(map[leaseFingerprintKey]*leaseMessage)
		for j := 0; j < 5; j++ {
			payload, action := newRandomPayload(prng, t), newRandomAction(prng, t)
			fp := newLeaseFingerprint(channelID, action, payload)
			all.messages[*channelID][fp.key()] = &leaseMessage{
				ChannelID: channelID,
				Action:    action,
				Payload:   payload,
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
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

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
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))
	channelID := newRandomChanID(prng, t)
	all.messages[*channelID] = make(map[leaseFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID: channelID,
			Action:    newRandomAction(prng, t),
			Payload:   newRandomPayload(prng, t),
			LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
		}
		fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		all.messages[*channelID][fp.key()] = lm
	}

	err := all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	loadedMessages, err := all.loadLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to load messages: %+v", err)
	}

	if !reflect.DeepEqual(all.messages[*channelID], loadedMessages) {
		t.Errorf("Loaded messages do not match original."+
			"\nexpected: %+v\nreceived: %+v",
			all.messages[*channelID], loadedMessages)
	}
}

// Tests that actionLeaseList.storeLeaseMessages deletes the lease message file
// from storage when the list is empty.
func Test_actionLeaseList_storeLeaseMessages_EmptyList(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))
	channelID := newRandomChanID(prng, t)
	all.messages[*channelID] = make(map[leaseFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID: channelID,
			Action:    newRandomAction(prng, t),
			Payload:   newRandomPayload(prng, t),
			LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
		}
		fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		all.messages[*channelID][fp.key()] = lm
	}

	err := all.storeLeaseMessages(channelID)
	if err != nil {
		t.Errorf("Failed to store messages: %+v", err)
	}

	all.messages[*channelID] = make(map[leaseFingerprintKey]*leaseMessage)
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
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))

	_, err := all.loadLeaseMessages(newRandomChanID(prng, t))
	if err == nil || all.kv.Exists(err) {
		t.Errorf("Failed to return expected error when nothing exists to load."+
			"\nexpected: %v\nreceived: %+v", os.ErrNotExist, err)
	}
}

// Tests that actionLeaseList.deleteLeaseMessages removes the lease messages
// from storage.
func Test_actionLeaseList_deleteLeaseMessages(t *testing.T) {
	prng := rand.New(rand.NewSource(32))
	all := newActionLeaseList(nil, versioned.NewKV(ekv.MakeMemstore()))
	channelID := newRandomChanID(prng, t)
	all.messages[*channelID] = make(map[leaseFingerprintKey]*leaseMessage)

	for i := 0; i < 15; i++ {
		lm := &leaseMessage{
			ChannelID: channelID,
			Action:    newRandomAction(prng, t),
			Payload:   newRandomPayload(prng, t),
			LeaseEnd:  newRandomLeaseEnd(prng, t).UnixNano(),
		}
		fp := newLeaseFingerprint(lm.ChannelID, lm.Action, lm.Payload)
		all.messages[*channelID][fp.key()] = lm
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
	channelID := newRandomChanID(prng, t)
	payload := []byte("payload")
	timestamp, lease := netTime.Now().Round(0), 6*time.Minute+30*time.Second
	nid1 := id.NewIdFromString("test01", id.Node, t)
	now := uint64(netTime.Now().UnixNano())
	ri := &mixmessages.RoundInfo{
		ID:        5,
		UpdateID:  1,
		State:     2,
		BatchSize: 150,
		Topology:  [][]byte{nid1.Bytes()},
		Timestamps: []uint64{now - 1000, now - 800, now - 600, now - 400,
			now - 200, now, now + 200},
		Errors: []*mixmessages.RoundError{{
			Id:     uint64(49),
			NodeId: nid1.Bytes(),
			Error:  "Test error",
		}},
		ResourceQueueTimeoutMillis: 0,
		AddressSpaceSize:           8,
	}
	lm := leaseMessage{
		ChannelID: channelID,
		MessageID: cryptoChannel.MakeMessageID(payload, channelID),
		Action:    newRandomAction(prng, t),
		Nickname:  "John",
		Payload:   payload,
		Timestamp: timestamp.UTC(),
		Lease:     lease,
		LeaseEnd:  timestamp.Add(lease).UnixNano(),
		Round:     rounds.MakeRound(ri),
		Status:    Delivered,
		FromAdmin: true,
		e:         nil,
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
		key := makeChannelLeaseMessagesKey(newRandomChanID(prng, t))

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
		"U/CjAdyKK79wyO4SFMq6Bf+h37gAqA6mp2qNO1ZWL2I=",
		"WIxQluAJ13lT+YpqJCx0u5GjduL9MB+ByRIaHezKyzo=",
		"o0whScIhzbA+ePEHZBNlq4mSmUrHPGrpeM7AzZhy/Us=",
		"Wq/1EuVsSIryFEeNnAXtStQrRHsadv+fixffeeO+wCU=",
		"n511MR4dhx4G16XivuR3ign9s3qU+Ayq4pDuzoXHqg0=",
		"hr7tF2vaLoKxRto4A31z/FjeqJVC7zMoyLZylMb6www=",
		"oV6kmT0ffG+GplPg8KMtCoBRUd120phQYUAqhsWXt+8=",
		"5CnNyI25TdA00deVNqXI0VAkchKtFF+4Dkxvi92LB3E=",
		"ZCyl4MoyK+yFNd9gmbSL3YNFQNZzdPx7zB/lzl4zwd8=",
		"2cV/KpakDNDHmgOXLJo8KTCoaY80n2H2gplLdZ3qX1c=",
		"EyQsAo1+ZGuAM+NYizt8NLpNm0i/1OzhTYs6E6pw7ec=",
		"xDwfdiI5Qs+iR9wye6FulKlBUToNyAGGHwCFMmGPjp4=",
		"+qQv435UETXr+NPT9oSIb8REqHU88x8stYad/uLlrVk=",
		"XdnLehSUjh55x/vevlyBr7rTl2VUzEsHS0M8Man35FI=",
		"bDL1K36g5RHXfeQNNe7na4YaM+1llLOl+LUuVHtBjYU=",
		"5JkB2bGQUHAeNxh03vegnvKClk7Vtw+DqgokZBZz7Og=",
	}

	for i, expected := range expectedFingerprints {
		fp := newLeaseFingerprint(newRandomChanID(prng, t),
			newRandomAction(prng, t), newRandomPayload(prng, t))

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
	chanIDs, payloads := make([]*id.ID, n), make([][]byte, n)
	for i := 0; i < n; i++ {
		chanIDs[i] = newRandomChanID(rng, t)
		payloads[i] = newRandomPayload(rng, t)

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

// newRandomChanID creates a new random channel id.ID for testing.
func newRandomChanID(rng io.Reader, t *testing.T) *id.ID {
	channelID, err := id.NewRandomID(rng, id.User)
	if err != nil {
		t.Fatalf("Failed to generate new channel ID: %+v", err)
	}

	return channelID
}

// newRandomPayload creates a new random payload for testing.
func newRandomPayload(rng io.Reader, t *testing.T) []byte {
	payload := make([]byte, 32)
	n, err := rng.Read(payload)
	if err != nil {
		t.Fatalf("Failed to generate new payload: %+v", err)
	} else if n != 32 {
		t.Fatalf(
			"Only generated %d bytes when %d bytes required for payload.", n, 32)
	}

	return payload
}

// newRandomAction creates a new random action MessageType for testing.
func newRandomAction(rng io.Reader, t *testing.T) MessageType {
	b := make([]byte, 4)
	n, err := rng.Read(b)
	if err != nil {
		t.Fatalf("Failed to generate new action bytes: %+v", err)
	} else if n != 4 {
		t.Fatalf(
			"Only generated %d bytes when %d bytes required for action.", n, 4)
	}

	num := binary.LittleEndian.Uint32(b)
	switch num % 3 {
	case 0:
		return Delete
	case 1:
		return Pinned
	case 2:
		return Mute
	}

	return 0
}

// newRandomLeaseEnd creates a new random action lease end for testing.
func newRandomLeaseEnd(rng io.Reader, t *testing.T) time.Time {
	b := make([]byte, 8)
	n, err := rng.Read(b)
	if err != nil {
		t.Fatalf("Failed to generate new lease time bytes: %+v", err)
	} else if n != 8 {
		t.Fatalf(
			"Only generated %d bytes when %d bytes required for lease.", n, 8)
	}

	lease := time.Duration(binary.LittleEndian.Uint64(b)%1000) * time.Minute

	return netTime.Now().Add(lease).Round(0)
}

// newRandomLease creates a new random lease duration end for testing.
func newRandomLease(rng io.Reader, t *testing.T) time.Duration {
	b := make([]byte, 8)
	n, err := rng.Read(b)
	if err != nil {
		t.Fatalf("Failed to generate new lease bytes: %+v", err)
	} else if n != 8 {
		t.Fatalf(
			"Only generated %d bytes when %d bytes required for lease.", n, 8)
	}

	return time.Duration(binary.LittleEndian.Uint64(b)%1000) * time.Minute
}
