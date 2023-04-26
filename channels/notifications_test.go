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
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/elixxir/ekv"
	primNotif "gitlab.com/elixxir/primitives/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Tests that newOrLoadNotifications returns a new notifications when none is
// saved and loads the expected notifications when one is saved.
func Test_newOrLoadNotifications(t *testing.T) {
	n := newOrLoadNotifications(
		makeEd25519PubKey(rand.New(rand.NewSource(42342)), t),
		nil, newMockNM(), versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient))
	expected := newNotifications(n.pubKey, nil, n.nm, n.kv, n.net)

	if !reflect.DeepEqual(expected, n) {
		t.Errorf("New notifications does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, n)
	}

	err := n.addChannel(id.NewIdFromString("channel", id.User, t))
	if err != nil {
		t.Fatalf("Failed to add new channel: %+v", err)
	}

	newN := newOrLoadNotifications(
		n.pubKey, nil, n.nm, n.kv, new(mockBroadcastClient))

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

	_ = newOrLoadNotifications(nil, nil, &mockNM{}, kv, new(mockBroadcastClient))
}

// Tests that newNotifications returns the expected new notifications object.
func Test_newNotifications(t *testing.T) {
	expected := &notifications{
		pubKey:   makeEd25519PubKey(rand.New(rand.NewSource(1219)), t),
		cb:       nil,
		channels: make(map[id.ID]NotificationLevel),
		kv:       versioned.NewKV(ekv.MakeMemstore()),
		nm:       newMockNM(),
		net:      new(mockBroadcastClient),
	}

	n := newNotifications(expected.pubKey, nil, expected.nm, expected.kv,
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
		nil, newMockNM(), versioned.NewKV(ekv.MakeMemstore()),
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

// Panic path: Tests that notifications.addChannel panics when adding a channel
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

// Tests that notifications.removeChannel removes the correct channel from the
// map and that channels with levels other than NotifyNone are unregistered.
func Test_notifications_removeChannel(t *testing.T) {
	me2e := newMockNM()
	n := newNotifications(makeEd25519PubKey(rand.New(rand.NewSource(7632)), t),
		func([]NotificationFilter) {}, me2e,
		versioned.NewKV(ekv.MakeMemstore()), new(mockBroadcastClient))

	channels := map[id.ID]NotificationLevel{
		*id.NewIdFromString("NotifyNone", id.User, t): NotifyNone,
		*id.NewIdFromString("NotifyPing", id.User, t): NotifyPing,
		*id.NewIdFromString("NotifyAll", id.User, t):  NotifyAll,
	}

	unregisterList := make(map[id.ID]struct{})
	for chanID, level := range channels {
		if err := n.addChannel(&chanID); err != nil {
			t.Errorf("Failed to add channel %s: %+v", chanID, err)
		}

		if level != NotifyNone {
			n.channels[chanID] = level
			unregisterList[chanID] = struct{}{}
		}
	}

	for chanID := range channels {
		n.removeChannel(&chanID)
		if level, exists := n.channels[chanID]; exists {
			t.Errorf("Channel %s with level %s not deleted", &chanID, level)
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

// Tests that notifications.removeChannel does not when trying to remove a
// channel that does not exist.
func Test_notifications_removeChannel_NoChannel(t *testing.T) {
	n := newNotifications(
		nil, nil, newMockNM(), versioned.NewKV(ekv.MakeMemstore()), nil)

	channels := map[id.ID]NotificationLevel{
		*id.NewIdFromString("NotifyNone", id.User, t): NotifyNone,
		*id.NewIdFromString("NotifyPing", id.User, t): NotifyNone,
		*id.NewIdFromString("NotifyAll", id.User, t):  NotifyNone,
	}

	for chanID := range channels {
		if err := n.addChannel(&chanID); err != nil {
			t.Errorf("Failed to add channel %s: %+v", chanID, err)
		}
	}

	n.removeChannel(id.NewIdFromString("NewChannel", id.User, t))

	if !reflect.DeepEqual(channels, n.channels) {
		t.Errorf("Channel list changed after removing channel that does not "+
			"exist.\nexpected: %+v\nreceived: %+v", channels, n.channels)
	}
}

// Tests that notifications.SetMobileNotificationsLevel sets the notification
// level in the map correctly and that is properly registers or unregisters
// the channel for notifications depending on the current and future level.
func Test_notifications_SetMobileNotificationsLevel(t *testing.T) {
	me2e := newMockNM()
	n := newNotifications(makeEd25519PubKey(rand.New(rand.NewSource(7632)), t),
		func([]NotificationFilter) {}, me2e,
		versioned.NewKV(ekv.MakeMemstore()), new(mockBroadcastClient))

	channels := map[id.ID]NotificationLevel{
		*id.NewIdFromString("NotifyNone", id.User, t): NotifyNone,
		*id.NewIdFromString("NotifyPing", id.User, t): NotifyPing,
		*id.NewIdFromString("NotifyAll", id.User, t):  NotifyAll,
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

// Error path: Tests that notifications.SetMobileNotificationsLevel returns an
// error when trying to modify a channel that does not exist.
func Test_notifications_SetMobileNotificationsLevel_NoChannelError(t *testing.T) {
	n := newNotifications(makeEd25519PubKey(rand.New(rand.NewSource(7632)), t),
		func([]NotificationFilter) {}, newMockNM(),
		versioned.NewKV(ekv.MakeMemstore()), new(mockBroadcastClient))

	err := n.SetMobileNotificationsLevel(
		id.NewIdFromString("NewChannel", id.User, t), NotifyNone)
	if err == nil || err != ChannelDoesNotExistsErr {
		t.Errorf("Did not return expected error when trying to set level of "+
			"channel that does not exist.\nexpected: %v\nreceived: %v",
			ChannelDoesNotExistsErr, err)
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
// For Me / Notification Report                                               //
////////////////////////////////////////////////////////////////////////////////

func TestGetNotificationReportsForMe(t *testing.T) {
	rng := rand.New(rand.NewSource(6584))
	types := []MessageType{Text, AdminText, Reaction, Delete, Pinned, Mute,
		AdminReplay, FileTransfer}
	levels := []NotificationLevel{NotifyPing, NotifyAll}

	var expected []NotificationReport
	var notifData []*primNotif.Data
	var nfs []NotificationFilter
	for _, mt := range types {
		for _, level := range levels {
			for _, includeTags := range []bool{true, false} {
				for _, includeChannel := range []bool{true, false} {
					chanID, _ := id.NewRandomID(rng, id.User)
					msgHash := make([]byte, 24)
					rng.Read(msgHash)
					identifier := append(chanID.Marshal(), []byte("identifier")...)
					tags := make([]string, 1+rng.Intn(4))
					for j := range tags {
						tags[j] = makeUserPingTag(makeEd25519PubKey(rng, t))
					}

					cSIH, err := sih.MakeCompressedSIH(
						chanID, msgHash, identifier, tags, mt.Marshal())
					if err != nil {
						t.Fatalf("Failed to make compressed SIH: %+v", err)
					}

					notifData = append(notifData, &primNotif.Data{
						IdentityFP:  cSIH,
						MessageHash: msgHash,
					})

					if includeChannel {
						var filterTags []string
						if includeTags {
							filterTags = []string{tags[rng.Intn(len(tags))]}
						}
						nfs = append(nfs, NotificationFilter{
							Identifier: identifier,
							ChannelID:  chanID,
							Tags:       filterTags,
							AllowLists: notificationLevelAllowLists[level],
						})

						if includeTags {
							if _, exists := notificationLevelAllowLists[level].AllowWithTags[mt]; !exists {
								break
							}
						} else if _, exists := notificationLevelAllowLists[level].AllowWithoutTags[mt]; !exists {
							break
						}

						expected = append(expected, NotificationReport{
							Channel: chanID,
							Type:    mt,
						})
					}
				}
			}
		}
	}

	nrs := GetNotificationReportsForMe(nfs, notifData)

	if !reflect.DeepEqual(nrs, expected) {
		t.Errorf("NotificationReport list does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, nrs)
	}

}

// Tests that a NotificationReport can be JSON marshalled and unmarshalled.
func TestNotificationReport_JSON(t *testing.T) {
	nr := NotificationReport{
		Channel: id.NewIdFromString("someChannel", id.User, t),
		Type:    Text,
	}

	data, err := json.Marshal(nr)
	if err != nil {
		t.Fatalf("Failed to JSON marshal %T: %+v", nr, err)
	}

	var newNr NotificationReport
	if err = json.Unmarshal(data, &newNr); err != nil {
		t.Fatalf("Failed to JSON unmarshal %T: %+v", newNr, err)
	}

	if !reflect.DeepEqual(nr, newNr) {
		t.Errorf("JSON marshalled and unmarshalled NotificationReport does "+
			"not match original.\nexpected: %+v\nreceivedL %+v", nr, newNr)
	}
}

// Tests that a slice of NotificationReport can be JSON marshalled and
// unmarshalled.
func TestNotificationReport_Slice_JSON(t *testing.T) {
	rng := rand.New(rand.NewSource(7632))

	nrs := make([]NotificationReport, 9)
	types := []MessageType{Text, AdminText, Reaction, Delete, Pinned, Mute,
		AdminReplay, FileTransfer}
	for i := range nrs {
		chanID, _ := id.NewRandomID(rng, id.User)
		nrs[i] = NotificationReport{
			Channel: chanID,
			Type:    types[i%len(types)],
		}
	}

	// data, err := json.MarshalIndent(nrs, "//  ", "  ")
	// if err != nil {
	// 	t.Fatalf("Failed to JSON marshal %T: %+v", nrs, err)
	// }
	// fmt.Printf("\n\n//  %s\n\n", data)

	data, err := json.Marshal(nrs)
	if err != nil {
		t.Fatalf("Failed to JSON marshal %T: %+v", nrs, err)
	}

	var newNfr []NotificationReport
	if err = json.Unmarshal(data, &newNfr); err != nil {
		t.Fatalf("Failed to JSON unmarshal %T: %+v", newNfr, err)
	}

	if !reflect.DeepEqual(nrs, newNfr) {
		t.Errorf("JSON marshalled and unmarshalled []NotificationReport does "+
			"not match original.\nexpected: %+v\nreceivedL %+v", nrs, newNfr)
	}
}

////////////////////////////////////////////////////////////////////////////////
// MessageTypeFilter                                                          //
////////////////////////////////////////////////////////////////////////////////

// Tests that a NotificationFilter can be JSON marshalled and unmarshalled.
func TestNotificationFilter_JSON(t *testing.T) {
	rng := rand.New(rand.NewSource(7632))
	chanID := id.NewIdFromString("someChannel", id.User, t)
	nf := NotificationFilter{
		Identifier: append(chanID.Marshal(), []byte("Identifier")...),
		ChannelID:  chanID,
		Tags:       makeUserPingTags(makeEd25519PubKey(rng, t)),
		AllowLists: notificationLevelAllowLists[NotifyPing],
	}

	data, err := json.Marshal(nf)
	if err != nil {
		t.Fatalf("Failed to JSON marshal %T: %+v", nf, err)
	}

	var newNf NotificationFilter
	if err = json.Unmarshal(data, &newNf); err != nil {
		t.Fatalf("Failed to JSON unmarshal %T: %+v", newNf, err)
	}

	if !reflect.DeepEqual(nf, newNf) {
		t.Errorf("JSON marshalled and unmarshalled NotificationFilter does "+
			"not match original.\nexpected: %+v\nreceivedL %+v", nf, newNf)
	}
}

// Tests that a slice of NotificationFilter can be JSON marshalled and
// unmarshalled.
func TestNotificationFilter_Slice_JSON(t *testing.T) {
	rng := rand.New(rand.NewSource(7632))

	nfs := make([]NotificationFilter, 3)
	levels := []NotificationLevel{NotifyPing, NotifyAll, NotifyPing}
	for i := range nfs {
		chanID, _ := id.NewRandomID(rng, id.User)
		nfs[i] = NotificationFilter{
			Identifier: append(chanID.Marshal(), []byte("Identifier")...),
			ChannelID:  chanID,
			Tags:       makeUserPingTags(makeEd25519PubKey(rng, t)),
			AllowLists: notificationLevelAllowLists[levels[i%len(levels)]],
		}
	}

	// data, err := json.MarshalIndent(nfs, "//  ", "  ")
	// if err != nil {
	// 	t.Fatalf("Failed to JSON marshal %T: %+v", nfs, err)
	// }
	// fmt.Printf("\n\n//  %s\n\n", data)

	data, err := json.Marshal(nfs)
	if err != nil {
		t.Fatalf("Failed to JSON marshal %T: %+v", nfs, err)
	}

	var newNfs []NotificationFilter
	if err = json.Unmarshal(data, &newNfs); err != nil {
		t.Fatalf("Failed to JSON unmarshal %T: %+v", newNfs, err)
	}

	if !reflect.DeepEqual(nfs, newNfs) {
		t.Errorf("JSON marshalled and unmarshalled []NotificationFilter does "+
			"not match original.\nexpected: %+v\nreceivedL %+v", nfs, newNfs)
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
