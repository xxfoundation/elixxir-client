package notifications

import (
	"bytes"
	pb "gitlab.com/elixxir/comms/mixmessages"
	notifCrypto "gitlab.com/elixxir/crypto/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"reflect"
	"testing"
	"time"
)

func TestManager_Set(t *testing.T) {

}

func TestManager_Delete(t *testing.T) {
	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	// test that it fails if it doesnt exist

	nid := id.NewIdFromUInt(42, id.User, t)

	if err := m.Delete(nid); err != nil {
		t.Fatalf("Got an error when deleting something that doesnt exist")
	}

	groupName := "oogabooga"

	reg := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{0, 2, 4},
			Status:   Mute,
		},
	}

	// test that deletions work when the status is not push and no
	// message is sent over comms

	mInternal.upsertNotificationUnsafeRAM(nid, reg)

	if err := mInternal.storeRegistration(nid, reg, time.Now()); err != nil {
		t.Errorf("Failed to store registeration, should not happen")
	}

	if err := m.Delete(nid); err != nil {
		t.Fatalf("Got an error when deleting something that exists and "+
			"shouldnt error: %+v", err)
	}

	if comms.receivedMessage != nil {
		t.Errorf("Message sent when it shouldnt be sent!")
	}

	notHasNotif(mInternal, nid, groupName, t)

}

func notHasNotif(mInternal *manager, nid *id.ID, group string, t *testing.T) {
	if _, err := mInternal.remote.GetMapElement(notificationsMap,
		makeElementName(nid), notificationsMapVersion); err == nil {
		t.Errorf("Got an element from remote when it should be deleted")
	}

	if _, exists := mInternal.notifications[*nid]; exists {
		t.Errorf("Notification still exists when it should not")
	}

	g, exists := mInternal.group[group]
	if !exists {
		return
	}

	if _, exists = g[*nid]; exists {
		t.Errorf("notification still exists in the groups registration!")
	}
}

func TestManager_GetGroup(t *testing.T) {

	m, _, _ := buildTestingManager(t)
	mInternal := m.(*manager)

	// test that get group works when it doesnt exist

	groupName := "MartyTheRabbitBoyAndHisMusicalBlender"
	if _, exists := m.GetGroup(groupName); exists {
		t.Errorf("Got a group when it shouldnt exist")
	}

	// test that it returns the correct group
	g := make(Group)

	for i := 0; i < 1000; i++ {
		elementName := id.NewIdFromUInt(uint64(i), id.User, t)
		g[*elementName] = State{
			Metadata: []byte{byte(i)},
			Status:   NotificationState(i % 3),
		}
	}

	mInternal.group[groupName] = g

	gReceived, exists := m.GetGroup(groupName)
	if !exists {
		t.Errorf("Failed to get group for %+v", gReceived)
	}
	if !reflect.DeepEqual(gReceived, g) {
		t.Errorf("Received group doesnt match expected")
	}

	// show that the two group objects are deep copies
	firstID := id.NewIdFromUInt(0, id.User, t)
	firstStat := gReceived[*firstID]
	firstStat.Metadata[0] = 69
	gReceived[*firstID] = firstStat

	if !reflect.DeepEqual(gReceived, g) {
		t.Errorf("Edit propogated!")
	}

}

func TestManager_registerNotification(t *testing.T) {

	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	nid := id.NewIdFromUInt(69, id.User, t)
	expectedIID, err := ephemeral.GetIntermediaryId(nid)
	if err != nil {
		t.Fatalf("Failed to get IID: %+v", err)
	}

	if err = mInternal.registerNotification(nid); err != nil {
		t.Fatalf("Failed to register notification: %+v", err)
	}

	// check the message
	message := comms.receivedMessage.(*pb.TrackedIntermediaryIDRequest)
	if !bytes.Equal(message.TrackedIntermediaryID, expectedIID[:]) {
		t.Errorf("IIDs do not match")
	}

	err = notifCrypto.VerifyIdentity(mInternal.transmissionRSA.Public(), expectedIID,
		time.Unix(0, message.RequestTimestamp), notifCrypto.RegisterTrackedIDTag,
		message.Signature)
	if err != nil {
		t.Errorf("Failed to verify the id unregister signature: %+v",
			err)
	}
}

func TestManager_UnregisterNotification(t *testing.T) {

	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	nid := id.NewIdFromUInt(69, id.User, t)
	expectedIID, err := ephemeral.GetIntermediaryId(nid)
	if err != nil {
		t.Fatalf("Failed to get IID: %+v", err)
	}

	if err = mInternal.unregisterNotification(nid); err != nil {
		t.Fatalf("Failed to register notification: %+v", err)
	}

	// check the message
	message := comms.receivedMessage.(*pb.TrackedIntermediaryIDRequest)
	if !bytes.Equal(message.TrackedIntermediaryID, expectedIID[:]) {
		t.Errorf("IIDs do not match")
	}

	err = notifCrypto.VerifyIdentity(mInternal.transmissionRSA.Public(), expectedIID,
		time.Unix(0, message.RequestTimestamp), notifCrypto.UnregisterTrackedIDTag,
		message.Signature)
	if err != nil {
		t.Errorf("Failed to verify the token unregister signature: %+v",
			err)
	}
}
