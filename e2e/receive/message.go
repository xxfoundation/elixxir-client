////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receive

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

type Message struct {
	MessageType catalog.MessageType
	ID          e2e.MessageID
	Payload     []byte

	Sender      *id.ID
	RecipientID *id.ID
	EphemeralID ephemeral.Id
	Timestamp   time.Time // Message timestamp of when the user sent

	Encrypted bool

	Round rounds.Round
}
