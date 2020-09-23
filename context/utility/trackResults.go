package utility

import (
	"fmt"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
)

// Function to track the results of events. It returns true if the collection of
// events resolved well, and then a count of how many rounds failed and how
// many roundEvents timed out.
func TrackResults(resultsCh chan ds.EventReturn, numResults int) (bool, int, int) {
	numTimeOut, numRoundFail := 0, 0
	for numResponses := 0; numResponses < numResults; numResponses++ {
		fmt.Printf("iterated: %v\n", numResponses)
		er := <-resultsCh
		if er.TimedOut {
			numTimeOut++
		} else if states.Round(er.RoundInfo.State) == states.FAILED {
			numRoundFail++
		}
	}

	return (numTimeOut + numRoundFail) > 0, numRoundFail, numTimeOut
}
