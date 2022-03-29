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

const (
	trackedIDChanSize = 1000
	deleteIDChanSize  = 1000
)

// DefaultExtraChecks is the default value for ExtraChecks on
// receptionID.Identity.
const DefaultExtraChecks = 10

type Tracker interface {
	StartProcesses() stoppable.Stoppable
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
	// Initialization
	t := &manager{
		tracked:        make([]trackedID, 0),
		session:        session,
		newIdentity:    make(chan trackedID, trackedIDChanSize),
		deleteIdentity: make(chan *id.ID, deleteIDChanSize),
		addrSpace:      addrSpace,
		mux:            &sync.Mutex{},
	}

	// Load this structure
	err := t.load()
	if err != nil && os.IsNotExist(err) {
		oldTimestamp, err2 := getOldTimestampStore(t.session)
		if err2 == nil {
			jww.WARN.Printf("No tracked identities found, creating a new " +
				"tracked identity from legacy stored timestamp.")

			t.tracked = append(t.tracked, trackedID{
				// Make the next generation now so a generation triggers on
				// first run
				NextGeneration: netTime.Now(),
				// It generated previously though oldTimestamp, denote that
				LastGeneration: oldTimestamp,
				Source:         t.session.GetReceptionID(),
				ValidUntil:     Forever,
				Persistent:     true,
			})
		} else {
			jww.WARN.Printf("No tracked identities found and no legacy " +
				"stored timestamp found; creating a new tracked identity " +
				"from scratch.")

			t.tracked = append(t.tracked, trackedID{
				// Make the next generation now so a generation triggers on
				// first run
				NextGeneration: netTime.Now(),
				// Start generation 24 hours ago to make sure all resent
				// ephemeral do pickups
				// TODO: Should we go back farther?
				LastGeneration: netTime.Now().Add(-time.Duration(ephemeral.Period)),
				Source:         t.session.GetReceptionID(),
				ValidUntil:     Forever,
				Persistent:     true,
			})
		}
	} else if err != nil {
		jww.FATAL.Panicf("Unable to create new Tracker: %+v", err)
	}

	t.store = receptionID.NewOrLoadStore(session.GetKV())

	return t
}

// StartProcesses track runs a thread which checks for past and present address
// ID.
func (t manager) StartProcesses() stoppable.Stoppable {
	stop := stoppable.NewSingle(ephemeralStoppable)

	go t.track(stop)

	return stop
}

// AddIdentity adds an identity to be tracked.
func (t *manager) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
	t.newIdentity <- trackedID{
		NextGeneration: netTime.Now().Add(-time.Second),
		LastGeneration: time.Time{},
		Source:         id,
		ValidUntil:     validUntil,
		Persistent:     persistent,
	}
}

// RemoveIdentity removes a currently tracked identity.
func (t *manager) RemoveIdentity(id *id.ID) {
	t.deleteIdentity <- id
}

// GetEphemeralIdentity returns an ephemeral Identity to poll the network with.
func (t *manager) GetEphemeralIdentity(rng io.Reader, addressSize uint8) (
	receptionID.IdentityUse, error) {
	return t.store.GetIdentity(rng, addressSize)
}

func (t *manager) track(stop *stoppable.Single) {
	// Wait until the ID size is retrieved from the network
	addressSize := t.addrSpace.GetAddressSpace()

	// Wait for next event
trackerLoop:
	for {
		edits := false
		var toRemove map[int]struct{}
		nextEvent := t.tracked[0].ValidUntil

		// Loop through every tracked ID and see if any operations are needed
		for i := range t.tracked {
			inQuestion := t.tracked[i]

			// Generate new ephemeral if is time for it
			if netTime.Now().After(inQuestion.NextGeneration) {
				edits = true

				// Ensure that ephemeral IDs will not be generated after the
				// identity is invalid
				generateUntil := inQuestion.NextGeneration
				if inQuestion.ValidUntil != Forever &&
					generateUntil.After(inQuestion.ValidUntil) {
					generateUntil = inQuestion.ValidUntil
				}

				// Generate all not yet existing ephemeral IDs
				identities, nextNextGeneration := generateIdentitiesOverRange(
					inQuestion.LastGeneration, inQuestion.NextGeneration,
					inQuestion.Source, addressSize)

				// Add all ephemeral IDs to the ephemeral handler
				for _, identity := range identities {
					// Move up the end time if the source identity is invalid
					// before the natural end of the ephemeral identity
					if inQuestion.ValidUntil != Forever &&
						identity.End.After(inQuestion.ValidUntil) {
						identity.End = inQuestion.ValidUntil
					}

					identity.Ephemeral = !inQuestion.Persistent
					if err := t.store.AddIdentity(identity); err != nil {
						jww.FATAL.Panicf("Could not insert identity: %+v", err)
					}
				}

				// Move forward the tracking of when generation should occur
				inQuestion.LastGeneration = inQuestion.NextGeneration
				inQuestion.NextGeneration = nextNextGeneration.Add(time.Millisecond)
			}

			// If it is time to delete the ID, then process the deletion
			if inQuestion.ValidUntil != Forever &&
				netTime.Now().After(inQuestion.ValidUntil) {
				edits = true
				toRemove[i] = struct{}{}
			} else {
				// Otherwise, see if it is responsible for the next event
				if inQuestion.NextGeneration.Before(nextEvent) {
					nextEvent = inQuestion.NextGeneration
				}
				if inQuestion.ValidUntil.Before(nextEvent) {
					nextEvent = inQuestion.ValidUntil
				}
			}
		}

		// Process any deletions
		if len(toRemove) > 0 {
			newTracked := make([]trackedID, 0, len(t.tracked))
			for i := range t.tracked {
				if _, remove := toRemove[i]; !remove {
					newTracked = append(newTracked, t.tracked[i])
				}
			}

			t.tracked = newTracked
		}

		if edits {
			t.save()
		}

		// Trigger events early. This will cause generations to happen early as
		// well as message pickup. As a result, if there are time sync issues
		// between clients, and they begin sending to ephemeral IDs early, then
		// messages will still be picked up.
		nextUpdate := nextEvent.Add(-validityGracePeriod)

		// Sleep until the last ID has expired
		select {
		case <-time.NewTimer(nextUpdate.Sub(nextUpdate)).C:
		case newIdentity := <-t.newIdentity:
			// If the identity is old, then update its properties
			for i := range t.tracked {
				inQuestion := t.tracked[i]
				if inQuestion.Source.Cmp(newIdentity.Source) {
					inQuestion.Persistent = newIdentity.Persistent
					inQuestion.ValidUntil = newIdentity.ValidUntil
					t.save()
					continue trackerLoop
				}
			}

			// Otherwise, add it to the list and run
			t.tracked = append(t.tracked, newIdentity)
			t.save()
			continue trackerLoop

		case deleteID := <-t.deleteIdentity:
			for i := range t.tracked {
				inQuestion := t.tracked[i]
				if inQuestion.Source.Cmp(deleteID) {
					t.tracked = append(t.tracked[:i], t.tracked[i+1:]...)
					t.save()
					t.store.RemoveIdentities(deleteID)
					break
				}
			}
		case <-stop.Quit():
			t.addrSpace.UnregisterAddressSpaceNotification(addressSpaceSizeChanTag)
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
	protoIds, err := ephemeral.GetIdsByRange(source, uint(addressSize),
		lastGeneration, generateThrough.Sub(lastGeneration))

	jww.DEBUG.Printf("Now: %s, LastCheck: %s, Different: %s",
		generateThrough, generateThrough, generateThrough.Sub(lastGeneration))
	jww.DEBUG.Printf("protoIds count: %d", len(protoIds))

	if err != nil {
		jww.FATAL.Panicf("Could not generate upcoming IDs: %+v", err)
	}

	// Generate identities off of that list
	identities := make([]receptionID.Identity, len(protoIds))

	// Add identities for every address ID
	for i, eid := range protoIds {
		// Expand the grace period for both start and end
		identities[i] = receptionID.Identity{
			EphemeralIdentity: receptionID.EphemeralIdentity{
				EphId:  eid.Id,
				Source: source,
			},
			AddressSize: addressSize,
			End:         eid.End,
			StartValid:  eid.Start.Add(-validityGracePeriod),
			EndValid:    eid.End.Add(validityGracePeriod),
			Ephemeral:   false,
			ExtraChecks: DefaultExtraChecks,
		}

	}

	jww.INFO.Printf("Number of identities generated: %d", len(identities))
	jww.INFO.Printf("Current Identity: %d (source: %s), Start: %s, End: %s",
		identities[len(identities)-1].EphId.Int64(),
		identities[len(identities)-1].Source,
		identities[len(identities)-1].StartValid,
		identities[len(identities)-1].EndValid)

	return identities, identities[len(identities)-1].End
}

func (t *manager) save() {
	t.mux.Lock()
	defer t.mux.Unlock()
	persistent := make([]trackedID, 0, len(t.tracked))

	for i := range t.tracked {
		if t.tracked[i].Persistent {
			persistent = append(persistent, t.tracked[i])
		}
	}

	if len(persistent) == 0 {
		return
	}

	data, err := json.Marshal(&persistent)
	if err != nil {
		jww.FATAL.Panicf("Unable to marshal trackedID list: %+v", err)
	}

	obj := &versioned.Object{
		Version:   TrackerListVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = t.session.GetKV().Set(TrackerListKey, TrackerListVersion, obj)
	if err != nil {
		jww.FATAL.Panicf("Unable to save trackedID list: %+v", err)
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
