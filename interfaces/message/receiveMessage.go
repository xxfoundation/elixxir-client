///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

type Receive struct {
	ID             e2e.MessageID
	Payload        []byte
	MessageType    Type
	Sender         *id.ID
	RecipientID    *id.ID
	EphemeralID    ephemeral.Id
	RoundId        id.Round
	RoundTimestamp time.Time
	Timestamp      time.Time // Message timestamp of when the user sent
	Encryption     string
}
