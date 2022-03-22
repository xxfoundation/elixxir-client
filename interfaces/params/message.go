///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"time"
)

type Messages struct {
	MessageReceptionBuffLen        uint
	MessageReceptionWorkerPoolSize uint
	MaxChecksInProcessMessage      uint
	InProcessMessageWait           time.Duration
	RealtimeOnly                   bool
}

func GetDefaultMessage() Messages {
	return Messages{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 4,
		MaxChecksInProcessMessage:      10,
		InProcessMessageWait:           15 * time.Minute,
		RealtimeOnly:                   false,
	}
}
