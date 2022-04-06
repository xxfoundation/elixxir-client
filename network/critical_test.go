package network

import (
	"errors"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/comms/mixmessages"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"testing"
	"time"
)

type mockMonitor struct{}

func (m *mockMonitor) AddHealthCallback(f func(bool)) uint64 {
	return 0
}
func (m *mockMonitor) RemoveHealthCallback(uint64) {
	return
}
func (m *mockMonitor) IsHealthy() bool {
	return true
}
func (m *mockMonitor) WasHealthy() bool {
	return true
}
func (m *mockMonitor) StartProcesses() (stoppable.Stoppable, error) {
	return stoppable.NewSingle("t"), nil
}

type mockRegistrar struct {
	statusReturn bool
}

func (mr *mockRegistrar) AddRoundEventChan(rid id.Round, eventChan chan ds.EventReturn,
	timeout time.Duration, validStates ...states.Round) *ds.EventCallback {
	eventChan <- ds.EventReturn{
		RoundInfo: &mixmessages.RoundInfo{
			ID:                         2,
			UpdateID:                   0,
			State:                      0,
			BatchSize:                  0,
			Topology:                   nil,
			Timestamps:                 nil,
			Errors:                     nil,
			ClientErrors:               nil,
			ResourceQueueTimeoutMillis: 0,
			Signature:                  nil,
			AddressSpaceSize:           0,
			EccSignature:               nil,
		},
		TimedOut: mr.statusReturn,
	}
	return &ds.EventCallback{}
}

func mockCriticalSender(msg format.Message, recipient *id.ID,
	params CMIXParams) (id.Round, ephemeral.Id, error) {
	return id.Round(1), ephemeral.Id{}, nil
}

func mockFailCriticalSender(msg format.Message, recipient *id.ID,
	params CMIXParams) (id.Round, ephemeral.Id, error) {
	return id.Round(1), ephemeral.Id{}, errors.New("Test error")
}

// TestCritical tests the basic functions of the critical messaging system
func TestCritical(t *testing.T) {
	// Init mock structures & start thread
	kv := versioned.NewKV(ekv.Memstore{})
	mr := &mockRegistrar{
		statusReturn: true,
	}
	c := newCritical(kv, &mockMonitor{}, mr, mockCriticalSender)
	s := stoppable.NewSingle("test")
	go c.runCriticalMessages(s)

	// Case 1 - should fail
	recipientID := id.NewIdFromString("zezima", id.User, t)
	c.Add(format.NewMessage(2048), recipientID, GetDefaultCMIXParams())
	c.trigger <- true
	time.Sleep(500 * time.Millisecond)

	// Case 2 - should succeed
	mr.statusReturn = false
	c.Add(format.NewMessage(2048), recipientID, GetDefaultCMIXParams())
	c.trigger <- true
	time.Sleep(500 * time.Millisecond)

	// Case 3 - should fail
	c.send = mockFailCriticalSender
	c.Add(format.NewMessage(2048), recipientID, GetDefaultCMIXParams())
	c.trigger <- true
	time.Sleep(time.Second)
	err := s.Close()
	if err != nil {
		t.Errorf("Failed to stop critical: %+v", err)
	}
}
