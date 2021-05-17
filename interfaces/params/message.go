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
	MaxChecksGarbledMessage        uint
	GarbledMessageWait             time.Duration
	// Use proxied (rather than direct) message sending
	ProxySending bool
}

func GetDefaultMessage() Messages {
	return Messages{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 4,
		MaxChecksGarbledMessage:        10,
		GarbledMessageWait:             15 * time.Minute,
		ProxySending:                   false,
	}
}
