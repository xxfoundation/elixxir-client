///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ephemeral

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/reception"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	ephemeralStore "gitlab.com/elixxir/client/storage/ephemeral"
	"time"
)

const checkInterval  =  time.Duration(500) * time.Second
const ephemeralIdSie = 64
const validityGracePeriod  = 5 * time.Minute

// Check runs a thread which checks for past and present ephemeral ids
func Check(session *storage.Session, ourId *id.ID)  stoppable.Stoppable {
	stop := stoppable.NewSingle("EphemeralCheck")

	go check(session, ourId, stop)

	return stop

}

// check is a thread which continuously processes ephemeral ids. If any error occurs,
// the thread crashes
func check(session *storage.Session, ourId *id.ID, stop *stoppable.Single) {
	t := time.NewTicker(checkInterval)
	ephStore := session.Ephemeral()
	identityStore := session.Reception()
	for true {
		select {
		case <-t.C:
			err := processEphemeralIds(ourId, ephStore, identityStore)
			if err != nil {
				globals.Log.FATAL.Panicf("Could not " +
					"process ephemeral ids: %v", err)
			}

			err = ephStore.UpdateTimestamp(time.Now())
			if err != nil {
				break
			}


		case <-stop.Quit():
			break
		}


	}

}

// processEphemeralIds periodically checks for past and present ephemeral ids.
// It then adds identities for these ids if needed
func processEphemeralIds(ourId *id.ID, ephemeralStore *ephemeralStore.Store,
	identityStore *reception.Store) error {
	// Get the timestamp of the last check
	lastCheck, err := ephemeralStore.GetTimestamp()
	if err != nil {
		return errors.Errorf("Could not get time stamp in " +
			"ephemeral store: %v", err)
	}

	// Find out how long that last check was
	timeSinceLastCheck := time.Now().Sub(lastCheck)

	// Generate ephemeral ids in the range of the last check
	eids, err := ephemeral.GetIdsByRange(ourId, ephemeralIdSie,
		time.Now().UnixNano(), timeSinceLastCheck)
	if err != nil {
		return errors.Errorf("Could not generate ephemeral ids: %v", err)
	}

	// Add identities for every ephemeral id
	for _, eid := range eids {
		err = identityStore.AddIdentity(reception.Identity{
			EphId:       eid.Id,
			Source:      ourId,
			End:         time.Now().Add(validityGracePeriod),
			StartValid:  eid.Start,
			EndValid:    eid.End,
			Ephemeral:   false,
		})
		if err != nil {
			return errors.Errorf("Could not add identity for " +
				"generated ephemeral ID: %v", err)
		}
	}

	return nil
}
