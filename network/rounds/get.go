///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
)

func (m *manager) GetMessagesFromRound(roundID id.Round, identity receptionID.EphemeralIdentity) {
	//get the round from the in ram store
	ri, err := m.instance.GetRound(roundID)

	// If we didn't find it, send to Historical Rounds Retrieval
	if err != nil || m.params.ForceHistoricalRounds {

		// store the round as an unretreived round without a round info
		// This will silently do nothing if the round is
		err = m.unchecked.AddRound(roundID, nil,
			identity.Source, identity.EphId)
		if err != nil {
			jww.FATAL.Panicf("Failed to denote Unchecked Round for round %d", roundID)
		}

		if m.params.ForceHistoricalRounds {
			jww.WARN.Printf("Forcing use of historical rounds for round ID %d.",
				roundID)
		}

		jww.INFO.Printf("Messages found in round %d for %d (%s), looking "+
			"up messages via historical lookup", roundID, identity.EphId.Int64(),
			identity.Source)

		err = m.historical.LookupHistoricalRound(roundID, func(info *pb.RoundInfo, success bool) {
			if !success {

			}
			// If found, send to Message Retrieval Workers
			m.lookupRoundMessages <- roundLookup{
				RoundInfo: info,
				Identity:  identity,
			}
		})
	} else { // if we did find it, send it to the round pickup thread
		jww.INFO.Printf("Messages found in round %d for %d (%s), looking "+
			"up messages via in ram lookup", roundID, identity.EphId.Int64(),
			identity.Source)
		//store the round as an unretreived round
		if !m.params.RealtimeOnly {
			err = m.unchecked.AddRound(roundID, ri,
				identity.Source, identity.EphId)
			if err != nil {
				jww.FATAL.Panicf("Failed to denote Unchecked Round for round %d", roundID)
			}
		}

		// If found, send to Message Retrieval Workers
		m.lookupRoundMessages <- roundLookup{
			RoundInfo: ri,
			Identity:  identity,
		}
	}

}
