////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"bytes"
	"crypto/ed25519"
	"io"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/dm"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/elixxir/ekv"
	primNotif "gitlab.com/elixxir/primitives/notifications"
	"gitlab.com/xx_network/primitives/id"
)

// Tests that newNotifications returns the expected new notifications object.
func Test_newNotifications(t *testing.T) {
	prng := rand.New(rand.NewSource(267335))
	me, _ := codename.GenerateIdentity(prng)
	receptionID := deriveReceptionID(
		ecdh.ECDHNIKE.DerivePublicKey(
			ecdh.Edwards2EcdhNikePrivateKey(me.Privkey)).Bytes(), me.GetDMToken())
	ps, err := newPartnerStore(collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote()))
	if err != nil {
		t.Fatal(err)
	}
	nm := newMockNM()
	expected := &notifications{
		id:            receptionID,
		pubKey:        me.PubKey,
		privKey:       me.Privkey,
		partnerTagMap: make(map[string]string),
		ps:            ps,
		nm:            nm,
	}

	n, err := newNotifications(receptionID, me.PubKey, me.Privkey, nil, ps, nm)
	if err != nil {
		t.Fatalf("Failed to make new notifications: %+v", err)
	}

	if !reflect.DeepEqual(expected, n) {
		t.Errorf("New notifications does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, n)
	}
}

// Tests that notifications.updateSihTagsCBUnsafe returns the expected filter,
// edits, and deleted list for each list of edits.
//
// This test sets up three calls to updateSihTagsCBUnsafe with structured inputs
// with known outputs.
func Test_notifications_updateSihTagsCBUnsafe(t *testing.T) {
	prng := rand.New(rand.NewSource(38496))
	me, _ := codename.GenerateIdentity(prng)
	receptionID := deriveReceptionID(
		ecdh.ECDHNIKE.DerivePublicKey(
			ecdh.Edwards2EcdhNikePrivateKey(me.Privkey)).Bytes(), me.GetDMToken())
	n := &notifications{
		id:            receptionID,
		pubKey:        me.PubKey,
		privKey:       me.Privkey,
		partnerTagMap: make(map[string]string),
	}

	key1, key2, key3, key4 :=
		newPubKey(prng), newPubKey(prng), newPubKey(prng), newPubKey(prng)
	tag1, tag2, tag3, tag4 := dm.MakeReceiverSihTag(key1, n.privKey),
		dm.MakeReceiverSihTag(key2, n.privKey), dm.MakeReceiverSihTag(key3, n.privKey),
		dm.MakeReceiverSihTag(key4, n.privKey)

	tests := []struct {
		edits         []elementEdit
		nf            NotificationFilter
		changed       []NotificationState
		deleted       []ed25519.PublicKey
		partnerTagMap map[string]string
	}{{
		// Test 0
		edits: []elementEdit{
			{nil, &dmPartner{key1, defaultStatus}, versioned.Loaded},
			{nil, &dmPartner{key2, defaultStatus}, versioned.Loaded},
			{nil, &dmPartner{key3, defaultStatus}, versioned.Loaded}},
		nf: NotificationFilter{
			Identifier:   n.pubKey,
			MyID:         n.id,
			Tags:         []string{tag1, tag2, tag3},
			PublicKeys:   map[string]ed25519.PublicKey{tag1: key1, tag2: key2, tag3: key3},
			AllowedTypes: allowList[NotifyAll],
		},
		changed:       []NotificationState{{key1, NotifyAll}, {key2, NotifyAll}, {key3, NotifyAll}},
		deleted:       []ed25519.PublicKey{},
		partnerTagMap: map[string]string{string(key1): tag1, string(key2): tag2, string(key3): tag3},
	}, {
		// Test 1
		edits: []elementEdit{
			{nil, &dmPartner{key4, defaultStatus}, versioned.Created},
			{&dmPartner{key2, defaultStatus}, &dmPartner{key2, statusMute}, versioned.Updated},
			{&dmPartner{key3, defaultStatus}, &dmPartner{key3, statusBlocked}, versioned.Updated}},
		nf: NotificationFilter{
			Identifier:   n.pubKey,
			MyID:         n.id,
			Tags:         []string{tag1, tag4},
			PublicKeys:   map[string]ed25519.PublicKey{tag1: key1, tag4: key4},
			AllowedTypes: allowList[NotifyAll],
		},
		changed:       []NotificationState{{key2, NotifyNone}, {key3, NotifyNone}, {key4, NotifyAll}},
		deleted:       []ed25519.PublicKey{},
		partnerTagMap: map[string]string{string(key1): tag1, string(key4): tag4},
	}, {
		// Test 2
		edits: []elementEdit{
			{&dmPartner{key1, defaultStatus}, nil, versioned.Deleted},
			{&dmPartner{key2, statusMute}, &dmPartner{key2, statusNotifyAll}, versioned.Updated},
			{&dmPartner{key3, statusBlocked}, &dmPartner{key3, statusMute}, versioned.Updated}},
		nf: NotificationFilter{
			Identifier:   n.pubKey,
			MyID:         n.id,
			Tags:         []string{tag2, tag4},
			PublicKeys:   map[string]ed25519.PublicKey{tag2: key2, tag4: key4},
			AllowedTypes: allowList[NotifyAll],
		},
		changed:       []NotificationState{{key2, NotifyAll}, {key3, NotifyNone}},
		deleted:       []ed25519.PublicKey{key1},
		partnerTagMap: map[string]string{string(key2): tag2, string(key4): tag4},
	}}

	for i, tt := range tests {
		nf, changed, deleted := n.updateSihTagsCBUnsafe(tt.edits)

		// Sort slices
		sort.SliceStable(tt.nf.Tags, func(i, j int) bool { return strings.Compare(tt.nf.Tags[i], tt.nf.Tags[j]) == -1 })
		sort.SliceStable(nf.Tags, func(i, j int) bool { return strings.Compare(nf.Tags[i], nf.Tags[j]) == -1 })
		sort.SliceStable(tt.changed, func(i, j int) bool { return bytes.Compare(tt.changed[i].PubKey, tt.changed[j].PubKey) == -1 })
		sort.SliceStable(changed, func(i, j int) bool { return bytes.Compare(changed[i].PubKey, changed[j].PubKey) == -1 })
		sort.SliceStable(tt.deleted, func(i, j int) bool { return bytes.Compare(tt.deleted[i], tt.deleted[j]) == -1 })
		sort.SliceStable(deleted, func(i, j int) bool { return bytes.Compare(deleted[i], deleted[j]) == -1 })

		if !reflect.DeepEqual(tt.nf, nf) {
			t.Errorf("Unexpected NotificationFilter (%d)."+
				"\nexpected: %#v\nreceived: %#v", i, tt.nf, nf)
		}
		if !reflect.DeepEqual(tt.changed, changed) {
			t.Errorf("Unexpected changed NotificationState (%d)."+
				"\nexpected: %v\nreceived: %v", i, tt.changed, changed)
		}
		if !reflect.DeepEqual(tt.deleted, deleted) {
			t.Errorf("Unexpected deleted public keys (%d)."+
				"\nexpected: %v\nreceived: %v", i, tt.deleted, deleted)
		}
		if !reflect.DeepEqual(tt.partnerTagMap, n.partnerTagMap) {
			t.Errorf("Unexpected partnerTagMap (%d)."+
				"\nexpected: %X\nreceived: %X", i, tt.partnerTagMap, n.partnerTagMap)
		}
	}

}

// Tests that notifications.GetNotificationLevel returns the correct level for
// all added IDs.
func Test_notifications_GetNotificationLevel(t *testing.T) {
	n := newTestNotifications(29817427, nil, t)
	prng := rand.New(rand.NewSource(63795))

	expected := map[string]NotificationLevel{
		string(newPubKey(prng)): NotifyNone,
		string(newPubKey(prng)): NotifyAll,
		string(newPubKey(prng)): NotifyAll,
	}

	for pubKey, level := range expected {
		err := n.SetMobileNotificationsLevel(ed25519.PublicKey(pubKey), level)
		if err != nil {
			t.Errorf("Failed to set level for partner %X: %+v", pubKey, err)
		}
	}

	for pubKey, level := range expected {
		l, err := n.GetNotificationLevel(ed25519.PublicKey(pubKey))
		if err != nil {
			t.Errorf("Failed to get notification level partner %s: %+v",
				pubKey, err)
		}

		if level != l {
			t.Errorf("Incorrect level for %X.\nexpected: %s\nreceived: %s",
				pubKey, level, l)
		}
	}

	_, err := n.GetNotificationLevel(newPubKey(prng))
	if err == nil {
		t.Errorf("Did not get error when getting level for a channel that " +
			"does not exist.")
	}
}

// Tests that notifications.SetMobileNotificationsLevel sets the notification
// level correctly.
func Test_notifications_SetMobileNotificationsLevel(t *testing.T) {
	n := newTestNotifications(29817427, nil, t)
	prng := rand.New(rand.NewSource(63795))

	expected := map[string]NotificationLevel{
		string(newPubKey(prng)): NotifyNone,
		string(newPubKey(prng)): NotifyAll,
		string(newPubKey(prng)): NotifyAll,
	}

	for pubKey := range expected {
		n.ps.set(ed25519.PublicKey(pubKey), 0)
	}

	for pubKey, level := range expected {
		err := n.SetMobileNotificationsLevel(ed25519.PublicKey(pubKey), level)
		if err != nil {
			t.Errorf("Failed to set level for partner %X: %+v", pubKey, err)
		}
	}

	partners := n.ps.getAll()

	if len(expected) != len(partners) {
		t.Errorf("Unexpected number of partners in storage."+
			"\nexpected: %d\nreceived: %d", len(expected), len(partners))
	}

	for _, partner := range partners {
		if _, exists := expected[string(partner.PublicKey)]; !exists {
			t.Errorf("Unexpected partner %X found.", partner.PublicKey)
		} else {
			delete(expected, string(partner.PublicKey))
		}
	}

	if len(expected) != 0 {
		t.Errorf("%d partners not found: %v", len(expected), expected)
	}
}

// Unit test of statusToLevel.
func Test_statusToLevel(t *testing.T) {
	tests := map[partnerStatus]NotificationLevel{
		statusMute:      NotifyNone,
		statusNotifyAll: NotifyAll,
		statusBlocked:   NotifyNone,
		6565:            NotifyAll,
	}

	for status, expected := range tests {
		level := statusToLevel(status)
		if expected != level {
			t.Errorf("Unexpected level for status %s."+
				"\nexpected: %s\nreceived: %s", status, expected, level)
		}
	}
}

// Unit test of levelToStatus.
func Test_levelToStatus(t *testing.T) {
	tests := map[NotificationLevel]partnerStatus{
		NotifyNone:    statusMute,
		NotifyAll:     statusNotifyAll,
		186:           statusNotifyAll,
	}

	for level, expected := range tests {
		status := levelToStatus(level)
		if expected != status {
			t.Errorf("Unexpected status for level %s."+
				"\nexpected: %s\nreceived: %s", level, expected, status)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// For Me / Notification Report                                               //
////////////////////////////////////////////////////////////////////////////////

// Tests GetNotificationReportsForMe.
func TestGetNotificationReportsForMe(t *testing.T) {
	prng := rand.New(rand.NewSource(38496))
	me, _ := codename.GenerateIdentity(prng)
	receptionID := deriveReceptionID(
		ecdh.ECDHNIKE.DerivePublicKey(
			ecdh.Edwards2EcdhNikePrivateKey(me.Privkey)).Bytes(), me.GetDMToken())
	types := []MessageType{TextType, ReplyType, ReactionType, SilentType}
	levels := []NotificationLevel{NotifyNone, NotifyAll}

	var expected []NotificationReport
	var notifData []*primNotif.Data
	nf := NotificationFilter{
		Identifier:   me.PubKey,
		MyID:         receptionID,
		Tags:         []string{},
		PublicKeys:   make(map[string]ed25519.PublicKey),
		AllowedTypes: allowList[NotifyAll],
	}
	for _, mt := range types {
		for _, level := range levels {
			pubKey := newPubKey(prng)
			tag := dm.MakeReceiverSihTag(pubKey, me.Privkey)

			msgHash := make([]byte, 24)
			prng.Read(msgHash)

			mtByte := mt.Marshal()
			cSIH, err := sih.MakeCompressedSIH(
				receptionID, msgHash, me.PubKey, []string{tag}, mtByte[:])
			if err != nil {
				t.Fatalf("Failed to make compressed SIH: %+v", err)
			}

			notifData = append(notifData, &primNotif.Data{
				IdentityFP:  cSIH,
				MessageHash: msgHash,
			})

			if level == NotifyAll && nf.AllowedTypes[mt] != struct{}{} {
				nf.Tags = append(nf.Tags, tag)
				nf.PublicKeys[tag] = pubKey
				expected = append(expected, NotificationReport{
					Partner: pubKey,
					Type:    mt,
				})
			}
		}
	}

	nrs := GetNotificationReportsForMe(nf, notifData)

	if !reflect.DeepEqual(nrs, expected) {
		t.Errorf("NotificationReport list does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, nrs)
	}
}

////////////////////////////////////////////////////////////////////////////////
// MessageTypeFilter                                                          //
////////////////////////////////////////////////////////////////////////////////

func TestNotificationFilter_match(t *testing.T) {
}

////////////////////////////////////////////////////////////////////////////////
// NotificationLevel                                                          //
////////////////////////////////////////////////////////////////////////////////

// Consistency test of NotificationLevel.String.
func TestNotificationLevel_String(t *testing.T) {
	tests := map[NotificationLevel]string{
		NotifyNone:    "none",
		NotifyAll:     "all",
		32:            "INVALID NOTIFICATION LEVEL: 32",
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
	tests := []NotificationLevel{NotifyNone, NotifyAll}

	for _, l := range tests {
		data := l.Marshal()
		newL := UnmarshalNotificationLevel(data)

		if l != newL {
			t.Errorf("Failed to marshal and unmarshal NotificationLevel %s."+
				"\nexpected: %d\nreceived: %d", l, l, newL)
		}
	}
}

func newPubKey(rng io.Reader) ed25519.PublicKey {
	pubKey, _, _ := ed25519.GenerateKey(rng)
	return pubKey
}

// newTestNotifications creates a new notifications for testing that contains
// a correctly generated ID.
func newTestNotifications(seed int64, cb NotificationUpdate, t testing.TB) *notifications {
	prng := rand.New(rand.NewSource(seed))
	me, _ := codename.GenerateIdentity(prng)
	receptionID := deriveReceptionID(
		ecdh.ECDHNIKE.DerivePublicKey(
			ecdh.Edwards2EcdhNikePrivateKey(me.Privkey)).Bytes(), me.GetDMToken())
	ps, err := newPartnerStore(collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote()))
	if err != nil {
		t.Fatal(err)
	}
	nm := newMockNM()

	if cb == nil {
		cb = func(NotificationFilter, []NotificationState, []ed25519.PublicKey) {}
	}

	n, err := newNotifications(receptionID, me.PubKey, me.Privkey, cb, ps, nm)
	if err != nil {
		t.Fatalf("Failed to make new notifications: %+v", err)
	}

	return n
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
