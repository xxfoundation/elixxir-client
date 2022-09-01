////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type Bundle struct {
	Round     id.Round
	RoundInfo rounds.Round
	Messages  []format.Message
	Finish    func()
	Identity  receptionID.EphemeralIdentity
}
