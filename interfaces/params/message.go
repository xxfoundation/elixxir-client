///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"encoding/json"
	"time"
)

type Messages struct {
	MessageReceptionBuffLen        uint
	MessageReceptionWorkerPoolSize uint
	MaxChecksGarbledMessage        uint
	GarbledMessageWait             time.Duration
}

func GetDefaultMessage() Messages {
	return Messages{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 4,
		MaxChecksGarbledMessage:        10,
		GarbledMessageWait:             15 * time.Minute,
	}
}

func (m *Messages) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Messages) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, m)
}
