////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import "strconv"

// SentStatus represents the current status of a channel message.
type SentStatus uint8

const (
	// Unsent is the status of a message when it is pending to be sent.
	Unsent SentStatus = 0

	// Sent is the status of a message once the round it is sent on completed.
	Sent SentStatus = 1

	// Delivered is the status of a message once is has been received.
	Delivered SentStatus = 2

	// Failed is the status of a message if it failed to send.
	Failed SentStatus = 3

	// SendProcessing is the status of a message when it has been added to the
	// event model, but it is being processes and is not yet ready to be sent.
	SendProcessing SentStatus = 4

	// SendProcessingComplete is the status of a message when it has been added
	// to the event model, is ready to be sent, but has not been sent yet.
	SendProcessingComplete SentStatus = 5

	// ReceptionProcessing is the status of a message when it has been received
	// and added to the event model, but it is being processes and is not yet
	// ready to be viewed.
	ReceptionProcessing SentStatus = 6

	// ReceptionProcessingComplete is the status of a message when it has been
	// received and processing is complete, but it has not been marked for
	// viewing yet.
	ReceptionProcessingComplete SentStatus = 7
)

// String returns a human-readable version of [SentStatus], used for debugging
// and logging. This function adheres to the [fmt.Stringer] interface.
func (ss SentStatus) String() string {
	switch ss {
	case Unsent:
		return "unsent"
	case Sent:
		return "sent"
	case Delivered:
		return "delivered"
	case Failed:
		return "failed"
	case SendProcessing:
		return "processing (send)"
	case SendProcessingComplete:
		return "processing complete (send)"
	case ReceptionProcessing:
		return "processing (receive)"
	case ReceptionProcessingComplete:
		return "processing complete (receive)"
	default:
		return "Invalid SentStatus: " + strconv.Itoa(int(ss))
	}
}
