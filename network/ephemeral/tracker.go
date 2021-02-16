///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ephemeral

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

const validityGracePeriod = 5 * time.Minute
const TimestampKey = "IDTrackingTimestamp"
const ephemeralStoppable = "EphemeralCheck"

// Track runs a thread which checks for past and present ephemeral ids
func Track(session *storage.Session, ourId *id.ID) stoppable.Stoppable {
	stop := stoppable.NewSingle(ephemeralStoppable)

	go track(session, ourId, stop)

	return stop
}

// track is a thread which continuously processes ephemeral ids.
// If any error occurs, the thread crashes
func track(session *storage.Session, ourId *id.ID, stop *stoppable.Single) {

	// Check that there is a timestamp in store at all
	err := checkTimestampStore(session)
	if err != nil {
		jww.FATAL.Panicf("Could not store timestamp "+
			"for ephemeral ID tracking: %v", err)
	}

	// Get the latest timestamp from store
	lastTimestampObj, err := session.Get(TimestampKey)
	if err != nil {
		jww.FATAL.Panicf("Could not get timestamp: %v", err)
	}

	lastCheck, err := unmarshalTimestamp(lastTimestampObj)
	if err != nil {
		jww.FATAL.Panicf("Could not parse stored timestamp: %v", err)
	}

	// Wait until we get the id size from the network
	receptionStore := session.Reception()
	receptionStore.WaitForIdSizeUpdate()

	for true {
		now := time.Now()
		// Generates the IDs since the last track
		protoIds, err := ephemeral.GetIdsByRange(ourId, receptionStore.GetIDSize(),
			now, now.Sub(lastCheck))

		jww.DEBUG.Printf("Now: %d, LastCheck: %d (%v), Different: %v",
			now.UnixNano(), lastCheck, lastCheck, now.Sub(lastCheck))

		jww.DEBUG.Printf("protoIds Count: %d", len(protoIds))

		if err != nil {
			jww.FATAL.Panicf("Could not generate "+
				"upcoming IDs: %v", err)
		}

		// Generate identities off of that list
		identities := generateIdentities(protoIds, ourId)

		jww.INFO.Printf("Number of Identities Generated: %d",
			len(identities))

		jww.INFO.Printf("Current Identity: %d (source: %s), Start: %s, End: %s",
			identities[len(identities)-1].EphId.Int64(), identities[len(identities)-1].Source,
			identities[len(identities)-1].StartValid, identities[len(identities)-1].EndValid)

		// Add identities to storage if unique
		for _, identity := range identities {
			if err = receptionStore.AddIdentity(identity); err != nil {
				jww.FATAL.Panicf("Could not insert "+
					"identity: %v", err)
			}
		}

		// Generate the time stamp for storage
		vo, err := marshalTimestamp(now)
		if err != nil {
			jww.FATAL.Panicf("Could not marshal "+
				"timestamp for storage: %v", err)

		}

		// Store the timestamp
		if err = session.Set(TimestampKey, vo); err != nil {
			jww.FATAL.Panicf("Could not store timestamp: %v", err)
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
		// Expand the grace period for both start and end
		eid.End.Add(validityGracePeriod)
		eid.Start.Add(-validityGracePeriod)
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

// Sanitation check of timestamp store. If a value has not been stored yet
// then the current time is stored
func checkTimestampStore(session *storage.Session) error {
	if _, err := session.Get(TimestampKey); err != nil {
		// only generate from the last hour because this is a new id, it
		// couldn't receive messages yet
		now, err := marshalTimestamp(time.Now().Add(-1 * time.Hour))
		if err != nil {
			return errors.Errorf("Could not marshal new timestamp for storage: %v", err)
		}
		return session.Set(TimestampKey, now)
	}

	return nil
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
func marshalTimestamp(timeToStore time.Time) (*versioned.Object, error) {
	data, err := timeToStore.MarshalBinary()

	return &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      data,
	}, err
}

// Helper function which calculates the time for the ticker based
// off of the last ephemeral ID to expire
func calculateTickerTime(baseIDs []ephemeral.ProtoIdentity) time.Duration {
	if len(baseIDs) == 0 {
		return time.Duration(0)
	}
	// Get the last identity in the list
	lastIdentity := baseIDs[len(baseIDs)-1]

	// Factor out the grace period previously expanded upon.
	// Calculate and return that duration
	gracePeriod := lastIdentity.End.Add(-validityGracePeriod)
	return lastIdentity.End.Sub(gracePeriod)
}
