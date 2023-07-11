package notifications

import (
	"encoding/json"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

func TestManager_SetMaxState(t *testing.T) {
	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	expectedLen := int(Push) + 1

	comms.reset()
	cbChan := make(chan struct{})
	group := "a"
	m.RegisterUpdateCallback(group, func(group Group, created, edits, deletions []*id.ID, maxState NotificationState) {
		cbChan <- struct{}{}
	})

	// add notification registrations
	for i := Mute; i <= Push; i++ {
		for x := 0; x < int(i)+1; x++ {
			nid := id.NewIdFromUInt(uint64(int(i)*100+x), id.User, t)
			if err := m.Set(nid, group, []byte{0}, i); err != nil {
				t.Errorf("errored in set: %+v", err)
			}
		}
	}
	for receivedUpdates := 0; receivedUpdates < 3; {
		to := time.NewTimer(time.Second)
		select {
		case <-cbChan:
			receivedUpdates++
		case <-to.C:
			t.Fatalf("Failed to receive on cb chan")
		}
	}

	// test Push -> Mute
	if err := m.SetMaxState(Mute); err != nil {
		t.Fatalf("errored in setMaxState: %+v", err)
	}

	// should unregister all 3 push registrations
	unReg := comms.receivedMessage.(*pb.UnregisterTrackedIdRequest)
	if len(unReg.Request.TrackedIntermediaryID) != expectedLen {
		t.Errorf("wrong number of ids unregistered")
	}

	// check that the internal data is at the right values
	if mInternal.maxState != Mute {
		t.Errorf("max state at wrong state internally")
	}

	if loadMaxState(mInternal, t) != Mute {
		t.Errorf("max state at wrong state in ekv")
	}

	// test push -> whenOpen
	comms.reset()
	if err := m.SetMaxState(WhenOpen); err != nil {
		t.Fatalf("errored in setMaxState: %+v", err)
	}

	// no messages should have been sent because we were not
	// moving into or out of the push state
	if comms.receivedMessage != nil {
		t.Errorf("message sent when it shouldnt be!")
	}

	// check that the internal data is at the right values
	if mInternal.maxState != WhenOpen {
		t.Errorf("max state at wrong state internally")
	}

	if loadMaxState(mInternal, t) != WhenOpen {
		t.Errorf("max state at wrong state in ekv")
	}

	// test WhenOpen -> Push
	comms.reset()
	if err := m.SetMaxState(Push); err != nil {
		t.Fatalf("errored in setMaxState: %+v", err)
	}

	// test that the correct comm was sent, registration of
	// 3 push notifications
	reg := comms.receivedMessage.(*pb.RegisterTrackedIdRequest)
	if len(reg.Request.TrackedIntermediaryID) != expectedLen {
		t.Errorf("wrong number of ids unregistered")
	}

	// check that the internal data is at the right values
	if mInternal.maxState != Push {
		t.Errorf("max state at wrong state internally")
	}

	if loadMaxState(mInternal, t) != Push {
		t.Errorf("max state at wrong state in ekv")
	}
}

func TestManager_GetMaxState(t *testing.T) {
	m, _, _ := buildTestingManager(t)
	mInternal := m.(*manager)

	// set to every value and get that value and see if the
	// correct result returns
	for i := Mute; i <= Push; i++ {
		mInternal.maxState = i
		got := m.GetMaxState()
		if i != got {
			t.Errorf("get didnt match value")
		}
	}
}

// gets the max state from the ekv and unmarshals it for testing
func loadMaxState(m *manager, t *testing.T) NotificationState {
	obj, err := m.remote.Get(maxStateKey, maxStateKetVersion)
	if err != nil {
		t.Fatalf("could not get max state: %+v", err)
	}
	var ms NotificationState
	err = json.Unmarshal(obj.Data, &ms)
	if err != nil {
		t.Fatalf("could not get max state: %+v", err)
	}
	return ms
}
