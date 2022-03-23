///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package identity

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/network/address"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"sort"
	"sync"
	"time"
)

const validityGracePeriod = 5 * time.Minute
const TrackerListKey = "TrackerListKey"
const TrackerListVersion = 0
const TimestampKey = "IDTrackingTimestamp"
const TimestampStoreVersion = 0
const ephemeralStoppable = "EphemeralCheck"
const addressSpaceSizeChanTag = "ephemeralTracker"
var Forever time.Time = time.Time{}


type Tracker struct{
	tracked []trackedID
	store *receptionID.Store
	session *storage.Session
	newIdentity chan trackedID
	deleteIdentity chan *id.ID
}

type trackedID struct{
	NextGeneration time.Time
	LastGeneration time.Time
	Source         *id.ID
	ValidUntil     time.Time
	Persistent     bool
}

func NewTracker(session *storage.Session, addrSpace *address.Space)*Tracker{
	//intilization
	//loading
	//if load fails, make a new
		//check if there is an old timestamp, recreate the basic ID from it
	//initilize the ephemeral identiies system
}

//make the GetIdentities call on the ephemerals system public on this

// Track runs a thread which checks for past and present address ID.
func (tracker Tracker)StartProcessies() stoppable.Stoppable {
	stop := stoppable.NewSingle(ephemeralStoppable)

	go track(session, addrSpace, ourId, stop)

	return stop
}



// AddIdentity adds an identity to be tracked
func (tracker *Tracker)AddIdentity(id *id.ID, validUntil time.Time, persistent bool){
	tracker.newIdentity<-trackedID{
		NextGeneration: netTime.Now().Add(-time.Second),
		LastGeneration: time.Time{},
		Source: id,
		ValidUntil: validUntil,
		Persistent: persistent,
	}
}

// RemoveIdentity removes a currently tracked identity.
func (tracker *Tracker) RemoveIdentity(id *id.ID){
	tracker.deleteIdentity<-id
}

func (tracker *Tracker)track(session *storage.Session, addrSpace *address.Space, ourId *id.ID, stop *stoppable.Single) {

	// Wait until we get the ID size from the network
	addressSize := addrSpace.Get()

	/*wait for next event*/
	trackerLoop:
	for {
		edits := false
		var toRemove map[int]struct{}
		nextEvent :=  tracker.tracked[0].ValidUntil

		//loop through every tracked ID and see if any operatrions are needed
		for i := range tracker.tracked {
			inQuestion := tracker.tracked[i]

			//generate new ephmerals if is time for it
			if netTime.Now().After(inQuestion.NextGeneration) {
				edits = true

				//ensure that ephemerals will not be generated after the identity is invalid
				generateUntil := inQuestion.NextGeneration
				if inQuestion.ValidUntil !=Forever && generateUntil.After(inQuestion.ValidUntil){
					generateUntil = inQuestion.ValidUntil
				}
				//generate all not yet existing ephemerals
				identities, nextNextGeneration := generateIdentitiesOverRange(inQuestion.LastGeneration,
					inQuestion.NextGeneration, inQuestion.Source, addressSize)
				//add all ephemerals to the ephemeral handler
				for _, identity := range identities {
					// move up the end time if the source identity is invalid before the natural end
					// of the ephemeral identity.
					if inQuestion.ValidUntil != Forever && identity.End.After(inQuestion.ValidUntil) {
						identity.End = inQuestion.ValidUntil
					}
					if err := tracker.store.AddIdentity(identity); err != nil {
						jww.FATAL.Panicf("Could not insert identity: %+v", err)
					}
				}
				//move forward the tracking of when generation should occur
				inQuestion.LastGeneration = inQuestion.NextGeneration
				inQuestion.NextGeneration = nextNextGeneration.Add(time.Millisecond)
			}

			// if it is time to delete the id, process the delete
			if inQuestion.ValidUntil != Forever && netTime.Now().After(inQuestion.ValidUntil) {
				edits = true
				toRemove[i] = struct{}{}
			} else {
				// otherwise see if it is responsible for the next event
				if inQuestion.NextGeneration.Before(nextEvent){
					nextEvent = inQuestion.NextGeneration
				}
				if inQuestion.ValidUntil.Before(nextEvent){
					nextEvent = inQuestion.ValidUntil
				}
			}
		}

		//process any deletions
		if len(toRemove)>0{
			newTracked := make([]trackedID,len(tracker.tracked))
			for i := range tracker.tracked{
				if _, remove := toRemove[i]; !remove {
					newTracked = append(newTracked, tracker.tracked[i])
				}
			}
			tracker.tracked=newTracked
		}

		if edits{
			tracker.save()
		}

		// trigger events early. this will cause generations to happen
		// early as well as message pickup. As a result, if there
		// are time sync issues between clients and they begin sending
		// to ephemerals early, messages will still be picked up
		nextUpdate := nextEvent.Add(-validityGracePeriod)

		// Sleep until the last ID has expired
		select {
		case <-time.NewTimer(nextUpdate.Sub(nextUpdate)).C:
		case newIdentity := <- tracker.newIdentity:
			// if the identity is old, just update its properties
			for i := range tracker.tracked{
				inQuestion := tracker.tracked[i]
				if inQuestion.Source.Cmp(newIdentity.Source){
					inQuestion.Persistent = newIdentity.Persistent
					inQuestion.ValidUntil = newIdentity.ValidUntil
					tracker.save()
					continue trackerLoop
				}
			}
			//otherwise, add it to the list and run
			tracker.tracked = append(tracker.tracked,newIdentity)
			tracker.save()
			continue trackerLoop
		case deleteID := <- tracker.deleteIdentity:
			for i := range tracker.tracked{
				inQuestion := tracker.tracked[i]
				if inQuestion.Source.Cmp(deleteID){
					tracker.tracked = append(tracker.tracked[:i], tracker.tracked[i+1:]...)
					tracker.save()
					break
				}
			}
		case <-stop.Quit():
			addrSpace.UnregisterNotification(addressSpaceSizeChanTag)
			stop.ToStopped()
			return
		}
	}
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

func generateIdentitiesOverRange(lastGeneration, generateThrough time.Time,
	source *id.ID, addressSize uint8 )([]receptionID.Identity, time.Time){
	protoIds, err := ephemeral.GetIdsByRange(
		source, uint(addressSize), lastGeneration, generateThrough.Sub(lastGeneration))

	jww.DEBUG.Printf("Now: %s, LastCheck: %s, Different: %s",
		generateThrough, generateThrough, generateThrough.Sub(lastGeneration))
	jww.DEBUG.Printf("protoIds Count: %d", len(protoIds))

	if err != nil {
		jww.FATAL.Panicf("Could not generate upcoming IDs: %+v", err)
	}

	// Generate identities off of that list
	identities := make([]receptionID.Identity, len(protoIds))

	// Add identities for every address ID
	for i, eid := range protoIds {
		// Expand the grace period for both start and end
		identities[i] = receptionID.Identity{
			EphId:       eid.Id,
			Source:      source,
			AddressSize: addressSize,
			End:         eid.End,
			StartValid:  eid.Start.Add(-validityGracePeriod),
			EndValid:    eid.End.Add(validityGracePeriod),
			Ephemeral:   false,
			ExtraChecks: interfaces.DefaultExtraChecks,
		}

	}

	jww.INFO.Printf("Number of Identities Generated: %d", len(identities))
	jww.INFO.Printf("Current Identity: %d (source: %s), Start: %s, End: %s",
		identities[len(identities)-1].EphId.Int64(),
		identities[len(identities)-1].Source,
		identities[len(identities)-1].StartValid,
		identities[len(identities)-1].EndValid)

	return identities, identities[len(identities)-1].End
}

func (tracker *Tracker)save(){
	persistant := make([]trackedID, 0, len(tracker.tracked))

	for i := range tracker.tracked{
		if tracker.tracked[i].Persistent{
			persistant = append(persistant, tracker.tracked[i])
		}
	}

	if len(persistant)==0{
		return
	}


	data, err := json.Marshal(&persistant)
	if err!=nil{
		jww.FATAL.Panicf("Failed to marshal the tracked users")
	}

	obj := &versioned.Object{
		Version:   ,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = tracker.session.GetKV().Set(TrackerListKey, TrackerListVersion, obj)
	if err!=nil{
		jww.FATAL.Panicf("Failed to store the tracked users")
	}
}