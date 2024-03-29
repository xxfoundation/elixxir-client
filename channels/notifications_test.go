////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/broadcast"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/crypto/sih"
	primNotif "gitlab.com/elixxir/primitives/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// Tests that newNotifications returns the expected new notifications object.
func Test_newNotifications(t *testing.T) {
	nm := newMockNM()
	expected := &notifications{
		pubKey:        makeEd25519PubKey(rand.New(rand.NewSource(1219)), t),
		cb:            nil,
		channelGetter: newMockCG(0, t),
		ext:           []ExtensionMessageHandler{newMockNotifExtension()},
		nm:            nm,
	}

	n := newNotifications(
		expected.pubKey, nil, newMockCG(0, t), expected.ext, nm)

	if !reflect.DeepEqual(expected, n) {
		t.Errorf("New notifications does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, n)
	}
}

// Tests that notifications.addChannel adds all the expected channels with the
// level NotifyNone.
func Test_notifications_addChannel(t *testing.T) {
	nm := newMockNM()
	n := notifications{nil, nil, nil, nil, nm}

	expected := clientNotif.Group{
		*id.NewIdFromString("channel1", id.User, t): {NotifyNone.Marshal(), clientNotif.Mute},
		*id.NewIdFromString("channel2", id.User, t): {NotifyNone.Marshal(), clientNotif.Mute},
		*id.NewIdFromString("channel3", id.User, t): {NotifyNone.Marshal(), clientNotif.Mute},
	}

	for chanID := range expected {
		n.addChannel(&chanID)
	}
	if !reflect.DeepEqual(expected, nm.channels[notificationGroup]) {
		t.Errorf("Notifications did not add expected channels."+
			"\nexpected: %+v\nreceived: %+v",
			expected, nm.channels[notificationGroup])
	}
}

// Tests that notifications.removeChannel removes the correct channel from the
// notification manager.
func Test_notifications_removeChannel(t *testing.T) {
	nm := newMockNM()
	n := notifications{nil, nil, nil, nil, nm}

	channels := map[id.ID]NotificationLevel{
		*id.NewIdFromString("NotifyNone", id.User, t): NotifyNone,
		*id.NewIdFromString("NotifyPing", id.User, t): NotifyPing,
		*id.NewIdFromString("NotifyAll", id.User, t):  NotifyAll,
	}

	for chanID := range channels {
		n.addChannel(&chanID)
	}

	for chanID := range channels {
		n.removeChannel(&chanID)

		if ni, exists := nm.channels[notificationGroup][chanID]; exists {
			t.Errorf("Channel %s with level %s not deleted",
				&chanID, UnmarshalNotificationLevel(ni.Metadata))
		}
	}
}

// Tests that notifications.GetNotificationLevel returns the correct
// NotificationLevel and that notifications.GetNotificationStatus return the
// correct notifications.NotificationState for all added IDs.
func TestNotifications_GetNotificationLevel_GetNotificationStatus(t *testing.T) {
	nm := newMockNM()
	n := notifications{nil, nil, nil, nil, nm}

	expected := map[id.ID]NotificationState{
		*id.NewIdFromString("channel1", id.User, t): {nil, NotifyNone, clientNotif.Mute},
		*id.NewIdFromString("channel2", id.User, t): {nil, NotifyPing, clientNotif.Push},
		*id.NewIdFromString("channel3", id.User, t): {nil, NotifyAll, clientNotif.WhenOpen},
	}

	for chanID, state := range expected {
		err := n.SetMobileNotificationsLevel(&chanID, state.Level, state.Status)
		if err != nil {
			t.Errorf("Failed to set level for channel %s: %+v", &chanID, err)
		}
	}

	for chanID, state := range expected {
		l, err := n.GetNotificationLevel(&chanID)
		if err != nil {
			t.Errorf("Failed to get notification level for %s: %+v", &chanID, err)
		}

		if state.Level != l {
			t.Errorf("Incorrect level for %s.\nexpected: %s\nreceived: %s",
				&chanID, state.Level, l)
		}

		s, err := n.GetNotificationStatus(&chanID)
		if err != nil {
			t.Errorf("Failed to get notification status for %s: %+v", &chanID, err)
		}

		if state.Status != s {
			t.Errorf("Incorrect status for %s.\nexpected: %s\nreceived: %s",
				&chanID, state.Status, s)
		}
	}

	_, err := n.GetNotificationLevel(id.NewIdFromString("chan", id.User, t))
	if err == nil {
		t.Errorf("Did not get error when getting level for a channel that " +
			"does not exist.")
	}
}

// Error path: Tests that notifications.GetNotificationLevel returns an error
// when no channel exists.
func TestNotifications_GetNotificationLevel_NoChannelError(t *testing.T) {
	n := notifications{nil, nil, nil, nil, newMockNM()}
	_, err := n.GetNotificationLevel(id.NewIdFromString("chan", id.User, t))
	if err == nil {
		t.Errorf("Did not get error when getting level for a channel that " +
			"does not exist.")
	}
}

// Error path: Tests that notifications.GetNotificationStatus returns an error
// when no channel exists.
func TestNotifications_GetNotificationStatus_NoChannelError(t *testing.T) {
	n := notifications{nil, nil, nil, nil, newMockNM()}
	_, err := n.GetNotificationStatus(id.NewIdFromString("chan", id.User, t))
	if err == nil {
		t.Errorf("Did not get error when getting state for a channel that " +
			"does not exist.")
	}
}

// Tests that notifications.SetMobileNotificationsLevel sets the notification
// level and status correctly.
func Test_notifications_SetMobileNotificationsLevel(t *testing.T) {
	nm := newMockNM()
	n := notifications{nil, nil, nil, nil, nm}

	expected := clientNotif.Group{
		*id.NewIdFromString("channel1", id.User, t): {NotifyNone.Marshal(), clientNotif.Mute},
		*id.NewIdFromString("channel2", id.User, t): {NotifyPing.Marshal(), clientNotif.Push},
		*id.NewIdFromString("channel3", id.User, t): {NotifyAll.Marshal(), clientNotif.WhenOpen},
	}

	for chanID := range expected {
		n.addChannel(&chanID)
	}

	for chanID, ni := range expected {
		err := n.SetMobileNotificationsLevel(&chanID, UnmarshalNotificationLevel(ni.Metadata), ni.Status)
		if err != nil {
			t.Errorf("Failed to add channel %s: %+v", &chanID, err)
		}
	}

	if !reflect.DeepEqual(expected, nm.channels[notificationGroup]) {
		t.Errorf("Notifications did not add expected channels."+
			"\nexpected: %+v\nreceived: %+v",
			expected, nm.channels[notificationGroup])
	}
}

// Error Path: Tests that notifications.SetMobileNotificationsLevel returns an
// error when the level is not NotifyNone and the status notifications.Mute at
// the same time.
func Test_notifications_SetMobileNotificationsLevel_StatusStateError(t *testing.T) {
	nm := newMockNM()
	n := notifications{nil, nil, nil, nil, nm}

	expected := clientNotif.Group{
		*id.NewIdFromString("channel1", id.User, t): {NotifyNone.Marshal(), clientNotif.WhenOpen},
		*id.NewIdFromString("channel2", id.User, t): {NotifyNone.Marshal(), clientNotif.Push},
		*id.NewIdFromString("channel3", id.User, t): {NotifyPing.Marshal(), clientNotif.Mute},
		*id.NewIdFromString("channel4", id.User, t): {NotifyAll.Marshal(), clientNotif.Mute},
	}
	for chanID, ni := range expected {
		l := UnmarshalNotificationLevel(ni.Metadata)
		err := n.SetMobileNotificationsLevel(&chanID, l, ni.Status)
		if err == nil {
			t.Errorf("Did not receive an error for an incompatible level (%s) "+
				"and state (%s)", l, ni.Status)
		}
	}
}

// Tests that notifications.getChannelStatuses properly processes the changed
// and edited channels into the NotificationState list.
func Test_notifications_getChannelStatuses(t *testing.T) {
	rng := rand.New(rand.NewSource(2323))
	cg, nm := newMockCG(12, t), newMockNM()
	n := notifications{makeEd25519PubKey(rng, t), nil, cg, nil, nm}

	nim := make(clientNotif.Group, len(cg.channels))
	var created, edits []*id.ID
	var expectedChanged []NotificationState
	levels := []NotificationLevel{NotifyNone, NotifyPing, NotifyAll}
	for chanId := range cg.channels {
		channelID := chanId.DeepCopy()
		level := levels[rng.Intn(len(levels))]
		state := clientNotif.Mute
		if level != NotifyNone {
			state = clientNotif.Push
		}
		nim[*channelID] = clientNotif.State{
			Metadata: level.Marshal(),
			Status:   state,
		}
		if rng.Intn(2) == 0 {
			edits = append(edits, channelID)
		} else {
			created = append(created, channelID)
		}
		expectedChanged = append(expectedChanged, NotificationState{
			ChannelID: channelID,
			Level:     level,
			Status:    state,
		})
	}

	// Add some channels that are not in the manager
	for i := 0; i < 2; i++ {
		channelID, _ := id.NewRandomID(rng, id.User)
		level := levels[rng.Intn(len(levels))]
		state := clientNotif.Mute
		if level != NotifyNone {
			state = clientNotif.Push
		}
		nim[*channelID] = clientNotif.State{
			Metadata: level.Marshal(),
			Status:   state,
		}
	}

	changed := n.getChannelStatuses(nim, created, edits)

	sort.SliceStable(expectedChanged, func(i, j int) bool {
		return bytes.Compare(expectedChanged[i].ChannelID[:],
			expectedChanged[j].ChannelID[:]) == -1
	})
	sort.SliceStable(changed, func(i, j int) bool {
		return bytes.Compare(changed[i].ChannelID[:],
			changed[j].ChannelID[:]) == -1
	})

	if !reflect.DeepEqual(expectedChanged, changed) {
		t.Errorf("Unexpected changed list."+
			"\nexpected: %+v\nreceived: %+v", expectedChanged, changed)
	}
}

// Tests that notifications.processesNotificationUpdates creates the expected
// filter list from the generated map of notification info.
func Test_notifications_processesNotificationUpdates(t *testing.T) {
	rng := rand.New(rand.NewSource(2323))
	cg, nm := newMockCG(5, t), newMockNM()
	ext := []ExtensionMessageHandler{newMockNotifExtension()}
	n := notifications{makeEd25519PubKey(rng, t), nil, cg, ext, nm}

	nim := make(clientNotif.Group, len(cg.channels))
	created := map[id.ID]struct{}{}
	var expectedChanged []NotificationState
	ex := make([]NotificationFilter, 0, len(cg.channels))
	levels := []NotificationLevel{NotifyNone, NotifyPing, NotifyAll}
	for chanId, ch := range cg.channels {
		channelID := chanId.DeepCopy()
		level := levels[rng.Intn(len(levels))]
		state := clientNotif.Mute
		if level != NotifyNone {
			state = clientNotif.Push
		}
		nim[*channelID] = clientNotif.State{
			Metadata: level.Marshal(),
			Status:   state,
		}
		created[*channelID] = struct{}{}
		expectedChanged = append(expectedChanged, NotificationState{
			ChannelID: channelID,
			Level:     level,
			Status:    state,
		})

		if level != NotifyNone {
			tags := makeUserPingTags(map[PingType][]ed25519.PublicKey{
				ReplyPing: {n.pubKey}, MentionPing: {n.pubKey}})
			sort.Strings(tags)
			ex = append(ex,
				NotificationFilter{
					Identifier: ch.broadcast.AsymmetricIdentifier(),
					ChannelID:  channelID,
					Tags:       tags,
					AllowLists: notificationLevelAllowLists[asymmetric][level],
				},
				NotificationFilter{
					Identifier: ch.broadcast.SymmetricIdentifier(),
					ChannelID:  channelID,
					Tags:       tags,
					AllowLists: notificationLevelAllowLists[symmetric][level],
				})
		}
	}

	// Add some channels that are not in the manager
	for i := 0; i < 2; i++ {
		channelID, _ := id.NewRandomID(rng, id.User)
		level := levels[rng.Intn(len(levels))]
		state := clientNotif.Mute
		if level != NotifyNone {
			state = clientNotif.Push
		}
		nim[*channelID] = clientNotif.State{
			Metadata: level.Marshal(),
			Status:   state,
		}
	}

	nf, changed := n.processesNotificationUpdates(nim, created, nil)

	sort.Slice(ex, func(i, j int) bool {
		return bytes.Compare(ex[i].Identifier, ex[j].Identifier) == -1
	})
	sort.Slice(nf, func(i, j int) bool {
		return bytes.Compare(nf[i].Identifier, nf[j].Identifier) == -1
	})

	for i := range nf {
		sort.Strings(nf[i].Tags)
	}

	if !reflect.DeepEqual(ex, nf) {
		t.Errorf("Unexpected filter list."+
			"\nexpected: %+v\nreceived: %+v", ex, nf)
	}

	sort.Slice(expectedChanged, func(i, j int) bool {
		return bytes.Compare(expectedChanged[i].ChannelID[:], expectedChanged[j].ChannelID[:]) == -1
	})
	sort.Slice(changed, func(i, j int) bool {
		return bytes.Compare(changed[i].ChannelID[:], changed[j].ChannelID[:]) == -1
	})

	if !reflect.DeepEqual(expectedChanged, changed) {
		t.Errorf("Unexpected changed list."+
			"\nexpected: %+v\nreceived: %+v", expectedChanged, changed)
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
	pingTypes := []PingType{ReplyPing, MentionPing}

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
						tags[j] = makeUserPingTag(makeEd25519PubKey(rng, t),
							pingTypes[rng.Intn(len(pingTypes))])
					}

					mtByte := mt.Marshal()
					cSIH, err := sih.MakeCompressedSIH(
						chanID, msgHash, identifier, tags, mtByte[:])
					if err != nil {
						t.Fatalf("Failed to make compressed SIH: %+v", err)
					}

					notifData = append(notifData, &primNotif.Data{
						IdentityFP:  cSIH,
						MessageHash: msgHash,
					})

					if includeChannel {
						var filterTags []string
						var pt PingType
						if includeTags {
							filterTags = []string{tags[rng.Intn(len(tags))]}
							pt, err = pingTypeFromTag(filterTags[0])
							if err != nil {
								t.Errorf("Failed to get Ping Type: %+v", err)
							}
						}
						nfs = append(nfs, NotificationFilter{
							Identifier: identifier,
							ChannelID:  chanID,
							Tags:       filterTags,
							AllowLists: notificationLevelAllowLists[symmetric][level],
						})

						if includeTags {
							if _, exists := notificationLevelAllowLists[symmetric][level].AllowWithTags[mt]; !exists {
								break
							}
						} else if _, exists := notificationLevelAllowLists[symmetric][level].AllowWithoutTags[mt]; !exists {
							break
						}

						expected = append(expected, NotificationReport{
							Channel:  chanID,
							Type:     mt,
							PingType: pt,
						})
					}
				}
			}
		}
	}

	nrs := GetNotificationReportsForMe(nfs, notifData)

	data, _ := json.MarshalIndent(nrs, "//  ", "  ")
	fmt.Printf("//  %s\n", data)

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
		Tags: makeUserPingTags(map[PingType][]ed25519.PublicKey{
			MentionPing: {makeEd25519PubKey(rng, t)}}),
		AllowLists: notificationLevelAllowLists[symmetric][NotifyPing],
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
	sourceTypes := []notificationSourceType{symmetric, asymmetric}
	for i := range nfs {
		chanID, _ := id.NewRandomID(rng, id.User)
		nfs[i] = NotificationFilter{
			Identifier: append(chanID.Marshal(), []byte("Identifier")...),
			ChannelID:  chanID,
			Tags: makeUserPingTags(map[PingType][]ed25519.PublicKey{
				MentionPing: {makeEd25519PubKey(rng, t)}}),
			AllowLists: notificationLevelAllowLists[sourceTypes[i%len(sourceTypes)]][levels[i%len(levels)]],
		}
	}

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

// Consistency test of NotificationLevel.String.
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

// Tests that a NotificationLevel marshaled via NotificationLevel and
// unmarshalled via UnmarshalNotificationLevel match the original.
func TestNotificationLevel_Marshal_UnmarshalMessageType(t *testing.T) {
	tests := []NotificationLevel{NotifyNone, NotifyPing, NotifyAll}

	for _, l := range tests {
		data := l.Marshal()
		newL := UnmarshalNotificationLevel(data)

		if l != newL {
			t.Errorf("Failed to marshal and unmarshal NotificationLevel %s."+
				"\nexpected: %d\nreceived: %d", l, l, newL)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Mock Notifications Manager                                                 //
////////////////////////////////////////////////////////////////////////////////

// Verify that mockNM adheres to the NotificationsManager interface.
var _ NotificationsManager = (*mockNM)(nil)

// mockNM adheres to the NotificationsManager interface.
type mockNM struct {
	channels map[string]clientNotif.Group
	cbs      map[string]clientNotif.Update
}

func newMockNM() *mockNM {
	return &mockNM{
		channels: make(map[string]clientNotif.Group),
		cbs:      make(map[string]clientNotif.Update),
	}
}

func (m *mockNM) Set(
	toBeNotifiedOn *id.ID, group string, metadata []byte, status clientNotif.NotificationState) error {
	if _, exists := m.channels[group]; !exists {
		m.channels[group] = clientNotif.Group{}
	}
	m.channels[group][*toBeNotifiedOn] = clientNotif.State{Metadata: metadata, Status: status}

	if _, exists := m.cbs[group]; exists {
		go m.cbs[group](m.channels[group], nil, nil, nil, clientNotif.Push)
	}
	return nil
}

func (m *mockNM) Get(toBeNotifiedOn *id.ID) (clientNotif.NotificationState, []byte, string, bool) {
	for group, ids := range m.channels {
		for chanID, ni := range ids {
			if chanID.Cmp(toBeNotifiedOn) {
				return ni.Status, ni.Metadata, group, true
			}
		}
	}

	return clientNotif.Mute, nil, "", false
}

func (m *mockNM) Delete(toBeNotifiedOn *id.ID) error {
	for group, ids := range m.channels {
		if _, exists := ids[*toBeNotifiedOn]; exists {
			delete(m.channels[group], *toBeNotifiedOn)
			if _, exists = m.cbs[group]; exists {
				go m.cbs[group](m.channels[group], nil, nil, nil, clientNotif.Push)
			}
			return nil
		}
	}
	return nil
}

func (m *mockNM) RegisterUpdateCallback(group string, nu clientNotif.Update) {
	m.cbs[group] = nu
}

////////////////////////////////////////////////////////////////////////////////
// Channel Getter                                                             //
////////////////////////////////////////////////////////////////////////////////

// Verify that mockCG adheres to the channelGetter interface.
var _ channelGetter = (*mockCG)(nil)

// mockNM adheres to the NotificationsManager interface.
type mockCG struct {
	channels map[id.ID]joinedChannel
	sync.RWMutex
}

// newMockCG returns a new mockCG with n new channels.
func newMockCG(n int, t testing.TB) *mockCG {
	rng := rand.New(rand.NewSource(2323))
	cg := &mockCG{channels: make(map[id.ID]joinedChannel)}
	for i := 0; i < n; i++ {
		chanID, err := id.NewRandomID(rng, id.User)
		if err != nil {
			t.Fatalf("Failed to generate new random ID for mockCG: %+v", err)
		}
		cg.channels[*chanID] = joinedChannel{&mockChannel{
			channelID:      chanID,
			asymIdentifier: append(chanID.Bytes(), []byte("asymIdentifier")...),
			symIdentifier:  append(chanID.Bytes(), []byte("symIdentifier")...),
		}, true}
	}
	return cg
}

func (m *mockCG) getChannel(channelID *id.ID) (*joinedChannel, error) {
	m.RLock()
	defer m.RUnlock()

	jc, exists := m.channels[*channelID]
	if !exists {
		return nil, ChannelDoesNotExistsErr
	}

	return &jc, nil
}

// Verify that mockChannel adheres to the broadcast.Channel interface.
var _ broadcast.Channel = (*mockChannel)(nil)

// mockChannel adheres to the broadcast.Channel interface.
type mockChannel struct {
	channelID      *id.ID
	asymIdentifier []byte
	symIdentifier  []byte
}

func (m *mockChannel) MaxPayloadSize() int            { panic("implement me") }
func (m *mockChannel) MaxRSAToPublicPayloadSize() int { panic("implement me") }
func (m *mockChannel) Get() *cryptoBroadcast.Channel  { panic("implement me") }
func (m *mockChannel) Broadcast([]byte, []string, [2]byte, cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	panic("implement me")
}
func (m *mockChannel) BroadcastWithAssembler(
	broadcast.Assembler, []string, [2]byte, cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {
	panic("implement me")
}
func (m *mockChannel) BroadcastRSAtoPublic(
	rsa.PrivateKey, []byte, []string, [2]byte, cmix.CMIXParams) (
	[]byte, rounds.Round, ephemeral.Id, error) {
	panic("implement me")
}
func (m *mockChannel) BroadcastRSAToPublicWithAssembler(
	rsa.PrivateKey, broadcast.Assembler, []string, [2]byte, cmix.CMIXParams) (
	[]byte, rounds.Round, ephemeral.Id, error) {
	panic("implement me")
}
func (m *mockChannel) RegisterRSAtoPublicListener(
	broadcast.ListenerFunc, []string) (broadcast.Processor, error) {
	panic("implement me")
}
func (m *mockChannel) RegisterSymmetricListener(
	broadcast.ListenerFunc, []string) (broadcast.Processor, error) {
	panic("implement me")
}
func (m *mockChannel) Stop() { panic("implement me") }

func (m *mockChannel) AsymmetricIdentifier() []byte { return m.asymIdentifier }
func (m *mockChannel) SymmetricIdentifier() []byte  { return m.symIdentifier }

////////////////////////////////////////////////////////////////////////////////
// Mock ExtensionMessageHandler                                               //
////////////////////////////////////////////////////////////////////////////////

// Tests that mockNotifExtension adheres to the ExtensionMessageHandler interface.
var _ ExtensionMessageHandler = (*mockNotifExtension)(nil)

// mockNotifExtension is a mock interface of ExtensionMessageHandler for
// testing.
type mockNotifExtension struct{}

func newMockNotifExtension() *mockNotifExtension                        { return &mockNotifExtension{} }
func (m *mockNotifExtension) GetType() MessageType                      { panic("implement me") }
func (m *mockNotifExtension) GetProperties() (string, bool, bool, bool) { panic("implement me") }
func (m *mockNotifExtension) Handle(*id.ID, message.ID, MessageType, string,
	[]byte, []byte, ed25519.PublicKey, uint32, uint8, time.Time, time.Time,
	time.Duration, id.Round, rounds.Round, SentStatus, bool, bool) uint64 {
	panic("implement me")
}

func (m *mockNotifExtension) GetNotificationTags(
	_ *id.ID, level NotificationLevel) (asymmetric, symmetric AllowLists) {
	switch level {
	case NotifyPing:
		return AllowLists{AllowWithTags: map[MessageType]struct{}{984: {}}},
			AllowLists{AllowWithTags: map[MessageType]struct{}{53345: {}}}
	case NotifyAll:
		return AllowLists{AllowWithoutTags: map[MessageType]struct{}{234: {}}},
			AllowLists{AllowWithoutTags: map[MessageType]struct{}{53345: {}}}
	}
	return AllowLists{}, AllowLists{}
}
