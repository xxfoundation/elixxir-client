package bindings

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type roundsList struct {
	rounds []int
}

func (rl roundsList) Marshal() []byte {

}

func unmarshalRoundsList(marshaled []byte) []id.Round {

}

func makeRoundsList(rounds []id.Round) roundsList {

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
func (c *Client) WaitForMessageDelivery(marshaledSendReport []byte,
	mdc MessageDeliveryCallback, timeoutMS int) error {
	jww.INFO.Printf("WaitForMessageDelivery(%v, _, %v)",
		marshaledSendReport, timeoutMS)
	sr, err := UnmarshalSendReport(marshaledSendReport)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to "+
			"WaitForMessageDelivery callback due to bad Send Report: %+v", err))
	}

	if sr == nil || sr.rl == nil || len(sr.rl.list) == 0 {
		return errors.New(fmt.Sprintf("Failed to "+
			"WaitForMessageDelivery callback due to invalid Send Report "+
			"unmarshal: %s", string(marshaledSendReport)))
	}

	f := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundLookupStatus) {
		results := make([]byte, len(sr.rl.list))
		jww.INFO.Printf("Processing WaitForMessageDelivery report "+
			"for %v, success: %v, timedout: %v", sr.mid, allRoundsSucceeded,
			timedOut)
		for i, r := range sr.rl.list {
			if result, exists := rounds[r]; exists {
				results[i] = byte(result)
			}
		}

		mdc.EventCallback(sr.mid.Marshal(), allRoundsSucceeded, timedOut, results)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond

	err = c.api.GetRoundResults(sr.rl.list, timeout, f)

	return err
}
