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
	"time"
)

type Receive struct {
	ID          e2e.MessageID
	Payload     []byte
	MessageType Type
	Sender      *id.ID
	Timestamp   time.Time
	Encryption  EncryptionType
}
