package bindings

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Example marshalled roundList object:
// [1001,1003,1006]
type RoundsList struct {
	Rounds []int
}

func (rl RoundsList) Marshal() ([]byte, error) {
	return json.Marshal(&rl)
}

// unmarshalRoundsList accepts a marshalled E2ESendReport object & unmarshalls it into a RoundsList object, returning a list of id.Round
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

func makeRoundsList(rounds []id.Round) RoundsList {
	rl := RoundsList{}
	for _, rid := range rounds {
		rl.Rounds = append(rl.Rounds, int(rid))
	}
	return rl
}

// MessageDeliveryCallback gets called on the determination if all events
// related to a message send were successful.
// If delivered == true, timedOut == false && roundResults != nil
// If delivered == false, roundResults == nil
// If timedOut == true, delivered == false && roundResults == nil
type MessageDeliveryCallback interface {
	EventCallback(delivered, timedOut bool, roundResults []byte)
}

// WaitForMessageDelivery allows the caller to get notified if the rounds a
// message was sent in successfully completed. Under the hood, this uses an API
// which uses the internal round data, network historical round lookup, and
// waiting on network events to determine what has (or will) occur.
//
// The callbacks will return at timeoutMS if no state update occurs
//
// This function takes the marshaled send report to ensure a memory leak does
// not occur as a result of both sides of the bindings holding a reference to
// the same pointer.
func (c *Cmix) WaitForMessageDelivery(roundList []byte,
	mdc MessageDeliveryCallback, timeoutMS int) error {
	jww.INFO.Printf("WaitForMessageDelivery(%v, _, %v)",
		roundList, timeoutMS)
	rl, err := unmarshalRoundsList(roundList)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to "+
			"WaitForMessageDelivery callback due to bad Send Report: %+v", err))
	}

	if rl == nil || len(rl) == 0 {
		return errors.New(fmt.Sprintf("Failed to "+
			"WaitForMessageDelivery callback due to invalid Send Report "+
			"unmarshal: %s", string(roundList)))
	}

	f := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundResult) {
		results := make([]byte, len(rl))
		jww.INFO.Printf("Processing WaitForMessageDelivery report "+
			"success: %v, timedout: %v", allRoundsSucceeded,
			timedOut)
		for i, r := range rl {
			if result, exists := rounds[r]; exists {
				results[i] = byte(result.Status)
			}
		}

		mdc.EventCallback(allRoundsSucceeded, timedOut, results)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond

	err = c.api.GetCmix().GetRoundResults(timeout, f, rl...)

	return err
}
