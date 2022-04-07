///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmix

import (
	jww "github.com/spf13/jwalterweatherman"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
)

// TrackResults follows the results of events. It returns true if the collection
// of events resolved well, and then a count of how many rounds failed and how
// many roundEvents timed out.
func TrackResults(resultsCh chan ds.EventReturn, numResults int) (bool, int, int) {
	numTimeOut, numRoundFail := 0, 0
	for numResponses := 0; numResponses < numResults; numResponses++ {
		er := <-resultsCh
		if er.TimedOut {
			numTimeOut++
		} else if states.Round(er.RoundInfo.State) == states.FAILED {
			jww.ERROR.Printf("RoundInfo FAILED: %+v", er.RoundInfo)
			numRoundFail++
		}
	}

	return (numTimeOut + numRoundFail) == 0, numRoundFail, numTimeOut
}
