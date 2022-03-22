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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

const validityGracePeriod = 5 * time.Minute
const TimestampKey = "IDTrackingTimestamp"
const TimestampStoreVersion = 0
const ephemeralStoppable = "EphemeralCheck"
const addressSpaceSizeChanTag = "ephemeralTracker"

// Track runs a thread which checks for past and present ephemeral ID.
func Track(session *storage.Session, addrSpace *AddressSpace, ourId *id.ID) stoppable.Stoppable {
	stop := stoppable.NewSingle(ephemeralStoppable)

	go track(session, addrSpace, ourId, stop)

	return stop
}

// track is a thread which continuously processes ephemeral IDs. Panics if any
// error occurs.
func track(session *storage.Session, addrSpace *AddressSpace, ourId *id.ID, stop *stoppable.Single) {

	// Check that there is a timestamp in store at all
	err := checkTimestampStore(session)
	if err != nil {
		jww.FATAL.Panicf("Could not store timestamp for ephemeral ID "+
			"tracking: %+v", err)
	}

	// get the latest timestamp from store
	lastTimestampObj, err := session.Get(TimestampKey)
	if err != nil {
		jww.FATAL.Panicf("Could not get timestamp: %+v", err)
	}

	lastCheck, err := unmarshalTimestamp(lastTimestampObj)
	if err != nil {
		jww.FATAL.Panicf("Could not parse stored timestamp: %+v", err)
	}

	// Wait until we get the ID size from the network
	receptionStore := session.Reception()
	addrSpace.UnregisterNotification(addressSpaceSizeChanTag)
	addressSizeUpdate, err := addrSpace.RegisterNotification(addressSpaceSizeChanTag)
	if err != nil {
		jww.FATAL.Panicf("failed to register address size notification "+
			"channel: %+v", err)
	}
	addressSize := addrSpace.Get()

	for {
		now := netTime.Now()

		// Hack for inconsistent time on android
		if now.Before(lastCheck) || now.Equal(lastCheck) {
			now = lastCheck.Add(time.Nanosecond)
		}

		// Generates the IDs since the last track
		protoIds, err := ephemeral.GetIdsByRange(
			ourId, uint(addressSize), lastCheck, now.Add(validityGracePeriod).Sub(lastCheck))

		jww.DEBUG.Printf("Now: %s, LastCheck: %s, Different: %s",
			now, lastCheck, now.Sub(lastCheck))
		jww.DEBUG.Printf("protoIds Count: %d", len(protoIds))

		if err != nil {
			jww.FATAL.Panicf("Could not generate upcoming IDs: %+v", err)
		}

		// Generate identities off of that list
		identities := generateIdentities(protoIds, ourId, addressSize)

		jww.INFO.Printf("Number of Identities Generated: %d", len(identities))
		jww.INFO.Printf("Current Identity: %d (source: %s), Start: %s, End: %s",
			identities[len(identities)-1].EphId.Int64(),
			identities[len(identities)-1].Source,
			identities[len(identities)-1].StartValid,
			identities[len(identities)-1].EndValid)

		// Add identities to storage, if unique
		for _, identity := range identities {
			if err = receptionStore.AddIdentity(identity); err != nil {
				jww.FATAL.Panicf("Could not insert identity: %+v", err)
			}
		}

		// Generate the timestamp for storage
		vo, err := marshalTimestamp(now)
		if err != nil {
			jww.FATAL.Panicf("Could not marshal timestamp for storage: %+v", err)

		}
		lastCheck = now

		// Store the timestamp
		if err = session.Set(TimestampKey, vo); err != nil {
			jww.FATAL.Panicf("Could not store timestamp: %+v", err)
		}

		// Sleep until the last ID has expired
		timeToSleep := calculateTickerTime(protoIds, now)
		select {
		case <-time.NewTimer(timeToSleep).C:
		case addressSize = <-addressSizeUpdate:
			receptionStore.SetToExpire(addressSize)
		case <-stop.Quit():
			addrSpace.UnregisterNotification(addressSpaceSizeChanTag)
			stop.ToStopped()
			return
		}
	}
}

// generateIdentities generates a list of identities off of the list of passed
// in ProtoIdentity.
func generateIdentities(protoIds []ephemeral.ProtoIdentity, ourId *id.ID,
	addressSize uint8) []reception.Identity {

	identities := make([]reception.Identity, len(protoIds))

	// Add identities for every ephemeral ID
	for i, eid := range protoIds {
		// Expand the grace period for both start and end
		identities[i] = reception.Identity{
			EphId:       eid.Id,
			Source:      ourId,
			AddressSize: addressSize,
			End:         eid.End,
			StartValid:  eid.Start.Add(-validityGracePeriod),
			EndValid:    eid.End.Add(validityGracePeriod),
			Ephemeral:   false,
			ExtraChecks: interfaces.DefaultExtraChecks,
		}

	}

	return identities
}

// checkTimestampStore performs a sanitation check of timestamp store. If a
// value has not been stored yet, then the current time is stored.
func checkTimestampStore(session *storage.Session) error {
	if _, err := session.Get(TimestampKey); err != nil {
		// Only generate from the last hour because this is a new ID; it could
		// not yet receive messages
		now, err := marshalTimestamp(netTime.Now().Add(-1 * time.Hour))
		if err != nil {
			return errors.Errorf("Could not marshal new timestamp for "+
				"storage: %+v", err)
		}

		return session.Set(TimestampKey, now)
	}

	return nil
}

// unmarshalTimestamp unmarshal the stored timestamp into a time.Time.
func unmarshalTimestamp(lastTimestampObj *versioned.Object) (time.Time, error) {
	if lastTimestampObj == nil || lastTimestampObj.Data == nil {
		return netTime.Now(), nil
	}

	lastTimestamp := time.Time{}
	err := lastTimestamp.UnmarshalBinary(lastTimestampObj.Data)
	return lastTimestamp, err
}

// marshalTimestamp marshals the timestamp and generates a storable object for
// ekv storage.
func marshalTimestamp(timeToStore time.Time) (*versioned.Object, error) {
	data, err := timeToStore.MarshalBinary()

	return &versioned.Object{
		Version:   TimestampStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}, err
}

// calculateTickerTime calculates the time for the ticker based off of the last
// ephemeral ID to expire.
func calculateTickerTime(baseIDs []ephemeral.ProtoIdentity, now time.Time) time.Duration {
	if len(baseIDs) == 0 {
		return time.Duration(0)
	}

	// get the last identity in the list
	lastIdentity := baseIDs[len(baseIDs)-1]

	// Factor out the grace period previously expanded upon
	// Calculate and return that duration
	gracePeriod := lastIdentity.End.Add(-1 * validityGracePeriod)
	return gracePeriod.Sub(now)
}
