////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/xx_network/primitives/id"
)

// dashboardBaseURL is the base of the xx network's round dashboard URL.
// This should be used by any type of send report's GetRoundURL method.
var dashboardBaseURL = "https://dashboard.xx.network"

// SetDashboardURL is a function which modifies the base dashboard URL that is
// returned as part of any send report. Internally, this is defaulted to
// "https://dashboard.xx.network". This should only be called if the user
// explicitly wants to modify the dashboard URL. This function is not
// thread-safe, and as such should only be called on setup.
//
// Parameters:
//   - newURL - A valid URL that will be used for round look up on any send
//     report.
func SetDashboardURL(newURL string) {
	dashboardBaseURL = newURL
}

// getRoundURL is a helper function which returns the specific round
// within any type of send report, if they have a round in their RoundsList.
// This helper function is messenger specific.
func getRoundURL(round id.Round) string {
	return fmt.Sprintf("%s/rounds/%d?xxmessenger=true", dashboardBaseURL, round)
}

// RoundsList contains a list of round IDs.
//
// JSON Example:
//
//	[1001,1003,1006]
type RoundsList struct {
	Rounds []uint64
}

// makeRoundsList converts a list of id.Round into a binding-compatible
// RoundsList.
func makeRoundsList(rounds ...id.Round) RoundsList {
	rl := RoundsList{make([]uint64, len(rounds))}
	for i, rid := range rounds {
		rl.Rounds[i] = uint64(rid)
	}
	return rl
}

// Marshal JSON marshals the RoundsList.
func (rl RoundsList) Marshal() ([]byte, error) {
	return json.Marshal(&rl)
}

// unmarshalRoundsList accepts a marshalled E2ESendReport object and unmarshalls
// it into a RoundsList object, returning a list of id.Round.
func unmarshalRoundsList(marshaled []byte) ([]id.Round, error) {
	sr := RoundsList{}
	err := json.Unmarshal(marshaled, &sr)
	if err != nil {
		return nil, err
	}

	realRl := make([]id.Round, len(sr.Rounds))

	for i, rid := range sr.Rounds {
		realRl[i] = id.Round(rid)
	}

	return realRl, nil
}

// MessageDeliveryCallback gets called on the determination if all events
// related to a message send were successful.
//
// If delivered == true, timedOut == false && roundResults != nil
//
// If delivered == false, roundResults == nil
//
// If timedOut == true, delivered == false && roundResults == nil
type MessageDeliveryCallback interface {
	EventCallback(delivered, timedOut bool, roundResults []byte)
}

// WaitForRoundResult allows the caller to get notified if the rounds a message
// was sent in successfully completed. Under the hood, this uses an API that
// uses the internal round data, network historical round lookup, and waiting on
// network events to determine what has (or will) occur.
//
// This function takes the marshaled send report to ensure a memory leak does
// not occur as a result of both sides of the bindings holding a reference to
// the same pointer.
//
// Parameters:
//   - roundList - JSON marshalled bytes of RoundsList or JSON of any send report
//     that inherits a [bindings.RoundsList] object
//   - mdc - callback that adheres to the MessageDeliveryCallback interface
//   - timeoutMS - timeout when the callback will return if no state update
//     occurs, in milliseconds
func (c *Cmix) WaitForRoundResult(
	roundList []byte, mdc MessageDeliveryCallback, timeoutMS int) error {
	jww.INFO.Printf("WaitForRoundResult(%s, _, %d)", roundList, timeoutMS)
	rl, err := unmarshalRoundsList(roundList)
	if err != nil {
		return errors.Errorf("Failed to WaitForRoundResult callback due to "+
			"bad Send Report: %+v", err)
	}

	if rl == nil || len(rl) == 0 {
		return errors.Errorf("Failed to WaitForRoundResult callback due to "+
			"invalid Send Report unmarshal: %s", roundList)
	}

	f := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundResult) {
		results := make([]byte, len(rl))
		jww.INFO.Printf(
			"Processing WaitForRoundResult report success: %t, timeout: %t",
			allRoundsSucceeded, timedOut)
		for i, r := range rl {
			if result, exists := rounds[r]; exists {
				results[i] = byte(result.Status)
			}
		}

		mdc.EventCallback(allRoundsSucceeded, timedOut, results)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond

	c.api.GetCmix().GetRoundResults(timeout, f, rl...)

	return nil
}
