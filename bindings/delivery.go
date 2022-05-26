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
type roundsList []int

func (rl roundsList) Marshal() ([]byte, error) {
	return json.Marshal(&rl)
}

func unmarshalRoundsList(marshaled []byte) ([]id.Round, error) {
	rl := roundsList{}
	err := json.Unmarshal(marshaled, &rl)
	if err != nil {
		return nil, err
	}

	realRl := make([]id.Round, len(rl))

	for _, rid := range rl {
		realRl = append(realRl, id.Round(rid))
	}

	return realRl, nil

}

func makeRoundsList(rounds []id.Round) roundsList {
	rl := make(roundsList, 0, len(rounds))
	for _, rid := range rounds {
		rl = append(rl, int(rid))
	}
	return rl
}

// MessageDeliveryCallback gets called on the determination if all events
// related to a message send were successful.
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
func (c *Client) WaitForMessageDelivery(roundList []byte,
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
