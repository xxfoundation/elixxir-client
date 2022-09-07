////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receive

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/xx_network/primitives/id"
)

// ID to respond to any message type
const AnyType = catalog.NoType

//ID to respond to any user
func AnyUser() *id.ID {
	return &id.ZeroUser
}
