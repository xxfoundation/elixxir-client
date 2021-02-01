///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ephemeral

import (
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

const ephemeralIdSie = 64
const validityGracePeriod = 5 * time.Minute
const TimestampKey = "IDTrackingTimestamp"
const ephemeralStoppable = "EphemeralCheck"

// Track runs a thread which checks for past and present ephemeral ids
func Track(session *storage.Session, ourId *id.ID, identityStore *IdentityStore) stoppable.Stoppable {
	stop := stoppable.NewSingle(ephemeralStoppable)

	go track(session, ourId, stop, identityStore)

	return stop
}

// track is a thread which continuously processes ephemeral ids.
// If any error occurs, the thread crashes
func track(session *storage.Session, ourId *id.ID, stop *stoppable.Single, identityStore *IdentityStore) {
	// Get the latest timestamp from store
	lastTimestampObj, err := session.Get(TimestampKey)
	if err != nil {
		globals.Log.FATAL.Panicf("Could not get timestamp: %v", err)
	}

	lastCheck, err := unmarshalTimestamp(lastTimestampObj)
	if err != nil {
		globals.Log.FATAL.Panicf("Could not parse stored timestamp: %v", err)
	}

	for true {
		// Generates the IDs since the last track
		now := time.Now()
		protoIds, err := getUpcomingIDs(ourId, now, lastCheck)
		if err != nil {
			globals.Log.FATAL.Panicf("Could not generate "+
				"upcoming IDs: %v", err)
		}

		// Generate identities off of that list
		identities := generateIdentities(protoIds, ourId)

		// Add identities to storage if unique
		for _, identity := range identities {
			// Track if identity has been generated already
			if !identityStore.IsAlreadyIdentity(identity) {
				// If not not, insert identity into store
				if err = identityStore.InsertIdentity(identity); err != nil {
					globals.Log.FATAL.Panicf("Could not insert "+
						"identity: %v", err)
				}
			}

		}

		// Generate the time stamp for storage
		vo, err := MarshalTimestamp(now)
		if err != nil {
			globals.Log.FATAL.Panicf("Could not marshal "+
				"timestamp for storage: %v", err)

		}

		// Store the timestamp
		if err = session.Set(TimestampKey, vo); err != nil {
			globals.Log.FATAL.Panicf("Could not store timestamp: %v", err)
		}

		// Sleep until the last Id has expired
		timeToSleep := calculateTickerTime(protoIds)
		t := time.NewTimer(timeToSleep)
		select {
		case <-t.C:
		case <-stop.Quit():
			return
		}
	}
}

// generateIdentities is a constructor which generates a list of
// identities off of the list of protoIdentities passed in
func generateIdentities(protoIds []ephemeral.ProtoIdentity,
	ourId *id.ID) []reception.Identity {

	identities := make([]reception.Identity, 0)

	// Add identities for every ephemeral id
	for _, eid := range protoIds {
		// Expand the grace period
		eid.End.Add(validityGracePeriod)

		identities = append(identities, reception.Identity{
			EphId:      eid.Id,
			Source:     ourId,
			End:        eid.End,
			StartValid: eid.Start,
			EndValid:   eid.End,
			Ephemeral:  false,
		})

	}

	return identities
}

// Takes the stored timestamp and unmarshal into a time object
func unmarshalTimestamp(lastTimestampObj *versioned.Object) (time.Time, error) {
	if lastTimestampObj == nil || lastTimestampObj.Data == nil {
		return time.Now(), nil
	}

	lastTimestamp := time.Time{}
	err := lastTimestamp.UnmarshalBinary(lastTimestampObj.Data)
	return lastTimestamp, err
}

// Marshals the timestamp for ekv storage. Generates a storable object
func MarshalTimestamp(timeToStore time.Time) (*versioned.Object, error) {
	data, err := timeToStore.MarshalBinary()

	return &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      data,
	}, err
}

// Wrapper for GetIdsByRange. Generates ephemeral ids in the time period
// since the last track
func getUpcomingIDs(ourId *id.ID, now,
	lastCheck time.Time) ([]ephemeral.ProtoIdentity, error) {
	return ephemeral.GetIdsByRange(ourId, ephemeralIdSie,
		now.UnixNano(), now.Sub(lastCheck))

}

// Helper function which calculates the time for the ticker based
// off of the last ephemeral ID to expire
func calculateTickerTime(baseIDs []ephemeral.ProtoIdentity) time.Duration {
	// Get the last identity in the list
	indx := 0
	if len(baseIDs)-1 >= 0 {
		indx = len(baseIDs) - 1
	}
	lastIdentity := baseIDs[indx]

	// Factor out the grace period previously expanded upon.
	// Calculate and return that duration
	gracePeriod := lastIdentity.End.Add(-validityGracePeriod)
	return lastIdentity.End.Sub(gracePeriod)
}
