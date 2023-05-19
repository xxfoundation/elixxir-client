////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package identity

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/cmix/address"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
)

var Forever = time.Time{}

const (
	validityGracePeriod     = 5 * time.Minute
	TrackerListKey          = "TrackerListKey"
	TrackerListVersion      = 0
	TimestampKey            = "IDTrackingTimestamp"
	ephemeralStoppable      = "EphemeralCheck"
	addressSpaceSizeChanTag = "ephemeralTracker"

	trackedIDChanSize = 1000
	deleteIDChanSize  = 1000

	// DefaultExtraChecks is the default value for ExtraChecks
	// on receptionID.Identity.
	DefaultExtraChecks = 5

	// NetworkRetention is how long messages are
	// retained on the network
	NetworkRetention = 500 * time.Hour

	// GenerationDelta is how far into the past
	// to go in order to ensure all relevant
	// ephemeral identities are generated
	GenerationDelta = time.Duration(ephemeral.Period) + (5 * time.Second)
)

type Tracker interface {
	StartProcesses() stoppable.Stoppable
	AddIdentityInternal(id *id.ID, validUntil time.Time, persistent bool)
	AddIdentityWithHistoryInternal(id *id.ID, validUntil, beginning time.Time, persistent bool)
	RemoveIdentityInternal(id *id.ID)
	ForEach(n int, rng io.Reader, addressSize uint8,
		operator func([]receptionID.IdentityUse) error) error
	GetIdentity(get *id.ID) (TrackedID, error)
}

type manager struct {
	tracked        []*TrackedID
	ephemeral      *receptionID.Store
	session        storage.Session
	newIdentity    chan TrackedID
	deleteIdentity chan *id.ID
	addrSpace      address.Space
	mux            *sync.Mutex
}

type TrackedID struct {
	NextGeneration time.Time
	LastGeneration time.Time
	Source         *id.ID
	ValidUntil     time.Time
	Persistent     bool
	Creation       time.Time
}

func NewOrLoadTracker(session storage.Session, addrSpace address.Space) Tracker {
	// Initialization
	t := &manager{
		tracked:        make([]*TrackedID, 0),
		session:        session,
		newIdentity:    make(chan TrackedID, trackedIDChanSize),
		deleteIdentity: make(chan *id.ID, deleteIDChanSize),
		addrSpace:      addrSpace,
		mux:            &sync.Mutex{},
	}

	// Load this structure
	err := t.load()
	if err != nil && !t.session.GetKV().Exists(err) {
		oldTimestamp, err2 := getOldTimestampStore(t.session)
		if err2 == nil {
			jww.WARN.Printf("No tracked identities found, creating a new " +
				"tracked identity from legacy stored timestamp.")

			t.tracked = append(t.tracked, &TrackedID{
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
				"stored timestamp found; no messages can be picked up until an " +
				"identity is added.")
		}
	} else if err != nil {
		jww.FATAL.Panicf("Unable to create new Tracker: %+v", err)
	}

	t.ephemeral = receptionID.NewOrLoadStore(session.GetKV())

	return t
}

// StartProcesses track runs a thread which checks for past and present address
// ID.
func (t *manager) StartProcesses() stoppable.Stoppable {
	stop := stoppable.NewSingle(ephemeralStoppable)

	go t.track(stop)

	return stop
}

// AddIdentityInternal adds an identity to be tracked.
func (t *manager) AddIdentityInternal(id *id.ID, validUntil time.Time, persistent bool) {
	lastGeneration := netTime.Now().Add(-GenerationDelta)
	t.newIdentity <- TrackedID{
		NextGeneration: netTime.Now().Add(-time.Second),
		LastGeneration: lastGeneration,
		Source:         id,
		ValidUntil:     validUntil,
		Persistent:     persistent,
		Creation:       netTime.Now(),
	}
}

// AddIdentityWithHistoryInternal adds an identity to be tracked which will slowly pick up history.
func (t *manager) AddIdentityWithHistoryInternal(id *id.ID, validUntil, historicalBeginning time.Time, persistent bool) {
	retention := netTime.Now().Add(-NetworkRetention)
	if historicalBeginning.Before(retention) {
		historicalBeginning = retention
	}

	if now := time.Now(); historicalBeginning.After(now) ||
		now.Sub(historicalBeginning) < GenerationDelta {
		historicalBeginning = now.Add(-GenerationDelta)
	}

	t.newIdentity <- TrackedID{
		NextGeneration: netTime.Now().Add(-time.Second),
		LastGeneration: historicalBeginning,
		Source:         id,
		ValidUntil:     validUntil,
		Persistent:     persistent,
		Creation:       netTime.Now(),
	}
}

// RemoveIdentityInternal removes a currently tracked identity.
func (t *manager) RemoveIdentityInternal(id *id.ID) {
	t.deleteIdentity <- id
}

// ForEach passes a fisher-yates shuffled list of up to 'num'
// ephemeral identities into the operation function. It will pass a
// fake identity if none are available
// and less than 'num' if less than 'num' are available.
// 'num' must be positive non-zero
func (t *manager) ForEach(n int, rng io.Reader, addressSize uint8,
	operator func([]receptionID.IdentityUse) error) error {
	return t.ephemeral.ForEach(n, rng, addressSize, operator)
}

// GetIdentity returns a currently tracked identity
func (t *manager) GetIdentity(get *id.ID) (TrackedID, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	for i := range t.tracked {
		if get.Cmp(t.tracked[i].Source) {
			return *t.tracked[i], nil
		}
	}
	return TrackedID{}, errors.Errorf("could not find id %s", get)
}

func (t *manager) track(stop *stoppable.Single) {
	// Wait until the ID size is retrieved from the network
	addressSize := t.addrSpace.GetAddressSpace()

	for {
		// Process new and old identities
		nextEvent := t.processIdentities(addressSize)
		waitPeriod := nextEvent.Sub(netTime.Now())

		if waitPeriod > validityGracePeriod {
			// Trigger events early. This will cause generations to happen early as
			// well as message pickup. As a result, if there are time collective issues
			// between clients, and they begin sending to ephemeral IDs early, then
			// messages will still be picked up.
			waitPeriod = waitPeriod - validityGracePeriod
		}

		// Sleep until the last ID has expired
		select {
		case <-time.After(waitPeriod):
		case newIdentity := <-t.newIdentity:
			jww.DEBUG.Printf("Receiving new identity %s :%+v",
				newIdentity.Source, newIdentity)

			// If the identity is old, then update its properties
			isOld := false
			for i := range t.tracked {
				inQuestion := t.tracked[i]
				if inQuestion.Source.Cmp(newIdentity.Source) {
					jww.DEBUG.Printf(
						"Updating old identity %s", newIdentity.Source)
					inQuestion.Persistent = newIdentity.Persistent
					inQuestion.ValidUntil = newIdentity.ValidUntil
					isOld = true
					break
				}
			}
			if !isOld {
				jww.DEBUG.Printf("Tracking new identity %s", newIdentity.Source)
				// Otherwise, add it to the list and run
				t.tracked = append(t.tracked, &newIdentity)
			}

			t.save()
			continue

		case deleteID := <-t.deleteIdentity:
			removed := false
			for i := range t.tracked {
				inQuestion := t.tracked[i]
				if inQuestion.Source.Cmp(deleteID) {
					removed = true
					t.tracked = append(t.tracked[:i], t.tracked[i+1:]...)
					t.save()
					// Requires manual deletion in case identity is deleted before expiration
					t.ephemeral.RemoveIdentities(deleteID)
					break
				}
			}
			if !removed {
				jww.WARN.Printf("Identity %s failed to be removed from tracker", deleteID)
			}
		case <-stop.Quit():
			t.addrSpace.UnregisterAddressSpaceNotification(addressSpaceSizeChanTag)
			stop.ToStopped()
			return
		}
	}
}

// processIdentities builds and adds new identities and removes old
// identities from the tracker and returns the timestamp of the next ID event.
func (t *manager) processIdentities(addressSize uint8) time.Time {
	edits := false
	toRemove := make(map[int]struct{})
	// Identities are rotated on a 24-hour time period. Set the event
	// to the latest possible time so that any sooner times will overwrite this
	nextEvent := netTime.Now().Add(time.Duration(ephemeral.Period))

	// Loop through every tracked ID and see if any operations are needed
	for i := range t.tracked {
		inQuestion := t.tracked[i]
		// Generate new ephemeral if is time for it
		if netTime.Now().After(inQuestion.NextGeneration) {
			nextGeneration := t.generateIdentitiesOverRange(inQuestion, addressSize)

			// Move forward the tracking of when generation should occur
			inQuestion.LastGeneration = inQuestion.NextGeneration
			inQuestion.NextGeneration = nextGeneration.Add(time.Millisecond)
			edits = true
		}

		// If it is time to delete the ID, then process the deletion
		if inQuestion.ValidUntil != Forever && netTime.Now().After(inQuestion.ValidUntil) {
			edits = true
			toRemove[i] = struct{}{}
		} else {
			// Otherwise, see if it is responsible for the next event
			if inQuestion.NextGeneration.Before(nextEvent) {
				nextEvent = inQuestion.NextGeneration
			}
			if !inQuestion.ValidUntil.IsZero() && inQuestion.ValidUntil.Before(nextEvent) {
				nextEvent = inQuestion.ValidUntil
			}
		}

	}

	jww.DEBUG.Printf("[TrackedIDS] NextEvent: %s", nextEvent)

	// Process any deletions
	if len(toRemove) > 0 {
		newTracked := make([]*TrackedID, 0, len(t.tracked))
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

	return nextEvent
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

// generateIdentitiesOverRange generates and adds all not yet existing ephemeral Ids
// and returns the timestamp of the next generation for the given TrackedID
func (t *manager) generateIdentitiesOverRange(inQuestion *TrackedID,
	addressSize uint8) time.Time {
	// Ensure that ephemeral IDs will not be generated after the
	// identity is invalid
	generateUntil := inQuestion.NextGeneration
	if inQuestion.ValidUntil != Forever && generateUntil.After(inQuestion.ValidUntil) {
		generateUntil = inQuestion.ValidUntil
	}

	// Generate list of identities
	protoIds, err := ephemeral.GetIdsByRange(inQuestion.Source, uint(addressSize),
		inQuestion.LastGeneration, generateUntil.Sub(inQuestion.LastGeneration))
	if err != nil {
		jww.FATAL.Panicf("Could not generate upcoming IDs: %+v", err)
	}

	identitiesToAdd := make([]receptionID.Identity, 0, len(protoIds))
	identitiesToChain := make([]receptionID.Identity, 0, len(protoIds))

	// Add identities for every address ID
	lastIdentityEnd := time.Time{}
	var NewestIdentity receptionID.Identity
	for i, _ := range protoIds {
		eid := protoIds[i]
		// Expand the grace period for both start and end
		newIdentity := receptionID.Identity{
			EphemeralIdentity: receptionID.EphemeralIdentity{
				EphId:  eid.Id,
				Source: inQuestion.Source,
			},
			AddressSize: addressSize,
			End:         eid.End,
			StartValid:  eid.Start.Add(-validityGracePeriod),
			EndValid:    eid.End.Add(validityGracePeriod),
			Ephemeral:   false,
			ExtraChecks: DefaultExtraChecks,
		}
		// Move up the end time if the source identity is invalid
		// before the natural end of the ephemeral identity
		if inQuestion.ValidUntil != Forever && newIdentity.End.
			After(inQuestion.ValidUntil) {
			newIdentity.End = inQuestion.ValidUntil
		}

		newIdentity.Ephemeral = !inQuestion.Persistent

		// If the identity expired before the current time, we know it
		// is no longer valid and should be added to the chain
		if netTime.Now().After(newIdentity.EndValid) {
			identitiesToChain = append(identitiesToChain, newIdentity)
		} else {
			identitiesToAdd = append(identitiesToAdd, newIdentity)
		}

		if newIdentity.End.After(lastIdentityEnd) {
			lastIdentityEnd = newIdentity.End
			NewestIdentity = newIdentity
		}
	}

	//link the chain
	if len(identitiesToChain) > 0 {
		firstLink := &identitiesToChain[len(identitiesToChain)-1]
		currentLink := firstLink
		if len(identitiesToChain) > 1 {
			for i := len(identitiesToChain) - 1; i >= 0; i-- {
				currentLink.ProcessNext = &identitiesToChain[i]
				currentLink = currentLink.ProcessNext
			}
		}
		identitiesToAdd = append(identitiesToAdd, *firstLink)
	}

	//add the identities
	for i := 0; i < len(identitiesToAdd); i++ {
		if err = t.ephemeral.AddIdentity(identitiesToAdd[i]); err != nil {
			jww.FATAL.Panicf("Could not insert identity: %+v", err)
		}
	}

	jww.INFO.Printf("Current Identity: %d (source: %s), Start: %s, "+
		"End: %s, addrSize: %d",
		NewestIdentity.EphId.Int64(),
		NewestIdentity.Source,
		NewestIdentity.StartValid,
		NewestIdentity.EndValid,
		addressSize)

	jww.INFO.Printf("Number of identities generated: %d", len(protoIds))
	return NewestIdentity.End
}

// save persistent TrackedID to storage
func (t *manager) save() {
	t.mux.Lock()
	defer t.mux.Unlock()
	persistent := make([]TrackedID, 0, len(t.tracked))

	for i := range t.tracked {
		if t.tracked[i].Persistent {
			persistent = append(persistent, *t.tracked[i])
		}
	}

	if len(persistent) == 0 {
		return
	}

	data, err := json.Marshal(&persistent)
	if err != nil {
		jww.FATAL.Panicf("Unable to marshal TrackedID list: %+v", err)
	}

	obj := &versioned.Object{
		Version:   TrackerListVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	err = t.session.GetKV().Set(TrackerListKey, obj)
	if err != nil {
		jww.FATAL.Panicf("Unable to save TrackedID list: %+v", err)
	}
}

// load persistent IDs from storage
func (t *manager) load() error {
	t.mux.Lock()
	defer t.mux.Unlock()
	obj, err := t.session.GetKV().Get(TrackerListKey, TrackerListVersion)
	if err != nil {
		return err
	}

	return json.Unmarshal(obj.Data, &t.tracked)
}
