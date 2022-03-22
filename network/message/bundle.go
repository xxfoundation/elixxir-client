///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"gitlab.com/elixxir/client/interfaces"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type Bundle struct {
	Round     id.Round
	RoundInfo *pb.RoundInfo
	Messages  []format.Message
	Finish    func()
	Identity  interfaces.Identity
}
