package rounds

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

func TestMakeRound(t *testing.T) {
	nid1 := id.NewIdFromString("test01", id.Node, t)
	now := uint64(time.Now().UnixNano())
	var timestamps = []uint64{now - 1000, now - 800, now - 600, now - 400, now - 200, now, now + 200}
	ri := &mixmessages.RoundInfo{
		ID:         5,
		UpdateID:   1,
		State:      2,
		BatchSize:  150,
		Topology:   [][]byte{nid1.Bytes()},
		Timestamps: timestamps,
		Errors: []*mixmessages.RoundError{
			{
				Id:     uint64(49),
				NodeId: nid1.Bytes(),
				Error:  "Test error",
			},
		},
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

	same := r.State == expected.State && r.ID == expected.ID && r.UpdateID == expected.UpdateID && r.AddressSpaceSize == expected.AddressSpaceSize && r.BatchSize == expected.BatchSize
	if !same {
		t.Errorf("Basic info not identical\n\tExpected: %+v\n\tReceived: %+v\n", expected, r)
		t.FailNow()
	}
	for i := 0; i < r.Topology.Len(); i++ {
		same = same && r.Topology.GetNodeAtIndex(i).Cmp(nid1)
	}
	if !same {
		t.Errorf("Topology info not identical\n\tExpected: %+v\n\tReceived: %+v\n", expected.Topology, r.Topology)
		t.FailNow()
	}
	for i := 0; i < len(r.Timestamps); i++ {
		same = same && r.Timestamps[states.Round(i)] == expectedTimestamps[states.Round(i)]
	}
	if !same {
		t.Errorf("Topology info not identical\n\tExpected: %+v\n\tReceived: %+v\n", expected.Topology, r.Topology)
		t.FailNow()
	}
	if r.Errors[0].Error != ri.Errors[0].Error {
		t.Errorf("Error info not identical\n\tExpected: %+v\n\tReceived: %+v\n", expected.Errors[0], r.Errors[0])
		t.FailNow()
	}
}

func TestRound_GetEndTimestamp(t *testing.T) {
	nid1 := id.NewIdFromString("test01", id.Node, t)
	now := uint64(time.Now().UnixNano())
	var timestamps = []uint64{now - 1000, now - 800, now - 600, now - 400, now - 200, now, now + 200}
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
			t.Errorf("Failed to get timestamp for state %s\n\tReceived: %s\n\tExpected: %s\n", r.State, r.GetEndTimestamp(), expected)
		}
	}

}
