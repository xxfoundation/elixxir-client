////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestGetNotificationReportsForMe(t *testing.T) {
}

func TestNotificationFilter_match(t *testing.T) {
}

func Test_newNotifications(t *testing.T) {
}

func Test_notifications_EnableNotifications(t *testing.T) {
}

func Test_notifications_GetNotificationLevel(t *testing.T) {
}

func Test_notifications_SetMobileNotificationsLevel(t *testing.T) {
}

func Test_notifications_updateSihTagsCB(t *testing.T) {
}

func Test_statusToLevel(t *testing.T) {
}



////////////////////////////////////////////////////////////////////////////////
// NotificationLevel                                                          //
////////////////////////////////////////////////////////////////////////////////



func TestNotificationLevel_String(t *testing.T) {
}

func TestNotificationLevel_Marshal(t *testing.T) {
}

func TestUnmarshalNotificationLevel(t *testing.T) {
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
