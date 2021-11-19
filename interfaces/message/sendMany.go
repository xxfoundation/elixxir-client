////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// TargetedCmixMessage defines a recipient target pair in a sendMany cMix
// message.
type TargetedCmixMessage struct {
	Recipient *id.ID
	Message   format.Message
}
