////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	e2eImport "gitlab.com/elixxir/client/v4/e2e"
	gs "gitlab.com/elixxir/client/v4/groupChat/groupStore"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Tests that manager adheres to the GroupChat interface.
var _ GroupChat = (*manager)(nil)

// Tests that groupCmix adheres to the cmix.Client interface.
var _ groupCmix = (cmix.Client)(nil)

// Tests that groupE2eHandler adheres to the e2e.Handler interface.
var _ groupE2eHandler = (e2eImport.Handler)(nil)

type mockProcessor struct{ receiveChan chan MessageReceive }

func (m mockProcessor) Process(msg MessageReceive, _ format.Message, _ []string, _ []byte,
	_ receptionID.EphemeralIdentity, _ rounds.Round) {
	m.receiveChan <- msg
}
func (m mockProcessor) String() string { return "mockProcessor" }

// Unit test of NewManager.
func TestNewManager(t *testing.T) {

	requestChan := make(chan gs.Group)
	requestFunc := func(g gs.Group) { requestChan <- g }
	receiveChan := make(chan MessageReceive)
	mockMess := newMockE2e(t, nil)
	gcInt, err := NewManager(mockMess, requestFunc,
		mockProcessor{receiveChan})
	if err != nil {
		t.Errorf("NewManager returned an error: %+v", err)
	}

	dhKeyPub := mockMess.GetE2E().GetHistoricalDHPubkey()
	user := group.Member{
		ID:    mockMess.GetReceptionIdentity().ID,
		DhKey: dhKeyPub,
	}

	m := gcInt.(*manager)

	if !m.gs.GetUser().Equal(user) {
		t.Errorf("NewManager failed to create a store with the correct user."+
			"\nexpected: %s\nreceived: %s", user, m.gs.GetUser())
	}

	if m.gs.Len() != 0 {
		t.Errorf("NewManager failed to create an empty store."+
			"\nexpected: %d\nreceived: %d", 0, m.gs.Len())
	}

	// Check if requestFunc works
	go m.requestFunc(gs.Group{})
	select {
	case <-requestChan:
	case <-time.NewTimer(5 * time.Millisecond).C:
		t.Errorf("Timed out waiting for requestFunc to be called.")
	}
}

// Tests that NewManager loads a group storage when it exists.
func TestNewManager_LoadStorage(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := group.Member{
		ID:    id.NewIdFromString("userID", id.User, t),
		DhKey: randCycInt(prng),
	}

	gStore, err := gs.NewStore(kv, user)
	if err != nil {
		t.Errorf("Failed to create new group storage: %+v", err)
	}

	expectedGroups := make([]gs.Group, 0)
	for i := 0; i < 10; i++ {
		grp := newTestGroup(getGroup(), getGroup().NewInt(42), prng, t)
		err := gStore.Add(grp)
		if err != nil {
			t.Errorf("Failed to add group %d: %+v", i, err)
		}
		expectedGroups = append(expectedGroups, grp)
	}

	mockMess := newMockE2e(t, kv)
	gcInt, err := NewManager(mockMess, nil, nil)
	if err != nil {
		t.Errorf("NewManager returned an error: %+v", err)
	}

	m := gcInt.(*manager)

	for _, grp := range expectedGroups {
		if _, exists := m.gs.Get(grp.ID); !exists {
			t.Errorf("NewManager failed to load the expected storage.")
		}
	}
}

// Error path: an error is returned when a group cannot be loaded from storage.
func TestNewManager_LoadError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	kv := versioned.NewKV(ekv.MakeMemstore())
	user := group.Member{
		ID:    id.NewIdFromString("userID", id.User, t),
		DhKey: randCycInt(rand.New(rand.NewSource(42))),
	}

	gStore, err := gs.NewStore(kv, user)
	if err != nil {
		t.Errorf("Failed to create new group storage: %+v", err)
	}

	g := newTestGroup(getGroup(), getGroup().NewInt(42), prng, t)
	err = gStore.Add(g)
	if err != nil {
		t.Errorf("Failed to add group: %+v", err)
	}
	_ = kv.Prefix("GroupChatListStore").Delete("GroupChat/"+g.ID.String(), 0)

	expectedErr := strings.SplitN(newGroupStoreErr, "%", 2)[0]

	mockMess := newMockE2e(t, kv)
	_, err = NewManager(mockMess, nil, nil)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("NewManager did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// FIXME: the test storage.Session used for each manager currently uses the same
//  user. To fix this test, they need to use different users, which requires
//  modifying
// storage.InitTestingSession.
// func Test_manager_StartProcesses(t *testing.T) {
// 	jww.SetLogThreshold(jww.LevelTrace)
// 	jww.SetStdoutThreshold(jww.LevelTrace)
// 	prng := rand.New(rand.NewSource(42))
// 	requestChan1 := make(chan gs.Group)
// 	requestFunc1 := func(g gs.Group) { requestChan1 <- g }
// 	receiveChan1 := make(chan MessageReceive)
// 	receiveFunc1 := func(msg MessageReceive) { receiveChan1 <- msg }
// 	requestChan2 := make(chan gs.Group)
// 	requestFunc2 := func(g gs.Group) { requestChan2 <- g }
// 	receiveChan2 := make(chan MessageReceive)
// 	receiveFunc2 := func(msg MessageReceive) { receiveChan2 <- msg }
// 	requestChan3 := make(chan gs.Group)
// 	requestFunc3 := func(g gs.Group) { requestChan3 <- g }
// 	receiveChan3 := make(chan MessageReceive)
// 	receiveFunc3 := func(msg MessageReceive) { receiveChan3 <- msg }
//
// 	m1, _ := newTestManagerWithStore(prng, 10, 0, requestFunc1, receiveFunc1, t)
// 	m2, _ := newTestManagerWithStore(prng, 10, 0, requestFunc2, receiveFunc2, t)
// 	m3, _ := newTestManagerWithStore(prng, 10, 0, requestFunc3, receiveFunc3, t)
//
// 	membership, err := group.NewMembership(m1.store.GetTransmissionIdentity().Contact(),
// 		m2.store.GetTransmissionIdentity().Contact(), m3.store.GetTransmissionIdentity().Contact())
// 	if err != nil {
// 		t.Errorf("Failed to generate new membership: %+v", err)
// 	}
//
// 	dhKeys := gs.GenerateDhKeyList(m1.gs.GetTransmissionIdentity().ID,
// 		m1.store.GetTransmissionIdentity().E2eDhPrivateKey, membership, m1.store.E2e().GetGroup())
//
// 	grp1 := newTestGroup(m1.store.E2e().GetGroup(), m1.store.GetTransmissionIdentity().E2eDhPrivateKey, prng, t)
// 	grp1.Members = membership
// 	grp1.DhKeys = dhKeys
// 	grp1.ID = group.NewID(grp1.IdPreimage, grp1.Members)
// 	grp1.Key = group.NewKey(grp1.KeyPreimage, grp1.Members)
// 	grp2 := grp1.DeepCopy()
// 	grp2.DhKeys = gs.GenerateDhKeyList(m2.gs.GetTransmissionIdentity().ID,
// 		m2.store.GetTransmissionIdentity().E2eDhPrivateKey, membership, m2.store.E2e().GetGroup())
// 	grp3 := grp1.DeepCopy()
// 	grp3.DhKeys = gs.GenerateDhKeyList(m3.gs.GetTransmissionIdentity().ID,
// 		m3.store.GetTransmissionIdentity().E2eDhPrivateKey, membership, m3.store.E2e().GetGroup())
//
// 	err = m1.gs.Add(grp1)
// 	if err != nil {
// 		t.Errorf("Failed to add group to member 1: %+v", err)
// 	}
// 	err = m2.gs.Add(grp2)
// 	if err != nil {
// 		t.Errorf("Failed to add group to member 2: %+v", err)
// 	}
// 	err = m3.gs.Add(grp3)
// 	if err != nil {
// 		t.Errorf("Failed to add group to member 3: %+v", err)
// 	}
//
// 	_ = m1.StartProcesses()
// 	_ = m2.StartProcesses()
// 	_ = m3.StartProcesses()
//
// 	// Build request message
// 	requestMarshaled, err := proto.Marshal(&Request{
// 		Name:        grp1.Name,
// 		IdPreimage:  grp1.IdPreimage.Bytes(),
// 		KeyPreimage: grp1.KeyPreimage.Bytes(),
// 		Members:     grp1.Members.Serialize(),
// 		Message:     grp1.InitMessage,
// 	})
// 	if err != nil {
// 		t.Errorf("Failed to proto marshal message: %+v", err)
// 	}
// 	msg := message.Receive{
// 		Payload:     requestMarshaled,
// 		MessageType: message.GroupCreationRequest,
// 		Sender:      m1.gs.GetTransmissionIdentity().ID,
// 	}
//
// 	m2.swb.(*switchboard.Switchboard).Speak(msg)
// 	m3.swb.(*switchboard.Switchboard).Speak(msg)
//
// 	select {
// 	case received := <-requestChan2:
// 		if !reflect.DeepEqual(grp2, received) {
// 			t.Errorf("Failed to receive expected group on requestChan."+
// 				"\nexpected: %#v\nreceived: %#v", grp2, received)
// 		}
// 	case <-time.NewTimer(5 * time.Millisecond).C:
// 		t.Error("Timed out waiting for request callback.")
// 	}
//
// 	select {
// 	case received := <-requestChan3:
// 		if !reflect.DeepEqual(grp3, received) {
// 			t.Errorf("Failed to receive expected group on requestChan."+
// 				"\nexpected: %#v\nreceived: %#v", grp3, received)
// 		}
// 	case <-time.NewTimer(5 * time.Millisecond).C:
// 		t.Error("Timed out waiting for request callback.")
// 	}
//
// 	contents := []byte("Test group message.")
// 	timestamp := netTime.Now()
//
// 	// Create cMix message and get public message
// 	cMixMsg, err := m1.newCmixMsg(grp1, contents, timestamp, m2.gs.GetTransmissionIdentity(), prng)
// 	if err != nil {
// 		t.Errorf("Failed to create new cMix message: %+v", err)
// 	}
//
// 	internalMsg, _ := newInternalMsg(cMixMsg.ContentsSize() - publicMinLen)
// 	internalMsg.SetTimestamp(timestamp)
// 	internalMsg.SetSenderID(m1.gs.GetTransmissionIdentity().ID)
// 	internalMsg.SetPayload(contents)
// 	expectedMsgID := group.NewMessageID(grp1.ID, internalMsg.Marshal())
//
// 	expectedMsg := MessageReceive{
// 		GroupID:        grp1.ID,
// 		ID:             expectedMsgID,
// 		Payload:        contents,
// 		SenderID:       m1.gs.GetTransmissionIdentity().ID,
// 		RoundTimestamp: timestamp.Local(),
// 	}
//
// 	msg = message.Receive{
// 		Payload:        cMixMsg.Marshal(),
// 		MessageType:    message.Raw,
// 		Sender:         m1.gs.GetTransmissionIdentity().ID,
// 		RoundTimestamp: timestamp.Local(),
// 	}
// 	m2.swb.(*switchboard.Switchboard).Speak(msg)
//
// 	select {
// 	case received := <-receiveChan2:
// 		if !reflect.DeepEqual(expectedMsg, received) {
// 			t.Errorf("Failed to receive expected group on receiveChan."+
// 				"\nexpected: %+v\nreceived: %+v", expectedMsg, received)
// 		}
// 	case <-time.NewTimer(5 * time.Millisecond).C:
// 		t.Error("Timed out waiting for receive callback.")
// 	}
// }

// Unit test of manager.JoinGroup.
func Test_manager_JoinGroup(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, t)
	g := newTestGroup(m.getE2eGroup(), m.getE2eHandler().GetHistoricalDHPubkey(), prng, t)

	err := m.JoinGroup(g)
	if err != nil {
		t.Errorf("JoinGroup returned an error: %+v", err)
	}

	if _, exists := m.gs.Get(g.ID); !exists {
		t.Errorf("JoinGroup failed to add the group %s.", g.ID)
	}
}

// Error path: an error is returned when a group is joined twice.
func Test_manager_JoinGroup_AddError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, t)
	expectedErr := strings.SplitN(joinGroupErr, "%", 2)[0]

	err := m.JoinGroup(g)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("JoinGroup failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of manager.LeaveGroup.
func Test_manager_LeaveGroup(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, t)

	err := m.LeaveGroup(g.ID)
	if err != nil {
		t.Errorf("LeaveGroup returned an error: %+v", err)
	}

	if _, exists := m.GetGroup(g.ID); exists {
		t.Error("LeaveGroup failed to delete the group.")
	}
}

// Error path: an error is returned when no group with the ID exists.
func Test_manager_LeaveGroup_NoGroupError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, t)
	expectedErr := strings.SplitN(leaveGroupErr, "%", 2)[0]

	err := m.LeaveGroup(id.NewIdFromString("invalidID", id.Group, t))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("LeaveGroup failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of manager.GetGroups.
func Test_manager_GetGroups(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, t)

	list := m.GetGroups()
	for i, gid := range list {
		if err := m.gs.Remove(gid); err != nil {
			t.Errorf("Group %s does not exist (%d): %+v", gid, i, err)
		}
	}

	if m.gs.Len() != 0 {
		t.Errorf("GetGroups returned %d IDs, which is %d less than is in "+
			"memory.", len(list), m.gs.Len())
	}
}

// Unit test of manager.GetGroup.
func Test_manager_GetGroup(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, t)

	testGrp, exists := m.GetGroup(g.ID)
	if !exists {
		t.Error("GetGroup failed to find a group that should exist.")
	}

	if !reflect.DeepEqual(g, testGrp) {
		t.Errorf("GetGroup failed to return the expected group."+
			"\nexpected: %#v\nreceived: %#v", g, testGrp)
	}

	testGrp, exists = m.GetGroup(id.NewIdFromString("invalidID", id.Group, t))
	if exists {
		t.Errorf("GetGroup returned a group that should not exist: %#v", testGrp)
	}
}

// Unit test of manager.NumGroups. First a manager is created with 10 groups
// and the initial number is checked. Then the number of groups is checked after
// leaving each until the number left is 0.
func Test_manager_NumGroups(t *testing.T) {
	expectedNum := 10
	m, _ := newTestManagerWithStore(rand.New(rand.NewSource(42)), expectedNum, 0, nil, t)

	groups := append([]*id.ID{{}}, m.GetGroups()...)

	for i, gid := range groups {
		_ = m.LeaveGroup(gid)

		if m.NumGroups() != expectedNum-i {
			t.Errorf("NumGroups failed to return the expected number of "+
				"groups (%d).\nexpected: %d\nreceived: %d",
				i, expectedNum-i, m.NumGroups())
		}
	}
}
