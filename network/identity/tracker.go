///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package identity

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

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
)

const validityGracePeriod = 5 * time.Minute
const TrackerListKey = "TrackerListKey"
const TrackerListVersion = 0
const TimestampKey = "IDTrackingTimestamp"
const ephemeralStoppable = "EphemeralCheck"
const addressSpaceSizeChanTag = "ephemeralTracker"

var Forever = time.Time{}

const trackedIDChanSize = 1000
const deleteIDChanSize = 1000

type Tracker interface {
	StartProcessies() stoppable.Stoppable
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	RemoveIdentity(id *id.ID)
	GetEphemeralIdentity(rng io.Reader, addressSize uint8) (receptionID.IdentityUse, error)
}

type manager struct {
	tracked        []trackedID
	store          *receptionID.Store
	session        storage.Session
	newIdentity    chan trackedID
	deleteIdentity chan *id.ID
	addrSpace      address.Space
	mux            *sync.Mutex
}

type trackedID struct {
	NextGeneration time.Time
	LastGeneration time.Time
	Source         *id.ID
	ValidUntil     time.Time
	Persistent     bool
}

func NewOrLoadTracker(session storage.Session, addrSpace address.Space) *manager {
	//intilization
	t := &manager{
		tracked:        make([]trackedID, 0),
		session:        session,
		newIdentity:    make(chan trackedID, trackedIDChanSize),
		deleteIdentity: make(chan *id.ID, deleteIDChanSize),
		addrSpace:      addrSpace,
		mux:            &sync.Mutex{},
	}
	//Load this structure
	err := t.load()
	if err != nil && os.IsNotExist(err) {
		oldTimestamp, err := getOldTimestampStore(t.session)
		if err == nil {
			jww.WARN.Printf("No tracked identities found, " +
				"creating a new tracked identity from legacy stored timestamp")
			t.tracked = append(t.tracked, trackedID{
				// make the next generation now so a generation triggers on first run
				NextGeneration: netTime.Now(),
				// it generated previously though oldTimestamp, denote that
				LastGeneration: oldTimestamp,
				Source:         t.session.GetReceptionID(),
				ValidUntil:     Forever,
				Persistent:     true,
			})
		} else {
			jww.WARN.Printf("No tracked identities found and no  legacy stored " +
				"timestamp found, creating a new tracked identity from scratch")
			t.tracked = append(t.tracked, trackedID{
				// make the next generation now so a generation triggers on first run
				NextGeneration: netTime.Now(),
				// start generation 24 hours ago to make sure all rescent ephemerals do pickups
				// todo: should we go back farther?
				LastGeneration: netTime.Now().Add(-time.Duration(ephemeral.Period)),
				Source:         t.session.GetReceptionID(),
				ValidUntil:     Forever,
				Persistent:     true,
			})
		}
	} else if err != nil {
		jww.FATAL.Panicf("unable to create new Tracker: %+v", err)
	}

	t.store = receptionID.NewOrLoadStore(session.GetKV())

	return t
}

// Track runs a thread which checks for past and present address ID.
func (tracker manager) StartProcessies() stoppable.Stoppable {
	stop := stoppable.NewSingle(ephemeralStoppable)

	go tracker.track(stop)

	return stop
}

// AddIdentity adds an identity to be tracked
func (tracker *manager) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
	tracker.newIdentity <- trackedID{
		NextGeneration: netTime.Now().Add(-time.Second),
		LastGeneration: time.Time{},
		Source:         id,
		ValidUntil:     validUntil,
		Persistent:     persistent,
	}
}

// RemoveIdentity removes a currently tracked identity.
func (tracker *manager) RemoveIdentity(id *id.ID) {
	tracker.deleteIdentity <- id
}

// GetEphemeralIdentity returns an ephemeral Identity to poll the network with.
func (tracker *manager) GetEphemeralIdentity(rng io.Reader, addressSize uint8) (receptionID.IdentityUse, error) {
	return tracker.store.GetIdentity(rng, addressSize)
}

func (tracker *manager) track(stop *stoppable.Single) {

	// Wait until we get the ID size from the network
	addressSize := tracker.addrSpace.GetAddressSpace()

	/*wait for next event*/
trackerLoop:
	for {
		edits := false
		var toRemove map[int]struct{}
		nextEvent := tracker.tracked[0].ValidUntil

		//loop through every tracked ID and see if any operatrions are needed
		for i := range tracker.tracked {
			inQuestion := tracker.tracked[i]

			//generate new ephmerals if is time for it
			if netTime.Now().After(inQuestion.NextGeneration) {
				edits = true

				//ensure that ephemerals will not be generated after the identity is invalid
				generateUntil := inQuestion.NextGeneration
				if inQuestion.ValidUntil != Forever && generateUntil.After(inQuestion.ValidUntil) {
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
					identity.Ephemeral = !inQuestion.Persistent
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
				if inQuestion.NextGeneration.Before(nextEvent) {
					nextEvent = inQuestion.NextGeneration
				}
				if inQuestion.ValidUntil.Before(nextEvent) {
					nextEvent = inQuestion.ValidUntil
				}
			}
		}

		//process any deletions
		if len(toRemove) > 0 {
			newTracked := make([]trackedID, len(tracker.tracked))
			for i := range tracker.tracked {
				if _, remove := toRemove[i]; !remove {
					newTracked = append(newTracked, tracker.tracked[i])
				}
			}
			tracker.tracked = newTracked
		}

		if edits {
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
		case newIdentity := <-tracker.newIdentity:
			// if the identity is old, just update its properties
			for i := range tracker.tracked {
				inQuestion := tracker.tracked[i]
				if inQuestion.Source.Cmp(newIdentity.Source) {
					inQuestion.Persistent = newIdentity.Persistent
					inQuestion.ValidUntil = newIdentity.ValidUntil
					tracker.save()
					continue trackerLoop
				}
			}
			//otherwise, add it to the list and run
			tracker.tracked = append(tracker.tracked, newIdentity)
			tracker.save()
			continue trackerLoop
		case deleteID := <-tracker.deleteIdentity:
			for i := range tracker.tracked {
				inQuestion := tracker.tracked[i]
				if inQuestion.Source.Cmp(deleteID) {
					tracker.tracked = append(tracker.tracked[:i], tracker.tracked[i+1:]...)
					tracker.save()
					tracker.store.RemoveIdentities(deleteID)
					break
				}
			}
		case <-stop.Quit():
			tracker.addrSpace.UnregisterAddressSpaceNotification(addressSpaceSizeChanTag)
			stop.ToStopped()
			return
		}
	}
}

func getOldTimestampStore(session storage.Session) (time.Time, error) {
	lastTimestampObj, err := session.Get(TimestampKey)
	if err != nil {
		return time.Time{}, err
	}

	return unmarshalTimestamp(lastTimestampObj)
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

func generateIdentitiesOverRange(lastGeneration, generateThrough time.Time,
	source *id.ID, addressSize uint8) ([]receptionID.Identity, time.Time) {
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
			EphemeralIdentity: interfaces.EphemeralIdentity{
				EphId:  eid.Id,
				Source: source,
			},
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

func (tracker *manager) save() {
	tracker.mux.Lock()
	defer tracker.mux.Unlock()
	persistant := make([]trackedID, 0, len(tracker.tracked))

	for i := range tracker.tracked {
		if tracker.tracked[i].Persistent {
			persistant = append(persistant, tracker.tracked[i])
		}
	}

	if len(persistant) == 0 {
		return
	}

	data, err := json.Marshal(&persistant)
	if err != nil {
		jww.FATAL.Panicf("unable to marshal trackedID list: %+v", err)
	}

	obj := &versioned.Object{
		Version:   TrackerListVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = tracker.session.GetKV().Set(TrackerListKey, TrackerListVersion,
		obj)
	if err != nil {
		jww.FATAL.Panicf("unable to save trackedID list: %+v", err)
	}
}

func (t *manager) load() error {
	t.mux.Lock()
	defer t.mux.Unlock()
	obj, err := t.session.GetKV().Get(TrackerListKey, TrackerListVersion)
	if err != nil {
		return err
	}

	return json.Unmarshal(obj.Data, &t.tracked)
}
