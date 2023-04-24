////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

// Tests that newOrLoadNotifications returns a new notifications when none is
// saved and loads the expected notifications when one is saved.
func Test_newOrLoadNotifications(t *testing.T) {
	n := newOrLoadNotifications(
		makeEd25519PubKey(rand.New(rand.NewSource(42342)), t),
		nil, newMockE2e(),
		versioned.NewKV(ekv.MakeMemstore()), new(mockBroadcastClient))
	expected := newNotifications(n.pubKey, nil, n.user, n.kv, n.net)

	if !reflect.DeepEqual(expected, n) {
		t.Errorf("New notifications does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, n)
	}

	err := n.addChannel(id.NewIdFromString("channel", id.User, t))
	if err != nil {
		t.Fatalf("Failed to add new channel: %+v", err)
	}

	newN := newOrLoadNotifications(
		n.pubKey, nil, n.user, n.kv, new(mockBroadcastClient))

	if !reflect.DeepEqual(n, newN) {
		t.Errorf("Loaded notifications does not match new."+
			"\nexpected: %+v\nreceived: %+v", n, newN)
	}
}

// Panic path: Tests that newOrLoadNotifications panics when trying to load
// invalid data.
func Test_newOrLoadNotifications_LoadInvalidDataPanic(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	err := kv.Set(notificationsKvKey, &versioned.Object{
		Version:   notificationsKvVersion,
		Timestamp: netTime.Now(),
		Data:      []byte("invalid data"),
	})
	if err != nil {
		t.Fatalf("Failed to save invalid data: %+v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Failed to panic when loading invalid data.")
		}
	}()

	_ = newOrLoadNotifications(nil, nil, mockE2e{}, kv, new(mockBroadcastClient))
}

// Tests that newNotifications returns the expected new notifications object.
func Test_newNotifications(t *testing.T) {
	expected := &notifications{
		pubKey:   makeEd25519PubKey(rand.New(rand.NewSource(1219)), t),
		cb:       nil,
		channels: make(map[id.ID]NotificationLevel),
		user:     newMockE2e(),
		net:      new(mockBroadcastClient),
		kv:       versioned.NewKV(ekv.MakeMemstore()),
	}

	n := newNotifications(expected.pubKey, nil, expected.user, expected.kv,
		new(mockBroadcastClient))

	if !reflect.DeepEqual(expected, n) {
		t.Errorf("New notifications does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, n)
	}
}

// Tests that notifications.addChannel adds all the expected channels with the
// level NotifyNone.
func Test_notifications_addChannel(t *testing.T) {
	n := newNotifications(makeEd25519PubKey(rand.New(rand.NewSource(7632)), t),
		nil, newMockE2e(), versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient))

	expected := map[id.ID]NotificationLevel{
		*id.NewIdFromString("channel1", id.User, t): NotifyNone,
		*id.NewIdFromString("channel2", id.User, t): NotifyNone,
		*id.NewIdFromString("channel3", id.User, t): NotifyNone,
	}

	for chanID := range expected {
		err := n.addChannel(&chanID)
		if err != nil {
			t.Errorf("Failed to add channel %s: %+v", chanID, err)
		}
	}
	if !reflect.DeepEqual(expected, n.channels) {
		t.Errorf("Notifications did not add expected channels."+
			"\nexpected: %+v\nreceived: %+v", expected, n.channels)
	}
}

// Panic path" Tests that notifications.addChannel panics when adding a channel
// that already exists.
func Test_notifications_addChannel_AddExistingChannelPanic(t *testing.T) {
	n := newNotifications(
		nil, nil, nil, versioned.NewKV(ekv.MakeMemstore()), nil)

	chanID := id.NewIdFromString("channel1", id.User, t)
	err := n.addChannel(chanID)
	if err != nil {
		t.Errorf("Failed to add channel %s: %+v", chanID, err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Failed to panic when adding channel that already exists.")
		}
	}()

	err = n.addChannel(chanID)
	if err != nil {
		t.Errorf("Failed to add channel %s: %+v", chanID, err)
	}
}

// TODO
func Test_notifications_removeChannel(t *testing.T) {
	n := newNotifications(
		nil, nil, newMockE2e(), versioned.NewKV(ekv.MakeMemstore()), nil)

	n.channels = map[id.ID]NotificationLevel{
		*id.NewIdFromString("channel1", id.User, t): NotifyNone,
		*id.NewIdFromString("channel2", id.User, t): NotifyPing,
		*id.NewIdFromString("channel3", id.User, t): NotifyAll,
	}

	for chanID := range n.channels {
		n.removeChannel(&chanID)
	}

}

// Tests that notifications.SetMobileNotificationsLevel sets the notification
// level in the map correctly and that is properly registers or unregisters
// the channel for notifications depending on the current and future level.
func Test_notifications_SetMobileNotificationsLevel(t *testing.T) {
	me2e := newMockE2e()
	n := newNotifications(makeEd25519PubKey(rand.New(rand.NewSource(7632)), t),
		func([]NotificationFilter) {}, me2e,
		versioned.NewKV(ekv.MakeMemstore()), new(mockBroadcastClient))

	channels := map[id.ID]NotificationLevel{
		*id.NewIdFromBase64String("NotifyNone", id.User, t): NotifyNone,
		*id.NewIdFromBase64String("NotifyPing", id.User, t): NotifyPing,
		*id.NewIdFromBase64String("NotifyAll", id.User, t):  NotifyAll,
	}

	registerList := make(map[id.ID]struct{})
	unregisterList := make(map[id.ID]struct{})
	for chanID, level := range channels {
		if err := n.addChannel(&chanID); err != nil {
			t.Errorf("Failed to add channel %s: %+v", chanID, err)
		}

		switch level {
		case NotifyNone:
			n.channels[chanID] = NotifyPing
			unregisterList[chanID] = struct{}{}
		case NotifyAll, NotifyPing:
			registerList[chanID] = struct{}{}
		}
	}

	for chanID, level := range channels {
		err := n.SetMobileNotificationsLevel(&chanID, level)
		if err != nil {
			t.Errorf("Failed to set level for channel %s: %+v", chanID, err)
		}
	}

	for chanID, level := range channels {
		if n.channels[chanID] != level {
			t.Errorf("Wrong level for channel %s.\nexpected: %s\nreceived: %s",
				&chanID, level, n.channels[chanID])
		}
	}

	for len(registerList) != 0 {
		select {
		case chanID := <-me2e.registeredIDs:
			if _, exists := registerList[chanID]; !exists {
				t.Errorf("Channel %s not expected to be registered.", chanID)
			} else {
				delete(registerList, chanID)
			}
		case <-time.After(15 * time.Millisecond):
			t.Fatal("Timed out waiting for registered IDs")
		}
	}

	for len(unregisterList) != 0 {
		select {
		case chanID := <-me2e.unregisteredIDs:
			if _, exists := unregisterList[chanID]; !exists {
				t.Errorf("Channel %s not expected to be unregistered.", chanID)
			} else {
				delete(unregisterList, chanID)
			}
		case <-time.After(15 * time.Millisecond):
			t.Fatal("Timed out waiting for unregistered IDs")
		}
	}
}

// Tests that notifications.createFilterList creates the expected filter list
// from the generated CompressedServiceList.
func Test_notifications_createFilterList(t *testing.T) {
	n := &notifications{
		channels: map[id.ID]NotificationLevel{
			*id.NewIdFromUInt(1, id.User, t): NotifyNone,
			*id.NewIdFromUInt(2, id.User, t): NotifyPing,
			*id.NewIdFromUInt(3, id.User, t): NotifyAll,
		},
		pubKey: makeEd25519PubKey(rand.New(rand.NewSource(42342)), t),
	}

	csl := make(message.CompressedServiceList, len(n.channels))
	ex := make([]NotificationFilter, 0, len(n.channels))
	for chanId, level := range n.channels {
		csl[chanId] = []message.CompressedService{
			{Identifier: []byte("Identifier for " + chanId.String())}}
		if level != NotifyNone {
			ex = append(ex, NotificationFilter{
				Identifier: csl[chanId][0].Identifier,
				ChannelID:  &chanId,
				Tags:       makeUserPingTags(n.pubKey),
				AllowLists: notificationLevelAllowLists[level],
			})
		}
	}

	nf := n.createFilterList(csl)

	sort.Slice(ex, func(i, j int) bool {
		return bytes.Compare(ex[i].ChannelID[:], ex[j].ChannelID[:]) == -1
	})
	sort.Slice(nf, func(i, j int) bool {
		return bytes.Compare(nf[i].ChannelID[:], nf[j].ChannelID[:]) == -1
	})

	if !reflect.DeepEqual(ex, nf) {
		t.Errorf("Unexpected filter list."+
			"\nexpected: %+v\nreceived: %+v", ex, nf)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage                                                                    //
////////////////////////////////////////////////////////////////////////////////

// Tests that a notifications that is saved and loaded to the KV matches the
// original.
func Test_notifications_save(t *testing.T) {
	n := &notifications{
		channels: map[id.ID]NotificationLevel{
			*id.NewIdFromString("channel1", id.User, t): NotifyNone,
			*id.NewIdFromString("channel2", id.User, t): NotifyPing,
			*id.NewIdFromString("channel3", id.User, t): NotifyAll,
		},
		kv: versioned.NewKV(ekv.MakeMemstore()),
	}

	err := n.save()
	if err != nil {
		t.Fatalf("Failed to save: %+v", err)
	}

	m := &notifications{kv: n.kv}
	err = m.load()
	if err != nil {
		t.Fatalf("Failed to load: %+v", err)
	}

	if !reflect.DeepEqual(n, m) {
		t.Errorf("Saved and loaded notifications does not match "+
			"original.\nexpected: %+v\nreceived: %+v", n, m)
	}
}

// Tests that a notifications can be JSON marshalled and unmarshalled.
func Test_notifications_MarshalJSON_UnmarshalJSON(t *testing.T) {
	n := &notifications{
		channels: map[id.ID]NotificationLevel{
			*id.NewIdFromString("channel1", id.User, t): NotifyNone,
			*id.NewIdFromString("channel2", id.User, t): NotifyPing,
			*id.NewIdFromString("channel3", id.User, t): NotifyAll,
		},
	}

	data, err := json.Marshal(n)
	if err != nil {
		t.Errorf("Failed to JSON marshal %T: %+v", n, err)
	}

	m := &notifications{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal %T: %+v", m, err)
	}

	if !reflect.DeepEqual(n, m) {
		t.Errorf("Marshalled and unmarshalled notifications does not match "+
			"original.\nexpected: %+v\nreceived: %+v", n, m)
	}
}

////////////////////////////////////////////////////////////////////////////////
// NotificationLevel                                                          //
////////////////////////////////////////////////////////////////////////////////

// Constancy test of NotificationLevel.String.
func TestNotificationLevel_String(t *testing.T) {
	tests := map[NotificationLevel]string{
		NotifyNone: "none",
		NotifyPing: "ping",
		NotifyAll:  "all",
		32:         "INVALID NOTIFICATION LEVEL: 32",
	}

	for l, expected := range tests {
		s := l.String()
		if s != expected {
			t.Errorf("Incorrect string for NotificationLevel %d."+
				"\nexpected: %s\nreceived: %s", l, expected, l)
		}
	}
}
