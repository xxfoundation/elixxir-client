////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package pickup

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/xx_network/primitives/id"
)

func (m *pickup) GetMessagesFromRound(
	roundID id.Round, identity receptionID.EphemeralIdentity) {
	// Get the round from the in-RAM store
	ri, err := m.instance.GetRound(roundID)

	// If we did not find it, then send to Historical Rounds Retrieval
	if err != nil || m.params.ForceHistoricalRounds {

		// Store the round as an un-retrieved round without a round info
		// This will silently do nothing if the round is
		err = m.unchecked.AddRound(roundID, nil,
			identity.Source, identity.EphId)
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to denote Unchecked Round for round %d", roundID)
		}

		if m.params.ForceHistoricalRounds {
			jww.WARN.Printf(
				"Forcing use of historical rounds for round ID %d.", roundID)
		}

		jww.INFO.Printf("Messages found in round %d for %d (%s), looking "+
			"up messages via historical lookup", roundID, identity.EphId.Int64(),
			identity.Source)

		err = m.historical.LookupHistoricalRound(
			roundID, func(round rounds.Round, success bool) {
				if !success {
					// TODO: Implement me
				}

				// If found, send to Message Retrieval Workers
				m.lookupRoundMessages <- roundLookup{
					Round:    round,
					Identity: identity,
				}
			})
	} else {
		// If we did find it, send it to the round pickup thread
		jww.INFO.Printf("Messages found in round %d for %d (%s), looking "+
			"up messages via in ram lookup", roundID, identity.EphId.Int64(),
			identity.Source)

		// store the round as an un-retrieved round
		err = m.unchecked.AddRound(roundID, ri,
			identity.Source, identity.EphId)
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to denote Unchecked Round for round %d", roundID)
		}

		// If found, send to Message Retrieval Workers
		m.lookupRoundMessages <- roundLookup{
			Round:    rounds.MakeRound(ri),
			Identity: identity,
		}
	}

}
