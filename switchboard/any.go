///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package switchboard

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/xx_network/primitives/id"
)

// ID to respond to any message type
const AnyType = message.NoType

//ID to respond to any user
func AnyUser() *id.ID {
	return &id.ZeroUser
}
