////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"strconv"
)

// SentPartStatus represents the current status of an individual sent file part.
type SentPartStatus uint8

const (
	// UnsentPart is the status when a part has not been sent yet.
	UnsentPart SentPartStatus = iota

	// SentPart is the status when a part has been sent and hte round has
	// successfully completed, but the recipient has yet to receive it.
	SentPart

	// ReceivedPart is the status when a part has been sent and received.
	ReceivedPart

	// numSentStates is the number of sent part states.
	numSentStates
)

// stateMap prevents illegal state changes for file parts.
//
//            unsent  sent  received
//    unsent    ✗      ✓       ✓
//      sent    ✓      ✓       ✓
//  received    ✗      ✗       ✗
//
// Each cell determines if the state in the column can transition to the state
// in the top row. For example, a part can go from sent to unsent or sent to
// received but cannot change states once received.
var stateMap = [][]bool{
	{false, true, true},
	{true, true, true},
	{false, false, false},
}

// String returns a human-readable form of SentPartStatus for debugging and
// logging. This function adheres to the fmt.Stringer interface.
func (sps SentPartStatus) String() string {
	switch sps {
	case UnsentPart:
		return "unsent"
	case SentPart:
		return "sent"
	case ReceivedPart:
		return "received"
	default:
		return "INVALID STATUS: " + strconv.Itoa(int(sps))
	}
}
