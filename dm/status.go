////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"strconv"
)

// Status represents the current status of a channel message.
type Status uint8

const (
	// Unsent is the status of a message when it is pending to be sent.
	Unsent Status = iota

	// Sent is the status of a message once the round it is sent
	// on completed.
	Sent

	// Received is the status of a message once is has been received.
	Received

	// Failed is the status of a message if it failed to send.
	Failed
)

// String returns a human-readable version of [SentStatus], used for debugging
// and logging. This function adheres to the [fmt.Stringer] interface.
func (ss Status) String() string {
	switch ss {
	case Unsent:
		return "unsent"
	case Sent:
		return "sent"
	case Received:
		return "received"
	case Failed:
		return "failed"
	default:
		return "Invalid SentStatus: " + strconv.Itoa(int(ss))
	}
}
