package notifications

import (
	"encoding/json"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestManager_SetMaxState(t *testing.T) {
	m, _, comms := buildTestingManager(t)
	mInternal := m.(*manager)

	expectedLen := int(Push) + 1

	comms.reset()

	for i := Mute; i <= Push; i++ {
		for x := 0; x < int(i)+1; x++ {
			nid := id.NewIdFromUInt(uint64(int(i)*100+x), id.User, t)
			if err := m.Set(nid, "a", []byte{0}, i); err != nil {
				t.Errorf("errored in set: %+v", err)
			}
		}
	}

	if err := m.SetMaxState(Mute); err != nil {
		t.Fatalf("errored in setMaxState: %+v", err)
	}

	unReg := comms.receivedMessage.(*pb.UnregisterTrackedIdRequest)
	if len(unReg.Request.TrackedIntermediaryID) != expectedLen {
		t.Errorf("wrong number of ids unregistered")
	}

	if mInternal.maxState != Mute {
		t.Errorf("max state at wrong state internally")
	}

	if loadMaxState(mInternal, t) != Mute {
		t.Errorf("max state at wrong state in ekv")
	}

	comms.reset()
	if err := m.SetMaxState(WhenOpen); err != nil {
		t.Fatalf("errored in setMaxState: %+v", err)
	}
	if comms.receivedMessage != nil {
		t.Errorf("message sent when it shouldnt be!")
	}
	if mInternal.maxState != WhenOpen {
		t.Errorf("max state at wrong state internally")
	}

	if loadMaxState(mInternal, t) != WhenOpen {
		t.Errorf("max state at wrong state in ekv")
	}
	comms.reset()
	if err := m.SetMaxState(Push); err != nil {
		t.Fatalf("errored in setMaxState: %+v", err)
	}

	reg := comms.receivedMessage.(*pb.RegisterTrackedIdRequest)
	if len(reg.Request.TrackedIntermediaryID) != expectedLen {
		t.Errorf("wrong number of ids unregistered")
	}

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

	for i := Mute; i <= Push; i++ {
		mInternal.maxState = i
		got := m.GetMaxState()
		if i != got {
			t.Errorf("get didnt match value")
		}
	}
}

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
