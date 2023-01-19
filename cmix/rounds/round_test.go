////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"encoding/json"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"testing"
	"time"
)

func TestMakeRound(t *testing.T) {
	nid1 := id.NewIdFromString("test01", id.Node, t)
	now := uint64(netTime.Now().UnixNano())
	var timestamps = []uint64{
		now - 1000, now - 800, now - 600, now - 400, now - 200, now, now + 200}
	ri := &mixmessages.RoundInfo{
		ID:         5,
		UpdateID:   1,
		State:      2,
		BatchSize:  150,
		Topology:   [][]byte{nid1.Bytes()},
		Timestamps: timestamps,
		Errors: []*mixmessages.RoundError{{
			Id:     uint64(49),
			NodeId: nid1.Bytes(),
			Error:  "Test error",
		}},
		ResourceQueueTimeoutMillis: 0,
		AddressSpaceSize:           8,
	}
	expectedTimestamps := map[states.Round]time.Time{}
	for i, ts := range timestamps {
		expectedTimestamps[states.Round(i)] = time.Unix(0, int64(ts))
	}
	expected := Round{
		ID:               id.Round(ri.ID),
		State:            states.Round(ri.State),
		Topology:         connect.NewCircuit([]*id.ID{nid1}),
		Timestamps:       expectedTimestamps,
		BatchSize:        ri.BatchSize,
		AddressSpaceSize: uint8(ri.AddressSpaceSize),
		UpdateID:         ri.UpdateID,
		Raw:              ri,
	}
	r := MakeRound(ri)

	same := r.State == expected.State &&
		r.ID == expected.ID &&
		r.UpdateID == expected.UpdateID &&
		r.AddressSpaceSize == expected.AddressSpaceSize &&
		r.BatchSize == expected.BatchSize
	if !same {
		t.Fatalf("Basic info not identical.\nexpected: %+v\nreceived: %+v",
			expected, r)
	}
	for i := 0; i < r.Topology.Len(); i++ {
		same = same && r.Topology.GetNodeAtIndex(i).Cmp(nid1)
	}
	if !same {
		t.Fatalf("Topology info not identical.\nexpected: %+v\nreceived: %+v",
			expected.Topology, r.Topology)
	}
	for i := 0; i < len(r.Timestamps); i++ {
		same = same &&
			r.Timestamps[states.Round(i)] == expectedTimestamps[states.Round(i)]
	}
	if !same {
		t.Fatalf("Topology info not identical.\nexpected: %+v\nreceived: %+v",
			expected.Topology, r.Topology)
	}
	if r.Errors[0].Error != ri.Errors[0].Error {
		t.Fatalf("Error info not identical.\nexpected: %+v\nreceived: %+v",
			expected.Errors[0], r.Errors[0])
	}
}

func TestRound_GetEndTimestamp(t *testing.T) {
	nid1 := id.NewIdFromString("test01", id.Node, t)
	now := uint64(netTime.Now().UnixNano())
	var timestamps = []uint64{
		now - 1000, now - 800, now - 600, now - 400, now - 200, now, now + 200}
	ri := &mixmessages.RoundInfo{
		ID:                         5,
		UpdateID:                   1,
		State:                      0,
		BatchSize:                  150,
		Topology:                   [][]byte{nid1.Bytes()},
		Timestamps:                 timestamps,
		ResourceQueueTimeoutMillis: 0,
		AddressSpaceSize:           8,
	}
	r := MakeRound(ri)
	for i, ts := range timestamps {
		r.State = states.Round(i)
		expected := time.Unix(0, int64(ts))
		received := r.GetEndTimestamp()
		if received != expected {
			t.Errorf("Failed to get timestamp for state %s."+
				"\nexpected: %s\nreceived: %s",
				r.State, expected, r.GetEndTimestamp())
		}
	}
}

// Tests that a Round JSON marshalled and unmarshalled matches the original.
func TestRound_JsonMarshalUnmarshal(t *testing.T) {
	nid1 := id.NewIdFromString("test01", id.Node, t)
	now := uint64(netTime.Now().UnixNano())
	ri := &mixmessages.RoundInfo{
		ID:        5,
		UpdateID:  1,
		State:     2,
		BatchSize: 150,
		Topology:  [][]byte{nid1.Bytes()},
		Timestamps: []uint64{now - 1000, now - 800, now - 600, now - 400,
			now - 200, now, now + 200},
		Errors: []*mixmessages.RoundError{{
			Id:     uint64(49),
			NodeId: nid1.Bytes(),
			Error:  "Test error",
		}},
		ResourceQueueTimeoutMillis: 0,
		AddressSpaceSize:           8,
	}

	r := MakeRound(ri)

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Failed to JSON marshal Round: %+v", err)
	}

	var newRound Round
	err = json.Unmarshal(data, &newRound)
	if err != nil {
		t.Fatalf("Failed to JSON ummarshal Round: %+v", err)
	}

	if !reflect.DeepEqual(r, newRound) {
		t.Errorf("JSON marshalled and unmarshalled Round does not match "+
			"original.\nexpected: %#v\nreceived: %#v", r, newRound)
	}
}

// Tests that a Round with all nil and default fields can be JSON marshalled and
// unmarshalled.
func TestRound_JsonMarshalUnmarshal_Nil(t *testing.T) {
	r := Round{}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Failed to JSON marshal Round: %+v", err)
	}

	var newRound Round
	err = json.Unmarshal(data, &newRound)
	if err != nil {
		t.Fatalf("Failed to JSON ummarshal Round: %+v", err)
	}

	newRound.Raw = r.Raw
	if !reflect.DeepEqual(r, newRound) {
		t.Errorf("JSON marshalled and unmarshalled Round does not match "+
			"original.\nexpected: %#v\nreceived: %#v", r, newRound)
	}
}
