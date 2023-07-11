package notifications

import (
	"bytes"
	"errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	notifCrypto "gitlab.com/elixxir/crypto/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"reflect"
	"testing"
	"time"
)

func TestManager_Set(t *testing.T) {
	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	groupName := "flexnard"

	cbChan := make(chan struct{})
	m.RegisterUpdateCallback(groupName, func(group Group, created, edits, deletions []*id.ID, maxState NotificationState) {
		cbChan <- struct{}{}
	})

	// Show that nothing happens if your set is the same as the local data
	nid1 := id.NewIdFromUInt(42, id.User, t)

	reg1 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{88, 2, 4},
			// not push so it will trigger an unregister that we can see
			// on the comms object in the event it isnt caught as identical
			Status: Mute,
		},
	}

	mInternal.upsertNotificationUnsafeRAM(nid1, reg1)

	if err := m.Set(nid1, reg1.Group, reg1.Metadata, reg1.Status); err != nil {
		t.Errorf("Got an error on calling set in a valid way")
	}

	if comms.receivedMessage != nil {
		t.Errorf("received message should not have been set becasue we " +
			"are setting a value equal to what is on remote")
	}

	// test a set where the local value doesnt exist at all works when
	// not doing a push
	comms.reset()
	nid2 := id.NewIdFromUInt(69, id.User, t)

	reg2 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{66, 33, 99},
			// not push so it will trigger an unregister that we can see
			// on the comms object in the event it isnt caught as identical
			Status: Mute,
		},
	}

	if err := m.Set(nid2, reg2.Group, reg2.Metadata, reg2.Status); err != nil {
		t.Errorf("Got an error on calling set in a valid way")
	}
	to := time.NewTimer(time.Second)
	select {
	case <-cbChan:
	case <-to.C:
		t.Fatalf("Failed to receive on cb chan")
	}

	if comms.receivedMessage == nil {
		t.Errorf("received message should not have been set becasue we " +
			"are setting a value equal to what is on remote")
	}

	hasNotif(mInternal, nid2, groupName, true, t)

	// test a set where the local value doesnt exist at all works when
	// doing a push

	comms.reset()
	nid3 := id.NewIdFromUInt(420, id.User, t)

	reg3 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{11, 22, 44},
			Status:   Push,
		},
	}

	if err := m.Set(nid3, reg3.Group, reg3.Metadata, reg3.Status); err != nil {
		t.Errorf("Got an error on calling set in a valid way")
	}
	to.Reset(time.Second)
	select {
	case <-cbChan:
	case <-to.C:
		t.Fatalf("Failed to receive on cb chan")
	}

	if comms.receivedMessage == nil {
		t.Errorf("received message should not have been set becasue we " +
			"are setting a value equal to what is on remote")
	}

	hasNotif(mInternal, nid3, groupName, true, t)

	// test a set where the local value does exist at all works when
	// doing a push, shouldn't do a comm but should update the storage
	comms.reset()
	nid4 := id.NewIdFromUInt(18, id.User, t)

	reg4 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{12, 34, 56},
			Status:   Push,
		},
	}

	mInternal.upsertNotificationUnsafeRAM(nid4, reg4)

	reg4.Metadata[0] = 21

	if err := m.Set(nid4, reg4.Group, reg4.Metadata, reg4.Status); err != nil {
		t.Errorf("Got an error on calling set in a valid way")
	}

	if comms.receivedMessage != nil {
		t.Errorf("received message should not have been set becasue we " +
			"are setting a value equal to what is on remote")
	}

	hasNotif(mInternal, nid3, groupName, true, t)

	if !bytes.Equal(mInternal.notifications[*nid4].Metadata, reg4.Metadata) {
		t.Errorf("notifications data not updated correctly")
	}

	// test a set where the local value doesnt exist and comms returns an error
	// stopping the insert without a push
	comms.reset()
	comms.returnedError = errors.New("dummy")
	nid5 := id.NewIdFromUInt(99, id.User, t)

	reg5 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{65, 43, 32},
			// not push so it will trigger an unregister that we can see
			// on the comms object in the event it isnt caught as identical
			Status: Mute,
		},
	}

	if err := m.Set(nid5, reg5.Group, reg5.Metadata, reg5.Status); err != nil {
		t.Errorf("Received error from comms (this should happen in the handler thread, and will not be seen here)")
	}
	time.Sleep(time.Second)

	if comms.receivedMessage == nil {
		t.Errorf("no message received when a push should occur")
	}

	hasNotif(mInternal, nid5, groupName, false, t)

	// test a set where the local value doesnt exist and comms returns an error
	// stoping the insert with a push
	comms.reset()
	comms.returnedError = errors.New("dummy")
	nid6 := id.NewIdFromUInt(88, id.User, t)

	reg6 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{81, 82, 83},
			Status:   Push,
		},
	}

	if err := m.Set(nid6, reg6.Group, reg6.Metadata, reg6.Status); err != nil {
		t.Errorf("Received error from comms (this should happen in the handler thread, and will not be seen here)")
	}
	time.Sleep(time.Second)

	if comms.receivedMessage == nil {
		t.Errorf("no message received when a push should occur")
	}

}

func TestManager_Get(t *testing.T) {

	m, _, _ := buildTestingManager(t)
	mInternal := m.(*manager)

	groupName := "DoorTechnicianRick"

	// test get when the element does not exist
	nid1 := id.NewIdFromUInt(42, id.User, t)
	if _, _, _, exists := m.Get(nid1); exists {
		t.Errorf("Element returns when it doest exist")
	}

	// test an element is returned when it does exist
	nid2 := id.NewIdFromUInt(69, id.User, t)

	reg2 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{88, 2, 4},
			Status:   Mute,
		},
	}
	mInternal.upsertNotificationUnsafeRAM(nid2, reg2)

	state, metadata, group, exists := m.Get(nid2)
	if !exists {
		t.Errorf("element not found when it exists")
	}

	if state != reg2.Status {
		t.Errorf("status returned is wrong")
	}

	if group != groupName {
		t.Errorf("returned the wrong group")
	}

	if !bytes.Equal(metadata, reg2.Metadata) {
		t.Errorf("Returned metadata does not match inserted metadata")
	}

	// check that the metadata is deep copied
	metadata[0] = metadata[0] * 2

	if bytes.Equal(metadata, reg2.Metadata) {
		t.Errorf("metadata not deep copied, internal copy can be edited")
	}

}

func TestManager_GetGroup(t *testing.T) {

	m, _, _ := buildTestingManager(t)
	mInternal := m.(*manager)

	groupName := "MartyTheRabbitBoyAndHisMusicalBlender"

	// test that get group works when it doesnt exist
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

func TestManager_Delete(t *testing.T) {
	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	// test that it fails if it doesnt exist

	nid := id.NewIdFromUInt(42, id.User, t)

	if err := m.Delete(nid); err != nil {
		t.Fatalf("Got an error when deleting something that doesnt exist")
	}

	groupName := "oogabooga"
	cbChan := make(chan struct{})
	m.RegisterUpdateCallback(groupName, func(group Group, created, edits, deletions []*id.ID, maxState NotificationState) {
		cbChan <- struct{}{}
	})

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
	to := time.NewTimer(time.Second)
	select {
	case <-cbChan:
	case <-to.C:
		t.Fatalf("Failed to receive on cb chan")
	}

	if comms.receivedMessage != nil {
		t.Errorf("Message sent when it shouldnt be sent!")
	}

	notHasNotif(mInternal, nid, groupName, t)

	// test that deletions work when the status is push and a message is sent
	// over comms
	comms.reset()
	nid2 := id.NewIdFromUInt(69, id.User, t)
	reg2 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{5, 22, 3},
			Status:   Push,
		},
	}

	mInternal.upsertNotificationUnsafeRAM(nid2, reg2)

	if err := mInternal.storeRegistration(nid2, reg2, time.Now()); err != nil {
		t.Errorf("Failed to store registeration, should not happen")
	}

	if err := m.Delete(nid2); err != nil {
		t.Fatalf("Got an error when deleting something that exists and "+
			"shouldnt error: %+v", err)
	}
	to.Reset(time.Second)
	select {
	case <-cbChan:
	case <-to.C:
		t.Fatalf("Failed to receive on cb chan")
	}

	if comms.receivedMessage == nil {
		t.Errorf("Message not sent when it should be sent!")
	}

	notHasNotif(mInternal, nid, groupName, t)

	// test that deletions do not occur when comms returns an error
	comms.reset()
	comms.returnedError = errors.New("dummy error")

	nid3 := id.NewIdFromUInt(420, id.User, t)
	reg3 := registration{
		Group: groupName,
		State: State{
			Metadata: []byte{89, 22, 39},
			Status:   Push,
		},
	}

	mInternal.upsertNotificationUnsafeRAM(nid3, reg3)

	if err := mInternal.storeRegistration(nid3, reg3, time.Now()); err != nil {
		t.Errorf("Failed to store registeration, should not happen")
	}

	if err := m.Delete(nid3); err != nil {
		t.Fatalf("Comms failures will occur in the handler thread and should not affect the delete call")
	}
	time.Sleep(time.Second)

	if comms.receivedMessage == nil {
		t.Errorf("Message not sent when it should be sent!")
	}

	hasNotif(mInternal, nid3, groupName, false, t)
}

func TestManager_registerNotification(t *testing.T) {

	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	nid := id.NewIdFromUInt(69, id.User, t)
	expectedIID, err := ephemeral.GetIntermediaryId(nid)
	if err != nil {
		t.Fatalf("Failed to get IID: %+v", err)
	}

	if err = mInternal.registerNotification([]*id.ID{nid}); err != nil {
		t.Fatalf("Failed to register notification: %+v", err)
	}

	// check the message
	message := comms.receivedMessage.(*pb.RegisterTrackedIdRequest).Request
	if !bytes.Equal(message.TrackedIntermediaryID[0], expectedIID[:]) {
		t.Errorf("IIDs do not match")
	}

	err = notifCrypto.VerifyIdentity(mInternal.transmissionRSA.Public(), [][]byte{expectedIID},
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

	if err = mInternal.unregisterNotification([]*id.ID{nid}); err != nil {
		t.Fatalf("Failed to register notification: %+v", err)
	}

	// check the message
	message := comms.receivedMessage.(*pb.UnregisterTrackedIdRequest).Request
	if !bytes.Equal(message.TrackedIntermediaryID[0], expectedIID[:]) {
		t.Errorf("IIDs do not match")
	}

	err = notifCrypto.VerifyIdentity(mInternal.transmissionRSA.Public(), [][]byte{expectedIID},
		time.Unix(0, message.RequestTimestamp), notifCrypto.UnregisterTrackedIDTag,
		message.Signature)
	if err != nil {
		t.Errorf("Failed to verify the token unregister signature: %+v",
			err)
	}
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

func hasNotif(mInternal *manager, nid *id.ID, group string, confirmed bool, t *testing.T) {
	if _, err := mInternal.remote.GetMapElement(notificationsMap,
		makeElementName(nid), notificationsMapVersion); err != nil {
		t.Errorf("did not get an element from remote when it should exist")
	}
	reg, exists := mInternal.notifications[*nid]
	if !exists {
		t.Errorf("notification does not exists when it should")
	}
	if reg.Confirmed != confirmed {
		t.Errorf("Expected registration confirmation status to be %+v, instead it was %+v", confirmed, reg.Confirmed)
	}

	g, exists := mInternal.group[group]
	if !exists {
		t.Errorf("notification's group doesnt exist when it " +
			"should because the notification should exist")
		return
	}

	if _, exists = g[*nid]; !exists {
		t.Errorf("Notification does not exists in group when it should")
	}
}
