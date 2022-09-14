////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// The round events interface allows the registration of an event which triggers
// when a round reaches one or more states

type RoundEvents interface {
	// designates a callback to call on the specified event
	// rid is the id of the round the event occurs on
	// callback is the callback the event is triggered on
	// timeout is the amount of time before an error event is returned
	// valid states are the states which the event should trigger on
	AddRoundEvent(rid id.Round, callback ds.RoundEventCallback,
		timeout time.Duration, validStates ...states.Round) *ds.EventCallback

	// designates a go channel to signal the specified event
	// rid is the id of the round the event occurs on
	// eventChan is the channel the event is triggered on
	// timeout is the amount of time before an error event is returned
	// valid states are the states which the event should trigger on
	AddRoundEventChan(rid id.Round, eventChan chan ds.EventReturn,
		timeout time.Duration, validStates ...states.Round) *ds.EventCallback

	//Allows the un-registration of a round event before it triggers
	Remove(rid id.Round, e *ds.EventCallback)
}
