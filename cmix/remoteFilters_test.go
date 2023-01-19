////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	jww "github.com/spf13/jwalterweatherman"
	bloom "gitlab.com/elixxir/bloomfilter"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

// Unit test NewRemoteFilter.
func TestNewRemoteFilter(t *testing.T) {
	bloomFilter := &mixmessages.ClientBloom{
		Filter:     nil,
		FirstRound: 0,
		RoundRange: 0,
	}

	rf := NewRemoteFilter(bloomFilter)
	if !reflect.DeepEqual(rf.data, bloomFilter) {
		t.Fatalf("RemoteFilter not initialized as expected."+
			"\nexpected: %+v\nreceived: %+v", bloomFilter, rf.data)
	}
}

// Unit test RemoteFilter.GetFilter.
func TestRemoteFilter_GetFilter(t *testing.T) {
	testFilter, err := bloom.InitByParameters(BloomFilterSize,
		BloomFilterHashes)
	if err != nil {
		t.Fatalf("Cannot initialize bloom filter for setup: %+v", err)
	}

	data, err := testFilter.MarshalBinary()
	if err != nil {
		t.Fatalf("Cannot marshal filter for setup: %+v", err)
	}

	bloomFilter := &mixmessages.ClientBloom{
		Filter:     data,
		FirstRound: 0,
		RoundRange: 0,
	}

	rf := NewRemoteFilter(bloomFilter)
	retrievedFilter := rf.GetFilter()
	if !reflect.DeepEqual(retrievedFilter, testFilter) {
		t.Fatalf("Did not retrieve expected filter."+
			"\nexpected: %+v\nreceived: %+v", testFilter, retrievedFilter)
	}
}

// Unit test for RemoteFilter.FirstRound and RemoteFilter.LastRound.
func TestRemoteFilter_FirstRound_LastRound(t *testing.T) {
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
		t.Fatalf("Did not receive expected round.\nexpected: %d\nreceived: %d",
			firstRound, receivedFirstRound)
	}

	// Test LastRound
	receivedLastRound := rf.LastRound()
	if receivedLastRound != id.Round(firstRound+uint64(roundRange)) {
		t.Fatalf("Did not receive expected round.\nexpected: %d\nreceived: %d",
			receivedLastRound, firstRound+uint64(roundRange))
	}

}
