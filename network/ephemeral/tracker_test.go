///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ephemeral

import (
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Smoke test for Track function
func TestCheck(t *testing.T) {
	session := storage.InitTestingSession(t)
	identityStore := NewTracker(session.Reception())

	/// Store a mock initial timestamp the store
	now := time.Now()
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
	twoDaysTimestamp, err := MarshalTimestamp(twoDaysAgo)
	if err != nil {
		t.Errorf("Could not marshal timestamp for test setup: %v", err)
	}
	err = session.Set(TimestampKey, twoDaysTimestamp)
	if err != nil {
		t.Errorf("Could not set mock timestamp for test setup: %v", err)
	}

	ourId := id.NewIdFromBytes([]byte("Sauron"), t)
	stop := Track(session, ourId, identityStore)

	err = stop.Close(3 * time.Second)
	if err != nil {
		t.Errorf("Could not close thread: %v", err)
	}

}

// Unit test for track
func TestCheck_Thread(t *testing.T) {

	session := storage.InitTestingSession(t)
	ourId := id.NewIdFromBytes([]byte("Sauron"), t)
	stop := stoppable.NewSingle(ephemeralStoppable)
	identityStore := NewTracker(session.Reception())


	/// Store a mock initial timestamp the store
	now := time.Now()
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
	twoDaysTimestamp, err := MarshalTimestamp(twoDaysAgo)
	if err != nil {
		t.Errorf("Could not marshal timestamp for test setup: %v", err)
	}
	err = session.Set(TimestampKey, twoDaysTimestamp)
	if err != nil {
		t.Errorf("Could not set mock timestamp for test setup: %v", err)
	}

	// Run the tracker
	go func() {
		track(session, ourId, stop, identityStore)
	}()

	time.Sleep(1 * time.Second)

	// Manually generate identities
	eids, err := getUpcomingIDs(ourId, time.Now(), twoDaysAgo)
	if err != nil {
		t.Errorf("Could not generate upcoming ids: %v", err)
	}

	identities := generateIdentities(eids, ourId)

	// Check if store has been updated for new identities
	if !identityStore.IsAlreadyIdentity(identities[0]) {
		t.Errorf("Store was not updated for newly generated identies")
	}

	err = stop.Close(3 * time.Second)
	if err != nil {
		t.Errorf("Could not close thread: %v", err)
	}


}
