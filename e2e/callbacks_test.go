////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
)

// Tests that newPartnerCallbacks returns the expected new partnerCallbacks.
func Test_newPartnerCallbacks(t *testing.T) {
	expected := &partnerCallbacks{
		callbacks: make(map[id.ID]Callbacks),
	}

	pcb := newPartnerCallbacks()

	if !reflect.DeepEqual(expected, pcb) {
		t.Errorf("Did not get expected new partnerCallbacks."+
			"\nexpected: %+v\nreceived: %+v", expected, pcb)
	}
}

// Tests that partnerCallbacks.add adds all the expected callbacks to the map.
func Test_partnerCallbacks_add(t *testing.T) {
	pcb := newPartnerCallbacks()

	const n = 10
	expected := make(map[id.ID]Callbacks, n)
	for i := uint64(0); i < n; i++ {
		expected[*id.NewIdFromUInt(i, id.User, t)] = &mockCallbacks{id: i}
	}

	for partnerID, cbs := range expected {
		pcb.add(&partnerID, cbs)
	}

	if !reflect.DeepEqual(expected, pcb.callbacks) {
		t.Errorf("Callback list does not match expected."+
			"\nexpected: %v\nreceived: %v", expected, pcb.callbacks)
	}
}

// Tests that partnerCallbacks.delete removes all callbacks from the map.
func Test_partnerCallbacks_delete(t *testing.T) {
	pcb := newPartnerCallbacks()

	const n = 10
	expected := make(map[id.ID]Callbacks, n)
	for i := uint64(0); i < n; i++ {
		partnerID, cbs := id.NewIdFromUInt(i, id.User, t), &mockCallbacks{id: i}
		expected[*partnerID] = cbs
		pcb.add(partnerID, cbs)
	}

	for partnerID := range expected {
		pcb.delete(&partnerID)
	}

	if len(pcb.callbacks) > 0 {
		t.Errorf("Callback map not empty: %v", pcb.callbacks)
	}
}

// Tests that partnerCallbacks.get returns the expected Callbacks for each
// partner ID.
func Test_partnerCallbacks_get(t *testing.T) {
	pcb := newPartnerCallbacks()

	const n = 10
	expected := make(map[id.ID]Callbacks, n)
	for i := uint64(0); i < n; i++ {
		partnerID, cbs := id.NewIdFromUInt(i, id.User, t), &mockCallbacks{id: i}
		expected[*partnerID] = cbs
		pcb.add(partnerID, cbs)
	}

	for partnerID, expectedCbs := range expected {
		cbs := pcb.get(&partnerID)
		if !reflect.DeepEqual(expectedCbs, cbs) {
			t.Errorf("Callbacks for parter %s do not match."+
				"\nexpected: %+v\nreceived: %+v", &partnerID, expectedCbs, cbs)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Mock Callbacks                                                             //
////////////////////////////////////////////////////////////////////////////////

// Verify that mockCallbacks adhere to the Callbacks interface
var _ Callbacks = (*mockCallbacks)(nil)

// mockCallbacks is a structure used for testing that adheres to the Callbacks
// interface.
type mockCallbacks struct {
	id                   uint64
	connectionClosedChan chan *id.ID
}

func (m *mockCallbacks) ConnectionClosed(partner *id.ID, _ rounds.Round) {
	m.connectionClosedChan <- partner
}
