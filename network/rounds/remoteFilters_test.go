///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
	bloom "gitlab.com/elixxir/bloomfilter"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

// Unit test NewRemoteFilter
func TestNewRemoteFilter(t *testing.T) {
	bloomFilter := &mixmessages.ClientBloom{
		Filter:     nil,
		FirstRound: 0,
		RoundRange: 0,
	}

	rf := NewRemoteFilter(bloomFilter)
	if !reflect.DeepEqual(rf.data, bloomFilter) {
		t.Fatalf("NewRemoteFilter() error: "+
			"RemoteFilter not initialized as expected."+
			"\n\tExpected: %v\n\tReceived: %v", bloomFilter, rf.data)
	}
}

// Unit test GetFilter
func TestRemoteFilter_GetFilter(t *testing.T) {
	testFilter, err := bloom.InitByParameters(interfaces.BloomFilterSize,
		interfaces.BloomFilterHashes)
	if err != nil {
		t.Fatalf("GetFilter error: "+
			"Cannot initialize bloom filter for setup: %v", err)
	}

	data, err := testFilter.MarshalBinary()
	if err != nil {
		t.Fatalf("GetFilter error: "+
			"Cannot marshal filter for setup: %v", err)
	}

	bloomFilter := &mixmessages.ClientBloom{
		Filter:     data,
		FirstRound: 0,
		RoundRange: 0,
	}

	rf := NewRemoteFilter(bloomFilter)
	retrievedFilter := rf.GetFilter()
	if !reflect.DeepEqual(retrievedFilter, testFilter) {
		t.Fatalf("GetFilter error: "+
			"Did not retrieve expected filter."+
			"\n\tExpected: %v\n\tReceived: %v", testFilter, retrievedFilter)
	}
}

// Unit test fro FirstRound and LastRound
func TestRemoteFilter_FirstLastRound(t *testing.T) {
	firstRound := uint64(25)
	roundRange := uint32(75)
	bloomFilter := &mixmessages.ClientBloom{
		Filter:     nil,
		FirstRound: firstRound,
		RoundRange: roundRange,
	}
	rf := NewRemoteFilter(bloomFilter)

	// Test FirstRound
	receivedFirstRound := rf.FirstRound()
	if receivedFirstRound != id.Round(firstRound) {
		t.Fatalf("FirstRound error: "+
			"Did not receive expected round."+
			"\n\tExpected: %v\n\tReceived: %v", firstRound, receivedFirstRound)
	}

	// Test LastRound
	receivedLastRound := rf.LastRound()
	if receivedLastRound != id.Round(firstRound+uint64(roundRange)) {
		t.Fatalf("LastRound error: "+
			"Did not receive expected round."+
			"\n\tExpected: %v\n\tReceived: %v", receivedLastRound, firstRound+uint64(roundRange))
	}

}

// In bounds test
func TestValidFilterRange(t *testing.T) {
	firstRound := uint64(25)
	roundRange := uint32(75)
	testFilter, err := bloom.InitByParameters(interfaces.BloomFilterSize,
		interfaces.BloomFilterHashes)
	if err != nil {
		t.Fatalf("GetFilter error: "+
			"Cannot initialize bloom filter for setup: %v", err)
	}

	data, err := testFilter.MarshalBinary()
	if err != nil {
		t.Fatalf("GetFilter error: "+
			"Cannot marshal filter for setup: %v", err)
	}

	// Construct an in bounds value
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)
	iu := reception.IdentityUse{
		Identity: reception.Identity{
			EphId:      expectedEphID,
			Source:     requestGateway,
			StartValid: time.Now().Add(-12 * time.Hour),
			EndValid:   time.Now().Add(24 * time.Hour),
		},
	}

	bloomFilter := &mixmessages.ClientBloom{
		Filter:     data,
		FirstRound: firstRound,
		RoundRange: roundRange,
	}

	// Fix for test on Windows machines: provides extra buffer between
	// time.Now() for the reception.Identity and the mixmessages.ClientBlooms
	time.Sleep(time.Millisecond)

	msg := &mixmessages.ClientBlooms{
		Period:         int64(12 * time.Hour),
		FirstTimestamp: time.Now().UnixNano(),
		Filters:        []*mixmessages.ClientBloom{bloomFilter},
	}

	start, end, outOfBounds := ValidFilterRange(iu, msg)
	if outOfBounds {
		t.Errorf("ValidFilterRange error: " +
			"Range should not be out of bounds")
	}

	if start != 0 && end != 1 {
		t.Errorf("ValidFilterRange error: "+
			"Unexpected indices returned. "+
			"\n\tExpected start: %v\n\tReceived start: %v"+
			"\n\tExpected end: %v\n\tReceived end: %v", 0, start, 1, end)
	}

}

// out of bounds test
func TestValidFilterRange_OutBounds(t *testing.T) {
	firstRound := uint64(25)
	roundRange := uint32(75)
	testFilter, err := bloom.InitByParameters(interfaces.BloomFilterSize,
		interfaces.BloomFilterHashes)
	if err != nil {
		t.Fatalf("GetFilter error: "+
			"Cannot initialize bloom filter for setup: %v", err)
	}

	data, err := testFilter.MarshalBinary()
	if err != nil {
		t.Fatalf("GetFilter error: "+
			"Cannot marshal filter for setup: %v", err)
	}

	// Construct an in bounds value
	expectedEphID := ephemeral.Id{1, 2, 3, 4, 5, 6, 7, 8}
	requestGateway := id.NewIdFromString(ReturningGateway, id.Gateway, t)
	iu := reception.IdentityUse{
		Identity: reception.Identity{
			EphId:      expectedEphID,
			Source:     requestGateway,
			StartValid: time.Now().Add(-24 * time.Hour),
			EndValid:   time.Now().Add(-36 * time.Hour),
		},
	}

	bloomFilter := &mixmessages.ClientBloom{
		Filter:     data,
		FirstRound: firstRound,
		RoundRange: roundRange,
	}

	msg := &mixmessages.ClientBlooms{
		Period:         int64(12 * time.Hour),
		FirstTimestamp: time.Now().UnixNano(),
		Filters:        []*mixmessages.ClientBloom{bloomFilter},
	}

	_, _, outOfBounds := ValidFilterRange(iu, msg)
	if !outOfBounds {
		t.Errorf("ValidFilterRange error: " +
			"Range should be out of bounds")
	}

}
